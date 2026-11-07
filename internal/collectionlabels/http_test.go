package collectionlabels

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// fixedAuth is a fictional resolver, not token verification or SQL grant evidence.
type fixedAuth struct {
	calls  int
	denied bool
}

func (a *fixedAuth) Authenticate(context.Context, string) (string, error) {
	a.calls++
	if a.denied {
		return "", errors.New("private resolver diagnostic")
	}
	return "independently-verified", nil
}

// fakeOperations records transport boundaries and returns a fixed SQL-shaped envelope.
type fakeOperations struct {
	calls     int
	operation string
	request   Request
	err       error
	override  string
}

func (o *fakeOperations) call(r Request, op string) (json.RawMessage, error) {
	o.calls++
	o.request = r
	o.operation = op
	body := o.override
	if body == "" {
		body = result(op)
	}
	return json.RawMessage(body), o.err
}
func (o *fakeOperations) Catalog(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "catalog")
}
func (o *fakeOperations) Query(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "query")
}
func (o *fakeOperations) Preview(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "preview")
}
func (o *fakeOperations) Prepare(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "prepare")
}
func (o *fakeOperations) Pending(_ context.Context, r Request) (json.RawMessage, error) {
	return o.call(r, "pending")
}
func (o *fakeOperations) Transition(_ context.Context, r Request, op string) (json.RawMessage, error) {
	return o.call(r, op)
}

// path selects only the frozen canonical public route for a test operation.
func path(op string) string {
	base := "/v1/collections/" + workspace
	switch op {
	case "catalog":
		return base + "/taxonomy"
	case "query":
		return base + "/review/query"
	case "preview":
		return base + "/labels/preview"
	case "prepare":
		return base + "/label-commands"
	case "pending":
		return base + "/label-commands/pending"
	default:
		return base + "/label-commands/" + commandID + "/" + op
	}
}

// httpCall injects synthetic identity and exact media framing without a listening socket.
func httpCall(h http.Handler, method, url, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, url, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer synthetic")
	if method == "POST" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// TestLabelHTTPRoutes verifies every route, actor/path separation and exact duplicate-key byte preservation.
func TestLabelHTTPRoutes(t *testing.T) {
	for _, op := range []string{"catalog", "query", "preview", "prepare", "pending", "execute", "cancel", "acknowledge"} {
		t.Run(op, func(t *testing.T) {
			a := &fixedAuth{}
			o := &fakeOperations{}
			method, body := "POST", ` {"duplicate":1,"duplicate":2,"actor":"foreign","amount":"9007199254740993.01"} `
			if op == "catalog" || op == "pending" {
				method = "GET"
				body = ""
			}
			w := httpCall(Handler(a, o), method, path(op), body)
			if w.Code != 200 || o.calls != 1 || a.calls != 1 || o.operation != op || o.request.Actor != "independently-verified" || o.request.Workspace != workspace || o.request.Body != body || w.Body.String() != result(op) {
				t.Fatalf("changed transport status%d", w.Code)
			}
			if op == "execute" || op == "cancel" || op == "acknowledge" {
				if o.request.Command != commandID {
					t.Fatal("command path lost")
				}
			} else if o.request.Command != "" {
				t.Fatal("unexpected command identity")
			}
			if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Type") != "application/json" {
				t.Fatal("private response cache/media boundary")
			}
		})
	}
}

// TestLabelHTTPFraming rejects bounded malformed framing before any private SQL operation.
func TestLabelHTTPFraming(t *testing.T) {
	for _, tc := range []struct {
		name, op, method, url, body string
		edit                        func(*http.Request)
		status                      int
	}{
		{"query_string", "query", "POST", path("query") + "?search=private", "{}", nil, 400},
		{"empty_query", "query", "POST", path("query") + "?", "{}", nil, 400},
		{"missing_auth", "query", "POST", path("query"), "{}", func(r *http.Request) { r.Header.Del("Authorization") }, 401},
		{"duplicate_auth", "query", "POST", path("query"), "{}", func(r *http.Request) { r.Header.Add("Authorization", "Bearer second") }, 401},
		{"blank_auth", "query", "POST", path("query"), "{}", func(r *http.Request) { r.Header.Set("Authorization", "Bearer   ") }, 401},
		{"idempotency", "prepare", "POST", path("prepare"), "{}", func(r *http.Request) { r.Header.Set("Idempotency-Key", "foreign") }, 400},
		{"encoded", "preview", "POST", path("preview"), "{}", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, 400},
		{"media", "query", "POST", path("query"), "{}", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=utf-8") }, 415},
		{"duplicate_media", "query", "POST", path("query"), "{}", func(r *http.Request) { r.Header.Add("Content-Type", "application/json") }, 415},
		{"nil_workspace", "query", "POST", strings.Replace(path("query"), workspace, "00000000-0000-0000-0000-000000000000", 1), "{}", nil, 400},
		{"uppercase_workspace", "query", "POST", strings.Replace(path("query"), workspace, "AAAAAAAA-0000-4000-8000-000000000001", 1), "{}", nil, 400},
		{"nil_command", "execute", "POST", strings.Replace(path("execute"), commandID, "00000000-0000-0000-0000-000000000000", 1), "{}", nil, 400},
		{"head_catalog", "catalog", "HEAD", path("catalog"), "", nil, 405},
		{"head_pending", "pending", "HEAD", path("pending"), "", nil, 405},
		{"get_body", "catalog", "GET", path("catalog"), "{}", nil, 400},
		{"get_chunked", "pending", "GET", path("pending"), "", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &fixedAuth{}
			o := &fakeOperations{}
			r := httptest.NewRequest(tc.method, tc.url, strings.NewReader(tc.body))
			r.Header.Set("Authorization", "Bearer synthetic")
			if tc.method == "POST" {
				r.Header.Set("Content-Type", "application/json")
			}
			if tc.edit != nil {
				tc.edit(r)
			}
			w := httptest.NewRecorder()
			Handler(a, o).ServeHTTP(w, r)
			if w.Code != tc.status || o.calls != 0 {
				t.Fatalf("status%d calls%d", w.Code, o.calls)
			}
			if tc.status == 401 && a.calls != 0 {
				t.Fatal("invalid authentication framing reached resolver")
			}
			if tc.method == "HEAD" && w.Header().Get("Allow") != "GET" {
				t.Fatal("missing HEAD Allow")
			}
		})
	}
	for _, op := range []string{"query", "preview", "prepare", "execute", "cancel", "acknowledge"} {
		for _, excess := range []int{0, 1} {
			o := &fakeOperations{}
			body := strings.Repeat("x", int(bodyLimit(op))+excess)
			w := httpCall(Handler(&fixedAuth{}, o), "POST", path(op), body)
			want, calls := 200, 1
			if excess == 1 {
				want = 413
				calls = 0
			}
			if w.Code != want || o.calls != calls {
				t.Fatalf("body limit %s excess%d: %d", op, excess, w.Code)
			}
		}
	}
	a := &fixedAuth{denied: true}
	o := &fakeOperations{}
	w := httpCall(Handler(a, o), "POST", path("query"), "private malformed")
	if w.Code != 401 || o.calls != 0 || strings.Contains(w.Body.String(), "private") {
		t.Fatal("auth failure escaped or private parsing reached service")
	}
}

// TestLabelHTTPSafeFailures relays only the closed status/code pairs and never private error bodies.
func TestLabelHTTPSafeFailures(t *testing.T) {
	for _, tc := range []struct {
		err    error
		body   string
		status int
		code   string
	}{
		{errors.New("private SQL"), result("query"), 503, "temporarily_unavailable"},
		{&ledger.Failure{Status: 409, Code: "collection_snapshot_changed"}, result("query"), 409, "collection_snapshot_changed"},
		{&ledger.Failure{Status: 200, Code: "collection_denied"}, result("query"), 503, "temporarily_unavailable"},
		{&ledger.Failure{Status: 500, Code: "private diagnostic"}, result("query"), 503, "temporarily_unavailable"},
		{nil, `null`, 503, "temporarily_unavailable"},
		{nil, strings.Replace(result("query"), workspace, commandID, 1), 503, "temporarily_unavailable"},
		{nil, result("preview"), 503, "temporarily_unavailable"},
	} {
		o := &fakeOperations{err: tc.err, override: tc.body}
		w := httpCall(Handler(&fixedAuth{}, o), "POST", path("query"), "{}")
		var body struct {
			RequestID string `json:"request_id"`
			Status    int    `json:"status"`
			Code      string `json:"code"`
		}
		if json.Unmarshal(w.Body.Bytes(), &body) != nil || !validID(body.RequestID) || body.Status != tc.status || body.Code != tc.code || w.Code != tc.status || strings.Contains(w.Body.String(), "private") {
			t.Fatal("unsafe public failure", w.Code)
		}
	}
	w := httpCall(Handler(&fixedAuth{}, nil), "GET", path("catalog"), "")
	if w.Code != 503 {
		t.Fatal("missing service available")
	}
	for _, op := range []string{"prepare", "pending", "execute", "cancel", "acknowledge"} {
		o := &fakeOperations{override: strings.Replace(result(op), `"command":{}`, `"command":null`, 1)}
		method, body := "POST", "{}"
		if op == "pending" {
			method = "GET"
			body = ""
		}
		w := httpCall(Handler(&fixedAuth{}, o), method, path(op), body)
		want := 503
		if op == "pending" {
			want = 200
		}
		if w.Code != want {
			t.Fatal("HTTP command-nullability fence", op, w.Code)
		}
	}
}
