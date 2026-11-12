package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func automaticStatus() string {
	return `{"profile":"private-collection-automation-v1","workspace_id":"` + workspace + `","revision":"1","enabled":true,"processing_state":"idle","blocked_reason":null,"pending_count":"0","attention_count":"0","last_run":null}`
}

// TestAutomationSettings verifies raw settings and independently trusted identity stay separate.
func TestAutomationSettings(t *testing.T) {
	for _, writing := range []bool{false, true} {
		d := &fakeDB{body: automaticStatus()}
		s := &AutomationService{DB: d}
		r := Request{Actor: commandID, Workspace: workspace, Body: ` {"key":"same","enabled":true,"enabled":false,"actor":"foreign"} `}
		var e error
		if writing {
			_, e = s.SetAutomation(context.Background(), r)
		} else {
			_, e = s.GetAutomation(context.Background(), r)
		}
		if e != nil || d.commits != 1 || d.rollbacks != 1 || !d.deadline || !d.cleanupDeadline {
			t.Fatal("settings transaction", e)
		}
		args := []any{r.Actor, r.Workspace}
		iso := pgx.RepeatableRead
		if writing {
			args = append(args, r.Body)
			iso = pgx.Serializable
		}
		if !reflect.DeepEqual(d.args, args) || d.options.IsoLevel != iso {
			t.Fatal("raw settings or isolation changed")
		}
		for _, err := range []error{errors.New("lost commit"), &pgconn.PgError{Code: "40003"}, &pgconn.PgError{Code: "08006"}} {
			d := &fakeDB{body: automaticStatus(), commitErr: err}
			body, e := (&AutomationService{DB: d}).SetAutomation(context.Background(), r)
			if e == nil || body != nil || d.calls != 1 {
				t.Fatal("unknown commit retried or exposed result")
			}
		}
	}
	d := &fakeDB{body: automaticStatus(), firstRowErr: &pgconn.PgError{Code: "40001"}}
	if _, e := (&AutomationService{DB: d}).GetAutomation(context.Background(), Request{Workspace: workspace}); e != nil || d.calls != 2 {
		t.Fatal("known abort not retried", e)
	}
	d = &fakeDB{body: automaticStatus(), commitErr: &pgconn.PgError{Code: "40P01"}}
	if _, e := (&AutomationService{DB: d}).GetAutomation(context.Background(), Request{Workspace: workspace}); e == nil || d.calls != 3 {
		t.Fatal("retry bound")
	}
}

type automaticPort struct {
	request Request
	calls   int
	body    string
}

func (p *automaticPort) GetAutomation(_ context.Context, r Request) (json.RawMessage, error) {
	p.request = r
	p.calls++
	return json.RawMessage(p.body), nil
}
func (p *automaticPort) SetAutomation(c context.Context, r Request) (json.RawMessage, error) {
	return p.GetAutomation(c, r)
}

// TestAutomationHTTP covers actual handler framing without requiring a listener or accepting computed results.
func TestAutomationHTTP(t *testing.T) {
	p := &automaticPort{body: automaticStatus()}
	h := AutomationHandler(&fixedAuth{}, p)
	url := "/v1/collections/" + workspace + "/automation"
	raw := ` {"key":"same","expected_revision":"1","enabled":true,"enabled":false} `
	w := httpCall(h, "POST", url, raw)
	if w.Code != 200 || p.request.Body != raw || p.request.Actor != "independently-verified" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("settings framing", w.Code)
	}
	for _, tc := range []struct {
		method, url, body string
		status            int
	}{{"GET", url, "", 200}, {"HEAD", url, "", 405}, {"GET", url, "{}", 400}, {"GET", url + "?actor=foreign", "", 400}, {"POST", url, strings.Repeat("x", 1025), 413}, {"GET", strings.Replace(url, workspace, "bad", 1), "", 400}} {
		if got := httpCall(h, tc.method, tc.url, tc.body); got.Code != tc.status {
			t.Fatalf("%s: %d", tc.method, got.Code)
		}
	}
	before := p.calls
	r := httptest.NewRequest(http.MethodPost, "/v1/collections/bad/automation?invalid=1", strings.NewReader("bad"))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var problemBody map[string]json.RawMessage
	if w.Code != 401 || w.Header().Get("WWW-Authenticate") != "Bearer" || json.Unmarshal(w.Body.Bytes(), &problemBody) != nil || len(problemBody) != 5 || p.calls != before {
		t.Fatal("auth-first problem boundary")
	}
	var id string
	if json.Unmarshal(problemBody["request_id"], &id) != nil || !validID(id) {
		t.Fatal("missing server request identity")
	}
	p.body = strings.Replace(automaticStatus(), `"attention_count":"0"`, `"attention_count":"00"`, 1)
	if got := httpCall(h, "GET", url, ""); got.Code != 503 {
		t.Fatal("invalid SQL result accepted")
	}
}

type automaticQuery struct {
	sql, body string
	err       error
	inspect   func([]any)
}
type automaticTxPlan struct {
	queries []automaticQuery
	commit  error
}
type automaticDB struct {
	t                         *testing.T
	plans                     []automaticTxPlan
	calls, commits, rollbacks int
}
type automaticTx struct {
	pgx.Tx
	db    *automaticDB
	plan  automaticTxPlan
	index int
}
type automaticRow struct {
	body string
	err  error
}

func (r automaticRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*json.RawMessage) = json.RawMessage(r.body)
	return nil
}
func (d *automaticDB) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 4*time.Second || o.IsoLevel != pgx.Serializable {
		d.t.Fatal("unbounded or wrong worker transaction")
	}
	if d.calls >= len(d.plans) {
		d.t.Fatal("unexpected new worker transaction")
	}
	p := d.plans[d.calls]
	d.calls++
	return &automaticTx{db: d, plan: p}, nil
}
func (tx *automaticTx) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if !strings.Contains(q, "set_config('statement_timeout'") || len(args) != 1 {
		tx.db.t.Fatal("unexpected worker SQL mutation")
	}
	return pgconn.CommandTag{}, nil
}
func (tx *automaticTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if tx.index >= len(tx.plan.queries) {
		tx.db.t.Fatal("unexpected worker query")
	}
	p := tx.plan.queries[tx.index]
	tx.index++
	if q != p.sql {
		tx.db.t.Fatalf("worker query %s wanted %s", q, p.sql)
	}
	if p.inspect != nil {
		p.inspect(args)
	}
	return automaticRow{p.body, p.err}
}
func (tx *automaticTx) Commit(context.Context) error { tx.db.commits++; return tx.plan.commit }
func (tx *automaticTx) Rollback(ctx context.Context) error {
	if ctx.Err() != nil {
		tx.db.t.Fatal("cancelled cleanup")
	}
	tx.db.rollbacks++
	return nil
}

const autoWorkSQL = "select banking.collection_automation_work()"
const autoContextSQL = "select banking.collection_automation_context($1::uuid,$2::uuid)"
const autoPrepareSQL = "select banking.prepare_collection_automatic_run($1::uuid,$2::uuid,$3::text,$4::text)"
const autoExecuteSQL = "select banking.execute_collection_automatic_run($1::uuid,$2::uuid,$3::uuid)"

func autoWork() string {
	return `{"profile":"private-collection-auto-work-v1","workspaces":[{"principal_id":"` + commandID + `","workspace_id":"` + workspace + `"}]}`
}
func autoRun(state string) string {
	return `{"profile":"private-collection-auto-run-v1","workspace_id":"` + workspace + `","run_id":"` + commandID + `","state":"` + state + `","processed_count":"0","attention_count":"0"}`
}
func autoContext(state string, last string) string {
	run := "null"
	c := "null"
	if last == "" {
		last = "null"
	}
	if state == "pending" {
		run = `"` + commandID + `"`
	}
	if state == "ready" {
		c = `{"profile":"private-rule-context-v1","principal_id":"` + commandID + `","workspace_id":"` + workspace + `","snapshot":{"taxonomy_revision":"1","tag_revision":"0","ruleset_revision":"0"},"action":{"kind":"rules_apply"},"topic_ids":[],"tag_ids":[],"rules":[],"facts":[{"source_id":"` + workspace + `","observation_id":"` + commandID + `","observation_revision":"1","booked_date":"2026-01-01","merchant":null,"description":"Fictional fact","currency_version_id":"` + workspace + `","direction":"outflow","source_state":"booked","money_state":"available","manual_topic":null,"manual_tags":null}]}`
	}
	return `{"profile":"private-collection-auto-context-v1","principal_id":"` + commandID + `","workspace_id":"` + workspace + `","state":"` + state + `","run_id":` + run + `,"context":` + c + `,"last_run":` + last + `}`
}

// TestAutomaticDurablePhases proves same-TX trusted evaluation, no immediate execution and durable pending recovery.
func TestAutomaticDurablePhases(t *testing.T) {
	work := automaticTxPlan{queries: []automaticQuery{{sql: autoWorkSQL, body: autoWork()}}}
	ready := automaticTxPlan{queries: []automaticQuery{{sql: autoContextSQL, body: autoContext("ready", "")}, {sql: autoPrepareSQL, body: autoRun("prepared"), inspect: func(args []any) {
		if len(args) != 4 || args[0] != commandID || args[1] != workspace {
			t.Fatal("untrusted target")
		}
		var evaluated struct {
			Results []ruleEvaluation `json:"results"`
		}
		if json.Unmarshal([]byte(args[3].(string)), &evaluated) != nil || len(evaluated.Results) != 1 || evaluated.Results[0].Status != "unclassified" {
			t.Fatal("existing matcher result not passed")
		}
	}}}, commit: errors.New("lost prepare commit")}
	pending := automaticTxPlan{queries: []automaticQuery{{sql: autoContextSQL, body: autoContext("pending", autoRun("prepared"))}, {sql: autoExecuteSQL, body: autoRun("committed"), inspect: func(args []any) {
		if args[2] != commandID {
			t.Fatal("changed durable run")
		}
	}}}, commit: errors.New("lost execute commit")}
	idle := automaticTxPlan{queries: []automaticQuery{{sql: autoContextSQL, body: autoContext("idle", autoRun("committed"))}}}
	d := &automaticDB{t: t, plans: []automaticTxPlan{work, ready, work, pending, work, idle}}
	s := &AutomaticService{DB: d}
	if s.Step(context.Background()) == nil || d.calls != 2 {
		t.Fatal("unknown preparation did not stop round")
	}
	if s.Step(context.Background()) == nil || d.calls != 4 {
		t.Fatal("unknown execution did not stop round")
	}
	if e := s.Step(context.Background()); e != nil || d.calls != 6 || d.rollbacks != 6 {
		t.Fatal("durable receipt recovery", e)
	}
}

// TestAutomaticClosedContext refuses mismatched scope/last receipt before preparing or executing more work.
func TestAutomaticClosedContext(t *testing.T) {
	for _, body := range []string{strings.Replace(autoContext("ready", ""), commandID, workspace, 1), strings.Replace(autoContext("ready", ""), `"last_run":null`, `"last_run":{}`, 1), autoContext("unexpected", "")} {
		d := &automaticDB{t: t, plans: []automaticTxPlan{{queries: []automaticQuery{{sql: autoWorkSQL, body: autoWork()}}}, {queries: []automaticQuery{{sql: autoContextSQL, body: body}}}}}
		if (&AutomaticService{DB: d}).Step(context.Background()) == nil || d.commits != 1 {
			t.Fatal("invalid context committed")
		}
	}
	d := &automaticDB{t: t, plans: []automaticTxPlan{{queries: []automaticQuery{{sql: autoWorkSQL, body: `{"profile":"private-collection-auto-work-v1","workspaces":null}`}}}}}
	if (&AutomaticService{DB: d}).Step(context.Background()) == nil || d.commits != 0 {
		t.Fatal("invalid inventory accepted")
	}
}

func TestAutomaticCatalogMarker(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		d := &fakeDB{body: result("catalog")}
		s := &Service{DB: d, AutomaticCategories: enabled}
		body, e := s.Catalog(context.Background(), Request{Workspace: workspace})
		if e != nil || strings.Contains(string(body), "automatic_categorization") != enabled {
			t.Fatal("optional marker", e)
		}
	}
}
