//go:build linux || darwin

package staging

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
)

const supported = true
const noFollow = syscall.O_NOFOLLOW
const nonBlock = syscall.O_NONBLOCK

// owned rejects inodes belonging to another local principal, even in privileged tests.
func owned(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

// lockFile polls an advisory exclusive lock without blocking past caller cancellation.
func lockFile(ctx context.Context, f *os.File) error {
	for {
		if e := checkContext(ctx); e != nil {
			return e
		}
		e := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if e == nil {
			return nil
		}
		if !errors.Is(e, syscall.EWOULDBLOCK) && !errors.Is(e, syscall.EAGAIN) && !errors.Is(e, syscall.EINTR) {
			return failure("storage_error")
		}
		timer := time.NewTimer(5 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return failure("cancelled")
		case <-timer.C:
		}
	}
}
