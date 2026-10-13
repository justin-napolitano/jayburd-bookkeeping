//go:build integration

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/importer"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/staging"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

const demoImportActor = "00000000-0000-4000-8000-000000000001"
const demoImportUSD = "00000000-0000-4000-8000-000000000002"
const demoImportRows = 1000

// demoAcquisition exists only in the disposable integration binary, never the production server.
type demoAcquisition struct {
	importer *importer.Service
	ledger   *ledger.Service
	batches  []demoImportBatch
	gate     sync.Mutex
}

// demoImportBatch retains runtime-generated identity; no caller chooses a scope or object.
type demoImportBatch struct {
	Name, Entity, Book, Source, Batch, Reference string
	Index                                        int
}

// demoBatchStatus exposes durable progress and navigation identity without storage references.
type demoBatchStatus struct {
	Name     string `json:"name"`
	Entity   string `json:"entity_id"`
	Book     string `json:"book_id"`
	Source   string `json:"source_account_id"`
	Batch    string `json:"batch_id"`
	State    string `json:"state"`
	Imported string `json:"imported_count"`
}

// demoAcquisitionStatus distinguishes durable imports from previews or in-flight requests.
type demoAcquisitionStatus struct {
	Profile  string            `json:"profile"`
	Total    string            `json:"total_count"`
	Imported string            `json:"imported_count"`
	Complete bool              `json:"complete"`
	Blocked  bool              `json:"blocked"`
	Pending  *string           `json:"pending_command_id"`
	Batches  []demoBatchStatus `json:"batches"`
}

// demoPreview holds the reviewed revisions required by the existing interactive gate.
type demoPreview struct {
	ID       string `json:"preview_id"`
	Revision string `json:"preview_revision"`
	Binding  string `json:"binding_revision"`
	Source   string `json:"source_revision"`
}

// demoBatchValue is a bounded projection of the authorized SQL batch response.
type demoBatchValue struct {
	ID       string       `json:"batch_id"`
	State    string       `json:"state"`
	Created  string       `json:"created_count"`
	Existing string       `json:"existing_count"`
	Preview  *demoPreview `json:"preview"`
}

// demoPending preserves the actual interactive command, never guessing recovery state.
type demoPending struct {
	ID        string          `json:"command_id"`
	Operation string          `json:"operation"`
	Entity    string          `json:"entity_id"`
	Book      string          `json:"book_id"`
	State     string          `json:"state"`
	Body      json.RawMessage `json:"body"`
}

// demoImportBody is the exact import approval shape; it carries no posting request.
type demoImportBody struct {
	Batch    string `json:"batch_id"`
	Preview  string `json:"preview_id"`
	Revision string `json:"preview_revision"`
	Binding  string `json:"binding_revision"`
	Source   string `json:"source_revision"`
	Profile  string `json:"profile"`
}

// demoJSON marshals only fixed typed synthetic request structures.
func demoJSON(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic("invalid synthetic request structure")
	}
	return string(b)
}

// demoID provides stable synthetic request identities, not object or ledger primary keys.
func demoID(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", 800000+n) }

// demoScope always binds the one fixture actor to server-owned entity and book IDs.
func demoScope(b demoImportBatch) importer.Scope {
	return importer.Scope{Actor: demoImportActor, Entity: b.Entity, Book: b.Book}
}

// demoCSV uses integer cents: 500 inflows at 100.01 and 500 outflows at 25.02 per LLC.
func demoCSV(index int, ref string) []byte {
	var out strings.Builder
	out.WriteString("source_account_ref,entry_id,booked_on,currency,direction,amount,description\r\n")
	for i := 0; i < demoImportRows; i++ {
		direction, amount, description := "inflow", 10001, "Fictional customer payment"
		if i%2 == 1 {
			direction, amount, description = "outflow", 2502, "Fictional operating expense"
		}
		fmt.Fprintf(&out, "%s,DEMO%d_%04d,2026-01-%02d,USD,%s,%d.%02d,%s\r\n", ref, index+1, i+1, 1+i%28, direction, amount/100, amount%100, description)
	}
	return []byte(out.String())
}

// demoSource owns an in-memory deterministic buffer and honors cancellation synchronously.
type demoSource struct{ *bytes.Reader }

// Read checks cancellation without creating an unowned blocking-reader goroutine.
func (s *demoSource) Read(ctx context.Context, p []byte) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	return s.Reader.Read(p)
}

// Close has no external resource or asynchronous cleanup.
func (s *demoSource) Close(context.Context) error { return nil }

// demoOwnerCall is fixture-only provisioning; runtime acquisition never receives the admin connection.
func demoOwnerCall(ctx context.Context, admin *pgx.Conn, q string, args ...any) ([]byte, error) {
	tx, e := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return nil, e
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, e = tx.Exec(ctx, "SET LOCAL ROLE bookkeeping_owner"); e != nil {
		return nil, e
	}
	var result []byte
	if e = tx.QueryRow(ctx, q, args...).Scan(&result); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return result, nil
}

// bootstrapDemoAcquisition provisions fictional ownership separately, then registers empty runtime batches.
// Reusing the same database requires retaining the passed private staging root.
func bootstrapDemoAcquisition(ctx context.Context, admin *pgx.Conn, pool *pgxpool.Pool, store *staging.Store) (*demoAcquisition, error) {
	d := &demoAcquisition{importer: importer.New(pool, store), ledger: &ledger.Service{DB: pool}}
	names := []string{"Fictional Harbor Design LLC", "Fictional Juniper Services LLC", "Fictional Summit Trading LLC"}
	for index, name := range names {
		setupBody := map[string]any{"entity_name": name, "entity_kind": "legal", "book_name": "Synthetic acquisition / USD", "starts_on": "2026-01-01", "ends_before": "2027-01-01", "functional_currency_version_id": demoImportUSD, "accounts": []map[string]any{demoAccount("1000", "Fictional operating bank", "asset", "debit", true), demoAccount("4000", "Fictional service income", "revenue", "credit", false), demoAccount("5000", "Fictional operating expenses", "expense", "debit", false)}}
		setup, e := d.ledger.Setup(ctx, ledger.Command{Actor: demoImportActor, Key: fmt.Sprintf("demo-acquisition-setup-v1-%d", index), Request: demoID(index), Body: demoJSON(setupBody)})
		if e != nil {
			return nil, e
		}
		var value struct {
			Entity   string `json:"entity_id"`
			Book     string `json:"book_id"`
			Accounts []struct {
				ID   string `json:"account_id"`
				Code string `json:"code"`
			} `json:"accounts"`
		}
		if json.Unmarshal(setup.Body, &value) != nil || value.Entity == "" || value.Book == "" {
			return nil, errors.New("demo_configuration")
		}
		// Setup grants read/post only. The fixture administrator separately grants
		// prepare; replay never restores a deliberately revoked membership.
		if _, e = demoOwnerCall(ctx, admin, `WITH granted AS (INSERT INTO ledger.membership(entity_id,principal_id,permission,revoked_at) VALUES($1,$2,'prepare',NULL) ON CONFLICT(entity_id,principal_id,permission) DO NOTHING RETURNING 1) SELECT '{}'::jsonb`, value.Entity, demoImportActor); e != nil {
			return nil, e
		}
		bank := ""
		for _, a := range value.Accounts {
			if a.Code == "1000" {
				bank = a.ID
			}
		}
		if bank == "" {
			return nil, errors.New("demo_configuration")
		}
		ref := fmt.Sprintf("DEMO-BANK-%d", index+1)
		manifest := map[string]any{"ledger_account_id": bank, "currency_version_id": demoImportUSD, "binding_date": "2026-01-01", "binding_evidence_ref": demoID(100 + index), "reader_principal_id": demoImportActor, "namespace": map[string]any{"profile": ingestion.Profile, "synthetic_dataset_id": demoID(200 + index), "source_account_ref": ref}}
		raw, e := demoOwnerCall(ctx, admin, "select banking.register_source($1::uuid,$2::uuid,$3::uuid,$4::text,$5::text)", demoImportActor, value.Entity, value.Book, "demo-acquisition-source-v1", demoJSON(manifest))
		if e != nil {
			return nil, e
		}
		var source struct {
			ID        string `json:"source_account_id"`
			Namespace string `json:"namespace_id"`
		}
		if json.Unmarshal(raw, &source) != nil || source.ID == "" || source.Namespace == "" {
			return nil, errors.New("demo_configuration")
		}
		permission := map[string]any{"expected_binding_revision": "1", "principal_id": demoImportActor, "enabled": true, "entity_import_enabled": true}
		raw, e = demoOwnerCall(ctx, admin, "select banking.set_source_import_permission($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", demoImportActor, value.Entity, value.Book, source.ID, "demo-acquisition-permission-v1", demoJSON(permission))
		if e != nil {
			return nil, e
		}
		var grant struct {
			Binding string `json:"binding_revision"`
		}
		if json.Unmarshal(raw, &grant) != nil || grant.Binding == "" {
			return nil, errors.New("demo_configuration")
		}
		reviewPermission := map[string]any{"expected_binding_revision": grant.Binding, "principal_id": demoImportActor, "enabled": true}
		raw, e = demoOwnerCall(ctx, admin, "select banking.set_source_review_permission($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::text,$6::text)", demoImportActor, value.Entity, value.Book, source.ID, "demo-acquisition-review-v1", demoJSON(reviewPermission))
		if e != nil {
			return nil, e
		}
		if json.Unmarshal(raw, &grant) != nil || grant.Binding == "" {
			return nil, errors.New("demo_configuration")
		}
		b := demoImportBatch{Name: name, Entity: value.Entity, Book: value.Book, Source: source.ID, Reference: ref, Index: index}
		registration := importer.Registration{SourceAccountID: source.ID, NamespaceID: source.Namespace, BindingRevision: grant.Binding, StatementRef: "DEMO-JANUARY-2026-V1", StartsOn: "2026-01-01", EndsBefore: "2026-02-01", Profile: ingestion.Profile}
		raw, e = d.importer.Register(ctx, demoScope(b), "demo-acquisition-batch-v1", registration)
		if e != nil {
			return nil, e
		}
		var batch demoBatchValue
		if json.Unmarshal(raw, &batch) != nil || batch.ID == "" {
			return nil, errors.New("demo_configuration")
		}
		b.Batch = batch.ID
		d.batches = append(d.batches, b)
	}
	return d, nil
}

// snapshot reads durable batches and the current interactive gate without mutating either.
func (d *demoAcquisition) snapshot(ctx context.Context) (demoAcquisitionStatus, []demoBatchValue, *demoPending, error) {
	status := demoAcquisitionStatus{Profile: "synthetic-acquisition-v1", Total: "3000", Imported: "0", Batches: make([]demoBatchStatus, 0, len(d.batches))}
	values := make([]demoBatchValue, 0, len(d.batches))
	total := 0
	for _, b := range d.batches {
		raw, e := d.importer.Batch(ctx, demoScope(b), b.Batch)
		if e != nil {
			return status, nil, nil, e
		}
		var v demoBatchValue
		if json.Unmarshal(raw, &v) != nil || v.ID != b.Batch {
			return status, nil, nil, errors.New("demo_configuration")
		}
		count := 0
		if v.State == "imported" {
			created, e1 := strconv.Atoi(v.Created)
			existing, e2 := strconv.Atoi(v.Existing)
			if e1 != nil || e2 != nil || created < 0 || existing < 0 || created+existing != demoImportRows {
				return status, nil, nil, errors.New("demo_configuration")
			}
			count = created + existing
		}
		total += count
		values = append(values, v)
		status.Batches = append(status.Batches, demoBatchStatus{b.Name, b.Entity, b.Book, b.Source, b.Batch, v.State, strconv.Itoa(count)})
		if v.State == "rejected" || v.State == "cancelled" {
			status.Blocked = true
		}
	}
	status.Imported = strconv.Itoa(total)
	pending, e := d.ledger.Recovery(ctx, ledger.Command{Actor: demoImportActor}, "pending")
	if e != nil {
		return status, nil, nil, e
	}
	var envelope struct {
		Command *demoPending `json:"command"`
	}
	if json.Unmarshal(pending.Body, &envelope) != nil {
		return status, nil, nil, errors.New("demo_configuration")
	}
	command := envelope.Command
	own := false
	if command != nil {
		status.Pending = &command.ID
		own = d.commandIndex(command, values) >= 0
		if !own || command.State == "cancelled" {
			status.Blocked = true
		}
	}
	status.Complete = total == 3000 && (!own || command == nil)
	if status.Complete {
		status.Blocked = false
	}
	return status, values, command, nil
}

// bodyFor binds explicit synthetic import approval to the latest immutable preview.
func bodyFor(b demoImportBatch, v demoBatchValue) demoImportBody {
	return demoImportBody{b.Batch, v.Preview.ID, v.Preview.Revision, v.Preview.Binding, v.Preview.Source, ingestion.Profile}
}

// commandIndex refuses to execute or acknowledge unrelated or differently reviewed work.
func (d *demoAcquisition) commandIndex(p *demoPending, values []demoBatchValue) int {
	if p.Operation != "import" {
		return -1
	}
	var body demoImportBody
	decoder := json.NewDecoder(bytes.NewReader(p.Body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&body) != nil {
		return -1
	}
	for i, b := range d.batches {
		if b.Entity == p.Entity && b.Book == p.Book && values[i].Preview != nil && body == bodyFor(b, values[i]) {
			return i
		}
	}
	return -1
}

// status uses a bounded request budget and derives progress entirely from committed SQL.
func (d *demoAcquisition) status(parent context.Context) (demoAcquisitionStatus, error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	s, _, _, e := d.snapshot(ctx)
	return s, e
}

// step performs exactly one durable stage; interruption resumes through SQL, never an in-memory job queue.
func (d *demoAcquisition) step(parent context.Context) (demoAcquisitionStatus, error) {
	if !d.gate.TryLock() {
		return demoAcquisitionStatus{}, errors.New("demo_busy")
	}
	defer d.gate.Unlock()
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	status, values, pending, e := d.snapshot(ctx)
	if e != nil {
		return status, e
	}
	if status.Complete {
		return status, nil
	}
	if status.Blocked {
		return status, errors.New("demo_blocked")
	}
	if pending != nil {
		index := d.commandIndex(pending, values)
		if index < 0 {
			return status, errors.New("demo_blocked")
		}
		action := "execute"
		if pending.State == "committed" {
			action = "acknowledge"
		} else if pending.State != "prepared" {
			return status, errors.New("demo_blocked")
		}
		_, e = d.ledger.Recovery(ctx, ledger.Command{Actor: demoImportActor, CommandID: pending.ID, Request: demoID(400 + index), Body: "{}"}, action)
	} else {
		for i, v := range values {
			b := d.batches[i]
			if v.State == "imported" {
				continue
			}
			switch v.State {
			case "received":
				_, e = d.importer.Upload(ctx, demoScope(b), b.Batch, &demoSource{bytes.NewReader(demoCSV(b.Index, b.Reference))})
			case "preview_ready":
				if v.Preview == nil {
					return status, errors.New("demo_configuration")
				}
				body := map[string]any{"operation": "import", "entity_id": b.Entity, "book_id": b.Book, "body": bodyFor(b, v)}
				_, e = d.ledger.Recovery(ctx, ledger.Command{Actor: demoImportActor, Key: fmt.Sprintf("demo-acquisition-import-v1-%d", i), Request: demoID(300 + i), Body: demoJSON(body)}, "register")
			default:
				return status, errors.New("demo_blocked")
			}
			break
		}
	}
	if e != nil {
		return demoAcquisitionStatus{}, e
	}
	status, _, _, e = d.snapshot(ctx)
	return status, e
}

// ServeHTTP is mounted only by the synthetic integration fixture under BrowserBoundary.
func (d *demoAcquisition) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if len(r.Header.Values("Authorization")) != 1 || r.Header.Get("Authorization") != "Bearer synthetic.browser.proof" {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"code":"unauthorized"}}`)
		return
	}
	var status demoAcquisitionStatus
	var err error
	switch {
	case r.URL.Path == "/v1/demo/acquisition" && r.Method == http.MethodGet:
		if r.URL.RawQuery != "" {
			err = errors.New("invalid_request")
		} else {
			status, err = d.status(r.Context())
		}
	case r.URL.Path == "/v1/demo/acquisition/step" && r.Method == http.MethodPost:
		raw, e := io.ReadAll(io.LimitReader(r.Body, 4))
		if e != nil || string(raw) != "{}" || r.URL.RawQuery != "" {
			err = errors.New("invalid_request")
		} else {
			status, err = d.step(r.Context())
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"code":"not_found"}}`)
		return
	}
	if err != nil {
		code := err.Error()
		httpStatus := http.StatusServiceUnavailable
		switch code {
		case "invalid_request":
			httpStatus = http.StatusBadRequest
		case "demo_busy", "demo_blocked":
			httpStatus = http.StatusConflict
		default:
			code = "demo_retry_required"
		}
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code}})
		return
	}
	_ = json.NewEncoder(w).Encode(status)
}
