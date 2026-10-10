package mask

import (
	"testing"
)

func TestMask(t *testing.T) {
	// Set HOME/USER to known values for consistent masking tests
	t.Setenv("HOME", "/home/testuser")
	t.Setenv("USER", "testuser")
	RefreshMaskingVars()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory path",
			input:    "/home/testuser/some/path",
			expected: "~/some/path",
		},
		{
			name:     "home directory with username",
			input:    "/home/testuser/project/main.go",
			expected: "~/project/main.go",
		},
		{
			name:     "username in path",
			input:    "error in /home/testuser/src/file.go",
			expected: "error in ~/src/file.go",
		},
		{
			name:     "key=value pattern",
			input:    "api_key=sk-abcdef123456",
			expected: "api_key=***",
		},
		{
			name:     "token=value pattern",
			input:    "token=ghp_xyz789",
			expected: "token=***",
		},
		{
			name:     "password=value pattern",
			input:    "password=secret123",
			expected: "password=***",
		},
		{
			name:     "secret=value pattern",
			input:    "secret=mysecret",
			expected: "secret=***",
		},
		{
			name:     "passwd=value pattern",
			input:    "passwd=dbpass",
			expected: "passwd=***",
		},
		{
			name:     "Bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: Bearer ***",
		},
		{
			name:     "Bearer with space",
			input:    "Bearer sk-1234567890abcdef",
			expected: "Bearer ***",
		},
		{
			name:     "quoted key value",
			input:    `api_key="sk-abcdef123456"`,
			expected: `api_key="***"`,
		},
		{
			name:     "json style key value",
			input:    `"api_key": "sk-abcdef123456"`,
			expected: `"api_key": "***"`,
		},
		{
			name:     "colon separator",
			input:    "token: abc123",
			expected: "token: ***",
		},
		{
			name:     "multiple secrets in one line",
			input:    "api_key=key1 token=key2 password=key3",
			expected: "api_key=*** token=*** password=***",
		},
		{
			name:     "keyword should not be masked",
			input:    "keyword=value",
			expected: "keyword=value",
		},
		{
			name:     "username masking with word boundaries",
			input:    "user testuser ran command",
			expected: "user <user> ran command",
		},
		{
			name:     "username not masked if too short",
			input:    "user ab ran command",
			expected: "user ab ran command",
		},
		{
			name:     "username not masked in middle of word",
			input:    "supertestuser ran command",
			expected: "supertestuser ran command",
		},
		{
			name:     "no secrets present",
			input:    "regular error message without secrets",
			expected: "regular error message without secrets",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "complex real-world error",
			input:    `Error: failed to authenticate: api_key=sk-live_abc123 token=ghp_xyz789 path=/home/testuser/project`,
			expected: `Error: failed to authenticate: api_key=*** token=*** path=~/project`,
		},
		{
			name:     "Bearer in header format",
			input:    `curl -H "Authorization: Bearer sk-test123" https://api.example.com`,
			expected: `curl -H "Authorization: Bearer ***" https://api.example.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Mask(tt.input)
			if result != tt.expected {
				t.Errorf("Mask(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			// Ensure no original secret survives
			if tt.input != tt.expected {
				for _, secret := range []string{"sk-", "ghp_", "sk-live_", "sk-test"} {
					if containsSecret(tt.input, secret) && containsSecret(result, secret) {
						t.Errorf("secret %q still present in masked output: %q", secret, result)
					}
				}
			}
		})
	}
}

func containsSecret(s, secret string) bool {
	return len(s) >= len(secret) && (s == secret || len(s) > len(secret) && (s[:len(secret)] == secret || s[len(s)-len(secret):] == secret || findSubstring(s, secret)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
