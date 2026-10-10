package mask

import (
	"testing"
)

func TestMask_ExtraSecrets(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("USER", "testuser")
	RefreshMaskingVars()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standalone sk key",
			input:    "failed with key sk-abcdef1234567890 end",
			expected: "failed with key *** end",
		},
		{
			name:     "sk live key",
			input:    "token sk-live_abc123XYZ7890 here",
			expected: "token *** here",
		},
		{
			name:     "github ghp token",
			input:    "push with ghp_abcdefgh1234567890ABC",
			expected: "push with ***",
		},
		{
			name:     "github gho token",
			input:    "oauth gho_abcdefgh1234567890ABC done",
			expected: "oauth *** done",
		},
		{
			name:     "github fine-grained pat",
			input:    "use github_pat_abcDEF1234567890_xyz end",
			expected: "use *** end",
		},
		{
			name:     "aws access key id",
			input:    "aws key AKIAIOSFODNN7EXAMPLE here",
			expected: "aws key *** here",
		},
		{
			name:     "jwt three parts",
			input:    "auth eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0In0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c done",
			expected: "auth *** done",
		},
		{
			name:     "url userinfo",
			input:    "failed https://admin:s3cret@api.example.com/v1 end",
			expected: "failed https://***@api.example.com/v1 end",
		},
		{
			name:     "url userinfo user only",
			input:    "clone http://deployer@git.example.com/repo end",
			expected: "clone http://***@git.example.com/repo end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mask(tt.input); got != tt.expected {
				t.Errorf("Mask(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMask_Negatives(t *testing.T) {
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("USER", "testuser")
	RefreshMaskingVars()

	// These must pass through untouched.
	inputs := []string{
		"keyword=value",
		"unexpected token: }",
		"token: }",
		"plain user@example.com email",
		"https://example.com/no-auth-here",
		"not.a.jwt two parts only",
		"short sk-12 not a key",
		"version 1.2.3 released",
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			if got := Mask(in); got != in {
				t.Errorf("Mask(%q) = %q, want unchanged", in, got)
			}
		})
	}
}
