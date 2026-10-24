package bankcredential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const repairAttempt = "00000000-0000-4000-8000-000000000004"

// repairDB exposes only a synthetic exact-attempt function result and independent commit faults.
type repairDB struct {
	raw                 []byte
	queryErr, commitErr error
	calls               int
	active              bool
	scope               connections.Scope
}

// BeginTx enforces SQL-only serializable work before any cryptographic call.
func (d *repairDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable {
		panic("unexpected isolation")
	}
	d.calls++
	d.active = true
	return &repairTx{db: d}, nil
}

// repairTx rejects any credential lookup/write other than the selected purpose function.
type repairTx struct {
	pgx.Tx
	db *repairDB
}

// Exec accepts only the existing adapter transaction settings.
func (t *repairTx) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if q != "SET LOCAL statement_timeout='3s'; SET LOCAL lock_timeout='1s'" || len(args) != 0 {
		panic("unexpected SQL mutation")
	}
	return pgconn.CommandTag{}, nil
}

// QueryRow checks that only verified scope and attempt, never a caller-selected reference, enter SQL.
func (t *repairTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if q != "SELECT integration.get_repair_bank_credential($1,$2,$3,$4)::text" || len(args) != 4 ||
		args[0] != t.db.scope.Actor || args[1] != t.db.scope.Entity || args[2] != t.db.scope.Book || args[3] != repairAttempt {
		panic("wrong credential purpose or authority tuple")
	}
	return repairRow{t.db}
}

// Commit independently confirms or loses the synthetic result boundary.
func (t *repairTx) Commit(context.Context) error { t.db.active = false; return t.db.commitErr }

// Rollback ensures cleanup does not retain the canceled request context.
func (t *repairTx) Rollback(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok || ctx.Err() != nil {
		panic("unbounded cleanup")
	}
	t.db.active = false
	return nil
}

// repairRow supplies only the closed SQL projection or a fixed denial.
type repairRow struct{ db *repairDB }

// Scan never exposes a projection after a selected query failure.
func (r repairRow) Scan(dest ...any) error {
	if r.db.queryErr != nil {
		return r.db.queryErr
	}
	*dest[0].(*[]byte) = r.db.raw
	return nil
}

// repairWrapper detects any new wrapping/write or unwrap before SQL has completed.
type repairWrapper struct {
	db             *repairDB
	wraps, unwraps int
}

// Wrap counts prohibited fresh ciphertext preparation.
func (w *repairWrapper) Wrap(ctx context.Context, key, aad []byte) (string, []byte, error) {
	w.wraps++
	return testWrapper{}.Wrap(ctx, key, aad)
}

// Unwrap uses the independent synthetic KEK after confirmed SQL-only work.
func (w *repairWrapper) Unwrap(ctx context.Context, version string, key, aad []byte) ([]byte, error) {
	if w.db.active {
		panic("key unwrap inside SQL")
	}
	w.unwraps++
	return testWrapper{}.Unwrap(ctx, version, key, aad)
}

// repairWire prepares a real AEAD envelope under an independently fixed synthetic binding.
func repairWire(t *testing.T) (Binding, map[string]any) {
	t.Helper()
	b := binding()
	v, e := Seal(context.Background(), testWrapper{}, b, connections.NewSecret("access-sandbox-fictional-private"))
	if e != nil {
		t.Fatal(e)
	}
	return b, map[string]any{"envelope": v, "credential_ref": b.Reference.ReferenceID,
		"credential_version": b.Reference.Version, "item_id": b.Reference.ItemID, "environment": b.Reference.Environment}
}

// TestRepairCredentialRead proves exact-address existing AEAD readback with no fresh wrap or generic lookup.
func TestRepairCredentialRead(t *testing.T) {
	b, wire := repairWire(t)
	db := &repairDB{scope: connections.Scope{Actor: repairAttempt, Entity: b.Entity, Book: b.Book}}
	db.raw, _ = json.Marshal(wire)
	w := &repairWrapper{db: db}
	store, _ := NewAPIStore(db, w)
	r := store.ForScope(db.scope).(connections.RepairCredentialReader)
	got, e := r.ReadRepair(context.Background(), repairAttempt, b.Reference)
	if e != nil || got.Reference != b.Reference || w.wraps != 0 || w.unwraps != 1 || db.calls != 1 {
		t.Fatal("wrong purpose read")
	}
	_ = got.Token.WithValue(func(v string) error {
		if v != "access-sandbox-fictional-private" {
			t.Fatal("wrong existing credential")
		}
		return nil
	})
	if _, e = store.ReadRepair(context.Background(), repairAttempt, b.Reference); e == nil || db.calls != 1 || w.unwraps != 1 {
		t.Fatal("unscoped store gained authority")
	}
}

// TestRepairCredentialDeniedBeforeUnwrap prevents SQL uncertainty, malformed identity and wrong-address results from reaching KMS.
func TestRepairCredentialDeniedBeforeUnwrap(t *testing.T) {
	for _, kind := range []string{"denied", "unknown_commit", "reference", "version", "item", "environment", "missing", "null", "unknown_field", "wrong_type"} {
		t.Run(kind, func(t *testing.T) {
			b, wire := repairWire(t)
			db := &repairDB{scope: connections.Scope{Actor: repairAttempt, Entity: b.Entity, Book: b.Book}}
			switch kind {
			case "denied":
				db.queryErr = &pgconn.PgError{Code: "P2704"}
			case "unknown_commit":
				db.commitErr = errors.New("synthetic reply loss")
			case "reference":
				wire["credential_ref"] = repairAttempt
			case "version":
				wire["credential_version"] = "2"
			case "item":
				wire["item_id"] = "foreign-item"
			case "environment":
				wire["environment"] = "production"
			case "missing":
				delete(wire, "credential_ref")
			case "null":
				wire["envelope"] = nil
			case "unknown_field":
				wire["extra"] = true
			case "wrong_type":
				wire["credential_version"] = 1
			}
			db.raw, _ = json.Marshal(wire)
			w := &repairWrapper{db: db}
			store, _ := NewAPIStore(db, w)
			r := store.ForScope(db.scope).(connections.RepairCredentialReader)
			if _, e := r.ReadRepair(context.Background(), repairAttempt, b.Reference); e == nil || w.wraps != 0 || w.unwraps != 0 || db.calls != 1 {
				t.Fatal("denied, uncertain or mismatched result used key authority")
			}
		})
	}
	b, _ := repairWire(t)
	db := &repairDB{scope: connections.Scope{Actor: "invalid", Entity: b.Entity, Book: b.Book}}
	w := &repairWrapper{db: db}
	store, _ := NewAPIStore(db, w)
	if _, e := store.ForScope(db.scope).(connections.RepairCredentialReader).ReadRepair(context.Background(), repairAttempt, b.Reference); e == nil || db.calls != 0 {
		t.Fatal("malformed caller reached SQL")
	}
}

// TestRepairCredentialTamper denies modified ciphertext and wrong AAD while never creating a replacement envelope.
func TestRepairCredentialTamper(t *testing.T) {
	for _, kind := range []string{"ciphertext", "foreign_aad"} {
		t.Run(kind, func(t *testing.T) {
			b, wire := repairWire(t)
			v := wire["envelope"].(Envelope)
			if kind == "ciphertext" {
				raw, _ := base64.StdEncoding.DecodeString(v.Ciphertext)
				raw[0] ^= 1
				v.Ciphertext = base64.StdEncoding.EncodeToString(raw)
			} else {
				foreign := b
				foreign.Book = repairAttempt
				v, _ = Seal(context.Background(), testWrapper{}, foreign, connections.NewSecret("foreign-private"))
			}
			wire["envelope"] = v
			db := &repairDB{scope: connections.Scope{Actor: repairAttempt, Entity: b.Entity, Book: b.Book}}
			db.raw, _ = json.Marshal(wire)
			w := &repairWrapper{db: db}
			store, _ := NewAPIStore(db, w)
			if _, e := store.ForScope(db.scope).(connections.RepairCredentialReader).ReadRepair(context.Background(), repairAttempt, b.Reference); e == nil || w.wraps != 0 || w.unwraps != 1 {
				t.Fatal("tampered or cross-scope envelope returned plaintext")
			}
		})
	}
}
