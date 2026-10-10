package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kuscadev/niye/internal/mask"
	"github.com/kuscadev/niye/internal/provider"
	"github.com/kuscadev/niye/internal/secret"
)

func TestRun_Help(t *testing.T) {
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--help"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stderr.String(), "Kullanım") && !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("help output missing: %s", stderr.String())
	}
}

func TestRun_Version(t *testing.T) {
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--version"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "niye") {
		t.Errorf("version output missing: %s", stdout.String())
	}
}

func TestRun_EmptyInput(t *testing.T) {
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "boş hata girişi") && !strings.Contains(stderr.String(), "empty error input") {
		t.Errorf("error message missing: %s", stderr.String())
	}
}

func TestRun_ArgumentInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "Test reason", "fix": "test fix"}`}},
			},
		})
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL, "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "● Neden:") && !strings.Contains(output, "● Cause:") {
		t.Errorf("missing reason label: %s", output)
	}
	if !strings.Contains(output, "● Çözüm:") && !strings.Contains(output, "● Fix:") {
		t.Errorf("missing fix label: %s", output)
	}
	if !strings.Contains(output, "test fix") {
		t.Errorf("missing fix content: %s", output)
	}
}

func TestRun_StdinInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "Stdin reason", "fix": "stdin fix"}`}},
			},
		})
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL}, strings.NewReader("stdin error"), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "Stdin reason") {
		t.Errorf("missing stdin reason: %s", output)
	}
	if !strings.Contains(output, "stdin fix") {
		t.Errorf("missing stdin fix: %s", output)
	}
}

func TestRun_MissingAPIKey(t *testing.T) {
	var stderr, stdout bytes.Buffer
	// No API key provided - ensure env vars are not set
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("NIYE_API_KEY", "")
	exitCode := run([]string{"niye", "--provider", "openai", "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1, stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "API anahtarı") && !strings.Contains(stderr.String(), "API key") {
		t.Errorf("missing API key error: %s", stderr.String())
	}
}

func TestRun_InvalidProvider(t *testing.T) {
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--provider", "invalid", "--api-key", "test", "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "geçersiz provider") && !strings.Contains(stderr.String(), "invalid provider") {
		t.Errorf("invalid provider error: %s", stderr.String())
	}
}

func TestRun_InvalidLang(t *testing.T) {
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--lang", "fr", "--api-key", "test", "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "geçersiz dil") && !strings.Contains(stderr.String(), "invalid language") {
		t.Errorf("invalid lang error: %s", stderr.String())
	}
}

func TestRun_ProviderError_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "bad", "--base-url", server.URL, "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "yetkisiz") && !strings.Contains(stderr.String(), "unauthorized") {
		t.Errorf("unauthorized error: %s", stderr.String())
	}
}

func TestRun_ProviderError_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL, "--model", "bad-model", "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "bulunamadı") && !strings.Contains(stderr.String(), "not found") {
		t.Errorf("model not found error: %s", stderr.String())
	}
}

func TestRun_ProviderError_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL, "--timeout", "10ms", "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "zaman aşımı") && !strings.Contains(stderr.String(), "timed out") && !strings.Contains(stderr.String(), "timeout") {
		t.Errorf("timeout error: %s", stderr.String())
	}
}

func TestRun_EnglishOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "Test reason.", "fix": "test fix"}`}},
			},
		})
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--lang", "en", "--api-key", "test", "--base-url", server.URL, "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "● Cause:") {
		t.Errorf("missing Cause label: %s", output)
	}
	if !strings.Contains(output, "● Fix:") {
		t.Errorf("missing Fix label: %s", output)
	}
}

func TestRun_Init_NoTTY(t *testing.T) {
	// Simulate piped input for init
	// Prompts: Provider, Base URL, Model, API Key, Language, Timeout, Store key
	// Input: openai, (empty), (empty), (empty for api key), (empty), (empty), n
	input := "openai\n\n\n\n\nn\n"
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "init"}, strings.NewReader(input), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}
	// Should not write key to config when 'n' is answered
	// Config file should be created without api_key
}

func TestRun_Init_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.toml")

	input := "gemini\nhttps://custom.api\ngemini-model\ntest-key\nen\n30s\ny\n"
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "init", "--config", configPath}, strings.NewReader(input), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}

	// Verify config was written
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `provider = "gemini"`) {
		t.Errorf("config missing provider: %s", content)
	}
	if !strings.Contains(content, `api_key = "test-key"`) {
		t.Errorf("config missing api_key: %s", content)
	}
}

func TestRun_Init_DontStoreKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Answer 'n' to store key prompt
	// Prompts: Provider, Base URL, Model, API Key, Language, Timeout, Store key
	// Input: openai, (empty), (empty), test-key, (empty), (empty), n
	input := "openai\n\n\ntest-key\n\n\nn\n"
	secret.ForceTerminalForTesting = true
	defer func() { secret.ForceTerminalForTesting = false }()
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "init", "--config", configPath}, strings.NewReader(input), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}

	// Verify config was written WITHOUT api_key
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "api_key") {
		t.Errorf("config should not have api_key: %s", content)
	}
	if !strings.Contains(stderr.String(), "diske yazılmadı") && !strings.Contains(stderr.String(), "not written to disk") {
		t.Errorf("missing key not stored message: %s", stderr.String())
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"content": `{"reason": "Test", "fix": "fix"}`}},
				},
			})
		}
	}))
	defer server.Close()

	var stderr, stdout bytes.Buffer
	// Very short timeout
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL, "--timeout", "1ms", "test error"}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 2 {
		t.Errorf("exit code = %d, want 2", exitCode)
	}
}

func TestRun_MaskingApplied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		messages := req["messages"].([]any)
		userMsg := messages[1].(map[string]any)["content"].(string)

		// Verify masking was applied
		if strings.Contains(userMsg, "sk-real-key") {
			t.Errorf("API key not masked in request: %s", userMsg)
		}
		if strings.Contains(userMsg, "/home/realuser") {
			t.Errorf("home path not masked in request: %s", userMsg)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "OK", "fix": "cmd"}`}},
			},
		})
	}))
	defer server.Close()

	// Set HOME to a known value for testing
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USER", "testuser")
	// Refresh masking vars to pick up the test env
	mask.RefreshMaskingVars()

	errorText := `Error: api_key=sk-real-key path=` + tmpDir + `/project`
	var stderr, stdout bytes.Buffer
	exitCode := run([]string{"niye", "--api-key", "test", "--base-url", server.URL, errorText}, strings.NewReader(""), &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0, stderr: %s", exitCode, stderr.String())
	}
}

func TestRun_ProviderInterface(t *testing.T) {
	// Test that Provider interface works with both providers
	ctx := context.Background()

	// OpenAI provider
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"reason": "OpenAI", "fix": "fix1"}`}},
			},
		})
	}))
	defer server1.Close()

	p1 := provider.NewOpenAIProvider(server1.URL, "key", "model", 5*time.Second)
	ans1, err := p1.Ask(ctx, provider.Input{ErrorText: "test"})
	if err != nil {
		t.Fatalf("OpenAI Ask error: %v", err)
	}
	if ans1.Reason != "OpenAI" {
		t.Errorf("OpenAI reason = %q", ans1.Reason)
	}

	// Gemini provider
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]any{{"text": `{"reason": "Gemini", "fix": "fix2"}`}}}},
			},
		})
	}))
	defer server2.Close()

	p2 := provider.NewGeminiProvider(server2.URL, "key", "gemini-1.5-flash", 5*time.Second)
	ans2, err := p2.Ask(ctx, provider.Input{ErrorText: "test"})
	if err != nil {
		t.Fatalf("Gemini Ask error: %v", err)
	}
	if ans2.Reason != "Gemini" {
		t.Errorf("Gemini reason = %q", ans2.Reason)
	}
}
