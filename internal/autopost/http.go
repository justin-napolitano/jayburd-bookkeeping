package autopost

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

const policyBodyLimit int64 = 128 << 10

var pathID = regexp.MustCompile(`^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$`)

// Handler exposes policy configuration and inert status only; automatic execution is never a public route.
func Handler(auth ledger.Authenticator, operations Operations, runtimeEnabled bool) http.Handler {
	mux := http.NewServeMux()
	base := "/v1/entities/{entity}/books/{book}/automatic-accounting"
	mux.HandleFunc("GET "+base, endpoint(auth, operations, "get", runtimeEnabled))
	mux.HandleFunc("POST "+base, endpoint(auth, operations, "set", runtimeEnabled))
	mux.HandleFunc("GET "+base+"/status", endpoint(auth, operations, "status", runtimeEnabled))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

// endpoint authenticates before inspecting private selectors/body; SQL owns mapping and current-authority semantics.
func endpoint(auth ledger.Authenticator, operations Operations, operation string, runtimeEnabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fail := func(status int, code string) { writeProblem(w, status, code) }
		headers := r.Header.Values("Authorization")
		if auth == nil || len(headers) != 1 || !strings.HasPrefix(headers[0], "Bearer ") || strings.TrimSpace(strings.TrimPrefix(headers[0], "Bearer ")) == "" {
			fail(401, "unauthenticated")
			return
		}
		actor, err := auth.Authenticate(r.Context(), strings.TrimPrefix(headers[0], "Bearer "))
		if err != nil || actor == "" {
			fail(401, "unauthenticated")
			return
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Allow", "GET")
			fail(405, "method_not_allowed")
			return
		}
		request := Request{Actor: actor, Entity: r.PathValue("entity"), Book: r.PathValue("book")}
		validID := func(value string) bool {
			return pathID.MatchString(value) && value != "00000000-0000-0000-0000-000000000000"
		}
		if !validID(request.Entity) || !validID(request.Book) || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.Header.Values("Content-Encoding")) != 0 || len(r.Header.Values("Idempotency-Key")) != 0 {
			fail(400, "invalid_automatic_accounting")
			return
		}
		if operation == "set" {
			if len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" {
				fail(415, "unsupported_media_type")
				return
			}
			if r.Body == nil {
				fail(400, "invalid_automatic_accounting")
				return
			}
			if r.ContentLength > policyBodyLimit {
				fail(413, "request_too_large")
				return
			}
			raw, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, policyBodyLimit))
			if readErr != nil {
				var size *http.MaxBytesError
				if errors.As(readErr, &size) {
					fail(413, "request_too_large")
				} else {
					fail(400, "invalid_automatic_accounting")
				}
				return
			}
			if !utf8.Valid(raw) {
				fail(400, "invalid_automatic_accounting")
				return
			}
			request.Body = string(raw)
		} else {
			if r.ContentLength > 0 || len(r.TransferEncoding) != 0 || len(r.Header.Values("Content-Type")) > 1 {
				fail(400, "invalid_automatic_accounting")
				return
			}
			if r.Body != nil {
				raw, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, 1))
				if readErr != nil || len(raw) != 0 {
					fail(400, "invalid_automatic_accounting")
					return
				}
			}
		}
		if operations == nil {
			fail(503, "temporarily_unavailable")
			return
		}
		var body json.RawMessage
		switch operation {
		case "get":
			body, err = operations.GetPolicy(r.Context(), request)
		case "set":
			body, err = operations.SetPolicy(r.Context(), request)
		case "status":
			body, err = operations.Status(r.Context(), request)
		}
		if err != nil {
			var failure *ledger.Failure
			if errors.As(err, &failure) && failure != nil {
				statuses := map[string]int{"invalid_automatic_accounting": 400, "automatic_accounting_denied": 404, "automatic_accounting_conflict": 409, "automatic_accounting_snapshot_changed": 409, "automatic_accounting_limit": 422, "temporarily_unavailable": 503}
				if status, ok := statuses[failure.Code]; ok && status == failure.Status {
					fail(status, failure.Code)
					return
				}
			}
			fail(503, "temporarily_unavailable")
			return
		}
		if !validResult(body, operation, request) {
			fail(503, "temporarily_unavailable")
			return
		}
		if operation == "status" {
			// Runtime availability comes from trusted process composition, never settings or SQL input.
			var status map[string]json.RawMessage
			if json.Unmarshal(body, &status) != nil {
				fail(503, "temporarily_unavailable")
				return
			}
			status["runtime_enabled"], _ = json.Marshal(runtimeEnabled)
			body, err = json.Marshal(status)
			if err != nil {
				fail(503, "temporarily_unavailable")
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

// writeProblem emits fixed codes with an independent correlation ID, never SQL or provider diagnostics.
func writeProblem(w http.ResponseWriter, status int, code string) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		http.Error(w, "Request unavailable", 503)
		return
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	id := hex.EncodeToString(value[:])
	w.Header().Set("Content-Type", "application/problem+json")
	if status == 401 {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	if status == 503 {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://bookkeeping.example.invalid/problems/" + code, "title": http.StatusText(status), "status": status, "code": code, "request_id": fmt.Sprintf("%s-%s-%s-%s-%s", id[:8], id[8:12], id[12:16], id[16:20], id[20:])})
}
