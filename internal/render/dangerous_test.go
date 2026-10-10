package render

import (
	"strings"
	"testing"

	"github.com/kuscadev/niye/internal/provider"
)

// Extended dangerous-command coverage. Format contract (unchanged):
// the warning line is printed ABOVE the command, single line.
func TestIsDangerous_Extended(t *testing.T) {
	dangerous := []struct {
		cmd     string
		comment string
	}{
		{"rm -rf /", "rm -rf root"},
		{"rm -rf /home/user", "rm -rf absolute path"},
		{"rm -rf ~", "rm -rf home tilde"},
		{"rm -rf ~/projects", "rm -rf tilde subdir"},
		{"rm -rf $HOME", "rm -rf $HOME"},
		{"rm -rf $HOME/projects", "rm -rf $HOME subdir"},
		{"rm -rf *", "rm -rf glob"},
		{"rm -rf .", "rm -rf dot"},
		{"sudo rm -rf /", "sudo rm -rf root"},
		{"chmod -R 777 /var/www", "chmod without sudo"},
		{"sudo chmod -R 777 /", "chmod with sudo"},
		{"git push --force", "git push force"},
		{"git push -f", "git push short force"},
		{"git push origin main --force", "git push force with args"},
		{"git reset --hard", "git reset hard"},
		{"git reset --hard HEAD~1", "git reset hard with ref"},
		{"echo data > /dev/sda", "redirect to sd device"},
		{"cat img.iso > /dev/sda1", "redirect to sd partition"},
		{"dd if=x of=y > /dev/nvme0n1", "redirect to nvme device"},
		{"cat f >> /dev/nvme0n1p2", "append to nvme partition"},
	}
	for _, tt := range dangerous {
		t.Run(tt.comment, func(t *testing.T) {
			if !isDangerous(tt.cmd) {
				t.Errorf("isDangerous(%q) = false, want true", tt.cmd)
			}
		})
	}
}

func TestIsDangerous_SafeCommands(t *testing.T) {
	safe := []string{
		"rm dosya.txt",
		"rm -rf node_modules",
		"rm -rf ./build",
		"chmod 644 dosya",
		"chmod -R 755 ./public",
		"git push",
		"git push origin main",
		"git reset --soft HEAD~1",
		"git status",
		"ls -la",
		"cat file.txt > /tmp/out.txt",
	}
	for _, cmd := range safe {
		t.Run(cmd, func(t *testing.T) {
			if isDangerous(cmd) {
				t.Errorf("isDangerous(%q) = true, want false", cmd)
			}
		})
	}
}

// End-to-end through RenderOutput: warning line sits above the command.
func TestRenderOutput_ExtendedDangerous(t *testing.T) {
	ans := provider.Answer{Reason: "Test.", Fix: "rm -rf ~"}
	output := RenderOutput(ans, "tr")
	warnIdx := strings.Index(output, "Uyarı")
	cmdIdx := strings.Index(output, "rm -rf ~")
	if warnIdx == -1 {
		t.Fatalf("missing warning in output: %s", output)
	}
	if cmdIdx == -1 {
		t.Fatalf("missing command in output: %s", output)
	}
	if warnIdx > cmdIdx {
		t.Errorf("warning must be printed above the command: %s", output)
	}
}
