package connections

import (
	"context"
	"testing"
	"time"
)

// profileSecrets models the separate current SQL/key gate, not authorization policy itself.
type profileSecrets struct {
	*memorySecrets
	denied bool
	checks int
}

// AuthorizeProvider injects the current preflight outcome and records invocation order.
func (s *profileSecrets) AuthorizeProvider(context.Context) error {
	s.checks++
	if s.denied {
		return failed("temporarily_unavailable")
	}
	return nil
}

// TestProductionPreflightBeforeBegin proves no durable attempt or provider call precedes current scope/key acceptance.
func TestProductionPreflightBeforeBegin(t *testing.T) {
	s, r, p, store := fixture()
	s.environment = "production"
	r.c.Environment = "production"
	ps := &profileSecrets{memorySecrets: store, denied: true}
	s.secrets = ps
	if _, e := s.Begin(context.Background(), scope, "production-begin", BeginRequest{Mode: "new"}); e == nil || r.begun || len(p.calls) != 0 || ps.checks != 1 {
		t.Fatal("denied preflight created work")
	}
	ps.denied = false
	out, e := s.Begin(context.Background(), scope, "production-begin", BeginRequest{Mode: "new"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Execute(context.Background(), scope, attemptID, ExecuteRequest{Nonce: out.Nonce, LinkToken: out.LinkToken, PublicToken: "synthetic-public"}); e != nil {
		t.Fatal(e)
	}
	if len(store.values) != 1 {
		t.Fatal("missing immutable credential")
	}
	for ref := range store.values {
		if ref.Environment != "production" {
			t.Fatal("Production reference fell back to Sandbox")
		}
	}
}

// TestProductionAttemptContextMismatch denies missing/foreign origin before provider work.
func TestProductionAttemptContextMismatch(t *testing.T) {
	for _, env := range []string{"", "sandbox", "Production"} {
		s, r, p, store := fixture()
		s.environment = "production"
		r.c.Environment = env
		s.secrets = &profileSecrets{memorySecrets: store}
		if _, e := s.Begin(context.Background(), scope, "production-begin", BeginRequest{Mode: "new"}); e == nil || len(p.calls) != 0 {
			t.Fatal("private projection crossed profile")
		}
	}
}

// TestLegacyServiceRejectsProduction cannot send a Production attempt to a legacy Sandbox provider.
func TestLegacyServiceRejectsProduction(t *testing.T) {
	s, r, p, _ := fixture()
	r.c.Environment = "production"
	if _, e := s.Begin(context.Background(), scope, "legacy-mismatch", BeginRequest{Mode: "new"}); e == nil || len(p.calls) != 0 {
		t.Fatal("legacy provider accepted Production projection")
	}
}

// TestProductionExpiredSQLRecovery retains discovery without provider or ciphertext adapters.
func TestProductionExpiredSQLRecovery(t *testing.T) {
	s, r, _, _ := fixture()
	s.environment = "production"
	s.provider = nil
	s.secrets = nil
	s.providerApproved = func() bool { return false }
	r.c.Environment = "production"
	if _, e := s.Attempt(context.Background(), scope, attemptID); e != nil {
		t.Fatal("SQL read blocked")
	}
	if _, e := s.Begin(context.Background(), scope, "denied", BeginRequest{Mode: "new"}); e == nil {
		t.Fatal("expired provider operation allowed")
	}
	if _, e := NewProductionWithApproval(nil, nil, nil, time.Now(), time.Now); e == nil {
		t.Fatal("missing SQL capability accepted")
	}
}
