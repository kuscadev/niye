//go:build linux

package secret

import (
	"syscall"
	"unsafe"
)

// disableEcho turns off terminal echo on fd and returns a restore func.
// It fails on non-terminals instead of silently leaving echo on.
func disableEcho(fd uintptr) (func(), error) {
	var termios syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&termios))); errno != 0 {
		return nil, errNotSupported
	}

	oldTermios := termios
	termios.Lflag &^= syscall.ECHO
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(&termios))); errno != 0 {
		return nil, errNotSupported
	}

	restored := false
	return func() {
		if restored {
			return
		}
		restored = true
		syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCSETS, uintptr(unsafe.Pointer(&oldTermios)))
	}, nil
}
