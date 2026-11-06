package collectionread

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

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

var workspaceID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Handler mounts only explicit reads; provisioning, mutation and provider operations are absent.
func Handler(auth ledger.Authenticator, reader Reader) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/collection", endpoint(auth, reader, "workspace"))
	mux.HandleFunc("POST /v1/collections/{workspace}/sources/query", endpoint(auth, reader, "sources"))
	mux.HandleFunc("POST /v1/collections/{workspace}/observations/query", endpoint(auth, reader, "observations"))
	return mux
}

// endpoint authenticates before parsing a private selector and never accepts caller identity.
func endpoint(auth ledger.Authenticator, reader Reader, operation string) http.HandlerFunc {
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
			fail(405, "method_not_allowed")
			return
		}
		if r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.Header.Values("Idempotency-Key")) != 0 || len(r.Header.Values("Content-Encoding")) != 0 {
			fail(400, "invalid_collection")
			return
		}
		request := Request{Actor: actor, Workspace: r.PathValue("workspace")}
		if operation != "workspace" {
			if !workspaceID.MatchString(request.Workspace) || request.Workspace == "00000000-0000-0000-0000-000000000000" {
				fail(400, "invalid_collection")
				return
			}
			if len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" {
				fail(415, "unsupported_media_type")
				return
			}
			raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8192))
			if err != nil {
				var size *http.MaxBytesError
				if errors.As(err, &size) {
					fail(413, "request_too_large")
				} else {
					fail(400, "invalid_collection")
				}
				return
			}
			// SQL owns duplicate/unknown keys and exact selector semantics; preserve these bytes.
			request.Body = string(raw)
		} else {
			if r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
				fail(400, "invalid_collection")
				return
			}
			if r.Body != nil {
				raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1))
				if err != nil || len(raw) != 0 {
					fail(400, "invalid_collection")
					return
				}
			}
		}
		if reader == nil {
			fail(503, "temporarily_unavailable")
			return
		}
		var body json.RawMessage
		switch operation {
		case "workspace":
			body, err = reader.Workspace(r.Context(), request)
		case "sources":
			body, err = reader.Sources(r.Context(), request)
		case "observations":
			body, err = reader.Observations(r.Context(), request)
		}
		if err != nil {
			var f *ledger.Failure
			if errors.As(err, &f) {
				statuses := map[string]int{"invalid_collection": 400, "collection_denied": 404, "collection_conflict": 409, "collection_snapshot_changed": 409, "collection_not_ready": 409, "collection_limit": 422, "temporarily_unavailable": 503}
				if status, ok := statuses[f.Code]; ok && status == f.Status {
					fail(status, f.Code)
					return
				}
			}
			fail(503, "temporarily_unavailable")
			return
		}
		if !validResult(body, operation) {
			fail(503, "temporarily_unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(body)
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
