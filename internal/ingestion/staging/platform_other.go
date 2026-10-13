//go:build !linux && !darwin

package staging

import (
	"context"
	"os"
)

const supported = false
const noFollow = 0
const nonBlock = 0

// owned fails closed on platforms without the reviewed local identity/locking profile.
func owned(os.FileInfo) bool { return false }

// lockFile cannot silently substitute an unsafe unsupported-platform lock.
func lockFile(context.Context, *os.File) error { return failure("unsupported_platform") }
