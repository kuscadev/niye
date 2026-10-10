package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const leakTestKey = "TESTKEY-SECRET-12345"

// The native Gemini adapter must send the API key in the
// x-goog-api-key header, never as a ?key= URL query parameter.
func TestGeminiProvider_UsesAPIKeyHeader(t *testing.T) {
	var gotHeader, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-goog-api-key")
		gotQuery = r.URL.Query().Get("key")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"reason\": \"R.\", \"fix\": \"cmd\"}"}]}}]}`))
	}))
	defer server.Close()

	p := NewGeminiProvider(server.URL, leakTestKey, "test-model", 5*time.Second)
	if _, err := p.Ask(context.Background(), Input{ErrorText: "boom"}); err != nil {
		t.Fatalf("Ask error: %v", err)
	}
	if gotHeader != leakTestKey {
		t.Errorf("x-goog-api-key header = %q, want %q", gotHeader, leakTestKey)
	}
	if gotQuery != "" {
		t.Errorf("URL ?key= param = %q, want empty (key must not be in URL)", gotQuery)
	}
}

// No error message from either adapter may contain the API key:
// connection refused, timeout, 4xx/5xx, malformed response.
func TestProviders_DoNotLeakAPIKeyInErrors(t *testing.T) {
	newAdapters := func(baseURL string, client *http.Client) map[string]Provider {
		return map[string]Provider{
			"openai": &openAIProvider{baseURL: baseURL, apiKey: leakTestKey, model: "m", client: client},
			// Native Gemini URLs end in ":generateContent"; Ask() leaves
			// such URLs untouched, so append a model path here to mirror
			// NewGeminiProvider's real behavior with custom base URLs.
			"gemini": &geminiProvider{baseURL: baseURL + "/models/m", apiKey: leakTestKey, model: "m", client: client},
		}
	}

	t.Run("connection refused", func(t *testing.T) {
		for name, p := range newAdapters("http://localhost:9999", &http.Client{Timeout: 5 * time.Second}) {
			_, err := p.Ask(context.Background(), Input{ErrorText: "boom"})
			if err == nil {
				t.Fatalf("%s: expected connection error", name)
			}
			if strings.Contains(err.Error(), leakTestKey) {
				t.Errorf("%s: error leaks API key: %v", name, err)
			}
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		for name, p := range newAdapters(server.URL, &http.Client{Timeout: 50 * time.Millisecond}) {
			_, err := p.Ask(context.Background(), Input{ErrorText: "boom"})
			if err == nil {
				t.Fatalf("%s: expected timeout error", name)
			}
			if strings.Contains(err.Error(), leakTestKey) {
				t.Errorf("%s: error leaks API key: %v", name, err)
			}
		}
	})

	t.Run("http errors", func(t *testing.T) {
		for _, code := range []int{400, 401, 403, 404, 429, 500} {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			for name, p := range newAdapters(server.URL, &http.Client{Timeout: 5 * time.Second}) {
				_, err := p.Ask(context.Background(), Input{ErrorText: "boom"})
				if err == nil {
					t.Fatalf("%s [%d]: expected HTTP error", name, code)
				}
				if strings.Contains(err.Error(), leakTestKey) {
					t.Errorf("%s [%d]: error leaks API key: %v", name, code, err)
				}
			}
			server.Close()
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{not json at all`))
		}))
		defer server.Close()
		for name, p := range newAdapters(server.URL, &http.Client{Timeout: 5 * time.Second}) {
			_, err := p.Ask(context.Background(), Input{ErrorText: "boom"})
			if err == nil {
				t.Fatalf("%s: expected parse error", name)
			}
			if strings.Contains(err.Error(), leakTestKey) {
				t.Errorf("%s: error leaks API key: %v", name, err)
			}
		}
	})

	t.Run("provider error payload", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// OpenAI-style error envelope; both adapters must surface it without the key.
			w.Write([]byte(`{"error":{"message":"bad request","type":"x","code":"y"}}`))
		}))
		defer server.Close()
		for name, p := range newAdapters(server.URL, &http.Client{Timeout: 5 * time.Second}) {
			_, err := p.Ask(context.Background(), Input{ErrorText: "boom"})
			if err == nil {
				t.Fatalf("%s: expected envelope error", name)
			}
			if strings.Contains(err.Error(), leakTestKey) {
				t.Errorf("%s: error leaks API key: %v", name, err)
			}
		}
	})
}
