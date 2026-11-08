package collectiononboarding

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Handler exposes only explicit reviewed onboarding under the caller's existing browser boundary.
func Handler(auth ledger.Authenticator, ops Operations) http.Handler {
	mux := http.NewServeMux()
	prefix := "/v1/collection-onboarding"
	for pattern, operation := range map[string]string{
		"GET " + prefix:                                      "catalog",
		"POST " + prefix + "/previews":                       "preview",
		"POST " + prefix + "/commands":                       "prepare",
		"GET " + prefix + "/commands/pending":                "pending",
		"POST " + prefix + "/commands/{command}/execute":     "execute",
		"POST " + prefix + "/commands/{command}/cancel":      "cancel",
		"POST " + prefix + "/commands/{command}/acknowledge": "acknowledge",
		"POST " + prefix + "/recovery/clear":                 "clear",
	} {
		mux.HandleFunc(pattern, endpoint(auth, ops, operation))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

// endpoint authenticates before reading path/body/framing and preserves original SQL-owned JSON.
func endpoint(auth ledger.Authenticator, ops Operations, operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fail := func(status int, code string) { problem(w, status, code) }
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
			w.Header().Set("Allow", http.MethodGet)
			fail(405, "method_not_allowed")
			return
		}
		if r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.Header.Values("Idempotency-Key")) != 0 || len(r.Header.Values("Content-Encoding")) != 0 {
			fail(400, "invalid_onboarding")
			return
		}
		request := Request{Actor: actor, Command: r.PathValue("command")}
		if (operation == "execute" || operation == "cancel" || operation == "acknowledge") && !validID(request.Command) {
			fail(400, "invalid_onboarding")
			return
		}
		if operation == "catalog" || operation == "pending" {
			if r.ContentLength > 0 || len(r.TransferEncoding) > 0 || len(r.Header.Values("Content-Type")) > 1 {
				fail(400, "invalid_onboarding")
				return
			}
			if r.Body != nil {
				raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 1))
				if e != nil || len(raw) != 0 {
					fail(400, "invalid_onboarding")
					return
				}
			}
		} else {
			if len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" {
				fail(415, "unsupported_media_type")
				return
			}
			if r.ContentLength > bodyLimit(operation) {
				fail(413, "request_too_large")
				return
			}
			raw, e := io.ReadAll(http.MaxBytesReader(w, r.Body, bodyLimit(operation)))
			if e != nil {
				var size *http.MaxBytesError
				if errors.As(e, &size) {
					fail(413, "request_too_large")
				} else {
					fail(400, "invalid_onboarding")
				}
				return
			}
			if !utf8.Valid(raw) {
				fail(400, "invalid_onboarding")
				return
			}
			request.Body = string(raw)
		}
		if ops == nil {
			fail(503, "temporarily_unavailable")
			return
		}
		var body json.RawMessage
		switch operation {
		case "catalog":
			body, err = ops.Catalog(r.Context(), request)
		case "preview":
			body, err = ops.Preview(r.Context(), request)
		case "prepare":
			body, err = ops.Prepare(r.Context(), request)
		case "pending":
			body, err = ops.Pending(r.Context(), request)
		case "clear":
			body, err = ops.Clear(r.Context(), request)
		default:
			body, err = ops.Transition(r.Context(), request, operation)
		}
		if err != nil {
			var f *ledger.Failure
			if errors.As(err, &f) && f != nil {
				statuses := map[string]int{"invalid_onboarding": 400, "onboarding_denied": 404, "onboarding_conflict": 409, "onboarding_snapshot_changed": 409, "onboarding_not_ready": 409, "onboarding_limit": 422, "temporarily_unavailable": 503}
				if status, ok := statuses[f.Code]; ok && status == f.Status {
					fail(status, f.Code)
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
		w.Header().Set("Content-Type", "application/json")
		status := http.StatusOK
		if operation == "prepare" {
			status = http.StatusCreated
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}
}

// bodyLimit is transport framing only; SQL validates closed keys and all business semantics.
func bodyLimit(operation string) int64 {
	switch operation {
	case "preview":
		return 8192
	case "prepare":
		return 4096
	default:
		return 1024
	}
}

// problem emits only fixed diagnostics and an independently generated correlation identifier.
func problem(w http.ResponseWriter, status int, code string) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		http.Error(w, "Request unavailable", 503)
		return
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	hexID := hex.EncodeToString(id[:])
	requestID := fmt.Sprintf("%s-%s-%s-%s-%s", hexID[:8], hexID[8:12], hexID[12:16], hexID[16:20], hexID[20:])
	w.Header().Set("Content-Type", "application/problem+json")
	if status == 401 {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	if status == 503 {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://bookkeeping.example.invalid/problems/" + code, "title": http.StatusText(status), "status": status, "code": code, "request_id": requestID})
}
