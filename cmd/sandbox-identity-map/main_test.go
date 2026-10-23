package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const syntheticFixture = `{"profile":"jayburd-sandbox-verification/1","fixture_id":"fresh-1","rows":[{"logical_id":"one","description":"Fictional one","revision":"1","current":true,"previous_revision":null,"source_state":"booked"}],"pending_links":[]}`
const syntheticPage = `{"added":[{"transaction_id":"Transaction-A","account_id":"Account-A","name":"Fictional one","amount":999999.99}],"modified":[],"removed":[],"has_more":false,"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`

// invocationFixture authors local synthetic expectations before any input pipe can be opened.
func invocationFixture(t *testing.T) ([]string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.json")
	if os.WriteFile(path, []byte(syntheticFixture), 0600) != nil {
		t.Fatal("fixture setup")
	}
	hash := sha256.Sum256([]byte(syntheticFixture))
	return []string{"-action=project", "-fixture=" + path, "-fixture-sha256=" + hex.EncodeToString(hash[:]), "-account-id=Account-A", "-input-fd=3"}, path
}

// TestDefaultAndInvalidAreInert proves no private input/credential/network work is implicit.
func TestDefaultAndInvalidAreInert(t *testing.T) {
	never := func(int) (io.ReadCloser, error) {
		t.Fatal("input opened before configuration accepted")
		return nil, errCommand
	}
	var out bytes.Buffer
	if execute(nil, &out, never) != nil || out.String() != "{\"offline\":true}\n" {
		t.Fatal("default not inert")
	}
	for _, args := range [][]string{{"-action=project"}, {"-secret=synthetic"}, {"-dsn=synthetic"}, {"-action=sync"}, {"-input-fd=0"}, {"trailing"}} {
		out.Reset()
		if execute(args, &out, never) == nil || out.Len() != 0 {
			t.Fatal("unsafe invocation accepted")
		}
	}
	args, path := invocationFixture(t)
	badAccount := append([]string(nil), args...)
	badAccount[3] = "-account-id=access-sandbox-rejected"
	out.Reset()
	if execute(badAccount, &out, never) == nil || out.Len() != 0 {
		t.Fatal("unsafe account reached private input")
	}
	if os.WriteFile(path, []byte(syntheticFixture+" "), 0600) != nil {
		t.Fatal("fixture change")
	}
	out.Reset()
	if execute(args, &out, never) == nil || out.Len() != 0 {
		t.Fatal("unpinned fixture accepted")
	}
}

// TestPrivateCreateOnlyProjection proves IDs are written privately and never appear in the bounded outcome.
func TestPrivateCreateOnlyProjection(t *testing.T) {
	args, path := invocationFixture(t)
	open := func(fd int) (io.ReadCloser, error) {
		if fd != 3 {
			t.Fatal("wrong descriptor")
		}
		return io.NopCloser(strings.NewReader(syntheticPage)), nil
	}
	var out bytes.Buffer
	if execute(args, &out, open) != nil || out.String() != "{\"identity_map_ready\":true,\"count\":\"1\"}\n" {
		t.Fatal("projection failed")
	}
	mapPath := filepath.Join(filepath.Dir(path), "identity-map.json")
	info, err := os.Stat(mapPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("map not private")
	}
	raw, err := os.ReadFile(mapPath)
	if err != nil || !bytes.Contains(raw, []byte("Transaction-A")) || bytes.Contains(raw, []byte("999999")) {
		t.Fatal("identity-only map not preserved")
	}
	out.Reset()
	if execute(args, &out, open) == nil || out.Len() != 0 {
		t.Fatal("existing map overwritten")
	}
	after, _ := os.ReadFile(mapPath)
	if !bytes.Equal(raw, after) {
		t.Fatal("immutable map changed")
	}
}

// TestPrivateInputCannotReadSavedPayloads distinguishes an actual private pipe from credential/payload files.
func TestPrivateInputCannotReadSavedPayloads(t *testing.T) {
	p := filepath.Join(t.TempDir(), "synthetic-input")
	if os.WriteFile(p, []byte("synthetic"), 0600) != nil {
		t.Fatal("input setup")
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal("input setup")
	}
	if _, err := privateInput(int(f.Fd())); err == nil {
		t.Fatal("saved input file accepted")
	}
	_ = f.Close()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal("pipe setup")
	}
	defer w.Close()
	input, err := privateInput(int(r.Fd()))
	if err != nil {
		t.Fatal("private pipe rejected")
	}
	_ = input.Close()
}
