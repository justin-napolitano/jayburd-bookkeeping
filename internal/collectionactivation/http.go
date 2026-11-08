package collectionactivation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Handler accepts only a plan reference after existing managed authentication.
// Server composition retains the existing same-origin browser boundary.
func Handler(auth ledger.Authenticator, s *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/collection-activation/adopt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fail := func(status int, code string) {
			w.Header().Set("Content-Type", "application/problem+json")
			if status == http.StatusUnauthorized {
				w.Header().Set("WWW-Authenticate", "Bearer")
			}
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": code, "status": status, "code": code, "request_id": requestID()})
		}
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
		if s == nil || s.plan == nil || actor != s.plan.value.Actor || !s.plan.deadline.After(s.now()) {
			fail(404, "activation_denied")
			return
		}
		if r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.Header.Values("Idempotency-Key")) != 0 || len(r.Header.Values("Content-Encoding")) != 0 {
			fail(400, "invalid_activation")
			return
		}
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
				fail(400, "invalid_activation")
			}
			return
		}
		var request struct {
			ID string `json:"plan_id"`
		}
		if decode(raw, &request) != nil || !validID(request.ID) {
			fail(400, "invalid_activation")
			return
		}
		body, err := s.adopt(r.Context(), actor, request.ID)
		if err != nil {
			var f *ledger.Failure
			if errors.As(err, &f) {
				fail(f.Status, f.Code)
			} else {
				fail(503, "temporarily_unavailable")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
	return mux
}

// requestID identifies a response independently of all private plan/actor IDs.
func requestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
