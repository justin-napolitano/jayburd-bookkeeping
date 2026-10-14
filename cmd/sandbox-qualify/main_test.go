package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestOfflineAndExplicitGuards ensures default/help/invalid modes cannot touch a descriptor or transport.
func TestOfflineAndExplicitGuards(t *testing.T) {
	for _, args := range [][]string{nil, {"-mode", "offline", "-allow-sandbox", "-input-fd", "3"}, {"-mode", "sandbox-sync"}, {"-mode", "sandbox-sync", "-allow-sandbox", "-input-fd", "0"}, {"-mode", "sandbox-sync", "-allow-sandbox", "-input-fd", "2"}, {"-mode", "production"}, {"-secret=must-not-echo"}} {
		r := execute(context.Background(), args, func(int) (io.ReadCloser, error) { t.Fatal("input read without permission"); return nil, nil }, func() http.RoundTripper { t.Fatal("network constructed"); return nil })
		if r.Status != "blocked" {
			t.Fatal("unqualified success")
		}
	}
}

// TestMalformedPipeDoesNotConstructTransport rejects owner input before any network resource is allocated.
func TestMalformedPipeDoesNotConstructTransport(t *testing.T) {
	r := execute(context.Background(), []string{"-mode", "sandbox-sync", "-allow-sandbox", "-input-fd", "3"}, func(int) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(`{"secret":"hidden"}`)), nil }, func() http.RoundTripper { t.Fatal("network constructed"); return nil })
	if r.Status != "blocked" || r.Checks[0].Reason != "input_invalid" {
		t.Fatal("malformed input accepted")
	}
}

// TestInheritedPipeAndDeadline exercises a real descriptor while rejecting ordinary files.
func TestInheritedPipeAndDeadline(t *testing.T) {
	reader, writer, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer reader.Close()
	defer writer.Close()
	fd, e := syscall.Dup(int(reader.Fd()))
	if e != nil {
		t.Fatal(e)
	}
	accepted, e := readPipe(fd)
	if e != nil {
		t.Fatal(e)
	}
	defer accepted.Close()
	file := accepted.(*os.File)
	if e = file.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); e != nil {
		t.Fatal(e)
	}
	var b [1]byte
	if _, e = accepted.Read(b[:]); e == nil {
		t.Fatal("pipe read was not bounded")
	}
	regular, e := os.CreateTemp(t.TempDir(), "synthetic")
	if e != nil {
		t.Fatal(e)
	}
	defer regular.Close()
	if r, e := readPipe(int(regular.Fd())); e == nil {
		r.Close()
		t.Fatal("regular file accepted")
	}
}
