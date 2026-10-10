package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExtractAnswer(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    Answer
		wantErr bool
	}{
		{
			name: "valid JSON response",
			body: `{"reason": "Invalid git command", "fix": "git push"}`,
			want: Answer{Reason: "Invalid git command", Fix: "git push"},
		},
		{
			name: "JSON with neden/cozum keys (Turkish)",
			body: `{"neden": "Geçersiz git komutu", "cozum": "git push"}`,
			want: Answer{Reason: "Geçersiz git komutu", Fix: "git push"},
		},
		{
			name: "JSON with cause/solution keys",
			body: `{"cause": "Command not found", "solution": "git push"}`,
			want: Answer{Reason: "Command not found", Fix: "git push"},
		},
		{
			name: "JSON with command key",
			body: `{"reason": "Error", "command": "git push"}`,
			want: Answer{Reason: "Error", Fix: "git push"},
		},
		{
			name: "JSON with extra whitespace",
			body: `  { "reason" : "Error" , "fix" : "git push" }  `,
			want: Answer{Reason: "Error", Fix: "git push"},
		},
		{
			name: "JSON with newlines",
			body: `{"reason": "Error", "fix": "git push\nwith newline"}`,
			want: Answer{Reason: "Error", Fix: "git push\nwith newline"},
		},
		{
			name: "malformed JSON - fallback to plain text with Neden/Çözüm",
			body: `Neden: Invalid command
Çözüm: git push`,
			want: Answer{Reason: "Invalid command", Fix: "git push"},
		},
		{
			name: "malformed JSON - fallback to plain text with Reason/Fix",
			body: `Reason: Command not found
Fix: git push`,
			want: Answer{Reason: "Command not found", Fix: "git push"},
		},
		{
			name: "malformed JSON - fallback to fenced code block",
			body: "The issue is a typo.\n```bash\ngit push\n```",
			want: Answer{Reason: "The issue is a typo.", Fix: "git push"},
		},
		{
			name: "malformed JSON - first line reason, rest fix",
			body: `Invalid git command
git push
git push --force`,
			want: Answer{Reason: "Invalid git command", Fix: "git push\ngit push --force"},
		},
		{
			name: "JSON with balanced braces in fix",
			body: `{"reason": "Error", "fix": "echo '{\"key\": \"value\"}'"}`,
			want: Answer{Reason: "Error", Fix: `echo '{"key": "value"}'`},
		},
		{
			name: "empty reason fallback",
			body: `{"reason": "", "fix": "git push"}`,
			want: Answer{Reason: "Açıklama alınamadı.", Fix: "git push"},
		},
		{
			name: "strips dollar prompt from fix",
			body: `{"reason": "Error", "fix": "$ git push\n$ git status"}`,
			want: Answer{Reason: "Error", Fix: "git push\ngit status"},
		},
		{
			name: "strips backticks from fix",
			body: `{"reason": "Error", "fix": "` + "`git push`" + `"}`,
			want: Answer{Reason: "Error", Fix: "git push"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractAnswer(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Reason != tt.want.Reason {
				t.Errorf("extractAnswer() reason = %q, want %q", got.Reason, tt.want.Reason)
			}
			if got.Fix != tt.want.Fix {
				t.Errorf("extractAnswer() fix = %q, want %q", got.Fix, tt.want.Fix)
			}
		})
	}
}

func TestOpenAIProvider_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong auth header")
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "Test reason", "fix": "test fix"}`}},
			},
		})
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "test-key",
		model:   "test-model",
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	ans, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err != nil {
		t.Fatalf("ask() error = %v", err)
	}
	if ans.Reason != "Test reason" || ans.Fix != "test fix" {
		t.Errorf("ask() = %+v, want {Reason:Test reason, Fix:test fix}", ans)
	}
}

func TestOpenAIProvider_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "bad-key",
		model:   "test-model",
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	_, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "yetkisiz") && !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("error should mention 401/unauthorized: %v", err)
	}
}

func TestOpenAIProvider_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "model not found"})
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "test-key",
		model:   "nonexistent-model",
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	_, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "model") && !strings.Contains(err.Error(), "bulunamadı") {
		t.Errorf("error should mention 404/model not found: %v", err)
	}
}

func TestOpenAIProvider_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "test-key",
		model:   "test-model",
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	_, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "429") && !strings.Contains(err.Error(), "rate") && !strings.Contains(err.Error(), "limit") {
		t.Errorf("error should mention 429/rate limit: %v", err)
	}
}

func TestOpenAIProvider_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `not valid json`}},
			},
		})
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "test-key",
		model:   "test-model",
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	// Should fall back to plain text extraction
	ans, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err != nil {
		t.Fatalf("ask() error = %v", err)
	}
	if ans.Reason == "" {
		t.Errorf("should have extracted reason from plain text")
	}
}

func TestOpenAIProvider_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "test-key",
		model:   "test-model",
		client:  &http.Client{Timeout: 100 * time.Millisecond},
	}

	_, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "zaman") && !strings.Contains(err.Error(), "context deadline") {
		t.Errorf("error should mention timeout: %v", err)
	}
}

func TestOpenAIProvider_ConnectionRefused(t *testing.T) {
	p := &openAIProvider{
		baseURL: "http://localhost:9999", // unlikely to be open
		apiKey:  "test-key",
		model:   "test-model",
		client:  &http.Client{Timeout: 100 * time.Millisecond},
	}

	_, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "bağlantı") && !strings.Contains(err.Error(), "dial") && !strings.Contains(err.Error(), "refused") {
		t.Errorf("error should mention connection failure: %v", err)
	}
}

func TestOpenAIProvider_OllamaNoAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ollama doesn't require auth
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Ollama should not require auth header")
		}
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "Ollama reason", "fix": "ollama fix"}`}},
			},
		})
	}))
	defer server.Close()

	p := &openAIProvider{
		baseURL: server.URL,
		apiKey:  "", // No auth for Ollama
		model:   "llama3",
		client:  &http.Client{Timeout: 5 * time.Second},
	}

	ans, err := p.Ask(context.Background(), Input{ErrorText: "test error"})
	if err != nil {
		t.Fatalf("ask() error = %v", err)
	}
	if ans.Reason != "Ollama reason" || ans.Fix != "ollama fix" {
		t.Errorf("ask() = %+v, want {Reason:Ollama reason, Fix:ollama fix}", ans)
	}
}
