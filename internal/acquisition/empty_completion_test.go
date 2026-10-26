package acquisition

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
)

// emptyCompletionDB returns a fixed SQL-shaped empty publication receipt, independent of production rendering.
type emptyCompletionDB struct {
	faultDB
	body string
}

// BeginTx reuses strict transaction checks and replaces only returned synthetic SQL metadata.
func (d *emptyCompletionDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable {
		panic("synthetic isolation mismatch")
	}
	return &emptyCompletionTx{transactionFault: &transactionFault{db: &d.faultDB}, body: d.body}, nil
}

// emptyCompletionTx preserves the existing role/cleanup/commit fixture behavior.
type emptyCompletionTx struct {
	*transactionFault
	body string
}

// QueryRow returns the real adapter's role proof or the independently authored receipt bytes.
func (tx *emptyCompletionTx) QueryRow(_ context.Context, q string, _ ...any) pgx.Row {
	if q == syncRoleQuery {
		return faultRow{role: true}
	}
	return emptyCompletionRow(tx.body)
}

// emptyCompletionRow models the JSON result column without deriving expected amounts or revisions.
type emptyCompletionRow string

func (r emptyCompletionRow) Scan(dest ...any) error { *dest[0].(*[]byte) = []byte(r); return nil }

const emptyCompletionFixture = `{"stream_id":"00000000-0000-4000-8000-000000000003","set_id":"00000000-0000-4000-8000-000000000004","cursor":"empty-history-cursor","completed_generation":"1","source_revision":"0","changed_count":"0","readiness":"NOT_READY"}`

// TestInitialEmptyCompletionAndResume proves both adapter paths accept SQL's legitimate initial zero revision.
func TestInitialEmptyCompletionAndResume(t *testing.T) {
	var c completion
	if json.Unmarshal([]byte(emptyCompletionFixture), &c) != nil {
		t.Fatal("fixture")
	}
	s := Scope{EntityID: "00000000-0000-4000-8000-000000000001", BookID: "00000000-0000-4000-8000-000000000002", StreamID: c.StreamID}
	a := attempt{SetID: c.SetID, Fence: "1"}
	db := &emptyCompletionDB{body: emptyCompletionFixture}
	r := SQLRepository{DB: db}
	out, e := r.Complete(context.Background(), s, a)
	if e != nil || out.SourceRevision != "0" || out.ChangedCount != "0" {
		t.Fatal("initial empty completion rejected", e)
	}
	db.body = `{"state":"completed","set_id":"` + c.SetID + `","fence":"1","completion":` + emptyCompletionFixture + `}`
	resumed, e := r.Resume(context.Background(), s, "00000000-0000-4000-8000-000000000005")
	if e != nil || resumed.Completion == nil || resumed.Completion.SourceRevision != "0" {
		t.Fatal("committed empty resume rejected", e)
	}
	for _, bad := range []string{"", "-1", "00", "01", "+0", " 0", "1.0", "9223372036854775808"} {
		v := c
		v.SourceRevision = bad
		if validCompletion(v, s, a) {
			t.Fatalf("invalid revision accepted: %q", bad)
		}
	}
	for _, good := range []string{"0", "1", "9223372036854775807"} {
		v := c
		v.SourceRevision = good
		if !validCompletion(v, s, a) {
			t.Fatalf("valid revision rejected: %q", good)
		}
	}
	c.CompletedGeneration = "0"
	if validCompletion(c, s, a) {
		t.Fatal("zero generation accepted")
	}
}
