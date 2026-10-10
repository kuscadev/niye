package render

import (
	"strings"
	"testing"

	"github.com/kuscadev/niye/internal/provider"
)

func TestRenderOutput(t *testing.T) {
	tests := []struct {
		name     string
		ans      provider.Answer
		lang     string
		expected string
	}{
		{
			name:     "Turkish basic",
			ans:      provider.Answer{Reason: "Geçersiz git komutu.", Fix: "git push"},
			lang:     "tr",
			expected: "● Neden: Geçersiz git komutu.\n● Çözüm:\n    git push",
		},
		{
			name:     "English basic",
			ans:      provider.Answer{Reason: "Invalid git command.", Fix: "git push"},
			lang:     "en",
			expected: "● Cause: Invalid git command.\n● Fix:\n    git push",
		},
		{
			name:     "Turkish multiline fix",
			ans:      provider.Answer{Reason: "Hata oluştu.", Fix: "git add .\ngit commit -m 'fix'\ngit push"},
			lang:     "tr",
			expected: "● Neden: Hata oluştu.\n● Çözüm:\n    git add .\n    git commit -m 'fix'\n    git push",
		},
		{
			name:     "Reason truncated to one sentence",
			ans:      provider.Answer{Reason: "First sentence. Second sentence.", Fix: "cmd"},
			lang:     "tr",
			expected: "● Neden: First sentence.\n● Çözüm:\n    cmd",
		},
		{
			name:     "Reason with exclamation",
			ans:      provider.Answer{Reason: "Error! Another sentence.", Fix: "cmd"},
			lang:     "tr",
			expected: "● Neden: Error!\n● Çözüm:\n    cmd",
		},
		{
			name:     "Reason with question mark",
			ans:      provider.Answer{Reason: "What happened? Another.", Fix: "cmd"},
			lang:     "tr",
			expected: "● Neden: What happened?\n● Çözüm:\n    cmd",
		},
		{
			name:     "Empty reason fallback",
			ans:      provider.Answer{Reason: "", Fix: "cmd"},
			lang:     "tr",
			expected: "● Neden: Açıklama alınamadı.\n● Çözüm:\n    cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderOutput(tt.ans, tt.lang)
			if output != tt.expected {
				t.Errorf("RenderOutput() = %q\nwant %q", output, tt.expected)
			}
		})
	}
}

func TestRenderOutput_DangerousCommands(t *testing.T) {
	dangerousCommands := []struct {
		name string
		cmd  string
	}{
		{"rm -rf /", "rm -rf /"},
		{"rm -rf /home/user", "rm -rf /home/user"},
		{"sudo chmod -R 777 /", "sudo chmod -R 777 /"},
		{"mkfs.ext4 /dev/sda1", "mkfs.ext4 /dev/sda1"},
		{"dd of=/dev/sda", "dd of=/dev/sda"},
		{"curl ... | bash", "curl https://example.com/script.sh | bash"},
		{"wget ... | sh", "wget -qO- https://example.com | sh"},
		{"pipe to bash at end", "some command | bash"},
	}

	for _, tc := range dangerousCommands {
		t.Run(tc.name, func(t *testing.T) {
			ans := provider.Answer{Reason: "Test.", Fix: tc.cmd}
			output := RenderOutput(ans, "tr")

			// Should contain warning
			if !strings.Contains(output, "Uyarı") && !strings.Contains(output, "Warning") {
				t.Errorf("missing warning for dangerous command %q: %s", tc.cmd, output)
			}
			// Should contain the command
			if !strings.Contains(output, tc.cmd) {
				t.Errorf("missing command in output: %s", output)
			}
		})
	}
}

func TestRenderOutput_NonDangerousCommands(t *testing.T) {
	safeCommands := []string{
		"git push",
		"git commit -m 'fix'",
		"rm file.txt",
		"chmod 644 file.txt",
		"ls -la",
		"curl https://api.example.com",
	}

	for _, cmd := range safeCommands {
		t.Run(cmd, func(t *testing.T) {
			ans := provider.Answer{Reason: "Test.", Fix: cmd}
			output := RenderOutput(ans, "tr")

			// Should NOT contain warning
			if strings.Contains(output, "Uyarı") || strings.Contains(output, "Warning") {
				t.Errorf("unexpected warning for safe command %q: %s", cmd, output)
			}
		})
	}
}

func TestTruncateToOneSentence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Single sentence.", "Single sentence."},
		{"First. Second.", "First."},
		{"First! Second?", "First!"},
		{"First? Second.", "First?"},
		{"No punctuation", "No punctuation"},
		{"", ""},
		{"Multiple. Sentences. Here.", "Multiple."},
		{"  Spaces  .  Next.  ", "Spaces  ."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateToOneSentence(tt.input)
			if result != tt.expected {
				t.Errorf("truncateToOneSentence(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsDangerous(t *testing.T) {
	tests := []struct {
		cmd     string
		want    bool
		comment string
	}{
		{"rm -rf /", true, "rm -rf root"},
		{"rm -rf /home/user", true, "rm -rf home"},
		{"sudo chmod -R 777 /", true, "sudo chmod -R 777"},
		{"mkfs.ext4 /dev/sda1", true, "mkfs"},
		{"dd of=/dev/sda", true, "dd of="},
		{"curl https://example.com | bash", true, "curl | bash"},
		{"wget -qO- https://example.com | sh", true, "wget | sh"},
		{"command | bash", true, "pipe to bash"},
		{"command | sh", true, "pipe to sh"},
		{"git push", false, "git push"},
		{"rm file.txt", false, "rm single file"},
		{"chmod 644 file.txt", false, "chmod single file"},
		{"curl https://api.example.com", false, "curl without pipe"},
		{"echo 'rm -rf /'", false, "rm -rf in quotes"},
	}

	for _, tt := range tests {
		t.Run(tt.comment, func(t *testing.T) {
			result := isDangerous(tt.cmd)
			if result != tt.want {
				t.Errorf("isDangerous(%q) = %v, want %v", tt.cmd, result, tt.want)
			}
		})
	}
}
