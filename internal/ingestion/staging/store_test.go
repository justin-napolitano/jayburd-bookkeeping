package staging

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const entityID = "00000000-0000-4000-8000-000000000041"
const bookID = "00000000-0000-4000-8000-000000000042"

var helperRoot = flag.String("staging-helper-root", "", "Synthetic subprocess fixture only")
var helperCrash = flag.String("staging-helper-crash", "", "Synthetic process exit point only")
var helperObject = flag.String("staging-helper-object", "", "Synthetic subprocess object UUID only")

// byteSource supplies deterministic synthetic bytes and records transferred reader ownership.
type byteSource struct {
	r          *bytes.Reader
	closed     atomic.Int32
	readBytes  int
	readError  error
	closeError error
}

// source creates a fresh reader for each attempted upload, never reusing consumed input.
func source(raw []byte) *byteSource { return &byteSource{r: bytes.NewReader(raw)} }

// Read honors context and can inject a private error for redaction acceptance.
func (s *byteSource) Read(ctx context.Context, p []byte) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	if s.readError != nil {
		return 0, s.readError
	}
	n, e := s.r.Read(p)
	s.readBytes += n
	return n, e
}

// Close records exactly one ownership release without exposing injected error text.
func (s *byteSource) Close(context.Context) error { s.closed.Add(1); return s.closeError }

// setup uses only owner-private temporary storage and generated object references.
func setup(t *testing.T) (*Store, string, Reference) {
	t.Helper()
	if !supported {
		t.Skip("reviewed local adapter supports Linux/macOS")
	}
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	r, e := NewReference(Scope{entityID, bookID})
	if e != nil {
		t.Fatal(e)
	}
	return s, dir, r
}

// wantCode ensures every failure is an owned bounded type without arbitrary diagnostics.
func wantCode(t testing.TB, err error, want string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != want {
		t.Fatalf("want %s got %v", want, err)
	}
}

// put checks the returned receipt against independently computed fixture bytes.
func put(t testing.TB, s *Store, r Reference, raw []byte) Receipt {
	t.Helper()
	src := source(raw)
	got, e := s.Put(context.Background(), r, src)
	if e != nil {
		t.Fatal(e)
	}
	if src.closed.Load() != 1 || got.Reference != r || got.ByteCount != len(raw) || got.SHA256 != sha256.Sum256(raw) {
		t.Fatal("receipt/ownership mismatch")
	}
	return got
}

// TestStagingImmutableReplay preserves exact bytes and rejects changed same-length content without replacement.
func TestStagingImmutableReplay(t *testing.T) {
	s, dir, r := setup(t)
	raw := []byte("synthetic evidence 1.01\r\n")
	first := put(t, s, r, raw)
	if next := put(t, s, r, raw); next != first {
		t.Fatal("retry changed receipt")
	}
	changed := append([]byte(nil), raw...)
	changed[0] = 'S'
	_, e := s.Put(context.Background(), r, source(changed))
	wantCode(t, e, "object_conflict")
	got, receipt, e := s.Read(context.Background(), r)
	if e != nil || !bytes.Equal(got, raw) || receipt != first {
		t.Fatal("immutable content replaced", e)
	}
	for _, suffix := range []string{".data", ".lock"} {
		info, e := os.Stat(filepath.Join(dir, stem(r)+suffix))
		if e != nil || info.Mode().Perm() != 0600 {
			t.Fatal("unsafe file mode", e)
		}
	}
	reopened, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if next := put(t, reopened, r, raw); next != first {
		t.Fatal("restart lost identity")
	}
}

// TestStagingConcurrentStores serializes independent handles, accepting identical writers and one changed-content winner.
func TestStagingConcurrentStores(t *testing.T) {
	s, dir, r := setup(t)
	other, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Go(func() { _, e := other.Put(context.Background(), r, source([]byte("same synthetic bytes"))); errs <- e })
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	r, e = NewReference(r.Scope)
	if e != nil {
		t.Fatal(e)
	}
	errs = make(chan error, 2)
	for i, store := range []*Store{s, other} {
		raw := []byte{byte('a' + i)}
		wg.Go(func() { _, e := store.Put(context.Background(), r, source(raw)); errs <- e })
	}
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for e := range errs {
		if e == nil {
			success++
		} else if hasCode(e, "object_conflict") {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("conflicting writers both won")
	}
}

// TestStagingProcessHelper runs only when explicitly given the synthetic fixture flags by the parent test.
func TestStagingProcessHelper(t *testing.T) {
	if *helperRoot == "" {
		return
	}
	s, e := Open(*helperRoot)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if *helperCrash != "" {
		s.fault = func(stage string) error {
			if stage == *helperCrash {
				os.Exit(0)
			}
			return nil
		}
	}
	_, e = s.Put(context.Background(), Reference{Scope{entityID, bookID}, *helperObject}, source([]byte("subprocess synthetic bytes")))
	if e != nil {
		t.Fatal(e)
	}
}

// TestStagingConcurrentProcesses proves advisory locking with real independent OS processes.
func TestStagingConcurrentProcesses(t *testing.T) {
	_, dir, r := setup(t)
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	commands := make([]*exec.Cmd, 4)
	for i := range commands {
		commands[i] = exec.Command(exe, "-test.run=^TestStagingProcessHelper$", "-staging-helper-root="+dir, "-staging-helper-object="+r.ObjectID)
		if e = commands[i].Start(); e != nil {
			t.Fatal(e)
		}
	}
	for _, cmd := range commands {
		if e = cmd.Wait(); e != nil {
			t.Fatal("independent process failed", e)
		}
	}
}

// TestStagingSizeAndSourceBoundary checks exact cap/sentinel, empty/failed sources and cancellation ownership.
func TestStagingSizeAndSourceBoundary(t *testing.T) {
	s, _, r := setup(t)
	put(t, s, r, bytes.Repeat([]byte("x"), MaxBytes))
	for _, tc := range []struct {
		size int
		code string
	}{{0, "empty_object"}, {MaxBytes + 1, "too_large"}, {MaxBytes + 5000, "too_large"}} {
		r, e := NewReference(r.Scope)
		if e != nil {
			t.Fatal(e)
		}
		src := source(bytes.Repeat([]byte("z"), tc.size))
		receipt, e := s.Put(context.Background(), r, src)
		wantCode(t, e, tc.code)
		if receipt != (Receipt{}) || src.closed.Load() != 1 || src.readBytes > MaxBytes+1 {
			t.Fatal("partial receipt or reader overrun")
		}
		_, _, e = s.Read(context.Background(), r)
		wantCode(t, e, "not_found")
	}
	for _, closeError := range []bool{false, true} {
		src := source([]byte("synthetic"))
		if closeError {
			src.closeError = errors.New("private/path/payload")
		} else {
			src.readError = errors.New("private/path/payload")
		}
		_, e := s.Put(context.Background(), r, src)
		wantCode(t, e, "source_error")
		if strings.Contains(e.Error(), "private") || src.closed.Load() != 1 {
			t.Fatal("error leaked or source not closed")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := source([]byte("synthetic"))
	_, e := s.Put(ctx, r, src)
	wantCode(t, e, "cancelled")
	if src.readBytes != 0 || src.closed.Load() != 1 {
		t.Fatal("cancelled source ownership")
	}
}

// waitSource cooperatively blocks to prove cancellation and delete-versus-upload races without leaked goroutines.
type waitSource struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	closed  atomic.Int32
	done    bool
}

// Read waits under the supplied context, then emits one synthetic chunk exactly once.
func (s *waitSource) Read(ctx context.Context, p []byte) (int, error) {
	if s.done {
		return 0, io.EOF
	}
	s.once.Do(func() { close(s.started) })
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-s.release:
		s.done = true
		return copy(p, "synthetic interrupted upload"), io.EOF
	}
}

// Close records the bounded source handoff independently of cancellation state.
func (s *waitSource) Close(context.Context) error { s.closed.Add(1); return nil }

// TestStagingCancellationAndDeleteRace requires deletion to prevent an in-flight upload from resurrecting its identity.
func TestStagingCancellationAndDeleteRace(t *testing.T) {
	s, _, r := setup(t)
	src := &waitSource{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, e := s.Put(context.Background(), r, src); done <- e }()
	<-src.started
	if e := s.Delete(context.Background(), r); e != nil {
		t.Fatal(e)
	}
	close(src.release)
	wantCode(t, <-done, "object_deleted")
	if src.closed.Load() != 1 {
		t.Fatal("source not closed")
	}
	src = &waitSource{started: make(chan struct{}), release: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, e := s.Put(ctx, r, src)
	wantCode(t, e, "cancelled")
	if src.closed.Load() != 1 {
		t.Fatal("cancel cleanup")
	}
}

// TestStagingPublicationFaults reconciles the same reference after faults before/after visible publication.
func TestStagingPublicationFaults(t *testing.T) {
	for _, stage := range []string{"before_file_sync", "before_publish", "after_publish", "directory_sync"} {
		t.Run(stage, func(t *testing.T) {
			s, dir, r := setup(t)
			s.fault = func(at string) error {
				if at == stage {
					return errors.New("protected fault")
				}
				return nil
			}
			receipt, e := s.Put(context.Background(), r, source([]byte("synthetic fault fixture")))
			want := "storage_error"
			if stage == "after_publish" || stage == "directory_sync" {
				want = "storage_unknown"
			}
			wantCode(t, e, want)
			if receipt != (Receipt{}) {
				t.Fatal("uncertain receipt exposed")
			}
			_, statErr := os.Stat(filepath.Join(dir, stem(r)+".data"))
			if want == "storage_error" && !os.IsNotExist(statErr) {
				t.Fatal("prepublication failure published")
			}
			s.fault = nil
			put(t, s, r, []byte("synthetic fault fixture"))
		})
	}
}

// TestStagingDeleteReplay preserves tombstone identity across failure, retry, reopen and changed-byte replay.
func TestStagingDeleteReplay(t *testing.T) {
	for _, stage := range []string{"after_tombstone", "after_delete", "directory_sync"} {
		t.Run(stage, func(t *testing.T) {
			s, dir, r := setup(t)
			put(t, s, r, []byte("synthetic deletion fixture"))
			s.fault = func(at string) error {
				if at == stage {
					return errors.New("private fault")
				}
				return nil
			}
			wantCode(t, s.Delete(context.Background(), r), "storage_unknown")
			s.fault = nil
			_, _, e := s.Read(context.Background(), r)
			wantCode(t, e, "object_deleted")
			if e = s.Delete(context.Background(), r); e != nil {
				t.Fatal(e)
			}
			if e = s.Delete(context.Background(), r); e != nil {
				t.Fatal(e)
			}
			if _, e = os.Stat(filepath.Join(dir, stem(r)+".data")); !os.IsNotExist(e) {
				t.Fatal("raw object remains")
			}
			reopened, e := Open(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer reopened.Close()
			_, e = reopened.Put(context.Background(), r, source([]byte("changed synthetic bytes")))
			wantCode(t, e, "object_deleted")
		})
	}
}

// TestStagingPathsAndModes rejects malformed scope, symlinks, permissive roots and substituted special entries.
func TestStagingPathsAndModes(t *testing.T) {
	s, dir, r := setup(t)
	for _, bad := range []string{"../escape", "/absolute", "00000000-0000-0000-0000-000000000000", "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"} {
		badRef := r
		badRef.ObjectID = bad
		src := source([]byte("synthetic"))
		_, e := s.Put(context.Background(), badRef, src)
		wantCode(t, e, "invalid_reference")
		if src.readBytes != 0 || src.closed.Load() != 1 {
			t.Fatal("invalid ref read source")
		}
		if _, e = NewReference(Scope{bad, bookID}); e == nil {
			t.Fatal("invalid scope accepted")
		}
	}
	outside := filepath.Join(t.TempDir(), "synthetic-outside")
	if e := os.WriteFile(outside, []byte("synthetic outside sentinel"), 0600); e != nil {
		t.Fatal(e)
	}
	for _, suffix := range []string{".data", ".lock", ".deleted"} {
		r, e := NewReference(r.Scope)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.Symlink(outside, filepath.Join(dir, stem(r)+suffix)); e != nil {
			t.Fatal(e)
		}
		_, e = s.Put(context.Background(), r, source([]byte("synthetic")))
		wantCode(t, e, "unsafe_storage")
		_, _, e = s.Read(context.Background(), r)
		wantCode(t, e, "unsafe_storage")
		wantCode(t, s.Delete(context.Background(), r), "unsafe_storage")
	}
	raw, e := os.ReadFile(outside)
	if e != nil || string(raw) != "synthetic outside sentinel" {
		t.Fatal("escaped root")
	}
	link := filepath.Join(t.TempDir(), "root-link")
	if e = os.Symlink(dir, link); e != nil {
		t.Fatal(e)
	}
	_, e = Open(link)
	wantCode(t, e, "unsafe_storage")
	permissive := t.TempDir()
	if e = os.Chmod(permissive, 0755); e != nil {
		t.Fatal(e)
	}
	_, e = Open(permissive)
	wantCode(t, e, "unsafe_storage")
}

// TestStagingCorruptDiskAndLockDeadline enforces disk bounds/modes and cancels actual lock contention.
func TestStagingCorruptDiskAndLockDeadline(t *testing.T) {
	s, dir, r := setup(t)
	put(t, s, r, []byte("synthetic"))
	path := filepath.Join(dir, stem(r)+".data")
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	_, _, e := s.Read(context.Background(), r)
	wantCode(t, e, "unsafe_storage")
	if e = os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Truncate(path, MaxBytes+1); e != nil {
		t.Fatal(e)
	}
	_, _, e = s.Read(context.Background(), r)
	wantCode(t, e, "unsafe_storage")
	unlock, e := s.lock(context.Background(), stem(r))
	if e != nil {
		t.Fatal(e)
	}
	defer unlock()
	other, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, _, e = other.Read(ctx, r)
	wantCode(t, e, "cancelled")
}

// TestStagingCloseAndPrivateOrphan treats temporary files as unaddressable evidence and rejects operations after close.
func TestStagingCloseAndPrivateOrphan(t *testing.T) {
	s, dir, r := setup(t)
	if e := os.WriteFile(filepath.Join(dir, "tmp_00000000-0000-4000-8000-000000000099"), []byte("synthetic interrupted temp"), 0600); e != nil {
		t.Fatal(e)
	}
	_, _, e := s.Read(context.Background(), r)
	wantCode(t, e, "not_found")
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	src := source([]byte("synthetic"))
	_, e = s.Put(context.Background(), r, src)
	wantCode(t, e, "storage_closed")
	if src.closed.Load() != 1 {
		t.Fatal("closed store lost source ownership")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
}

// TestStagingProcessExitRecovery reconciles actual abrupt process exit, leaving private orphan temporaries outside object addressing.
func TestStagingProcessExitRecovery(t *testing.T) {
	for _, stage := range []string{"before_publish", "after_publish"} {
		t.Run(stage, func(t *testing.T) {
			s, dir, r := setup(t)
			exe, e := os.Executable()
			if e != nil {
				t.Fatal(e)
			}
			cmd := exec.Command(exe, "-test.run=^TestStagingProcessHelper$", "-staging-helper-root="+dir, "-staging-helper-object="+r.ObjectID, "-staging-helper-crash="+stage)
			if e = cmd.Run(); e != nil {
				t.Fatal("crash fixture child", e)
			}
			raw, _, e := s.Read(context.Background(), r)
			if stage == "before_publish" {
				wantCode(t, e, "not_found")
			} else if e != nil || string(raw) != "subprocess synthetic bytes" {
				t.Fatal("published bytes not recoverable", e)
			}
			put(t, s, r, []byte("subprocess synthetic bytes"))
			entries, e := os.ReadDir(dir)
			if e != nil {
				t.Fatal(e)
			}
			orphans := 0
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "tmp_") {
					orphans++
					info, e := entry.Info()
					if e != nil || info.Mode().Perm() != 0600 {
						t.Fatal("unprotected crash orphan", e)
					}
				}
			}
			if orphans == 0 {
				t.Fatal("exit did not exercise orphan boundary")
			}
		})
	}
}

// TestStagingCancellationPublicationBoundary distinguishes clean prepublication cancellation from an uncertain committed name.
func TestStagingCancellationPublicationBoundary(t *testing.T) {
	for _, stage := range []string{"before_publish", "after_publish"} {
		t.Run(stage, func(t *testing.T) {
			s, _, r := setup(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s.fault = func(at string) error {
				if at == stage {
					cancel()
				}
				return nil
			}
			receipt, e := s.Put(ctx, r, source([]byte("synthetic cancel boundary")))
			want := "cancelled"
			if stage == "after_publish" {
				want = "storage_unknown"
			}
			wantCode(t, e, want)
			if receipt != (Receipt{}) {
				t.Fatal("cancelled/unknown receipt exposed")
			}
			s.fault = nil
			put(t, s, r, []byte("synthetic cancel boundary"))
		})
	}
}
