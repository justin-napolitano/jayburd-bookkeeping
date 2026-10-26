package bankcredential

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const prodKey = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/bank"
const prodLinkKey = "projects/fictional-production/locations/us-central1/keyRings/pilot/cryptoKeys/link"

// keyedWrapper independently authenticates a fictional key identity and supplied AAD without network.
type keyedWrapper struct {
	key   string
	calls int
}

// Wrap authenticates fictional key material and AAD independently of the production wrapper.
func (w *keyedWrapper) Wrap(_ context.Context, key, aad []byte) (string, []byte, error) {
	w.calls++
	b, _ := aes.NewCipher(bytes.Repeat([]byte{31}, 32))
	g, _ := cipher.NewGCM(b)
	return w.key + "/cryptoKeyVersions/7", g.Seal(nil, make([]byte, 12), key, aad), nil
}

// Unwrap rejects unexpected immutable versions before independent AEAD verification.
func (w *keyedWrapper) Unwrap(_ context.Context, version string, raw, aad []byte) ([]byte, error) {
	w.calls++
	if version != w.key+"/cryptoKeyVersions/7" {
		return nil, ErrUnavailable
	}
	b, _ := aes.NewCipher(bytes.Repeat([]byte{31}, 32))
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, make([]byte, 12), raw, aad)
}

// TestProductionCredentialEnvironmentBinding exercises real AEAD with independent fixture keys and identity changes.
func TestProductionCredentialEnvironmentBinding(t *testing.T) {
	ctx := context.Background()
	b := binding()
	b.Reference.Environment = "production"
	w := &keyedWrapper{key: prodKey}
	v, e := Seal(ctx, w, b, connections.NewSecret("synthetic-production-token"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Open(ctx, w, b, v); e != nil {
		t.Fatal("Production roundtrip rejected")
	}
	for _, field := range []string{"environment", "item", "book", "entity", "reference", "version"} {
		t.Run(field, func(t *testing.T) {
			wrong := b
			switch field {
			case "environment":
				wrong.Reference.Environment = "sandbox"
			case "item":
				wrong.Reference.ItemID = "foreign-item"
			case "book":
				wrong.Book = b.Entity
			case "entity":
				wrong.Entity = b.Book
			case "reference":
				wrong.Reference.ReferenceID = b.Entity
			case "version":
				wrong.Reference.Version = "2"
			}
			if _, e := Open(ctx, w, wrong, v); e == nil {
				t.Fatal("substituted identity decrypted")
			}
		})
	}
	if _, e = Open(ctx, &keyedWrapper{key: prodLinkKey}, b, v); e == nil {
		t.Fatal("foreign key accepted")
	}
	sc := connections.Scope{Actor: b.Reference.ReferenceID, Entity: b.Entity, Book: b.Book}
	id := "00000000-0000-4000-8000-000000000004"
	lw := &keyedWrapper{key: prodLinkKey}
	link, e := sealLinkEnvironment(ctx, lw, sc, id, connections.NewSecret("synthetic-link"), "production")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = openLinkEnvironment(ctx, lw, sc, id, link, "production"); e != nil {
		t.Fatal(e)
	}
	if _, e = openLink(ctx, lw, sc, id, link); e == nil {
		t.Fatal("Production handoff opened as Sandbox")
	}
	old, e := sealLink(ctx, testWrapper{}, sc, id, connections.NewSecret("synthetic-legacy-link"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = openLink(ctx, testWrapper{}, sc, id, old); e != nil {
		t.Fatal("legacy tuple changed")
	}
}

// projectionDB returns only fictional SQL metadata while recording when network capabilities could be used.
type projectionDB struct {
	raw     []byte
	queries []string
}

// projectionTx models the bounded SQL transaction without remote effects.
type projectionTx struct {
	pgx.Tx
	db *projectionDB
}

// projectionRow supplies one original fictional SQL projection.
type projectionRow struct{ raw []byte }

// BeginTx opens only an injected metadata snapshot.
func (d *projectionDB) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return &projectionTx{db: d}, nil
}

// Exec accepts adapter-local transaction settings without changing fixture state.
func (t *projectionTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// QueryRow records the selected operation for dependency-order assertions.
func (t *projectionTx) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	t.db.queries = append(t.db.queries, q)
	return projectionRow{t.db.raw}
}

// Commit confirms the synthetic metadata result.
func (t *projectionTx) Commit(context.Context) error { return nil }

// Rollback releases the synthetic snapshot after adapter cleanup.
func (t *projectionTx) Rollback(context.Context) error { return nil }

// Scan returns only originally authored fictional metadata bytes.
func (r projectionRow) Scan(dest ...any) error { *dest[0].(*[]byte) = r.raw; return nil }

// TestProductionPreflightBeforeBegin proves mismatched current SQL metadata cannot create an attempt or invoke KMS/provider.
func TestProductionPreflightBeforeBegin(t *testing.T) {
	for _, change := range []string{"environment", "origin", "bank", "link", "revision", "missing"} {
		t.Run(change, func(t *testing.T) {
			c := map[string]any{"environment": "production", "data_origin": "provider", "scope_revision": "1", "bank_key": prodKey, "link_key": prodLinkKey}
			switch change {
			case "environment":
				c["environment"] = "sandbox"
			case "origin":
				c["data_origin"] = "test"
			case "bank":
				c["bank_key"] = prodLinkKey
			case "link":
				c["link_key"] = prodKey
			case "revision":
				c["scope_revision"] = "01"
			case "missing":
				delete(c, "environment")
			}
			raw, _ := json.Marshal(c)
			db := &projectionDB{raw: raw}
			bank, link := &keyedWrapper{key: prodKey}, &keyedWrapper{key: prodLinkKey}
			store, e := NewProductionAPIStore(db, bank, link, ProductionPolicy{prodKey, prodLinkKey})
			if e != nil {
				t.Fatal(e)
			}
			scope := connections.Scope{Actor: binding().Reference.ReferenceID, Entity: binding().Entity, Book: binding().Book}
			scoped := store.ForScope(scope).(*scopedStore)
			if scoped.AuthorizeProvider(context.Background()) == nil || bank.calls != 0 || link.calls != 0 || len(db.queries) != 1 || !strings.Contains(db.queries[0], "connection_provider_context") {
				t.Fatal("mismatch escaped current scoped preflight")
			}
		})
	}
}

// TestProductionResolverRejectsMixedContext proves foreign expected environment fails before SQL/KMS/static reads.
func TestProductionResolverRejectsMixedContext(t *testing.T) {
	db := &projectionDB{}
	w := &keyedWrapper{key: prodKey}
	static := 0
	r, e := NewProductionResolver(db, w, ProductionPolicy{prodKey, prodLinkKey}, func(context.Context) (acquisition.ProviderCredentials, error) {
		static++
		return acquisition.ProviderCredentials{}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	for _, env := range []string{"sandbox", "", "Production"} {
		_, e = r.ResolveScoped(context.Background(), acquisition.Scope{}, "", "", acquisition.CredentialReference{Environment: env})
		if e == nil || len(db.queries) != 0 || w.calls != 0 || static != 0 {
			t.Fatal("mixed profile reached dependency")
		}
	}
}
