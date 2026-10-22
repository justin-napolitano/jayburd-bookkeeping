package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
)

// manifestFixture writes only fixed synthetic expectation bytes and reviewed metadata into a private temporary folder.
func manifestFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fixture := []byte(`{"synthetic-fixture":true}`)
	identities := []byte(`{"synthetic-map":true}`)
	fh := sha256.Sum256(fixture)
	ih := sha256.Sum256(identities)
	m := manifest{Profile: "jayburd-sandbox-verification-manifest/1", Instance: "jayburd-bookkeeping-dev:us-central1:jayburd-dev", MandateID: "00000000-0000-4000-8000-000000000001", MandateRevision: "1", FixtureSHA256: hex.EncodeToString(fh[:]), IdentityMapSHA256: hex.EncodeToString(ih[:])}
	raw, _ := json.Marshal(m)
	for name, data := range map[string][]byte{"manifest.json": raw, "fixture.json": fixture, "identity-map.json": identities} {
		if os.WriteFile(filepath.Join(dir, name), data, 0600) != nil {
			t.Fatal("synthetic fixture setup failed")
		}
	}
	return filepath.Join(dir, "manifest.json")
}
func TestOfflineAndInvalidSelection(t *testing.T) {
	connect := func(context.Context, clouddb.Config) (connection, error) {
		t.Fatal("inert invocation constructed metadata transport")
		return connection{}, nil
	}
	for _, args := range [][]string{nil, {"-action=verify"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out, connect); e != nil || out.String() != "{\"offline\":true}\n" {
			t.Fatal("default not inert")
		}
	}
	for _, args := range [][]string{{"-action=compare"}, {"-action=verify", "-manifest=/private/nonexistent"}, {"-action=compare", "-manifest=relative.json"}, {"-action=compare", "-manifest=/private/../manifest.json"}, {"-action=sync"}, {"-dsn=synthetic"}, {"trailing"}} {
		var out bytes.Buffer
		if e := run(context.Background(), args, &out, connect); e == nil || out.Len() != 0 {
			t.Fatal("ambiguous configuration accepted")
		}
	}
}
func TestManifestStrictness(t *testing.T) {
	path := manifestFixture(t)
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal("synthetic manifest unavailable")
	}
	for _, bad := range [][]byte{
		bytes.Replace(raw, []byte(`"profile":`), []byte(`"profile":"duplicate","profile":`), 1),
		bytes.Replace(raw, []byte(`"profile":`), []byte(`"unknown":true,"profile":`), 1),
		bytes.Replace(raw, []byte("jayburd-bookkeeping-dev:us-central1"), []byte("production-project:us-central1"), 1),
		bytes.Replace(raw, []byte(`"mandate_revision":"1"`), []byte(`"mandate_revision":"01"`), 1),
		append(raw, []byte(`{}`)...),
		bytes.Repeat([]byte(" "), 8193),
	} {
		if _, e := parseManifest(bad); e == nil {
			t.Fatal("unsafe manifest accepted")
		}
	}
}
func TestFileBoundaryBeforeConnector(t *testing.T) {
	for _, mode := range []string{"digest", "symlink", "oversize", "directory"} {
		t.Run(mode, func(t *testing.T) {
			path := manifestFixture(t)
			fixture := filepath.Join(filepath.Dir(path), "fixture.json")
			switch mode {
			case "digest":
				if os.WriteFile(fixture, []byte(`{"changed":true}`), 0600) != nil {
					t.Fatal("fixture write")
				}
			case "symlink":
				target := filepath.Join(filepath.Dir(path), "other.json")
				if os.Rename(fixture, target) != nil || os.Symlink(target, fixture) != nil {
					t.Fatal("synthetic symlink setup")
				}
			case "oversize":
				if os.WriteFile(fixture, make([]byte, (2<<20)+1), 0600) != nil {
					t.Fatal("fixture write")
				}
			case "directory":
				if os.Remove(fixture) != nil || os.Mkdir(fixture, 0700) != nil {
					t.Fatal("synthetic directory setup")
				}
			}
			var out bytes.Buffer
			connect := func(context.Context, clouddb.Config) (connection, error) {
				t.Fatal("unsafe file reached transport")
				return connection{}, nil
			}
			if e := run(context.Background(), []string{"-action=compare", "-manifest=" + path}, &out, connect); e == nil || out.Len() != 0 {
				t.Fatal("unsafe file accepted")
			}
		})
	}
}
func TestFixedDependencyErrorsAndCleanup(t *testing.T) {
	path := manifestFixture(t)
	for _, mode := range []string{"connect", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			var out bytes.Buffer
			calls, closed := 0, 0
			connect := func(_ context.Context, c clouddb.Config) (connection, error) {
				calls++
				if c.Component != "verifier" || c.Project() != "jayburd-bookkeeping-dev" {
					t.Fatal("role substitution")
				}
				if mode == "connect" {
					return connection{}, errors.New("synthetic-private-diagnostic")
				}
				return connection{nil, func() { closed++ }}, nil
			}
			e := run(context.Background(), []string{"-action=compare", "-manifest=" + path}, &out, connect)
			if e == nil || strings.Contains(e.Error(), "private") || out.Len() != 0 || calls != 1 || closed != map[string]int{"connect": 0, "unavailable": 1}[mode] {
				t.Fatal("dependency failure leaked/retained evidence")
			}
		})
	}
}

// commandPool supplies only the existing adapter's static SQL result shape, never cloud or provider behavior.
type commandPool struct{ tx *commandTx }

func (p commandPool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) { return p.tx, nil }

type commandTx struct {
	pgx.Tx
	raw         string
	commitError bool
	commits     int
}

func (t *commandTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *commandTx) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	return commandRow{identity: strings.Contains(q, "session_user"), raw: t.raw}
}
func (t *commandTx) Commit(context.Context) error {
	t.commits++
	if t.commitError {
		return errors.New("synthetic-private-commit")
	}
	return nil
}
func (*commandTx) Rollback(context.Context) error { return nil }

type commandRow struct {
	identity bool
	raw      string
}

func (r commandRow) Scan(dest ...any) error {
	if r.identity {
		*dest[0].(*bool) = true
	} else {
		*dest[0].(*[]byte) = []byte(r.raw)
	}
	return nil
}

func TestResultEmissionAndCommitUncertainty(t *testing.T) {
	const result = `{"equal":true,"missing_count":"0","extra_count":"0","field_mismatch_count":"0","review_mismatch_count":"0","lineage_mismatch_count":"0","journal_count":"0","journal_outbox_count":"0","no_automatic_journals":true}`
	for _, mode := range []string{"equal", "unequal", "unknown-commit"} {
		t.Run(mode, func(t *testing.T) {
			path := manifestFixture(t)
			tx := &commandTx{raw: result, commitError: mode == "unknown-commit"}
			closed := 0
			if mode == "unequal" {
				tx.raw = strings.Replace(strings.Replace(result, `"equal":true`, `"equal":false`, 1), `"extra_count":"0"`, `"extra_count":"1"`, 1)
			}
			var out bytes.Buffer
			connect := func(context.Context, clouddb.Config) (connection, error) {
				return connection{commandPool{tx}, func() { closed++ }}, nil
			}
			e := run(context.Background(), []string{"-action=compare", "-manifest=" + path}, &out, connect)
			if closed != 1 || tx.commits != 1 {
				t.Fatal("cleanup/commit repeated")
			}
			if mode == "unknown-commit" {
				if e != errCommand || out.Len() != 0 {
					t.Fatal("unconfirmed comparison emitted evidence")
				}
				return
			}
			if mode == "equal" && e != nil || mode == "unequal" && e != errMismatch {
				t.Fatal("comparison exit classification")
			}
			fields := map[string]any{}
			if json.Unmarshal(out.Bytes(), &fields) != nil || len(fields) != 9 || strings.Contains(out.String(), "synthetic") || strings.Contains(out.String(), "mandate") {
				t.Fatal("unsafe result output")
			}
		})
	}
}
