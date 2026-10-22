package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhookhost"
)

// arguments supplies public synthetic resource names with no actual deployment or credentials.
func arguments() []string {
	return []string{"--cloud-sql-instance=synthetic-project:us-central1:synthetic-db", "--project-number=123456", "--plaid-client-id-version=projects/123456/secrets/jayburd-dev-bank-sync-client-id/versions/1", "--plaid-client-secret-version=projects/123456/secrets/jayburd-dev-bank-sync-client-secret/versions/2"}
}

// TestParse permits only explicit reviewed public configuration and fixed local listener default.
func TestParse(t *testing.T) {
	c, err := parse(arguments())
	if err != nil || c.Listen != "127.0.0.1:8080" {
		t.Fatal("valid identifiers rejected")
	}
	for _, args := range [][]string{nil, append(arguments(), "--role=bookkeeping_api"), append(arguments(), "--dsn=synthetic-private"), append(arguments(), "--secret=synthetic-private"), append(arguments(), "--environment=production"), append(arguments(), "extra"), append(arguments(), "--listen=localhost:8080")} {
		if _, err := parse(args); err != webhookhost.ErrConfiguration || strings.Contains(err.Error(), "synthetic") {
			t.Fatal("unsafe flag accepted or reflected")
		}
	}
}

// fakeRuntime observes lifecycle only and never opens a socket or contacts an external service.
type fakeRuntime struct {
	runs, closes int
	err          error
}

func (f *fakeRuntime) Run(context.Context) error { f.runs++; return f.err }
func (f *fakeRuntime) Close()                    { f.closes++ }

// TestExecute validates before constructor, preserves cancellation and closes after runtime failure.
func TestExecute(t *testing.T) {
	never := func(context.Context, webhookhost.Config) (runtime, error) {
		t.Fatal("invalid or cancelled command initialized runtime")
		return nil, nil
	}
	if execute(context.Background(), nil, never) != webhookhost.ErrConfiguration {
		t.Fatal("missing identifiers accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if execute(ctx, arguments(), never) != webhookhost.ErrServe {
		t.Fatal("cancelled command initialized")
	}
	if execute(nil, arguments(), never) != webhookhost.ErrConfiguration {
		t.Fatal("nil context accepted")
	}
	for _, runErr := range []error{nil, webhookhost.ErrServe} {
		f := &fakeRuntime{err: runErr}
		if err := execute(context.Background(), arguments(), func(context.Context, webhookhost.Config) (runtime, error) { return f, nil }); err != runErr || f.runs != 1 || f.closes != 1 {
			t.Fatal("runtime lifecycle incorrect")
		}
	}
	if err := execute(context.Background(), arguments(), func(context.Context, webhookhost.Config) (runtime, error) {
		return nil, errors.New("synthetic-private-diagnostic")
	}); err != webhookhost.ErrServe {
		t.Fatal("constructor diagnostic propagated")
	}
	partial := &fakeRuntime{}
	if err := execute(context.Background(), arguments(), func(context.Context, webhookhost.Config) (runtime, error) {
		return partial, errors.New("synthetic-private-diagnostic")
	}); err != webhookhost.ErrServe || partial.closes != 1 || partial.runs != 0 {
		t.Fatal("partial constructor resource leaked")
	}
}
