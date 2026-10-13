package ledger

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
)

// TestReportJournalAdapter proves bounded retries and no payload before commit acknowledgement.
func TestReportJournalAdapter(t *testing.T) {
	for _, tc := range []struct {
		name        string
		row, commit error
		code        string
		calls       int
	}{
		{name: "success", calls: 1},
		{name: "stale", row: &pgconn.PgError{Code: "P2114"}, code: "report_snapshot_changed", calls: 1},
		{name: "retry", row: &pgconn.PgError{Code: "40001"}, code: "temporarily_unavailable", calls: 2},
		{name: "lost_commit", commit: errors.New("private transport detail"), code: "temporarily_unavailable", calls: 1},
		{name: "missing", row: &pgconn.PgError{Code: "P2101"}, code: "not_found", calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &reportDBStub{errorRow: tc.row, commitErr: tc.commit}
			s := Service{DB: d}
			raw := `{"snapshot_revision":"1","snapshot_revision":"2"}`
			r, err := s.ReportJournal(context.Background(), Command{Body: raw})
			if d.calls != tc.calls || d.options.IsoLevel != pgx.RepeatableRead || d.options.AccessMode == pgx.ReadOnly {
				t.Fatal("unsafe transaction", d.options)
			}
			if tc.code != "" {
				var f *Failure
				if !errors.As(err, &f) || f.Code != tc.code || len(r.Body) != 0 {
					t.Fatal("unsafe failure", err)
				}
			} else if err != nil || r.Status != 200 || d.raw != raw {
				t.Fatal("changed request boundary", err)
			}
		})
	}
}
