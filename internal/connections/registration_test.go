package connections

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// registrationRepo observes the resolved SQL boundary without modeling authority or provider behavior.
type registrationRepo struct {
	*memoryRepo
	args    []any
	failure error
}

// call records new preparation while delegating private context to the existing fixture.
func (r *registrationRepo) call(ctx context.Context, op string, args ...any) (json.RawMessage, error) {
	if op != "prepare_registration" && op != "registration_options" {
		return r.memoryRepo.call(ctx, op, args...)
	}
	r.calls = append(r.calls, op)
	r.args = append([]any(nil), args...)
	return json.RawMessage(`{"command_id":"00000000-0000-4000-8000-000000000006","operation":"register_accounts","state":"prepared"}`), r.failure
}

// TestRegistrationOptionsTransport keeps the SQL snapshot scoped and never loads bank credentials.
func TestRegistrationOptionsTransport(t *testing.T) {
	s, r, p, store := fixture()
	repo := &registrationRepo{memoryRepo: r}
	s.repo = repo
	_, err := s.RegistrationOptions(context.Background(), scope, attemptID)
	if err != nil || !reflect.DeepEqual(repo.args, []any{actorID, entityID, bookID, attemptID}) || len(p.calls) != 0 || store.reads != 0 || store.writes != 0 {
		t.Fatal("options escaped scoped SQL-only boundary")
	}
	repo.failure = failed("credential_recovery_required")
	_, err = s.RegistrationOptions(context.Background(), scope, attemptID)
	code(t, err, "registration_not_ready")
	for _, expired := range []bool{false, true} {
		s, r, p, store := fixture()
		id := "invalid"
		want := "invalid_request"
		if expired {
			s.providerApproved = func() bool { return false }
			id, want = attemptID, "temporarily_unavailable"
		}
		_, err := s.RegistrationOptions(context.Background(), scope, id)
		code(t, err, want)
		if len(r.calls) != 0 || len(p.calls) != 0 || store.reads != 0 {
			t.Fatal("invalid/expired options touched dependencies")
		}
	}
}

// TestRegistrationSelectionTransport pins trusted actor/scope/key/audit identity and never calls provider/crypto.
func TestRegistrationSelectionTransport(t *testing.T) {
	s, r, p, store := fixture()
	repo := &registrationRepo{memoryRepo: r}
	s.repo = repo
	in := RegistrationSelection{AccountIDs: []string{"selected-one", "selected-two"}}
	first, err := s.PrepareRegistration(context.Background(), scope, attemptID, "saved-selection", claimID, in)
	if err != nil {
		t.Fatal(err)
	}
	expected := []any{actorID, entityID, bookID, attemptID, "saved-selection", claimID, `{"account_ids":["selected-one","selected-two"]}`}
	if !reflect.DeepEqual(repo.args, expected) || len(p.calls) != 0 || store.reads != 0 || store.writes != 0 {
		t.Fatal("selection escaped trusted SQL-only boundary")
	}
	again, err := s.PrepareRegistration(context.Background(), scope, attemptID, "saved-selection", claimID, in)
	if err != nil || string(first) != string(again) || !reflect.DeepEqual(repo.args, expected) {
		t.Fatal("retry changed identity or selection")
	}
	repo.failure = failed("credential_recovery_required")
	_, err = s.PrepareRegistration(context.Background(), scope, attemptID, "unsupported", claimID, in)
	code(t, err, "registration_not_ready")
}

// TestRegistrationSelectionDenials rejects malformed selection before even the scoped SQL context read.
func TestRegistrationSelectionDenials(t *testing.T) {
	cases := []RegistrationSelection{{}, {}, {AccountIDs: []string{""}}, {AccountIDs: []string{"one", "one"}}, {AccountIDs: []string{"not printable"}}, {AccountIDs: []string{strings.Repeat("x", 257)}}}
	for n := 0; n < 26; n++ {
		cases[0].AccountIDs = append(cases[0].AccountIDs, strings.Repeat("x", n+1))
	}
	for _, in := range cases {
		s, r, p, store := fixture()
		_, err := s.PrepareRegistration(context.Background(), scope, attemptID, "selection", claimID, in)
		code(t, err, "invalid_request")
		if len(r.calls) != 0 || len(p.calls) != 0 || store.reads != 0 {
			t.Fatal("invalid selection reached dependency")
		}
	}
	s, r, _, _ := fixture()
	s.providerApproved = func() bool { return false }
	_, err := s.PrepareRegistration(context.Background(), scope, attemptID, "selection", claimID, RegistrationSelection{[]string{"one"}})
	code(t, err, "temporarily_unavailable")
	if len(r.calls) != 0 {
		t.Fatal("expired runtime prepared new authority")
	}
	for _, change := range []string{"profile", "preflight"} {
		s, r, p, store := fixture()
		s.environment = "production"
		r.c.Environment = "production"
		scoped := &profileSecrets{memorySecrets: store}
		s.secrets = scoped
		if change == "profile" {
			r.c.Environment = "sandbox"
		} else {
			scoped.denied = true
		}
		_, err = s.PrepareRegistration(context.Background(), scope, attemptID, "selection", claimID, RegistrationSelection{[]string{"one"}})
		if err == nil || len(p.calls) != 0 || store.reads != 0 {
			t.Fatal("profile denial permitted work")
		}
		for _, op := range r.calls {
			if op == "prepare_registration" {
				t.Fatal("profile mismatch prepared command")
			}
		}
	}
}

// TestRegistrationPendingFailure exposes only the stable command gate code, never PostgreSQL detail.
func TestRegistrationPendingFailure(t *testing.T) {
	err := sqlFailure(&pgconn.PgError{Code: "P2112", Message: "private synthetic detail"})
	code(t, err, "pending_command")
	if strings.Contains(err.Error(), "private") {
		t.Fatal("database detail escaped")
	}
}
