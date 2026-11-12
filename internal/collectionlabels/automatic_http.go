package collectionlabels

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// AutomationHandler exposes only authenticated policy settings, never trusted work or computed results.
func AutomationHandler(auth ledger.Authenticator, operations AutomationOperations) http.Handler {
	mux := http.NewServeMux()
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		mux.HandleFunc(method+" /v1/collections/{workspace}/automation", func(w http.ResponseWriter, r *http.Request) {
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
			request := Request{Actor: actor, Workspace: r.PathValue("workspace")}
			if !validID(request.Workspace) || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.Header.Values("Content-Encoding")) != 0 || len(r.Header.Values("Idempotency-Key")) != 0 {
				fail(400, "invalid_collection")
				return
			}
			if r.Method == http.MethodPost {
				if len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" {
					fail(415, "unsupported_media_type")
					return
				}
				raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
				if err != nil {
					var size *http.MaxBytesError
					if errors.As(err, &size) {
						fail(413, "request_too_large")
					} else {
						fail(400, "invalid_collection")
					}
					return
				}
				request.Body = string(raw) // SQL checks closed keys/duplicates; preserve exact request bytes.
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
			if r.Method == http.MethodPost {
				body, err = operations.SetAutomation(r.Context(), request)
			} else {
				body, err = operations.GetAutomation(r.Context(), request)
			}
			if err != nil {
				var f *ledger.Failure
				if errors.As(err, &f) {
					allowed := map[string]int{"invalid_collection": 400, "collection_denied": 404, "collection_conflict": 409, "collection_snapshot_changed": 409, "collection_not_ready": 409, "collection_limit": 422, "temporarily_unavailable": 503}
					if status, ok := allowed[f.Code]; ok && status == f.Status {
						fail(status, f.Code)
						return
					}
				}
				fail(503, "temporarily_unavailable")
				return
			}
			if !validAutomationStatus(body, request.Workspace) {
				fail(503, "temporarily_unavailable")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(body)
		})
	}
	return mux
}
