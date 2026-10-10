//go:build darwin

package secret

import (
	"bufio"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/kuscadev/niye/internal/i18n"
)

// readSecretNoEcho reads a secret from stdin with echo disabled on macOS.
func readSecretNoEcho(prompt string) (string, error) {
	os.Stderr.WriteString(prompt + " " + i18n.EchoDisabledNote + "\n")

	fd := int(os.Stdin.Fd())
	var termios syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGETA, uintptr(unsafe.Pointer(&termios))); errno != 0 {
		return "", errNotSupported
	}

	// Save original state
	oldTermios := termios

	// Disable echo
	termios.Lflag &^= syscall.ECHO
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&termios))); errno != 0 {
		return "", errNotSupported
	}

	// Read the line using a bufio reader on os.Stdin
	reader := bufio.NewReader(os.Stdin)
	secret, err := reader.ReadString('\n')

	// Restore terminal state
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&oldTermios)))
	os.Stderr.WriteString("\n")

	if err != nil {
		return "", err
	}
	return strings.TrimSpace(secret), nil
}
