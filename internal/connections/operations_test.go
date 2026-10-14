package connections

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// operationRepo captures exact bigint strings and fixed SQL operation selection.
type operationRepo struct {
	op    string
	args  []any
	err   error
	calls int
}

// call simulates only transport; actual current authorization belongs to SQL proofs.
func (r *operationRepo) call(_ context.Context, op string, args ...any) (json.RawMessage, error) {
	r.calls++
	r.op = op
	r.args = args
	return json.RawMessage(`{"queued":true}`), r.err
}

// TestConnectionOperations checks exact pins, invalid input and uncertain outcome forwarding without credentials.
func TestConnectionOperations(t *testing.T) {
	r := &operationRepo{}
	s := &Service{repo: r}
	ctx := context.Background()
	in := StreamOperation{"request", "9007199254740993", "2", "3"}
	if _, err := s.OperateStream(ctx, scope, attemptID, "same-key", in); err != nil {
		t.Fatal(err)
	}
	var wire StreamOperation
	if json.Unmarshal([]byte(r.args[5].(string)), &wire) != nil || wire != in || r.op != "mutate_connection_stream" {
		t.Fatal("operation pins changed")
	}
	for _, bad := range []StreamOperation{{"refresh", "1", "1", "1"}, {"pause", "01", "1", "1"}, {"resume", "1", "1", ""}, {"request", "1", "9223372036854775808", "1"}} {
		before := r.calls
		if _, err := s.OperateStream(ctx, scope, attemptID, "key", bad); err == nil || r.calls != before {
			t.Fatal("invalid operation reached SQL")
		}
	}
	if _, err := s.Attempts(ctx, scope, 100, nil); err != nil || r.op != "list_connection_attempts" {
		t.Fatal("discovery failed")
	}
	for _, n := range []int{0, 101} {
		before := r.calls
		if _, err := s.Attempts(ctx, scope, n, nil); err == nil || r.calls != before {
			t.Fatal("unbounded discovery")
		}
	}
	if _, err := s.StreamStatus(ctx, scope, attemptID); err != nil || r.op != "connection_stream_status" {
		t.Fatal("status failed")
	}
	r.err = failed("commit_unknown")
	before := r.calls
	if _, err := s.OperateStream(ctx, scope, attemptID, "same-key", in); err == nil || err.Error() != "commit_unknown" || r.calls != before+1 {
		t.Fatal("unknown commit retried or hidden")
	}
	for sql, code := range map[string]string{"P2800": "invalid_request", "P2801": "not_found", "P2802": "connection_conflict", "P2803": "stale_connection", "P2804": "connection_operation_blocked", "P2805": "connection_limit"} {
		if got := sqlFailure(&pgconn.PgError{Code: sql, Message: "private detail"}); got.Error() != code {
			t.Fatal("unsafe SQL failure mapping")
		}
	}
}
