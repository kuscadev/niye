//go:build linux

package secret

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// openPty opens a pseudoterminal pair using only stdlib + syscall.
func openPty(t *testing.T) (master, slave *os.File) {
	t.Helper()

	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatalf("open /dev/ptmx: %v", err)
	}
	var unlock int32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); errno != 0 {
		master.Close()
		t.Fatalf("unlockpt: %v", errno)
	}
	var ptn uint32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&ptn))); errno != 0 {
		master.Close()
		t.Fatalf("pty number: %v", errno)
	}
	slave, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", ptn), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		t.Fatalf("open slave: %v", err)
	}
	return master, slave
}

func echoEnabled(t *testing.T, f *os.File) bool {
	t.Helper()
	var termios syscall.Termios
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&termios))); errno != 0 {
		t.Fatalf("TCGETS: %v", errno)
	}
	return termios.Lflag&syscall.ECHO != 0
}

// disableEcho must really clear the ECHO bit, and restore must set it back.
func TestDisableEcho_TogglesEchoBit(t *testing.T) {
	master, slave := openPty(t)
	defer master.Close()
	defer slave.Close()

	if !echoEnabled(t, slave) {
		t.Fatal("pty slave should start with echo on")
	}
	restore, err := disableEcho(slave.Fd())
	if err != nil {
		t.Fatalf("disableEcho: %v", err)
	}
	if echoEnabled(t, slave) {
		t.Error("ECHO bit still set after disableEcho")
	}
	restore()
	if !echoEnabled(t, slave) {
		t.Error("ECHO bit not restored")
	}
}

// End to end: secret readable through the pty, echo restored afterwards.
func TestPromptSecretTerminal_ReadsSecretAndRestores(t *testing.T) {
	master, slave := openPty(t)
	defer master.Close()
	defer slave.Close()

	go func() {
		time.Sleep(200 * time.Millisecond)
		master.Write([]byte("gizli-anahtar\n"))
	}()
	got, err := PromptSecretTerminal("Anahtar: ", bufio.NewReader(slave), slave)
	if err != nil {
		t.Fatalf("PromptSecretTerminal: %v", err)
	}
	if got != "gizli-anahtar" {
		t.Errorf("got %q, want %q", got, "gizli-anahtar")
	}
	if !echoEnabled(t, slave) {
		t.Error("echo not restored after read")
	}
}

// Echo cannot be disabled on a pipe: must fail loudly, never silently
// fall back to visible input.
func TestPromptSecretTerminal_PipeFailsLoudly(t *testing.T) {
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	go func() {
		w.Write([]byte("x\n"))
		w.Close()
	}()
	if _, err := PromptSecretTerminal("Anahtar: ", bufio.NewReader(r), r); err == nil {
		t.Fatal("expected error when echo cannot be disabled, got nil (silent fallback?)")
	}
}
