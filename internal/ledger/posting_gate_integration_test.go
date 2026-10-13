//go:build integration

package ledger

import (
	"context"
	"testing"
)

// TestDeletedUnusedPrincipalDenied distinguishes a physically removed actor from an invented UUID.
func TestDeletedUnusedPrincipalDenied(t *testing.T) {
	c, s, _ := fixtureDB(t, "deleted_unused_principal")
	deleted := "00000000-0000-4000-8000-000000000090"
	sql(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','deleted-before-use')`, deleted)
	sql(t, c, `DELETE FROM ledger.principal WHERE principal_id=$1`, deleted)
	if scalar(t, c, `SELECT count(*)::text FROM ledger.principal WHERE principal_id=$1`, deleted) != "0" {
		t.Fatal("fixture actor was not physically deleted")
	}
	before := ledgerProofSnapshot(t, c)
	cmd := command("deleted-actor")
	cmd.Actor = deleted
	_, err := s.Post(context.Background(), cmd)
	requireFailure(t, err, "not_found")
	requireLedgerUnchanged(t, c, before)
	assertEmpty(t, c)
}

// TestHistoricalPrincipalRetentionAndDisable proves retention and access denial are separate boundaries.
func TestHistoricalPrincipalRetentionAndDisable(t *testing.T) {
	c, s, _ := fixtureDB(t, "historical_principal_retention")
	// A separate poster has no policy-approval reference; after removing membership,
	// the remaining principal references come from committed posting history.
	historical := "00000000-0000-4000-8000-000000000091"
	sql(t, c, `INSERT INTO ledger.principal(principal_id,issuer,subject) VALUES($1,'synthetic','historical-poster')`, historical)
	sql(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL),($1,$2,'read',NULL)`, entity, historical)
	cmd := command("historical-actor")
	cmd.Actor = historical
	posted, err := s.Post(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	sql(t, c, `DELETE FROM ledger.membership WHERE principal_id=$1`, historical)
	before := ledgerProofSnapshot(t, c)
	_, err = c.Exec(context.Background(), `DELETE FROM ledger.principal WHERE principal_id=$1`, historical)
	requireLedgerSQLState(t, err, "23503")
	requireLedgerUnchanged(t, c, before)
	if scalar(t, c, `SELECT count(*)::text FROM ledger.principal WHERE principal_id=$1`, historical) != "1" {
		t.Fatal("historical actor was deleted")
	}
	// Restore memberships before disabling: denial must come from the disabled
	// principal, not missing membership, including a previously committed receipt.
	sql(t, c, `INSERT INTO ledger.membership VALUES($1,$2,'post',NULL),($1,$2,'read',NULL)`, entity, historical)
	sql(t, c, `UPDATE ledger.principal SET disabled_at=now() WHERE principal_id=$1`, historical)
	before = ledgerProofSnapshot(t, c)
	_, err = s.Post(context.Background(), cmd)
	requireFailure(t, err, "not_found")
	requireLedgerUnchanged(t, c, before)
	fresh := change(cmd, func(v map[string]any) { v["expected_revision"] = "1" })
	fresh.Key = "disabled-new-command"
	_, err = s.Post(context.Background(), fresh)
	requireFailure(t, err, "not_found")
	requireLedgerUnchanged(t, c, before)
	read := cmd
	read.Journal = postedID(t, posted)
	_, err = s.Journal(context.Background(), read)
	requireFailure(t, err, "not_found")
	requireLedgerUnchanged(t, c, before)
	// A different still-authorized actor retains historical visibility.
	read.Actor = actor
	if _, err = s.Journal(context.Background(), read); err != nil {
		t.Fatal("authorized historical read failed", err)
	}
	requireLedgerUnchanged(t, c, before)
}
