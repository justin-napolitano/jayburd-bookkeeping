package ledger

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestImportCommandFailureRedaction proves new SQL failures remain safe at the existing recovery HTTP boundary.
func TestImportCommandFailureRedaction(t *testing.T) {
	for _, tc := range []struct {
		state, code string
		status      int
	}{
		{"P2400", "invalid_import", 400}, {"P2401", "import_denied", 404},
		{"P2402", "import_conflict", 409}, {"P2403", "stale_preview", 409},
		{"P2404", "import_limit", 422},
	} {
		t.Run(tc.state, func(t *testing.T) {
			private := "synthetic-private-description"
			err := failure(&pgconn.PgError{Code: tc.state, Message: private, Detail: private})
			ops := &recoveryStub{operationsStub: &operationsStub{err: err}}
			req := httptest.NewRequest("POST", "/v1/commands/synthetic-command/execute", strings.NewReader("{}"))
			req.Header.Set("Authorization", "Bearer synthetic")
			response := httptest.NewRecorder()
			Handler(authStub{}, ops).ServeHTTP(response, req)
			var body struct{ Code string }
			if e := json.Unmarshal(response.Body.Bytes(), &body); e != nil || body.Code != tc.code || response.Code != tc.status {
				t.Fatal("import recovery failure contract changed")
			}
			if strings.Contains(response.Body.String(), private) || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("import error exposed protected details or permitted caching")
			}
		})
	}
}
