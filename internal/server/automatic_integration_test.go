//go:build integration

package server

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionlabels"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/collectionread"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var automaticServerProof = flag.Bool("automatic-server-proof", false, "use only the disposable automatic runner's internal rules_template48")

const automaticProofOwner = "a4880000-0000-4000-8000-000000000901"
const automaticProofForeign = "a4880000-0000-4000-8000-000000000902"

// automaticProofAuth supplies fictional identities only; no Firebase or hosted identity claim.
type automaticProofAuth struct{}

func (automaticProofAuth) Authenticate(_ context.Context, token string) (string, error) {
	switch token {
	case "synthetic.automatic.owner":
		return automaticProofOwner, nil
	case "synthetic.automatic.foreign":
		return automaticProofForeign, nil
	default:
		return "", errors.New("synthetic identity denied")
	}
}

// TestAuto48ServerReadinessSettings proves actual role grants and mounted HTTP without a host listener.
// The explicit flag and fixed internal host cannot consume an inherited managed DSN.
func TestAuto48ServerReadinessSettings(t *testing.T) {
	if !*automaticServerProof {
		t.Skip("explicit disposable automatic server proof only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	control, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname=postgres sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer control.Close(context.Background())
	db := fmt.Sprintf("automatic_server_%d", time.Now().UnixNano())
	if _, e = control.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{db}.Sanitize()+" TEMPLATE rules_template"); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := control.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{db}.Sanitize()); e != nil {
			t.Error("owned automatic server clone cleanup failed", e)
		}
	}()
	admin, e := pgx.Connect(ctx, "host=database port=5432 user=postgres dbname="+db+" sslmode=disable connect_timeout=5")
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(context.Background())
	pool, e := pgxpool.New(ctx, "host=database port=5432 user=synthetic_runtime dbname="+db+" sslmode=disable connect_timeout=5 pool_max_conns=3")
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	if runtimeReady(ctx, pool) != nil || collectionReady(ctx, pool) != nil || automaticReady(ctx, pool) != nil {
		t.Fatal("restricted matching runtime not ready")
	}
	for _, signature := range []string{
		"banking.get_collection_automation(uuid,uuid)", "banking.set_collection_automation(uuid,uuid,text)", "banking.collection_automation_work()",
		"banking.collection_automation_context(uuid,uuid)", "banking.prepare_collection_automatic_run(uuid,uuid,text,text)", "banking.execute_collection_automatic_run(uuid,uuid,uuid)",
	} {
		t.Run(signature, func(t *testing.T) {
			if _, e := admin.Exec(ctx, "REVOKE EXECUTE ON FUNCTION "+signature+" FROM bookkeeping_api"); e != nil {
				t.Fatal(e)
			}
			if automaticReady(ctx, pool) == nil {
				t.Fatal("missing automatic entrypoint accepted")
			}
			if _, e := admin.Exec(ctx, "GRANT EXECUTE ON FUNCTION "+signature+" TO bookkeeping_api"); e != nil {
				t.Fatal(e)
			}
			if automaticReady(ctx, pool) != nil {
				t.Fatal("restored automatic grant rejected")
			}
		})
	}
	for _, q := range []string{"SELECT count(*) FROM banking.collection_automatic_run", "SELECT banking.collection_automatic_status(NULL,NULL)", "SELECT banking.provision_collection_owner(NULL,NULL,NULL)"} {
		_, e := pool.Exec(ctx, q)
		var pg *pgconn.PgError
		if !errors.As(e, &pg) || pg.Code != "42501" {
			t.Fatal("runtime acquired table/helper/operator authority")
		}
	}
	if _, e = admin.Exec(ctx, `INSERT INTO ledger.principal(principal_id,issuer,subject)VALUES($1,'synthetic','automatic-owner'),($2,'synthetic','automatic-foreign')`, automaticProofOwner, automaticProofForeign); e != nil {
		t.Fatal(e)
	}
	provision := func(key string, expected any, caps []string) string {
		t.Helper()
		tx, e := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback(context.Background())
		if _, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner"); e != nil {
			t.Fatal(e)
		}
		body, _ := json.Marshal(map[string]any{"expected_revision": expected, "capabilities": caps, "evidence_ref": "a4880000-0000-4000-8000-000000000903"})
		var raw []byte
		if e = tx.QueryRow(ctx, "SELECT banking.provision_collection_owner($1,$2,$3)", automaticProofOwner, key, string(body)).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		if e = tx.Commit(ctx); e != nil {
			t.Fatal(e)
		}
		var result struct {
			Workspace string `json:"workspace_id"`
		}
		if json.Unmarshal(raw, &result) != nil || result.Workspace == "" {
			t.Fatal("fixture workspace missing")
		}
		return result.Workspace
	}
	w := provision("fixture", nil, []string{"read", "acquire", "classify"})
	c := Config{Mode: "managed", Listen: "0.0.0.0:8080", Origin: "http://localhost:39083", Project: "jayburd-bookkeeping-dev", DatabaseURL: "synthetic-injected-pool", CollectionEnabled: true, AutomaticCategories: true}
	d := Dependencies{Auth: automaticProofAuth{}, Operations: &ledger.Service{DB: pool}, Collection: &collectionread.Service{DB: pool}, Labels: &collectionlabels.Service{DB: pool, Rules: collectionlabels.RuleProcessor{}, AutomaticCategories: true}, Automation: &collectionlabels.AutomationService{DB: pool}, Automatic: &collectionlabels.AutomaticService{DB: pool}, Ready: func(ctx context.Context) error {
		if e := runtimeReady(ctx, pool); e != nil {
			return e
		}
		return automaticReady(ctx, pool)
	}}
	app, e := New(c, d)
	if e != nil {
		t.Fatal(e)
	}
	url := "/v1/collections/" + w + "/automation"
	call := func(method, path, token, body string, want int) map[string]any {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", c.Origin)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if method == http.MethodPost {
			r.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		app.HTTP.Handler.ServeHTTP(res, r)
		if res.Code != want || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s status%d expected%d", method, res.Code, want)
		}
		var v map[string]any
		if json.Unmarshal(res.Body.Bytes(), &v) != nil {
			t.Fatal("invalid response JSON")
		}
		if want != 200 {
			if v["profile"] != nil || v["workspace_id"] != nil {
				t.Fatal("denial disclosed private metadata")
			}
			if want == 401 && res.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatal("missing auth challenge")
			}
		}
		return v
	}
	owner := "synthetic.automatic.owner"
	call(http.MethodGet, url, "", "", 401)
	call(http.MethodGet, url, "synthetic.automatic.foreign", "", 404)
	status := call(http.MethodGet, url, owner, "", 200)
	if status["profile"] != "private-collection-automation-v1" || status["revision"] != "0" || status["enabled"] != true {
		t.Fatal("initial current settings")
	}
	call(http.MethodPost, url, owner, `{"key":"bad","expected_revision":"0","enabled":true,"enabled":false}`, 400)
	call(http.MethodPost, url, owner, `{"key":"bad","expected_revision":"0","enabled":true,"actor":"foreign"}`, 400)
	raw := ` {"key":"same-policy","expected_revision":"0","enabled":false} `
	first := call(http.MethodPost, url, owner, raw, 200)
	again := call(http.MethodPost, url, owner, raw, 200)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(again)
	if string(a) != string(b) || first["revision"] != "1" || first["enabled"] != false {
		t.Fatal("policy receipt replay changed")
	}
	call(http.MethodPost, url, owner, `{"key":"same-policy","expected_revision":"0","enabled":true}`, 409)
	provision("remove-classify", "1", []string{"read"})
	call(http.MethodPost, url, owner, `{"key":"denied","expected_revision":"1","enabled":true}`, 404)
	call(http.MethodGet, url, owner, "", 200)
	if _, e = admin.Exec(ctx, "UPDATE ledger.principal SET disabled_at=clock_timestamp() WHERE principal_id=$1", automaticProofOwner); e != nil {
		t.Fatal(e)
	}
	call(http.MethodGet, url, owner, "", 404)
	for table, want := range map[string]int64{"ledger.journal_entry": 0, "integration.outbox_message": 0, "banking.collection_automatic_run": 0, "banking.collection_automatic_decision": 0, "banking.collection_taxonomy_edition": 1, "banking.collection_automation_setting_receipt": 1} {
		var got int64
		if e = admin.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&got); e != nil || got != want {
			t.Fatalf("read/settings effect %s=%d want%d", table, got, want)
		}
	}
}
