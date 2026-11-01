package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// arguments is independent public synthetic scope; no secret values are command inputs.
func arguments() []string {
	return []string{"-action=run", "-allow-sandbox", "-cloud-sql-instance=" + instance, "-project-number=" + projectNumber, "-expires-at=" + time.Now().Add(5*time.Minute).UTC().Format(time.RFC3339), "-entity-id=00000000-0000-4000-8000-000000000001", "-book-id=00000000-0000-4000-8000-000000000002", "-stream-id=00000000-0000-4000-8000-000000000003", "-claim-id=00000000-0000-4000-8000-000000000004", "-client-id-version=1", "-client-secret-version=2"}
}

func TestCommandOfflineAndGuards(t *testing.T) {
	called := 0
	execute := func(context.Context, options) (result, error) { called++; return result{true, true}, nil }
	var out bytes.Buffer
	if run(context.Background(), nil, &out, execute) != nil || out.String() != "{\"offline\":true}\n" || called != 0 {
		t.Fatal("default not inert")
	}
	for _, change := range []struct{ old, new string }{
		{"-allow-sandbox", "-allow-sandbox=false"},
		{"-action=run", "-action=production"},
		{"-project-number=" + projectNumber, "-project-number=999999999"},
		{"-cloud-sql-instance=" + instance, "-cloud-sql-instance=jayburd-bookkeeping-dev:us-central1:jayburd-prod"},
		{"-client-id-version=1", "-client-id-version=latest"},
		{"-client-secret-version=2", "-client-secret-version=9223372036854775808"},
		{"-stream-id=00000000-0000-4000-8000-000000000003", "-stream-id=00000000-0000-0000-0000-000000000000"},
	} {
		args := arguments()
		for i, a := range args {
			if a == change.old {
				args[i] = change.new
			}
		}
		out.Reset()
		if run(context.Background(), args, &out, execute) == nil || called != 0 || out.Len() != 0 {
			t.Fatal("invalid invocation reached executor")
		}
	}
	for _, deadline := range []string{"2020-01-01T00:00:00Z", time.Now().Add(time.Hour).UTC().Format(time.RFC3339), time.Now().Add(time.Minute).UTC().Format("2006-01-02T15:04:05+00:00")} {
		args := arguments()
		for i, a := range args {
			if strings.HasPrefix(a, "-expires-at=") {
				args[i] = "-expires-at=" + deadline
			}
		}
		if run(context.Background(), args, &out, execute) == nil || called != 0 {
			t.Fatal("invalid expiry reached executor")
		}
	}
	args := append(arguments(), "-provider-environment=production")
	if run(context.Background(), args, &out, execute) == nil || called != 0 {
		t.Fatal("Production option accepted")
	}
}

func TestCommandScopedExecution(t *testing.T) {
	var out bytes.Buffer
	err := run(context.Background(), arguments(), &out, func(ctx context.Context, o options) (result, error) {
		deadline, ok := ctx.Deadline()
		if !ok || !deadline.Equal(o.expires) || o.scope.BookID != "00000000-0000-4000-8000-000000000002" || o.clientSecretVersion != "2" {
			t.Fatal("lost scoped metadata")
		}
		return result{true, true}, nil
	})
	if err != nil || out.String() != "{\"webhook_requested\":true,\"sync_completed\":true}\n" {
		t.Fatal("wrong Boolean result")
	}
	out.Reset()
	if run(context.Background(), arguments(), &out, func(context.Context, options) (result, error) { return result{true, false}, nil }) == nil {
		t.Fatal("incomplete sync passed")
	}
}
