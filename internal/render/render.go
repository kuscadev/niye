package render

import (
	"regexp"
	"strings"

	"github.com/kuscadev/niye/internal/provider"
)

// RenderOutput renders the answer in the required two-line format.
func RenderOutput(ans provider.Answer, lang string) string {
	var reasonLabel, fixLabel, warningPrefix string
	if lang == "en" {
		reasonLabel = "Cause"
		fixLabel = "Fix"
		warningPrefix = "⚠ Warning:"
	} else {
		reasonLabel = "Neden"
		fixLabel = "Çözüm"
		warningPrefix = "⚠ Uyarı:"
	}

	// Truncate reason to one sentence, use fallback if empty
	reason := truncateToOneSentence(ans.Reason)
	if reason == "" {
		if lang == "en" {
			reason = "No explanation received."
		} else {
			reason = "Açıklama alınamadı."
		}
	}

	// Check for dangerous commands in fix
	fixLines := strings.Split(ans.Fix, "\n")
	var output strings.Builder

	// Print reason line
	output.WriteString("● " + reasonLabel + ": " + reason + "\n")

	// Check each fix line for dangerous patterns
	fixStarted := false
	for i, line := range fixLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && !fixStarted {
			continue
		}
		fixStarted = true

		if isDangerous(trimmed) {
			output.WriteString("● " + fixLabel + ":\n")
			output.WriteString("    " + warningPrefix + " Bu komut tehlikeli olabilir, dikkatle kullanın.\n")
			output.WriteString("    " + line + "\n")
		} else {
			if i == 0 {
				output.WriteString("● " + fixLabel + ":\n")
			}
			output.WriteString("    " + line + "\n")
		}
	}

	return strings.TrimRight(output.String(), "\n")
}

// truncateToOneSentence keeps only the first sentence.
func truncateToOneSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Find first sentence ending (. ! ?) followed by space or end
	re := regexp.MustCompile(`([.!?])(\s+|$)`)
	match := re.FindStringIndex(s)
	if match != nil {
		return strings.TrimSpace(s[:match[1]])
	}
	return s
}

// isDangerous checks if a command line contains dangerous patterns.
func isDangerous(cmd string) bool {
	// A redirection into a raw disk device is destructive no matter which
	// command performs it, so check it on the unquoted text first — even an
	// `echo ... > /dev/sdX` line destroys data.
	if deviceRedirectRe.MatchString(stripQuotes(cmd)) {
		return true
	}

	// Skip if the dangerous pattern appears only inside quotes
	// Simple heuristic: if the command starts with echo/printf and contains the pattern, it's likely a string
	trimmed := strings.TrimSpace(cmd)
	if strings.HasPrefix(trimmed, "echo ") || strings.HasPrefix(trimmed, "printf ") {
		return false
	}

	if hasDestructiveRmTarget(cmd) {
		return true
	}

	dangerousPatterns := []string{
		`rm\s+-rf\s+/`,
		`sudo\s+chmod\s+-R\s+777`,
		`chmod\s+-R\s+777`,
		`git\s+push\b[^|;&]*--force\b`,
		`git\s+push\b[^|;&]*\s-f(\s|$)`,
		`git\s+reset\b[^|;&]*--hard\b`,
		`mkfs(\.|\s+)`,
		`dd\s+of=`,
		`curl\s+.*\|\s*(bash|sh)`,
		`wget\s+.*\|\s*(bash|sh)`,
		`\|\s*(bash|sh)\s*$`,
	}
	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, cmd); matched {
			return true
		}
	}
	return false
}

// deviceRedirectRe matches shell redirections into raw disk devices.
var deviceRedirectRe = regexp.MustCompile(`>{1,2}\s*/dev/(sd|nvme)[A-Za-z0-9]*`)

// stripQuotes removes single- and double-quoted spans so that quoted text
// is not mistaken for shell syntax (best effort, no escape handling).
func stripQuotes(s string) string {
	var b strings.Builder
	var quote rune
	for _, r := range s {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// hasDestructiveRmTarget reports whether cmd contains an rm invocation with
// recursive+force flags and a target of /, ~, $HOME, * or ..
func hasDestructiveRmTarget(cmd string) bool {
	segments := strings.FieldsFunc(cmd, func(r rune) bool {
		return r == ';' || r == '&' || r == '|' || r == '\n'
	})
	for _, seg := range segments {
		fields := strings.Fields(seg)
		for i := 0; i < len(fields); i++ {
			f := fields[i]
			if f == "sudo" {
				continue
			}
			if f != "rm" && !strings.HasSuffix(f, "/rm") {
				continue
			}
			var flags strings.Builder
			j := i + 1
			for j < len(fields) && len(fields[j]) > 1 && strings.HasPrefix(fields[j], "-") && fields[j] != "--" {
				flags.WriteString(fields[j][1:])
				j++
			}
			fl := strings.ToLower(flags.String())
			if !strings.Contains(fl, "r") || !strings.Contains(fl, "f") {
				continue
			}
			for ; j < len(fields); j++ {
				if fields[j] == "--" {
					continue
				}
				if isDestructiveRmTarget(fields[j]) {
					return true
				}
			}
		}
	}
	return false
}

// isDestructiveRmTarget reports whether an rm argument is a dangerous target:
// an absolute path, home (~, $HOME), a glob, or the current directory.
func isDestructiveRmTarget(tok string) bool {
	tok = strings.Trim(tok, `"'`)
	switch {
	case tok == "":
		return false
	case strings.HasPrefix(tok, "/"):
		return true
	case strings.HasPrefix(tok, "~"):
		return true
	case tok == "$HOME" || strings.HasPrefix(tok, "$HOME/"):
		return true
	case tok == "${HOME}" || strings.HasPrefix(tok, "${HOME}/"):
		return true
	case tok == "*" || tok == "." || tok == "./":
		return true
	}
	return false
}
