package ledger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

// Authenticator must validate issuer, audience, expiry and subject before mapping
// a principal. The identity package supplies the managed provider adapter.
type Authenticator interface {
	Authenticate(context.Context, string) (string, error)
}

// Operations allows transport tests without treating mocks as accounting proof.
type Operations interface {
	Setup(context.Context, Command) (Result, error)
	Post(context.Context, Command) (Result, error)
	Reverse(context.Context, Command) (Result, error)
	Journal(context.Context, Command) (Result, error)
	TrialBalance(context.Context, Command) (Result, error)
}

// Handler exposes only the reviewed operations; callers explicitly supply verified
// identity and database adapters. Server composition owns listener policy.
func Handler(auth Authenticator, ops Operations) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/commands", endpoint(auth, ops, "command_register"))
	mux.HandleFunc("GET /v1/commands/pending", endpoint(auth, ops, "command_pending"))
	mux.HandleFunc("POST /v1/commands/{command}/execute", endpoint(auth, ops, "command_execute"))
	mux.HandleFunc("POST /v1/commands/{command}/cancel", endpoint(auth, ops, "command_cancel"))
	mux.HandleFunc("POST /v1/commands/{command}/acknowledge", endpoint(auth, ops, "command_acknowledge"))
	mux.HandleFunc("GET /v1/workspace", endpoint(auth, ops, "workspace"))
	mux.HandleFunc("POST /v1/setup", endpoint(auth, ops, "setup"))
	base := "/v1/entities/{entity}/books/{book}"
	mux.HandleFunc("POST "+base+"/journals", endpoint(auth, ops, "post"))
	mux.HandleFunc("POST "+base+"/journals/{journal}/reversals", endpoint(auth, ops, "reverse"))
	mux.HandleFunc("GET "+base+"/journals/{journal}", endpoint(auth, ops, "journal"))
	mux.HandleFunc("GET "+base+"/trial-balance", endpoint(auth, ops, "balance"))
	for _, kind := range []string{"trial-balance", "general-ledger", "balance-sheet", "profit-and-loss"} {
		mux.HandleFunc("GET "+base+"/reports/"+kind, endpoint(auth, ops, "report_"+kind))
	}
	return mux
}

// requestID generates a server-controlled UUID, never an actor-supplied identifier.
func requestID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	s := hex.EncodeToString(b[:])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}

// problem serializes only safe stable fields, never the underlying failure value.
func problem(w http.ResponseWriter, id string, f *Failure) {
	w.Header().Set("Content-Type", "application/problem+json")
	if f.Status == 401 {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	if f.Status == 503 {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(f.Status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "https://bookkeeping.example.invalid/problems/" + f.Code, "title": http.StatusText(f.Status), "status": f.Status, "code": f.Code, "request_id": id})
}

// endpoint authenticates before data access and preserves raw request JSON for SQL.
func endpoint(auth Authenticator, ops Operations, op string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		id := requestID()
		fail := func(status int, code string) { problem(w, id, &Failure{status, code}) }
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
		c := Command{Actor: actor, Entity: r.PathValue("entity"), Book: r.PathValue("book"), Journal: r.PathValue("journal"), CommandID: r.PathValue("command"), Request: id}
		isRecovery := strings.HasPrefix(op, "command_")
		if isRecovery && r.URL.RawQuery != "" {
			fail(400, "invalid_request")
			return
		}
		if op == "post" || op == "reverse" || op == "setup" || (isRecovery && op != "command_pending") {
			if !isRecovery || op == "command_register" {
				keys := r.Header.Values("Idempotency-Key")
				if len(keys) != 1 || keys[0] == "" {
					fail(400, "invalid_request")
					return
				}
				c.Key = keys[0]
			}
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
			if err != nil {
				var tooBig *http.MaxBytesError
				if errors.As(err, &tooBig) {
					fail(413, "request_too_large")
				} else {
					fail(400, "invalid_request")
				}
				return
			}
			if !utf8.Valid(body) {
				fail(400, "invalid_request")
				return
			}
			c.Body = string(body)
		} else if op == "balance" {
			q, exists := r.URL.Query()["through_date"]
			if !exists || len(q) != 1 || len(r.URL.Query()) != 1 {
				fail(400, "invalid_request")
				return
			}
			c.Body = q[0]
		}
		var result Result
		if strings.HasPrefix(op, "report_") {
			kind := strings.TrimPrefix(op, "report_")
			c.Body, err = reportRequest(r, kind)
			if err != nil {
				fail(400, "invalid_request")
				return
			}
			reports, ok := ops.(interface {
				FinancialReport(context.Context, Command) (Result, error)
			})
			if !ok {
				fail(503, "temporarily_unavailable")
				return
			}
			result, err = reports.FinancialReport(r.Context(), c)
			if err == nil {
				result.Body, err = reportResponse(result.Body, kind)
			}
		}
		switch op {
		case "command_register", "command_pending", "command_execute", "command_cancel", "command_acknowledge":
			if op == "command_pending" && (r.ContentLength > 0 || len(r.TransferEncoding) > 0) {
				fail(400, "invalid_request")
				return
			}
			recovery, ok := ops.(interface {
				Recovery(context.Context, Command, string) (Result, error)
			})
			if !ok {
				fail(503, "temporarily_unavailable")
				return
			}
			result, err = recovery.Recovery(r.Context(), c, strings.TrimPrefix(op, "command_"))
		case "workspace":
			if r.URL.RawQuery != "" || r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
				fail(400, "invalid_request")
				return
			}
			workspace, ok := ops.(interface {
				Workspace(context.Context, Command) (Result, error)
			})
			if !ok {
				fail(503, "temporarily_unavailable")
				return
			}
			result, err = workspace.Workspace(r.Context(), c)
		case "setup":
			result, err = ops.Setup(r.Context(), c)
		case "post":
			result, err = ops.Post(r.Context(), c)
		case "reverse":
			result, err = ops.Reverse(r.Context(), c)
		case "journal":
			result, err = ops.Journal(r.Context(), c)
		case "balance":
			result, err = ops.TrialBalance(r.Context(), c)
		}
		if err != nil {
			var f *Failure
			if !errors.As(err, &f) {
				f = &Failure{500, "internal_error"}
			}
			problem(w, id, f)
			return
		}
		if op == "post" || op == "reverse" || op == "setup" {
			w.Header().Set("Idempotent-Replay", map[bool]string{true: "true", false: "false"}[result.Replayed])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(result.Status)
		_, _ = w.Write(result.Body)
	}
}
