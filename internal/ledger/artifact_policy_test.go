package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TestArtifactPublicBoundary proves the application constructor cannot select primitive posting.
func TestArtifactPublicBoundary(t *testing.T) {
	for _, tc := range []struct {
		path, body, token string
		status            int
	}{
		{"/v1/entities/e/books/b/journals", "not json", "", 401},
		{"/v1/entities/e/books/b/journals/j/reversals", "not json", "", 401},
		{"/v1/entities/e/books/b/journals", "not json", "Bearer synthetic", 422},
		{"/v1/entities/e/books/b/journals/j/reversals", "{}", "Bearer synthetic", 422},
		{"/v1/commands", `{"operation":"post","entity_id":"e","book_id":"b","body":{}}`, "Bearer synthetic", 422},
		{"/v1/commands", `{"operation":"post"}`, "", 401},
		{"/v1/commands", `{"operation":"future_post"}`, "Bearer synthetic", 400},
		{"/v1/commands", `{}`, "Bearer synthetic", 400},
		{"/v1/commands", `{"operation":null}`, "Bearer synthetic", 400},
		{"/v1/commands", `{"operation":42}`, "Bearer synthetic", 400},
	} {
		ops := &recoveryStub{operationsStub: &operationsStub{}}
		r := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
		r.Header.Set("Authorization", tc.token)
		r.Header.Set("Idempotency-Key", "synthetic")
		w := httptest.NewRecorder()
		Handler(authStub{}, ops).ServeHTTP(w, r)
		if w.Code != tc.status || ops.calls != 0 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("posting reached operations: %d %d", w.Code, ops.calls)
		}
		if tc.status == 422 && !strings.Contains(w.Body.String(), `"code":"artifact_required"`) {
			t.Fatal("missing fixed evidence requirement")
		}
		if tc.status == 401 && w.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Fatal("missing authentication challenge")
		}
	}
	for _, path := range []string{"/v1/commands", "/v1/commands/id/execute"} {
		// An adapter with generic Recovery but no artifact-aware port must not become a fallback.
		ops := &genericRecoveryOnly{operationsStub: &operationsStub{}}
		r := httptest.NewRequest("POST", path, strings.NewReader(`{"operation":"source_post"}`))
		r.Header.Set("Authorization", "Bearer synthetic")
		r.Header.Set("Idempotency-Key", "synthetic")
		w := httptest.NewRecorder()
		Handler(authStub{}, ops).ServeHTTP(w, r)
		if w.Code != 503 || ops.calls != 0 {
			t.Fatalf("unsafe generic fallback %d %d", w.Code, ops.calls)
		}
	}
}

type genericRecoveryOnly struct{ *operationsStub }

func (s *genericRecoveryOnly) Recovery(ctx context.Context, c Command, _ string) (Result, error) {
	return s.Post(ctx, c)
}

// policyDB owns explicit synthetic attempts; no row result is an accounting oracle.
type policyDB struct {
	txs     []*policyTx
	calls   int
	options []pgx.TxOptions
}

func (d *policyDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.options = append(d.options, o)
	if d.calls >= len(d.txs) {
		return nil, errors.New("unexpected extra attempt")
	}
	tx := d.txs[d.calls]
	d.calls++
	return tx, nil
}

type policyTx struct {
	pgx.Tx
	pending                              string
	canonical                            string
	pendingErr, transitionErr, commitErr error
	queries                              []string
	args                                 [][]any
	commits, rollbacks                   int
	cleanupOK                            bool
}

func (t *policyTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *policyTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.queries = append(t.queries, q)
	t.args = append(t.args, args)
	return policyRow{tx: t, pending: strings.Contains(q, "pending_command") && len(args) == 2}
}
func (t *policyTx) Commit(context.Context) error { t.commits++; return t.commitErr }
func (t *policyTx) Rollback(ctx context.Context) error {
	t.rollbacks++
	_, deadline := ctx.Deadline()
	t.cleanupOK = deadline && ctx.Err() == nil
	return nil
}

type policyRow struct {
	tx      *policyTx
	pending bool
}

func (r policyRow) Scan(dest ...any) error {
	if r.pending {
		if r.tx.pendingErr != nil {
			return r.tx.pendingErr
		}
		*dest[0].(*json.RawMessage) = json.RawMessage(r.tx.pending)
		*dest[1].(*string) = r.tx.canonical
		return nil
	}
	if r.tx.transitionErr != nil {
		return r.tx.transitionErr
	}
	*dest[0].(*json.RawMessage) = json.RawMessage(`{"state":"committed","receipt":"synthetic"}`)
	return nil
}

const artifactPolicyCommandID = "81000000-0000-4000-8000-000000000007"

func policyPending(operation, state string) string {
	return `{"command":{"command_id":"` + artifactPolicyCommandID + `","operation":"` + operation + `","state":"` + state + `"}}`
}

// TestArtifactExecutionTransaction proves check and transition use one serializable attempt with no denied effects.
func TestArtifactExecutionTransaction(t *testing.T) {
	for _, tc := range []struct {
		name, pending    string
		rowErr           error
		code             string
		queries, commits int
	}{
		{"prepared_manual", policyPending("post", "prepared"), nil, "artifact_required", 1, 0},
		{"committed_manual_receipt", policyPending("post", "committed"), nil, "", 2, 1},
		{"cancelled_history", policyPending("post", "cancelled"), nil, "", 2, 1},
		{"source_post", policyPending("source_post", "prepared"), nil, "", 2, 1},
		{"setup", policyPending("setup", "prepared"), nil, "", 2, 1},
		{"import", policyPending("import", "prepared"), nil, "", 2, 1},
		{"register_accounts", policyPending("register_accounts", "prepared"), nil, "", 2, 1},
		{"acknowledged_history", `{"command":null}`, nil, "", 2, 1},
		{"malformed_authority", `{}`, nil, "temporarily_unavailable", 1, 0},
		{"future_operation", policyPending("future_post", "prepared"), nil, "temporarily_unavailable", 1, 0},
		{"unknown_historical_operation", policyPending("future_post", "committed"), nil, "temporarily_unavailable", 1, 0},
		{"denied_current_authority", "", &pgconn.PgError{Code: "P2102"}, "permission_denied", 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx := &policyTx{pending: tc.pending, canonical: artifactPolicyCommandID, pendingErr: tc.rowErr}
			db := &policyDB{txs: []*policyTx{tx}}
			s := Service{DB: db}
			c := Command{Actor: "synthetic-actor", CommandID: artifactPolicyCommandID, Body: `{}`, Request: "synthetic-request"}
			r, err := s.ArtifactRecovery(context.Background(), c, "execute")
			if tc.code == "" {
				if err != nil || r.Status != 200 || len(r.Body) == 0 {
					t.Fatal("missing committed response", err)
				}
			} else {
				var f *Failure
				if !errors.As(err, &f) || f.Code != tc.code || len(r.Body) != 0 {
					t.Fatal("unsafe policy error", err)
				}
				if tc.code == "artifact_required" && f.Status != 422 {
					t.Fatal("policy failure lost fixed422")
				}
			}
			if len(tx.queries) != tc.queries || tx.commits != tc.commits || tx.rollbacks != 1 || !tx.cleanupOK || db.options[0].IsoLevel != pgx.Serializable {
				t.Fatal("unsafe transaction boundary", tx.queries, tx.commits)
			}
			if tx.args[0][0] != c.Actor || tx.args[0][1] != c.CommandID {
				t.Fatal("authority identity changed")
			}
			if tc.queries == 2 && (!strings.Contains(tx.queries[1], "transition_command") || tx.args[1][1] != c.CommandID || tx.args[1][3] != "execute" || tx.args[1][4] != c.Body) {
				t.Fatal("wrong transition")
			}
		})
	}
}

// TestArtifactHTTPExecution uses the real service policy behind the public constructor, never a permissive stub.
func TestArtifactHTTPExecution(t *testing.T) {
	for _, state := range []string{"prepared", "committed"} {
		t.Run(state, func(t *testing.T) {
			tx := &policyTx{pending: policyPending("post", state), canonical: artifactPolicyCommandID}
			db := &policyDB{txs: []*policyTx{tx}}
			r := httptest.NewRequest("POST", "/v1/commands/"+artifactPolicyCommandID+"/execute", strings.NewReader(`{}`))
			r.Header.Set("Authorization", "Bearer synthetic")
			w := httptest.NewRecorder()
			Handler(authStub{}, &Service{DB: db}).ServeHTTP(w, r)
			if state == "prepared" {
				if w.Code != 422 || len(tx.queries) != 1 || tx.commits != 0 || strings.Contains(w.Body.String(), "receipt") {
					t.Fatal("public execute bypassed the evidence policy", w.Code, tx.queries)
				}
			} else if w.Code != 200 || len(tx.queries) != 2 || tx.commits != 1 || !strings.Contains(w.Body.String(), `"receipt":"synthetic"`) {
				t.Fatal("historical committed replay changed", w.Code, tx.queries)
			}
		})
	}
}

// TestArtifactRetriesAndReplay retains known-abort retry and unknown-commit redaction semantics.
func TestArtifactRetriesAndReplay(t *testing.T) {
	for _, mode := range []string{"pending_abort", "commit_abort", "commit_unknown"} {
		unknown := mode == "commit_unknown"
		first := &policyTx{pending: policyPending("source_post", "prepared"), canonical: artifactPolicyCommandID}
		if unknown {
			first.commitErr = errors.New("synthetic lost acknowledgement")
		} else if mode == "commit_abort" {
			first.commitErr = &pgconn.PgError{Code: "40001"}
		} else {
			first.pendingErr = &pgconn.PgError{Code: "40001"}
		}
		second := &policyTx{pending: policyPending("post", "prepared"), canonical: artifactPolicyCommandID}
		db := &policyDB{txs: []*policyTx{first, second}}
		r, err := (&Service{DB: db}).ArtifactRecovery(context.Background(), Command{CommandID: artifactPolicyCommandID, Body: `{}`}, "execute")
		var f *Failure
		if !errors.As(err, &f) || len(r.Body) != 0 {
			t.Fatal("exposed uncommitted result")
		}
		if unknown {
			if db.calls != 1 || f.Code != "temporarily_unavailable" {
				t.Fatal("retried unknown commit")
			}
		} else if db.calls != 2 || f.Code != "artifact_required" || len(second.queries) != 1 {
			t.Fatal("retry did not recheck current authority")
		}
	}
	// UUID spellings accepted by PostgreSQL cannot bypass matching the current manual command.
	tx := &policyTx{pending: policyPending("post", "prepared"), canonical: artifactPolicyCommandID}
	_, err := (&Service{DB: &policyDB{txs: []*policyTx{tx}}}).ArtifactRecovery(context.Background(), Command{CommandID: strings.ReplaceAll(artifactPolicyCommandID, "-", "")}, "execute")
	if f := failure(err); f.Code != "artifact_required" || len(tx.queries) != 1 {
		t.Fatal("noncanonical identifier bypassed policy")
	}
}

// TestArtifactRegistrationAndSafeTransitions preserves exact permitted bodies and safe terminal operations.
func TestArtifactRegistrationAndSafeTransitions(t *testing.T) {
	db := &policyDB{}
	_, err := (&Service{DB: db}).ArtifactRecovery(context.Background(), Command{Body: `{"operation":"post"}`}, "register")
	if f := failure(err); f.Code != "artifact_required" || db.calls != 0 {
		t.Fatal("new manual command reached SQL")
	}
	for _, raw := range []string{`{}`, `{"operation":null}`, `{"operation":"future_post"}`, `{"operation":42}`} {
		db := &policyDB{}
		_, err := (&Service{DB: db}).ArtifactRecovery(context.Background(), Command{Body: raw}, "register")
		var f *Failure
		if !errors.As(err, &f) || f.Code != "invalid_request" || f.Status != 400 || db.calls != 0 {
			t.Fatal("unknown command reached SQL", err)
		}
	}
	for _, operation := range []string{"setup", "source_post", "import", "register_accounts"} {
		raw := `{"operation":"` + operation + `","body":{"x":1,"x":2}}`
		tx := &policyTx{}
		db := &policyDB{txs: []*policyTx{tx}}
		_, err := (&Service{DB: db}).ArtifactRecovery(context.Background(), Command{Body: raw}, "register")
		if err != nil || tx.args[0][3] != raw || !strings.Contains(tx.queries[0], "register_command") {
			t.Fatal("allowed raw registration changed", err)
		}
	}
	// Duplicate validation remains SQL-owned: even top-level duplicates are sent unchanged for a known operation.
	raw := `{"operation":"source_post","operation":"source_post","body":{"x":1,"x":2}}`
	tx := &policyTx{}
	_, err = (&Service{DB: &policyDB{txs: []*policyTx{tx}}}).ArtifactRecovery(context.Background(), Command{Body: raw}, "register")
	if err != nil || tx.args[0][3] != raw {
		t.Fatal("duplicate raw keys were normalized before SQL", err)
	}
	for _, action := range []string{"cancel", "acknowledge", "pending", "clear"} {
		tx := &policyTx{}
		db := &policyDB{txs: []*policyTx{tx}}
		_, err := (&Service{DB: db}).ArtifactRecovery(context.Background(), Command{Body: `{}`}, action)
		if err != nil || len(tx.queries) != 1 || strings.Contains(tx.queries[0], "post_journal") {
			t.Fatal("safe recovery changed", action, err)
		}
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	tx = &policyTx{}
	db = &policyDB{txs: []*policyTx{tx}}
	_, err = (&Service{DB: db}).ArtifactRecovery(ctx, Command{}, "execute")
	if failure(err).Code != "temporarily_unavailable" || len(tx.queries) != 0 {
		t.Fatal("expired context reached policy SQL")
	}
}
