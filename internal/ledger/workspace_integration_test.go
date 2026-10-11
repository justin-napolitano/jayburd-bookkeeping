//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
)

// workspaceResult reads only the new bounded discovery contract, not private tables.
type workspaceResult struct {
	Entities []struct {
		ID          string   `json:"entity_id"`
		Permissions []string `json:"permissions"`
		Books       []struct {
			ID       string            `json:"book_id"`
			Revision string            `json:"book_revision"`
			Accounts []json.RawMessage `json:"accounts"`
			Periods  []json.RawMessage `json:"periods"`
			Policies []json.RawMessage `json:"policies"`
		} `json:"books"`
	} `json:"entities"`
	Currencies []json.RawMessage `json:"currencies"`
	Policy     string            `json:"development_policy"`
}

// readWorkspace fails tests on incomplete discovery rather than silently using zero values.
func readWorkspace(t *testing.T, s *Service) workspaceResult {
	t.Helper()
	r, err := s.Workspace(context.Background(), Command{Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	var v workspaceResult
	if r.Status != 200 || json.Unmarshal(r.Body, &v) != nil {
		t.Fatal("invalid workspace")
	}
	return v
}

// TestWorkspaceAuthorization proves read-only grants, hidden scopes and setup discovery.
func TestWorkspaceAuthorization(t *testing.T) {
	c, s, db := fixtureDB(t, "workspace")
	v := readWorkspace(t, s)
	if len(v.Entities) != 1 || v.Entities[0].ID != entity || len(v.Entities[0].Books) != 1 || len(v.Entities[0].Books[0].Accounts) != 2 || len(v.Entities[0].Books[0].Periods) != 2 || len(v.Entities[0].Books[0].Policies) != 1 || v.Entities[0].Books[0].Revision != "0" || len(v.Currencies) != 2 || v.Policy != "dev-accrual-usd-v1" {
		t.Fatal("workspace configuration lost")
	}
	sql(t, c, "INSERT INTO ledger.entity(entity_id,name,kind) VALUES(gen_random_uuid(),'Never readable','legal')")
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='post'")
	v = readWorkspace(t, s)
	if len(v.Entities) != 1 || len(v.Entities[0].Permissions) != 1 || v.Entities[0].Permissions[0] != "read" {
		t.Fatal("read privilege depended on post")
	}
	_, err := s.Post(context.Background(), command("read_only"))
	requireFailure(t, err, "permission_denied")
	sql(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE permission='post'")
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'")
	v = readWorkspace(t, s)
	if len(v.Entities) != 0 || len(v.Currencies) != 2 {
		t.Fatal("post-only actor saw readable scopes")
	}
	sql(t, c, "DELETE FROM ledger.membership")
	v = readWorkspace(t, s)
	if len(v.Entities) != 0 || len(v.Currencies) != 2 {
		t.Fatal("enrolled unconfigured actor cannot bootstrap")
	}
	sql(t, c, "UPDATE ledger.principal SET disabled_at=now() WHERE principal_id=$1", actor)
	_, err = s.Workspace(context.Background(), Command{Actor: actor})
	requireFailure(t, err, "not_found")
	_, err = s.Workspace(context.Background(), Command{Actor: "00000000-0000-4000-8000-000000000099"})
	requireFailure(t, err, "not_found")
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(context.Background())
	if _, err = runtime.Exec(context.Background(), "SELECT reporting.workspace($1)", actor); err == nil {
		t.Fatal("read committed bypass accepted")
	}
}

// TestWorkspaceRevocation uses real lock ordering for both revoker-first and reader-first.
func TestWorkspaceRevocation(t *testing.T) {
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			c, s, db := fixtureDB(t, fmt.Sprintf("workspace_revoke_%t", first))
			ctx := context.Background()
			runtime := connection(t, db, "synthetic_runtime")
			defer runtime.Close(ctx)
			if first {
				tx, err := c.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(ctx)
				if _, err = tx.Exec(ctx, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'"); err != nil {
					t.Fatal(err)
				}
				s.DB = runtime
				done := make(chan error, 1)
				go func() {
					r, err := s.Workspace(ctx, Command{Actor: actor})
					if err == nil {
						var v workspaceResult
						err = json.Unmarshal(r.Body, &v)
						if len(v.Entities) != 0 {
							err = fmt.Errorf("revoked scope exposed")
						}
					}
					done <- err
				}()
				waitBlocked(t, c, runtime.PgConn().PID())
				if err = tx.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				if err = <-done; err != nil {
					t.Fatal(err)
				}
			} else {
				tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(ctx)
				var result []byte
				if err = tx.QueryRow(ctx, "SELECT reporting.workspace($1)", actor).Scan(&result); err != nil {
					t.Fatal(err)
				}
				revoker := connection(t, db, "postgres")
				defer revoker.Close(ctx)
				done := make(chan error, 1)
				go func() {
					_, err := revoker.Exec(ctx, "UPDATE ledger.membership SET revoked_at=now() WHERE permission='read'")
					done <- err
				}()
				waitBlocked(t, c, revoker.PgConn().PID())
				if err = tx.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				if err = <-done; err != nil {
					t.Fatal(err)
				}
				if len(readWorkspace(t, s).Entities) != 0 {
					t.Fatal("revocation ignored on next read")
				}
			}
		})
	}
}

// TestWorkspaceBounds ensures oversized configuration fails instead of partial disclosure.
func TestWorkspaceBounds(t *testing.T) {
	c, s, _ := fixtureDB(t, "workspace_bounds")
	sql(t, c, "INSERT INTO ledger.account SELECT $1,$2,gen_random_uuid(),'synthetic-'||n,'Synthetic account','asset','debit',true,'2026-01-01',NULL FROM generate_series(1,199)n", entity, book)
	_, err := s.Workspace(context.Background(), Command{Actor: actor})
	requireFailure(t, err, "request_too_large")
}
