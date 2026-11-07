package collectionlabels

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestVersionedLabelTransport proves v2 selects a fixed SQL name and cannot expose a v1 envelope.
func TestVersionedLabelTransport(t *testing.T) {
	for _, op := range []string{"catalog", "query", "preview", "prepare", "pending", "execute", "cancel", "acknowledge"} {
		t.Run(op, func(t *testing.T) {
			db := &fakeDB{body: strings.Replace(result(op), "-v1", "-v2", 1)}
			req := httptest.NewRequest(http.MethodPost, strings.Replace(path(op), "/v1/", "/v2/", 1), strings.NewReader("{}"))
			if op == "catalog" || op == "pending" {
				req = httptest.NewRequest(http.MethodGet, req.URL.String(), nil)
			} else {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("Authorization", "Bearer synthetic")
			out := httptest.NewRecorder()
			Handler(&fixedAuth{}, &Service{DB: db}).ServeHTTP(out, req)
			if out.Code != 200 || db.commits != 1 || !strings.Contains(db.query, "_v2(") {
				t.Fatalf("version boundary: status%d query%q", out.Code, db.query)
			}
			db.body = result(op)
			if _, err := invoke(&Service{DB: db}, context.Background(), Request{Workspace: workspace, Version: 2}, op); err == nil {
				t.Fatal("v1 result accepted as v2")
			}
		})
	}
}

// TestOpaqueRecoveryTransport checks exact output minimization and serializable committed receipts.
func TestOpaqueRecoveryTransport(t *testing.T) {
	status := `{"profile":"private-collection-recovery-v1","state":"inaccessible","recovery_id":"` + commandID + `"}`
	receipt := `{"profile":"private-collection-recovery-clear-v1","recovery_id":"` + commandID + `","cleared":true}`
	for _, clear := range []bool{false, true} {
		db := &fakeDB{body: status}
		method, path, body := http.MethodGet, "/v1/collection-recovery", ""
		if clear {
			db.body = receipt
			method = http.MethodPost
			path += "/clear"
			body = `{"recovery_id":"` + commandID + `"}`
		}
		call := func() int {
			req := httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer synthetic")
			if clear {
				req.Header.Set("Content-Type", "application/json")
			}
			out := httptest.NewRecorder()
			Handler(&fixedAuth{}, &Service{DB: db}).ServeHTTP(out, req)
			return out.Code
		}
		if call() != 200 || db.options.IsoLevel != pgx.Serializable || db.commits != 1 {
			t.Fatal("recovery must commit serializably")
		}
		for _, leak := range []string{`"workspace_id":"private"`, `"preview":{}`, `"amount":"0.10"`, `"state":"prepared"`} {
			saved := db.body
			db.body = strings.TrimSuffix(saved, "}") + "," + leak + "}"
			commits := db.commits
			if call() != 503 || db.commits != commits {
				t.Fatal("unexpected recovery field exposed or committed")
			}
			db.body = saved
		}
		db.commitErr = errors.New("synthetic lost commit reply")
		calls := db.calls
		if call() != 503 || db.calls != calls+1 {
			t.Fatal("unknown commit was retried or exposed")
		}
	}
}
