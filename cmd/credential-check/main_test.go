package main

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// args supplies only public identifiers and synthetic numeric pins; no default transport is used.
func args() []string {
	return []string{"-project-id=jayburd-bookkeeping-dev", "-project-number=9088794002", "-client-id-version=7", "-client-secret-version=9", "-approval-expires-at=" + time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05Z")}
}

// TestCredentialCheckInvocation proves pre-I/O denial, explicit execution and fixed output on private dependency errors.
func TestCredentialCheckInvocation(t *testing.T) {
	var out bytes.Buffer
	calls := 0
	execute := func(ctx context.Context, c config) error {
		calls++
		d, ok := ctx.Deadline()
		if !ok || d.Format("2006-01-02T15:04:05Z") != c.expiresAt {
			t.Fatal("missing finite context")
		}
		return nil
	}
	if run(context.Background(), append(args(), "-check"), &out, execute) != 0 || calls != 0 {
		t.Fatal("offline check called dependency")
	}
	out.Reset()
	if run(context.Background(), append(args(), "-execute"), &out, execute) != 0 || calls != 1 || !strings.Contains(out.String(), "provider authentication untested") {
		t.Fatal("explicit readback failed")
	}
	for _, pair := range [][2]string{{"9088794002", "123456789012"}, {"jayburd-bookkeeping-dev", "foreign-project"}, {"-client-id-version=7", "-client-id-version=latest"}, {"-client-secret-version=9", "-client-secret-version=01"}, {"-client-secret-version=9", "-client-secret-version=9223372036854775808"}} {
		bad := args()
		for i := range bad {
			bad[i] = strings.ReplaceAll(bad[i], pair[0], pair[1])
		}
		out.Reset()
		if run(context.Background(), append(bad, "-execute"), &out, execute) == 0 || calls != 1 {
			t.Fatal("invalid profile reached dependency")
		}
	}
	for _, expiry := range []time.Time{time.Now().Add(-time.Hour), time.Now().Add(8 * 24 * time.Hour)} {
		bad := args()
		bad[len(bad)-1] = "-approval-expires-at=" + expiry.UTC().Format("2006-01-02T15:04:05Z")
		if run(context.Background(), append(bad, "-execute"), &out, execute) == 0 || calls != 1 {
			t.Fatal("expired/excess authority reached dependency")
		}
	}
	for _, bad := range [][]string{nil, args(), append(args(), "-check", "-execute"), append(args(), "-execute", "-secret=synthetic-private")} {
		if run(context.Background(), bad, &out, execute) == 0 || calls != 1 {
			t.Fatal("unsafe invocation reached dependency")
		}
	}
	out.Reset()
	if run(context.Background(), append(args(), "-execute"), &out, func(context.Context, config) error { return errors.New("synthetic-private-value") }) == 0 || out.String() != "PARTIAL: credential readback unconfirmed\n" {
		t.Fatal("dependency diagnostic escaped")
	}
}

// fakeReader observes exact resources and retains test buffers solely to check disposal.
type fakeReader struct {
	names []string
	data  [][]byte
}

// Read supplies only synthetic buffers and never constructs a transport.
func (r *fakeReader) Read(_ context.Context, name string) ([]byte, error) {
	r.names = append(r.names, name)
	data := []byte("synthetic-private-value")
	r.data = append(r.data, data)
	return data, nil
}

// TestReadbackPins verifies both literal resources and that caller-visible buffers are cleared.
func TestReadbackPins(t *testing.T) {
	r := &fakeReader{}
	if readback(context.Background(), config{number: "9088794002", idVersion: "7", secretVersion: "9"}, r) != nil || !reflect.DeepEqual(r.names, []string{"projects/9088794002/secrets/jayburd-prod-bank-sync-client-id/versions/7", "projects/9088794002/secrets/jayburd-prod-bank-sync-client-secret/versions/9"}) {
		t.Fatal("wrong static resource")
	}
	for _, data := range r.data {
		if !bytes.Equal(data, make([]byte, len(data))) {
			t.Fatal("payload buffer retained")
		}
	}
}
