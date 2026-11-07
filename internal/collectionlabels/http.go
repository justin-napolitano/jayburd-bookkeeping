package collectionlabels

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

// Handler mounts only the frozen private query and reviewed label command routes.
func Handler(auth ledger.Authenticator, operations Operations) http.Handler {
	mux := http.NewServeMux()
	prefix := "/v1/collections/{workspace}"
	for pattern, operation := range map[string]string{
		"GET " + prefix + "/taxonomy":                              "catalog",
		"POST " + prefix + "/review/query":                         "query",
		"POST " + prefix + "/labels/preview":                       "preview",
		"POST " + prefix + "/label-commands":                       "prepare",
		"GET " + prefix + "/label-commands/pending":                "pending",
		"POST " + prefix + "/label-commands/{command}/execute":     "execute",
		"POST " + prefix + "/label-commands/{command}/cancel":      "cancel",
		"POST " + prefix + "/label-commands/{command}/acknowledge": "acknowledge",
	} {
		mux.HandleFunc(pattern, endpoint(auth, operations, operation))
		mux.HandleFunc(strings.Replace(pattern, "/v1/", "/v2/", 1), endpoint(auth, operations, "v2_"+operation))
	}
	mux.HandleFunc("GET /v1/collection-recovery", endpoint(auth, operations, "recovery"))
	mux.HandleFunc("POST /v1/collection-recovery/clear", endpoint(auth, operations, "clear_recovery"))
	return mux
}

// endpoint authenticates before parsing a private selector and never accepts caller identity.
func endpoint(auth ledger.Authenticator, operations Operations, operation string) http.HandlerFunc {
	version := 1
	if strings.HasPrefix(operation, "v2_") {
		version = 2
		operation = strings.TrimPrefix(operation, "v2_")
	}
	global := operation == "recovery" || operation == "clear_recovery"
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
			fail(400, "invalid_collection")
			return
		}
		request := Request{Actor: actor, Workspace: r.PathValue("workspace"), Command: r.PathValue("command"), Version: version}
		if (!global && !validID(request.Workspace)) || ((operation == "execute" || operation == "cancel" || operation == "acknowledge") && !validID(request.Command)) {
			fail(400, "invalid_collection")
			return
		}
		if operation != "catalog" && operation != "pending" && operation != "recovery" {
			if len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" {
				fail(415, "unsupported_media_type")
				return
			}
			raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, bodyLimit(operation)))
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
		if operations == nil {
			fail(503, "temporarily_unavailable")
			return
		}
		var body json.RawMessage
		switch operation {
		case "catalog":
			body, err = operations.Catalog(r.Context(), request)
		case "query":
			body, err = operations.Query(r.Context(), request)
		case "preview":
			body, err = operations.Preview(r.Context(), request)
		case "prepare":
			body, err = operations.Prepare(r.Context(), request)
		case "pending":
			body, err = operations.Pending(r.Context(), request)
		case "recovery", "clear_recovery":
			recovery, ok := operations.(RecoveryOperations)
			if !ok {
				fail(503, "temporarily_unavailable")
				return
			}
			if operation == "recovery" {
				body, err = recovery.Recovery(r.Context(), request)
			} else {
				body, err = recovery.ClearRecovery(r.Context(), request)
			}
		default:
			body, err = operations.Transition(r.Context(), request, operation)
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
		if !validRequestResult(body, operation, request) {
			fail(503, "temporarily_unavailable")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(body)
	}
}

// validID accepts canonical nonzero path identifiers; SQL still owns their authority.
func validID(value string) bool {
	return workspaceID.MatchString(value) && value != "00000000-0000-0000-0000-000000000000"
}

// bodyLimit bounds transport memory without reproducing SQL object/selection validation.
func bodyLimit(operation string) int64 {
	switch operation {
	case "query":
		return 8192
	case "preview":
		return 524288
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
