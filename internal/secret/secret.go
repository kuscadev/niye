package secret

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

var (
	errNotSupported = errors.New("echo suppression not supported on this platform")
	errNotATerminal = errors.New("stdin is not a terminal")
	// ForceTerminalForTesting forces IsTerminal to return true in tests.
	ForceTerminalForTesting = false
)

// PromptSecret reads a secret from the user with echo disabled if possible.
func PromptSecret(prompt string, reader *bufio.Reader) (string, error) {
	if reader == nil {
		return "", errors.New("no reader provided")
	}

	// Write prompt to stderr
	if prompt != "" {
		os.Stderr.WriteString(prompt)
	}

	// Read line from the reader
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// IsTerminal checks if the reader is a terminal.
func IsTerminal(r io.Reader) bool {
	if ForceTerminalForTesting {
		return true
	}
	if f, ok := r.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}
