package sandboxidentity

import (
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestInheritedPrivatePipeDeadline uses a genuinely blocking Unix descriptor without another Go file owner.
func TestInheritedPrivatePipeDeadline(t *testing.T) {
	fds := make([]int, 2)
	if syscall.Pipe(fds) != nil {
		t.Fatal("pipe setup")
	}
	defer syscall.Close(fds[1])
	f, err := OpenPrivateInput(fds[0])
	if err != nil {
		t.Fatal("inherited pipe rejected")
	}
	defer f.Close()
	if f.SetReadDeadline(time.Now().Add(25*time.Millisecond)) != nil {
		t.Fatal("pipe not pollable")
	}
	var one [1]byte
	if _, err = f.Read(one[:]); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("inherited read ignored deadline")
	}
}

// TestPrivateSocketAndStandardDescriptorBoundary covers the second permitted descriptor type without networking.
func TestPrivateSocketAndStandardDescriptorBoundary(t *testing.T) {
	for _, fd := range []int{-1, 0, 1, 2, 1025} {
		if _, err := OpenPrivateInput(fd); err != ErrProjection {
			t.Fatal("standard/unbounded descriptor accepted")
		}
	}
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal("socket setup")
	}
	f, err := OpenPrivateInput(fds[0])
	if err != nil {
		_ = syscall.Close(fds[1])
		t.Fatal("private socket rejected")
	}
	defer f.Close()
	if _, err = syscall.Write(fds[1], []byte("synthetic")); err != nil {
		t.Fatal("socket write")
	}
	_ = syscall.Close(fds[1])
	raw, err := io.ReadAll(f)
	if err != nil || string(raw) != "synthetic" {
		t.Fatal("private socket read failed")
	}
}
