//go:build integration

package connectionops

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const canonicalHandoffKey = "projects/jayburd-synthetic-test/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1"

// handoffEnvelope is a fixed SQL-profile fixture, not cryptographic acceptance or provider evidence.
func handoffEnvelope(key string) map[string]any {
	return map[string]any{"profile": "jayburd-hosted-link-aead-v1", "key_version": key, "wrapped_key": "AQ==",
		"nonce": strings.Repeat("A", 16), "ciphertext": strings.Repeat("A", 23) + "="}
}

// handoffPut uses the actual API login; no direct runtime table access is introduced.
func handoffPut(t testing.TB, db string, a map[string]any, envelope map[string]any) (string, error) {
	t.Helper()
	return invoke(t, db, "synthetic_runtime", "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], raw(envelope))
}

// handoffMigration executes only the named public migration in a disposable proof database.
func handoffMigration(t testing.TB, c *pgx.Conn, direction string) error {
	t.Helper()
	if direction != "up" && direction != "down" {
		t.Fatal("unsupported fixture direction")
	}
	b, e := os.ReadFile("/proof/migrations/000033_owned_handoff_kms." + direction + ".sql")
	if e != nil {
		t.Fatal(e)
	}
	_, e = c.Exec(context.Background(), string(b))
	if e != nil {
		_, _ = c.Exec(context.Background(), "ROLLBACK")
	}
	return e
}

// TestHostedRepairSQLCanonicalHandoff proves33 restores reviewed20 grammar without widening scope or effects.
func TestHostedRepairSQLCanonicalHandoff(t *testing.T) {
	for _, key := range []string{canonicalHandoffKey, "projects/9088794002/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials/cryptoKeyVersions/1"} {
		name := "canonical"
		if strings.Contains(key, "/9088794002/") {
			name = "numeric"
		}
		t.Run(name, func(t *testing.T) {
			c, db, _, _ := connected(t)
			a := repairStart(t, c, db, "repair")
			envelope := handoffEnvelope(key)
			first, e := handoffPut(t, db, a, envelope)
			if e != nil {
				t.Fatal(e)
			}
			envelope["wrapped_key"] = "Ag=="
			second, e := handoffPut(t, db, a, envelope)
			if e != nil || first != second || value(t, c, "SELECT count(*)::text FROM integration.link_handoff") != "1" {
				t.Fatal("immutable winner changed")
			}
			if call(t, db, "synthetic_runtime", "SELECT integration.get_link_handoff($1,$2,$3,$4)::text", actor, entity, book, a["attempt_id"]) != first {
				t.Fatal("read changed stored bytes")
			}
			_, e = invoke(t, db, "synthetic_webhook", "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", actor, entity, book, a["attempt_id"], raw(envelope))
			requireCode(t, e, "42501")
			_, e = invoke(t, db, "synthetic_runtime", "SELECT integration.put_link_handoff($1,$2,$3,$4,$5)::text", other, entity, book, a["attempt_id"], raw(envelope))
			requireCode(t, e, "P2701")
			requireNoRepair(t, c)
		})
	}
}

// TestHostedRepairSQLHandoffProfile rejects independently chosen malformed key/envelope fields before publication.
func TestHostedRepairSQLHandoffProfile(t *testing.T) {
	c, db, _, _ := connected(t)
	a := repairStart(t, c, db, "repair")
	for _, key := range []string{
		strings.Replace(canonicalHandoffKey, "jayburd-synthetic-test", "Bad_Project", 1),
		strings.Replace(canonicalHandoffKey, "jayburd-synthetic-test", "abcd", 1),
		strings.Replace(canonicalHandoffKey, "jayburd-synthetic-test", "0001", 1),
		strings.Replace(canonicalHandoffKey, "us-central1", "us-east1", 1),
		strings.Replace(canonicalHandoffKey, "bank-credentials", "other-key", 1),
		strings.Replace(canonicalHandoffKey, "/1", "/0", 1), canonicalHandoffKey + "/extra",
	} {
		_, e := handoffPut(t, db, a, handoffEnvelope(key))
		requireCode(t, e, "P2700")
	}
	for _, field := range []string{"profile", "nonce", "ciphertext", "extra"} {
		v := handoffEnvelope(canonicalHandoffKey)
		v[field] = "invalid"
		_, e := handoffPut(t, db, a, v)
		requireCode(t, e, "P2700")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.link_handoff") != "0" {
		t.Fatal("invalid envelope was retained")
	}
	requireNoRepair(t, c)
}

// TestHostedRepairSQLHandoffMigration proves32 rejection→33 acceptance and a fail-before-change rollback guard.
func TestHostedRepairSQLHandoffMigration(t *testing.T) {
	c, db, _, _ := connected(t)
	a := repairStart(t, c, db, "repair")
	if e := handoffMigration(t, c, "down"); e != nil {
		t.Fatal(e)
	}
	_, e := handoffPut(t, db, a, handoffEnvelope(canonicalHandoffKey))
	requireCode(t, e, "P2700")
	if e = handoffMigration(t, c, "up"); e != nil {
		t.Fatal(e)
	}
	first, e := handoffPut(t, db, a, handoffEnvelope(canonicalHandoffKey))
	if e != nil {
		t.Fatal(e)
	}
	e = handoffMigration(t, c, "down")
	requireCode(t, e, "55000")
	// Failure rolls back before function replacement or any ciphertext/receipt deletion.
	if second, e := handoffPut(t, db, a, handoffEnvelope(canonicalHandoffKey)); e != nil || second != first {
		t.Fatal("guard changed function or canonical history")
	}
	if value(t, c, "SELECT count(*)::text FROM integration.link_handoff") != "1" {
		t.Fatal("rollback erased handoff")
	}
	requireNoRepair(t, c)
}

// TestHostedRepairSQLHandoffNewMode retains20 canonical support through the untouched nonrepair fallback.
func TestHostedRepairSQLHandoffNewMode(t *testing.T) {
	c, db := setup(t)
	a := link(t, db, begin(t, db, "new", "new", nil, nil, nil))
	first, e := handoffPut(t, db, a, handoffEnvelope(canonicalHandoffKey))
	if e != nil {
		t.Fatal(e)
	}
	// A canonical new-mode envelope does not make33→32 incompatible:20 still handles it.
	if e = handoffMigration(t, c, "down"); e != nil {
		t.Fatal(e)
	}
	if second, e := handoffPut(t, db, a, handoffEnvelope(canonicalHandoffKey)); e != nil || second != first {
		t.Fatal("new-mode canonical support regressed")
	}
	if e = handoffMigration(t, c, "up"); e != nil {
		t.Fatal(e)
	}
}
