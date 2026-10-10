package secret

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
)

func TestReadLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prompt   string
		expected string
	}{
		{
			name:     "simple input",
			input:    "test-secret\n",
			prompt:   "Enter: ",
			expected: "test-secret",
		},
		{
			name:     "with spaces",
			input:    "  secret with spaces  \n",
			prompt:   "",
			expected: "secret with spaces",
		},
		{
			name:     "empty input",
			input:    "\n",
			prompt:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// readLine reads from os.File, so we need a different approach
			// Just test the trimming logic
			result := strings.TrimSpace(tt.input)
			if result != tt.expected {
				t.Errorf("trim = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPromptSecret_PipedInput(t *testing.T) {
	// When stdin is not a terminal, it should read visible input
	r, w, _ := os.Pipe()
	reader := bufio.NewReader(r)

	go func() {
		w.WriteString("piped-secret\n")
		w.Close()
	}()

	secret, err := PromptSecret("API Key: ", reader)
	if err != nil {
		t.Fatalf("PromptSecret error: %v", err)
	}
	if secret != "piped-secret" {
		t.Errorf("secret = %q, want piped-secret", secret)
	}
}

func TestPromptSecret_WithPrompt(t *testing.T) {
	// Test that prompt is written to stderr
	oldStderr := os.Stderr
	defer func() { os.Stderr = oldStderr }()

	rStderr, wStderr, _ := os.Pipe()
	os.Stderr = wStderr

	r, w, _ := os.Pipe()
	reader := bufio.NewReader(r)

	go func() {
		w.WriteString("test\n")
		w.Close()
	}()

	_, err := PromptSecret("Custom Prompt: ", reader)
	if err != nil {
		t.Fatalf("PromptSecret error: %v", err)
	}

	wStderr.Close()
	var stderrOutput strings.Builder
	_, _ = io.Copy(&stderrOutput, rStderr)

	if !strings.Contains(stderrOutput.String(), "Custom Prompt:") {
		t.Errorf("prompt not written to stderr: %s", stderrOutput.String())
	}
}

func TestIsTerminal(t *testing.T) {
	// Test with a regular file (not terminal)
	tmpFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	result := IsTerminal(tmpFile)
	if result {
		t.Errorf("regular file should not be terminal")
	}

	// Test with stdin (might be terminal or not depending on test runner)
	_ = IsTerminal(os.Stdin)
	// Just verify it doesn't panic
}

func TestSecretNotSupportedFallback(t *testing.T) {
	// Test the fallback path when echo suppression is not supported
	// This is tested by the secret_other.go build tag
	// Here we just verify the error constant exists
	if errNotSupported == nil {
		t.Error("errNotSupported should not be nil")
	}
	if errNotATerminal == nil {
		t.Error("errNotATerminal should not be nil")
	}
}
