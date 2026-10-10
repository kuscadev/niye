package mask

import (
	"os"
	"regexp"
	"strings"
)

var (
	homeDir  string
	username string
	// Pattern 1a: key="value" (double quoted)
	secretPattern1a = regexp.MustCompile(`(?i)(key|token|password|secret|passwd)\s*([:=])\s*"([^"]+)"`)
	// Pattern 1b: key='value' (single quoted)
	secretPattern1b = regexp.MustCompile(`(?i)(key|token|password|secret|passwd)\s*([:=])\s*'([^']+)'`)
	// Pattern 1c: key=value or key: value (unquoted, value at least 4 chars
	// so that fragments like "token: }" are left alone)
	secretPattern1c = regexp.MustCompile(`(?i)(key|token|password|secret|passwd)\s*([:=])\s*([^"'\s]{4,})`)
	// Pattern 2: "key_name": "value" or 'key_name': 'value' (JSON style) - key_name contains sensitive word
	secretPattern2 = regexp.MustCompile(`(?i)["']([A-Za-z0-9_]*?(?:key|token|password|secret|passwd)[A-Za-z0-9_]*?)["']\s*:\s*(["'])([^"']+)(["'])`)
	// Pattern 3: Bearer token
	bearerPattern = regexp.MustCompile(`(?i)Bearer\s+([A-Za-z0-9_\-\.]+)`)
	// Standalone secrets without a key=/token= prefix
	skKeyPattern       = regexp.MustCompile(`\bsk-[A-Za-z0-9_\-.]{8,}\b`)
	githubTokenPattern = regexp.MustCompile(`\b((ghp|gho)_[A-Za-z0-9]{8,}|github_pat_[A-Za-z0-9_]{8,})\b`)
	awsKeyPattern      = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	jwtPattern         = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	// Userinfo in URLs: scheme://user...@host
	urlUserinfoPattern = regexp.MustCompile(`(?i)(https?://)[^/\s@]+@`)
	usernamePattern    *regexp.Regexp
)

func init() {
	RefreshMaskingVars()
}

// RefreshMaskingVars re-reads environment variables for masking.
// Exported for testing purposes.
func RefreshMaskingVars() {
	// Use HOME env var if available, otherwise fall back to os.UserHomeDir()
	homeDir = os.Getenv("HOME")
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}
	if u := os.Getenv("USER"); u != "" {
		username = u
	} else if u := os.Getenv("USERNAME"); u != "" {
		username = u
	}
	if len(username) >= 3 {
		usernamePattern = regexp.MustCompile(`\b` + regexp.QuoteMeta(username) + `\b`)
	} else {
		usernamePattern = nil
	}
}

// Mask redacts sensitive data (home dir, username, secrets, Bearer tokens)
// before the error text is sent to the LLM.
func Mask(input string) string {
	result := input

	// Mask home directory path
	if homeDir != "" {
		result = strings.ReplaceAll(result, homeDir, "~")
	}

	// Mask username
	if usernamePattern != nil {
		result = usernamePattern.ReplaceAllString(result, "<user>")
	}

	// Mask userinfo in URLs (scheme://user...@host -> scheme://***@host)
	result = urlUserinfoPattern.ReplaceAllString(result, "${1}***@")

	// Mask standalone secrets (no key= prefix needed)
	result = skKeyPattern.ReplaceAllString(result, "***")
	result = githubTokenPattern.ReplaceAllString(result, "***")
	result = awsKeyPattern.ReplaceAllString(result, "***")
	result = jwtPattern.ReplaceAllString(result, "***")

	// Mask secrets - pattern 1a: key="value" (double quoted)
	result = secretPattern1a.ReplaceAllStringFunc(result, func(match string) string {
		matches := secretPattern1a.FindStringSubmatch(match)
		if len(matches) >= 4 {
			keyName := matches[1]
			separator := matches[2]
			return keyName + separator + `"***"`
		}
		return match
	})

	// Mask secrets - pattern 1b: key='value' (single quoted)
	result = secretPattern1b.ReplaceAllStringFunc(result, func(match string) string {
		matches := secretPattern1b.FindStringSubmatch(match)
		if len(matches) >= 4 {
			keyName := matches[1]
			separator := matches[2]
			return keyName + separator + `'***'`
		}
		return match
	})

	// Mask secrets - pattern 1c: key=value or key: value (unquoted)
	result = secretPattern1c.ReplaceAllStringFunc(result, func(match string) string {
		matches := secretPattern1c.FindStringSubmatch(match)
		if len(matches) >= 4 {
			keyName := matches[1]
			separator := matches[2]
			if separator == ":" {
				return keyName + separator + " ***"
			}
			return keyName + separator + "***"
		}
		return match
	})

	// Mask secrets - pattern 2: "key_name": "value" (JSON style)
	result = secretPattern2.ReplaceAllStringFunc(result, func(match string) string {
		matches := secretPattern2.FindStringSubmatch(match)
		if len(matches) >= 5 {
			keyName := matches[1]
			quote := matches[2]
			return `"` + keyName + `": ` + quote + "***" + quote
		}
		return match
	})

	// Mask Bearer tokens
	result = bearerPattern.ReplaceAllString(result, "Bearer ***")

	// Cleanup: fix any double spaces from replacements
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	return result
}
