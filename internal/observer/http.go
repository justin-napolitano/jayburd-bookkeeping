package observer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	PublisherPath = "/internal/observer/publish"
	ConsumerPath  = "/internal/observer/consume"
)

var serviceEmail = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]@[a-z][a-z0-9-]{4,28}[a-z0-9]\.iam\.gserviceaccount\.com$`)
var serviceSubject = regexp.MustCompile(`^[1-9][0-9]{5,29}$`)
var runHost = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.([a-z0-9-]+\.)?run\.app$`)

// VerifiedToken contains only claims returned after cryptographic verification.
type VerifiedToken struct {
	Issuer, Audience, Subject, Email string
	EmailVerified                    bool
	IssuedAt, Expires                int64
}

// TokenVerifier must verify Google's signature and honor context before returning claims.
type TokenVerifier interface {
	Verify(context.Context, string, string) (VerifiedToken, error)
}

// CallerPolicy pins one workload, including its immutable service-account subject.
type CallerPolicy struct{ Audience, Email, Subject string }

// Ticker is the bounded publisher entry point; SQL owns its configured mandate.
type Ticker interface {
	Tick(context.Context) (TickResult, error)
}

// WakeupHandler acknowledges only committed receipt observation or durable terminal state.
type WakeupHandler interface {
	Handle(context.Context, []byte) error
}

// workerHTTP isolates publisher and consumer routes and caps per-process concurrency.
type workerHTTP struct {
	verifier  TokenVerifier
	policy    CallerPolicy
	publisher Ticker
	consumer  WakeupHandler
	slots     chan struct{}
}

// NewPublisherHTTP mounts only the fixed publisher route; it loads no credentials.
func NewPublisherHTTP(p Ticker, v TokenVerifier, policy CallerPolicy) (http.Handler, error) {
	if p == nil || !validPolicy(v, policy) {
		return nil, reject()
	}
	return &workerHTTP{verifier: v, policy: policy, publisher: p, slots: make(chan struct{}, 1)}, nil
}

// NewConsumerHTTP mounts only the fixed consumer route; callers cannot select scope.
func NewConsumerHTTP(c WakeupHandler, v TokenVerifier, policy CallerPolicy) (http.Handler, error) {
	if c == nil || !validPolicy(v, policy) {
		return nil, reject()
	}
	return &workerHTTP{verifier: v, policy: policy, consumer: c, slots: make(chan struct{}, 1)}, nil
}

// runURL limits this first transport to canonical managed Cloud Run destinations.
func runURL(raw string) (*url.URL, bool) {
	u, e := url.Parse(raw)
	if e != nil || len(raw) > 512 || u.Scheme != "https" || !runHost.MatchString(u.Host) || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(raw, "#") || u.RawPath != "" {
		return nil, false
	}
	return u, true
}

// validPolicy rejects missing or ambiguous configured workload identities.
func validPolicy(v TokenVerifier, p CallerPolicy) bool {
	u, ok := runURL(p.Audience)
	return v != nil && ok && (u.Path == "" || u.Path == "/") && serviceEmail.MatchString(p.Email) && serviceSubject.MatchString(p.Subject)
}

// authorize rejects spoofed delivery metadata before any request body or database work.
func (h *workerHTTP) authorize(ctx context.Context, r *http.Request) bool {
	values := r.Header.Values("Authorization")
	if len(values) != 1 || len(values[0]) > 8192 || !strings.HasPrefix(values[0], "Bearer ") || len(r.Header.Values("X-Serverless-Authorization")) != 0 {
		return false
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" || strings.ContainsAny(token, " \t\r\n,") {
		return false
	}
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	claims, e := h.verifier.Verify(verifyCtx, token, h.policy.Audience)
	now := time.Now().Unix()
	return e == nil && verifyCtx.Err() == nil && claims.Issuer == "https://accounts.google.com" && claims.Audience == h.policy.Audience && claims.Email == h.policy.Email && claims.Subject == h.policy.Subject && claims.EmailVerified && claims.IssuedAt > 0 && claims.IssuedAt <= now && claims.Expires > now && claims.Expires > claims.IssuedAt && claims.Expires-claims.IssuedAt <= 3600
}

// ServeHTTP never acknowledges uncertain work or returns infrastructure diagnostics.
func (h *workerHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		workerProblem(w, 503)
		return
	}
	if !h.authorize(ctx, r) {
		workerProblem(w, 401)
		return
	}
	path := ConsumerPath
	if h.publisher != nil {
		path = PublisherPath
	}
	if r.URL.Path != path || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		workerProblem(w, 404)
		return
	}
	if r.Method != http.MethodPost {
		workerProblem(w, 405)
		return
	}
	media, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil || media != "application/json" || len(params) != 0 || len(r.Header.Values("Content-Type")) != 1 || len(r.Header.Values("Content-Encoding")) != 0 {
		workerProblem(w, 400)
		return
	}
	// Real network reads are bounded by the composing server and this read deadline.
	// An arbitrary injected io.Reader cannot be made cancellable by a context alone.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(5 * time.Second))
	raw, e := io.ReadAll(io.LimitReader(r.Body, 1025))
	if e != nil || len(raw) > 1024 || ctx.Err() != nil {
		workerProblem(w, 400)
		return
	}
	if h.publisher != nil {
		if !bytes.Equal(bytes.TrimSpace(raw), []byte("{}")) {
			workerProblem(w, 400)
			return
		}
		result, e := h.publisher.Tick(ctx)
		if e != nil || ctx.Err() != nil {
			workerProblem(w, 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"discovered": result.Discovered, "candidates": result.Candidates, "enqueue_calls": result.EnqueueCalls, "recorded": result.Recorded, "skipped": result.Skipped})
		return
	}
	if _, e = ParseWakeup(raw); e != nil {
		workerProblem(w, 400)
		return
	}
	if e = h.consumer.Handle(ctx, raw); e != nil || ctx.Err() != nil {
		workerProblem(w, 503)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// workerProblem exposes a fixed status without claims, task identifiers or raw errors.
func workerProblem(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/problem+json")
	if status == 503 {
		w.Header().Set("Retry-After", "1")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": http.StatusText(status), "status": status})
}
