package ledger

import (
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
)

// TestRegistrationCommandFailures keeps execute-time SQL denials actionable without leaking private error detail.
func TestRegistrationCommandFailures(t *testing.T) {
	for _, tc := range []struct {
		sql, code string
		status    int
	}{{"P2700", "invalid_request", 400}, {"P2701", "not_found", 404}, {"P2702", "connection_conflict", 409}, {"P2703", "stale_connection", 409}, {"P2704", "registration_not_ready", 409}, {"P2705", "connection_limit", 422}} {
		f := failure(&pgconn.PgError{Code: tc.sql, Message: "private synthetic account detail"})
		if f.Code != tc.code || f.Status != tc.status || strings.Contains(f.Error(), "private") {
			t.Fatal("registration denial lost classification or redaction")
		}
	}
}
