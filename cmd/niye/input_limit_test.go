package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kuscadev/niye/internal/secret"
)

// Inputs larger than 32 KB must be truncated (head + tail kept, middle
// dropped) before reaching the model, with a one-line stderr note.
func TestRun_InputTruncated(t *testing.T) {
	for _, lang := range []string{"tr", "en"} {
		t.Run(lang, func(t *testing.T) {
			var gotUserText string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("decode request: %v", err)
				}
				msgs := req["messages"].([]any)
				gotUserText = msgs[1].(map[string]any)["content"].(string)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"choices":[{"message":{"content":"{\"reason\": \"R.\", \"fix\": \"cmd\"}"}}]}`))
			}))
			defer server.Close()

			head := "HEAD-MARKER-" + strings.Repeat("a", 100)
			tail := "TAIL-MARKER-" + strings.Repeat("z", 100)
			big := head + strings.Repeat("m", 50*1024) + "MIDDLE-UNIQUE-SENTINEL-12345" + strings.Repeat("m", 50*1024) + tail

			var stderr, stdout bytes.Buffer
			exitCode := run([]string{"niye", "--lang", lang, "--api-key", "test", "--base-url", server.URL, big},
				strings.NewReader(""), &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
			}
			if len(gotUserText) > 32*1024 {
				t.Errorf("model input = %d bytes, want at most %d", len(gotUserText), 32*1024)
			}
			if !strings.Contains(gotUserText, "HEAD-MARKER-") {
				t.Errorf("head part missing from truncated input")
			}
			if !strings.Contains(gotUserText, "TAIL-MARKER-") {
				t.Errorf("tail part missing from truncated input")
			}
			if strings.Contains(gotUserText, "MIDDLE-UNIQUE-SENTINEL-12345") {
				t.Errorf("middle part was not dropped")
			}
			note := stderr.String()
			if lang == "tr" && !strings.Contains(note, "32") {
				t.Errorf("missing Turkish truncation note in stderr: %q", note)
			}
			if lang == "en" && !strings.Contains(note, "32") {
				t.Errorf("missing English truncation note in stderr: %q", note)
			}
			if strings.Count(note, "\n") > 2 {
				t.Errorf("truncation note must be a single line, got: %q", note)
			}
		})
	}
}

// Small inputs must pass through untouched with no truncation note.
func TestRun_InputNotTruncated(t *testing.T) {
	var gotUserText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		msgs := req["messages"].([]any)
		gotUserText = msgs[1].(map[string]any)["content"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"reason\": \"R.\", \"fix\": \"cmd\"}"}}]}`))
	}))
	defer server.Close()

	small := "small error: something failed"
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL, small},
		strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}
	if gotUserText != small {
		t.Errorf("small input modified: got %q, want %q", gotUserText, small)
	}
	if strings.Contains(stderr.String(), "32") {
		t.Errorf("unexpected truncation note for small input: %q", stderr.String())
	}
}

// No args + stdin is a terminal: print usage and exit 1 without blocking.
func TestRun_TerminalNoArgs_PrintsUsage(t *testing.T) {
	secret.ForceTerminalForTesting = true
	defer func() { secret.ForceTerminalForTesting = false }()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "Kullanım") && !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("usage message missing in stderr: %q", stderr.String())
	}
}
