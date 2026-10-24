package connections

import (
	"context"
	"testing"
	"time"
)

// TestExpiredApprovalDeniesProviderEntries proves expiry cannot claim work or read credentials.
func TestExpiredApprovalDeniesProviderEntries(t *testing.T) {
	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, instant := range []time.Time{deadline, deadline.Add(time.Nanosecond)} {
		s, repo, _, _ := fixture()
		s.providerApproved = func() bool { return deadline.After(instant) }
		ctx := context.Background()
		calls := []func() error{
			func() error { _, e := s.Begin(ctx, scope, "same-key", BeginRequest{Mode: "new"}); return e },
			func() error { _, e := s.Execute(ctx, scope, attemptID, ExecuteRequest{Nonce: nonceID}); return e },
			func() error { _, e := s.CompleteHosted(ctx, scope, attemptID); return e },
			func() error { _, e := s.Reconcile(ctx, scope, attemptID); return e },
			func() error { _, e := s.Bind(ctx, scope, attemptID, "same-key", BindingRequest{}); return e },
			func() error { _, e := s.Abandon(ctx, scope, attemptID); return e },
		}
		// Nil adapters are tripwires: any provider/secret use would panic or fail differently.
		s.provider, s.secrets = nil, nil
		for i, call := range calls {
			code(t, call(), "temporarily_unavailable")
			if len(repo.calls) != 0 {
				t.Fatalf("entry %d reached SQL", i)
			}
		}
	}
}

// TestExpiredApprovalPreservesSQLAuthority keeps metadata/control operations under current SQL denial.
func TestExpiredApprovalPreservesSQLAuthority(t *testing.T) {
	r := &operationRepo{}
	s := &Service{repo: r, providerApproved: func() bool { return false }}
	ctx := context.Background()
	calls := []struct {
		op  string
		run func() error
	}{
		{"get_connection_attempt", func() error { _, e := s.Attempt(ctx, scope, attemptID); return e }},
		{"list_connections", func() error { _, e := s.List(ctx, scope); return e }},
		{"list_connection_attempts", func() error { _, e := s.Attempts(ctx, scope, 10, nil); return e }},
		{"connection_stream_status", func() error { _, e := s.StreamStatus(ctx, scope, attemptID); return e }},
		{"disconnect_stream", func() error { _, e := s.Disconnect(ctx, scope, attemptID, "same-key", "1"); return e }},
		{"mutate_connection_stream", func() error {
			_, e := s.OperateStream(ctx, scope, attemptID, "same-key", StreamOperation{"pause", "1", "1", "1"})
			return e
		}},
	}
	for _, c := range calls {
		r.err = nil
		if e := c.run(); e != nil || r.op != c.op {
			t.Fatalf("SQL operation %s unavailable: %v", c.op, e)
		}
		r.err = failed("not_found")
		code(t, c.run(), "not_found")
	}
}

// TestApprovalDeadlineTransition rejects at the exact deadline without sleeping or replacing the service.
func TestApprovalDeadlineTransition(t *testing.T) {
	s, _, _, _ := fixture()
	deadline := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	now := deadline.Add(-time.Nanosecond)
	s.providerApproved = func() bool { return deadline.After(now) }
	if e := s.ready(scope); e != nil {
		t.Fatal(e)
	}
	now = deadline
	code(t, s.ready(scope), "temporarily_unavailable")
	now = deadline.Add(time.Nanosecond)
	code(t, s.ready(scope), "temporarily_unavailable")
}
