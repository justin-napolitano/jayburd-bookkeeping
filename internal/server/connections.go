package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// ConnectionOperations is explicitly injected; ordinary managed startup cannot enable provider calls.
type ConnectionOperations interface {
	Begin(context.Context, connections.Scope, string, connections.BeginRequest) (connections.BeginResult, error)
	Execute(context.Context, connections.Scope, string, connections.ExecuteRequest) (json.RawMessage, error)
	Reconcile(context.Context, connections.Scope, string) (json.RawMessage, error)
	Attempt(context.Context, connections.Scope, string) (json.RawMessage, error)
	List(context.Context, connections.Scope) (json.RawMessage, error)
	Bind(context.Context, connections.Scope, string, string, connections.BindingRequest) (json.RawMessage, error)
	Disconnect(context.Context, connections.Scope, string, string, string) (json.RawMessage, error)
}

var connectionUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var connectionField = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var connectionKey = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// mountConnections uses the same verified identity and browser origin boundary as the books.
func mountConnections(mux *http.ServeMux, auth ledger.Authenticator, ops ConnectionOperations) {
	base := "/v1/entities/{entity}/books/{book}"
	for pattern, operation := range map[string]string{
		"POST " + base + "/connection-attempts":                     "begin",
		"GET " + base + "/connection-attempts/{attempt}":            "attempt",
		"POST " + base + "/connection-attempts/{attempt}/execute":   "execute",
		"POST " + base + "/connection-attempts/{attempt}/reconcile": "reconcile",
		"POST " + base + "/connection-attempts/{attempt}/bindings":  "bind",
		"GET " + base + "/connections":                              "list",
		"POST " + base + "/connections/streams/{stream}/disconnect": "disconnect",
	} {
		mux.HandleFunc(pattern, connectionEndpoint(auth, ops, operation))
	}
}

// connectionEndpoint authenticates before parsing memory-only tokens, and never logs request bodies.
func connectionEndpoint(auth ledger.Authenticator, ops ConnectionOperations, operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fail := func(status int, code string) { writeProblem(w, status, code) }
		h := r.Header.Values("Authorization")
		if auth == nil || len(h) != 1 || len(h[0]) > 8192 || !strings.HasPrefix(h[0], "Bearer ") || strings.ContainsAny(strings.TrimPrefix(h[0], "Bearer "), " \t\r\n,") || len(h[0]) == 7 {
			fail(401, "unauthenticated")
			return
		}
		actor, err := auth.Authenticate(r.Context(), strings.TrimPrefix(h[0], "Bearer "))
		if err != nil || !connectionUUID.MatchString(actor) {
			fail(401, "unauthenticated")
			return
		}
		scope := connections.Scope{Actor: actor, Entity: r.PathValue("entity"), Book: r.PathValue("book")}
		if !connectionUUID.MatchString(scope.Entity) || !connectionUUID.MatchString(scope.Book) || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" {
			fail(400, "invalid_request")
			return
		}
		attempt, stream := r.PathValue("attempt"), r.PathValue("stream")
		if (attempt != "" && !connectionUUID.MatchString(attempt)) || (stream != "" && !connectionUUID.MatchString(stream)) {
			fail(400, "invalid_request")
			return
		}
		if r.Method == http.MethodHead {
			fail(405, "method_not_allowed")
			return
		}
		key := ""
		if operation == "begin" || operation == "bind" || operation == "disconnect" {
			keys := r.Header.Values("Idempotency-Key")
			if len(keys) != 1 || !connectionKey.MatchString(keys[0]) {
				fail(400, "invalid_request")
				return
			}
			key = keys[0]
		}
		var raw []byte
		if r.Method == http.MethodPost {
			media, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if e != nil || media != "application/json" || len(params) != 0 || len(r.Header.Values("Content-Type")) != 1 || len(r.Header.Values("Content-Encoding")) != 0 {
				fail(400, "invalid_request")
				return
			}
			raw, err = io.ReadAll(io.LimitReader(r.Body, (32<<10)+1))
			if len(raw) > 32<<10 {
				fail(413, "request_too_large")
				return
			}
			if err != nil || !validConnectionJSON(raw) {
				fail(400, "invalid_request")
				return
			}
		} else if r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
			fail(400, "invalid_request")
			return
		}
		var result any
		switch operation {
		case "begin":
			var in connections.BeginRequest
			if !decodeConnectionJSON(raw, &in) {
				fail(400, "invalid_request")
				return
			}
			result, err = ops.Begin(r.Context(), scope, key, in)
		case "execute":
			var in connections.ExecuteRequest
			if !decodeConnectionJSON(raw, &in) {
				fail(400, "invalid_request")
				return
			}
			result, err = ops.Execute(r.Context(), scope, attempt, in)
		case "bind":
			var in connections.BindingRequest
			if !decodeConnectionJSON(raw, &in) {
				fail(400, "invalid_request")
				return
			}
			result, err = ops.Bind(r.Context(), scope, attempt, key, in)
		case "reconcile":
			var in struct{}
			if !decodeConnectionJSON(raw, &in) {
				fail(400, "invalid_request")
				return
			}
			result, err = ops.Reconcile(r.Context(), scope, attempt)
		case "disconnect":
			var in struct {
				ExpectedBindingRevision string `json:"expected_binding_revision"`
			}
			if !decodeConnectionJSON(raw, &in) || in.ExpectedBindingRevision == "" {
				fail(400, "invalid_request")
				return
			}
			result, err = ops.Disconnect(r.Context(), scope, stream, key, in.ExpectedBindingRevision)
		case "attempt":
			result, err = ops.Attempt(r.Context(), scope, attempt)
		case "list":
			result, err = ops.List(r.Context(), scope)
		}
		if err != nil {
			var f *connections.Failure
			if !errors.As(err, &f) {
				fail(500, "internal_error")
				return
			}
			status := map[string]int{"invalid_request": 400, "permission_denied": 403, "not_found": 404, "connection_conflict": 409, "stale_connection": 409, "credential_recovery_required": 409, "connection_limit": 422, "unsupported_connection_mode": 422, "commit_unknown": 503, "temporarily_unavailable": 503}[f.Code]
			if status == 0 {
				fail(500, "internal_error")
				return
			}
			fail(status, f.Code)
			return
		}
		encoded, err := json.Marshal(result)
		if err != nil || len(encoded) > 256<<10 || r.Context().Err() != nil {
			fail(503, "temporarily_unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encoded)
	}
}

// decodeConnectionJSON disallows unknown request fields after the duplicate-key scan.
func decodeConnectionJSON(raw []byte, target any) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil
}

// validConnectionJSON rejects duplicate keys at every depth and requires one bounded object.
func validConnectionJSON(raw []byte) bool {
	if !utf8.Valid(raw) || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !connectionJSONValue(d, 0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}

// connectionJSONValue bounds nested request parsing and prevents last-key-wins ambiguity.
func connectionJSONValue(d *json.Decoder, depth int) bool {
	if depth > 16 {
		return false
	}
	token, err := d.Token()
	if err != nil {
		return false
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return true
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			key, ok := k.(string)
			if e != nil || !ok || !connectionField.MatchString(key) || seen[key] {
				return false
			}
			seen[key] = true
			if !connectionJSONValue(d, depth+1) {
				return false
			}
		}
		end, e := d.Token()
		return e == nil && end == json.Delim('}')
	case '[':
		for d.More() {
			if !connectionJSONValue(d, depth+1) {
				return false
			}
		}
		end, e := d.Token()
		return e == nil && end == json.Delim(']')
	default:
		return false
	}
}
