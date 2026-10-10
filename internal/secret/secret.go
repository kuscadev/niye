package secret

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/kuscadev/niye/internal/i18n"
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

// PromptSecretTerminal reads a secret from a terminal with echo disabled.
// The line itself is read through reader (which must wrap f) so already
// buffered input is not lost. If echo cannot be disabled it returns an
// error and never falls back to visible input.
func PromptSecretTerminal(prompt string, reader *bufio.Reader, f *os.File) (string, error) {
	if reader == nil {
		return "", errors.New("no reader provided")
	}
	if f == nil {
		return "", errNotATerminal
	}

	if prompt != "" {
		os.Stderr.WriteString(prompt + " " + i18n.EchoDisabledNote + "\n")
	}

	restore, err := disableEcho(f.Fd())
	if err != nil {
		return "", fmt.Errorf("cannot disable terminal echo: %w", err)
	}
	stopWatch := watchSignals(restore)
	defer stopWatch()
	defer restore()

	line, err := reader.ReadString('\n')
	os.Stderr.WriteString("\n")
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// watchSignals restores the terminal if SIGINT (Ctrl+C) arrives while echo
// is off, then re-raises the signal so the process still dies from it
// instead of surviving with a broken terminal.
func watchSignals(restore func()) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	stopCh := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		select {
		case sig := <-ch:
			restore()
			signal.Stop(ch)
			if p, err := os.FindProcess(os.Getpid()); err == nil {
				_ = p.Signal(sig)
			}
		case <-stopCh:
		}
	}()
	return func() {
		signal.Stop(ch)
		close(stopCh)
		<-finished
	}
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
