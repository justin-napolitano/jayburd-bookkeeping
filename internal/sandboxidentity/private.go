package sandboxidentity

import (
	"os"
	"syscall"
	"time"
)

// OpenPrivateInput takes ownership only of an explicit pipe/socket and makes reads interruptible.
// Inherited Unix descriptors are often blocking; wrapping one without O_NONBLOCK can defeat Close/deadlines.
func OpenPrivateInput(fd int) (*os.File, error) {
	if fd < 3 || fd > 1024 {
		return nil, ErrProjection
	}
	var stat syscall.Stat_t
	if syscall.Fstat(fd, &stat) != nil {
		return nil, ErrProjection
	}
	if stat.Mode&syscall.S_IFMT != syscall.S_IFIFO && stat.Mode&syscall.S_IFMT != syscall.S_IFSOCK {
		_ = syscall.Close(fd)
		return nil, ErrProjection
	}
	if syscall.SetNonblock(fd, true) != nil {
		_ = syscall.Close(fd)
		return nil, ErrProjection
	}
	f := os.NewFile(uintptr(fd), "private-sandbox-input")
	if f == nil {
		_ = syscall.Close(fd)
		return nil, ErrProjection
	}
	// Require pollable deadline support rather than assuming closing a blocking descriptor interrupts a read.
	if f.SetReadDeadline(time.Now().Add(15*time.Second)) != nil {
		_ = f.Close()
		return nil, ErrProjection
	}
	return f, nil
}
