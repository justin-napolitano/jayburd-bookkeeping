package acquisition

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
)

const testEntity = "00000000-0000-4000-8000-000000000002"
const testBook = "00000000-0000-4000-8000-000000000003"
const testStream = "00000000-0000-4000-8000-000000000004"
const testSet = "00000000-0000-4000-8000-000000000005"
const testClaim = "00000000-0000-4000-8000-000000000006"

// scopeFixture returns only fictional scope identifiers.
func scopeFixture() Scope { return Scope{testEntity, testBook, testStream} }

// localEvidence opens a temporary private store and owns its close.
func localEvidence(t testing.TB) *LocalEvidence {
	t.Helper()
	path := t.TempDir()
	if e := os.Chmod(path, 0700); e != nil {
		t.Fatal(e)
	}
	s, e := staging.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	v, e := NewLocalEvidence(s)
	if e != nil {
		t.Fatal(e)
	}
	return v
}

// TestRawEvidenceLifecycle proves exact bytes, scoped isolation, multi-chunk publication and tombstoned deletion.
func TestRawEvidenceLifecycle(t *testing.T) {
	store := localEvidence(t)
	key := EvidenceKey{Scope: scopeFixture(), SetID: testSet, Page: 0}
	ctx := context.Background()
	raw := bytes.Repeat([]byte("x"), MaxRawBytes)
	if e := store.Put(ctx, key, raw); e != nil {
		t.Fatal(e)
	}
	got, found, e := store.Get(ctx, key)
	if e != nil || !found || !bytes.Equal(raw, got) {
		t.Fatal("raw bytes not retained")
	}
	if e = store.Put(ctx, key, raw); e != nil {
		t.Fatal(e)
	}
	different := bytes.Clone(raw)
	different[len(different)-1] = 'y'
	if e = store.Put(ctx, key, different); e == nil {
		t.Fatal("changed bytes reused identity")
	}
	other := key
	other.Scope.StreamID = testClaim
	if _, found, e = store.Get(ctx, other); e != nil || found {
		t.Fatal("cross-stream evidence")
	}
	if e = store.Put(ctx, other, append(raw, 'x')); e == nil {
		t.Fatal("oversized accepted")
	}
	if e = store.Delete(ctx, key); e != nil {
		t.Fatal(e)
	}
	if e = store.Put(ctx, key, raw); e == nil {
		t.Fatal("deleted evidence resurrected")
	}
}

// TestRawEvidenceConcurrentPublication rejects conflicting complete bytes and tolerates an orphan chunk.
func TestRawEvidenceConcurrentPublication(t *testing.T) {
	store := localEvidence(t)
	key := EvidenceKey{Scope: scopeFixture(), SetID: testSet, Page: 1}
	ctx := context.Background()
	raw := bytes.Repeat([]byte("a"), staging.MaxBytes+5)
	// A crash may leave a first chunk without a manifest; no evidence is readable yet.
	if _, e := store.store.Put(ctx, evidenceRef(key, "chunk-0"), &byteSource{bytes.NewReader(raw[:staging.MaxBytes])}); e != nil {
		t.Fatal(e)
	}
	if _, found, e := store.Get(ctx, key); e != nil || found {
		t.Fatal("partial object became evidence")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Go(func() { errs <- store.Put(ctx, key, raw) })
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	changed := bytes.Clone(raw)
	changed[len(changed)-1] = 'b'
	errs = make(chan error, 2)
	wg.Go(func() { errs <- store.Put(ctx, key, raw) })
	wg.Go(func() { errs <- store.Put(ctx, key, changed) })
	wg.Wait()
	close(errs)
	failed := 0
	for e := range errs {
		if e != nil {
			failed++
		}
	}
	if failed != 1 {
		t.Fatal("conflicting retry not rejected")
	}
	cursor := "different"
	wrong := key
	wrong.Cursor = &cursor
	if _, _, e := store.Get(ctx, wrong); e == nil {
		t.Fatal("cursor mismatch accepted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if e := store.Put(cancelled, EvidenceKey{Scope: scopeFixture(), SetID: testSet, Page: 2}, []byte("x")); e == nil {
		t.Fatal("cancelled write accepted")
	}
}
