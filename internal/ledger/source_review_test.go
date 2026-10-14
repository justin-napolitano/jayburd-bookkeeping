package ledger

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

// reviewTransportStub records only wire dispatch, never accounting effects.
type reviewTransportStub struct {
	operationsStub
	action string
}

// SourceReview proves verified scope and unchanged draft input reach the SQL adapter.
func (s *reviewTransportStub) SourceReview(ctx context.Context, c Command, action string) (Result, error) {
	s.action = action
	return s.Post(ctx, c)
}

// TestSourceReviewTransport denies malformed queries and unauthorized access before calls.
func TestSourceReviewTransport(t *testing.T) {
	base := "/v1/entities/e/books/b"
	for _, tc := range []struct {
		name, method, path, body, key string
		status, calls                 int
	}{
		{"head", "HEAD", base + "/sources", "", "", 405, 0},
		{"sources", "GET", base + "/sources", "", "", 200, 1},
		{"detail", "GET", base + "/review-items/r", "", "", 200, 1},
		{"categories", "GET", base + "/review-items/r/categories", "", "", 200, 1},
		{"list", "GET", base + "/review-items?source_account_id=00000000-0000-4000-8000-000000000001&limit=100", "", "", 200, 1},
		{"correction_hold", "GET", base + "/review-items?source_account_id=00000000-0000-4000-8000-000000000001&state=correction_hold", "", "", 200, 1},
		{"duplicate", "GET", base + "/review-items?source_account_id=x&source_account_id=y", "", "", 400, 0},
		{"unknown", "GET", base + "/sources?ignored=yes", "", "", 400, 0},
		{"missing_source", "GET", base + "/review-items", "", "", 400, 0},
		{"limit", "GET", base + "/review-items?source_account_id=00000000-0000-4000-8000-000000000001&limit=101", "", "", 400, 0},
		{"getbody", "GET", base + "/sources", "{}", "", 400, 0},
		{"draft", "POST", base + "/review-items/r/drafts", `{"duplicate":1,"duplicate":2}`, "same-key", 200, 1},
		{"draftkey", "POST", base + "/review-items/r/drafts", "{}", "", 400, 0},
		{"draftlimit", "POST", base + "/review-items/r/drafts", strings.Repeat("a", 65537), "key", 413, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &reviewTransportStub{operationsStub: operationsStub{result: Result{Status: 200, Body: []byte(`{}`)}}}
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer synthetic")
			if tc.key != "" {
				r.Header.Set("Idempotency-Key", tc.key)
			}
			w := httptest.NewRecorder()
			Handler(authStub{}, s).ServeHTTP(w, r)
			if w.Code != tc.status || s.calls != tc.calls {
				t.Fatalf("status %d calls %d", w.Code, s.calls)
			}
			if tc.name == "draft" && (s.command.Body != tc.body || s.command.ReviewID != "r" || s.command.Actor == "") {
				t.Fatal("draft scope/body changed")
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cacheable")
			}
		})
	}
	s := &reviewTransportStub{}
	r := httptest.NewRequest("GET", base+"/sources", nil)
	r.Header.Set("Authorization", "Bearer synthetic")
	w := httptest.NewRecorder()
	Handler(authStub{errors.New("private identity")}, s).ServeHTTP(w, r)
	if w.Code != 401 || s.calls != 0 || strings.Contains(w.Body.String(), "private") {
		t.Fatal("identity boundary")
	}
}
