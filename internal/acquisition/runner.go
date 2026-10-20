package acquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
)

// Runner composes real SQL ownership, an injected provider/resolver and durable evidence.
type Runner struct {
	repo        repository
	provider    Provider
	credentials CredentialResolver
	evidence    EvidenceStore
	slots       chan struct{}
}

// New validates dependencies without reading credentials, touching storage or beginning SQL work.
func New(db Beginner, p Provider, c CredentialResolver, e EvidenceStore) (*Runner, error) {
	if db == nil || p == nil || c == nil || e == nil {
		return nil, failure("configuration_error", false)
	}
	return &Runner{&SQLRepository{DB: db}, p, c, e, make(chan struct{}, 1)}, nil
}

// Run reuses a stable claim identity and never confuses transport or staged progress with publication.
func (r *Runner) Run(parent context.Context, s Scope, claimID string) (out Outcome, err error) {
	defer func() {
		if err != nil {
			err = publicError(err)
		}
	}()
	if !validScope(s) || !validID(claimID) {
		return Outcome{}, failure("invalid_request", false)
	}
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	default:
		return Outcome{}, failure("busy", true)
	}
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	for restart := 0; restart < 2; restart++ {
		key := claimID
		if restart == 1 {
			key = stableID("acquisition-mutation-restart-v1", s.EntityID, s.BookID, s.StreamID, claimID)
		}
		state, e := r.repo.Resume(ctx, s, key)
		if e != nil {
			return Outcome{}, e
		}
		if state.State == "completed" {
			return Outcome{Status: "completed", ChangedCount: state.Completion.ChangedCount}, nil
		}
		var a attempt
		if state.State == "absent" {
			a, e = r.repo.Claim(ctx, s, key)
			if e != nil {
				return Outcome{}, e
			}
		} else {
			a = attempt{*state.SetID, *state.Fence}
			if state.State != "staging" {
				if restart == 0 && state.State == "abandoned" && state.Reason != nil && *state.Reason == "pagination_mutation" {
					if e = r.cleanup(ctx, s, a); e != nil {
						return Outcome{}, e
					}
					continue
				}
				return Outcome{Status: state.State}, failure("attempt_terminal", false)
			}
		}
		out, mutation, e := r.acquire(ctx, s, a)
		if !mutation {
			return out, e
		}
		if restart == 1 {
			return Outcome{Status: "abandoned", RetryAfter: 5 * time.Second}, failure("pagination_mutation", true)
		}
	}
	return Outcome{}, failure("temporarily_unavailable", true)
}

// sameContext prevents using credentials or response bytes after scoped metadata changes.
func sameContext(a, b workContext) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// refresh reauthorizes and checks the exact expected page and credential pins.
func (r *Runner) refresh(ctx context.Context, s Scope, a attempt, expected workContext) error {
	current, e := r.repo.Context(ctx, s, a)
	if e != nil {
		return e
	}
	if !sameContext(current, expected) {
		return failure("stale_attempt", false)
	}
	return nil
}

// prefix verifies retained earlier pages and enforces the stricter complete-set raw-byte budget on resume.
func (r *Runner) prefix(ctx context.Context, s Scope, a attempt, c workContext) (int, error) {
	cursor := c.StartingCursor
	total := 0
	for i := 0; i < c.PageCount; i++ {
		raw, found, e := r.evidence.Get(ctx, EvidenceKey{s, a.SetID, i, cursor})
		if e != nil {
			return 0, e
		}
		if !found {
			return 0, failure("evidence_missing", false)
		}
		total += len(raw)
		if total > MaxRawBytes {
			return 0, failure("limit_exceeded", false)
		}
		page, e := plaid.DecodePage(raw, c.expected())
		if e != nil {
			return 0, failure("invalid_evidence", false)
		}
		cursor = nextCursor(page.NextCursor)
	}
	if !sameCursor(cursor, c.RequestCursor) {
		return 0, failure("invalid_evidence", false)
	}
	return total, nil
}

// sameCursor compares optional opaque values without coercing initial absence.
func sameCursor(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// nextCursor preserves the provider's special empty initialization as SQL's absent cursor.
func nextCursor(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// acquire performs one whole attempt, publishing only after every retained page is staged.
func (r *Runner) acquire(ctx context.Context, s Scope, a attempt) (Outcome, bool, error) {
	for steps := 0; steps <= 10; steps++ {
		c, e := r.repo.Context(ctx, s, a)
		if e != nil {
			return Outcome{}, false, e
		}
		total, e := r.prefix(ctx, s, a, c)
		if e != nil {
			return Outcome{}, false, e
		}
		if c.FinalPage {
			receipt, e := r.repo.Complete(ctx, s, a)
			if e != nil {
				return Outcome{}, false, e
			}
			return Outcome{Status: "completed", ChangedCount: receipt.ChangedCount}, false, nil
		}
		if c.NextPageIndex >= 10 {
			return r.abort(ctx, s, a, "decoder_rejected", failure("limit_exceeded", false))
		}
		key := EvidenceKey{s, a.SetID, c.NextPageIndex, c.RequestCursor}
		raw, found, e := r.evidence.Get(ctx, key)
		if e != nil {
			return Outcome{}, false, e
		}
		if !found {
			if e = r.refresh(ctx, s, a, c); e != nil {
				return Outcome{}, false, e
			}
			credentialCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			var credentials ProviderCredentials
			if scoped, ok := r.credentials.(interface {
				ResolveScoped(context.Context, Scope, string, string, CredentialReference) (ProviderCredentials, error)
			}); ok {
				credentials, e = scoped.ResolveScoped(credentialCtx, s, c.SetID, c.Fence, c.reference())
			} else {
				credentials, e = r.credentials.Resolve(credentialCtx, c.reference())
			}
			expired := credentialCtx.Err() != nil
			stop()
			if e != nil || expired {
				return r.abort(ctx, s, a, "provider_failure", failure("credentials_unavailable", true))
			}
			if e = r.refresh(ctx, s, a, c); e != nil {
				return Outcome{}, false, e
			}
			fetchCtx, stop := context.WithTimeout(ctx, 10*time.Second)
			raw, e = r.provider.Fetch(fetchCtx, FetchRequest{AccountID: c.AccountID, Cursor: c.RequestCursor, Credentials: credentials})
			expired = fetchCtx.Err() != nil
			stop()
			if e != nil || expired {
				var p *ProviderFailure
				if errors.As(e, &p) && p.Code == "pagination_mutation" && !expired {
					return r.abort(ctx, s, a, "pagination_mutation", e)
				}
				if expired {
					e = failure("provider_transient", true)
				}
				return r.abort(ctx, s, a, "provider_failure", e)
			}
		}
		if len(raw) == 0 || total+len(raw) > MaxRawBytes {
			return r.abort(ctx, s, a, "decoder_rejected", failure("limit_exceeded", false))
		}
		page, e := plaid.DecodePage(raw, c.expected())
		if e != nil {
			return r.abort(ctx, s, a, "decoder_rejected", failure("decoder_rejected", false), plaid.ErrorCode(e))
		}
		body, e := json.Marshal(pageWire{PageIndex: c.NextPageIndex, RequestCursor: c.RequestCursor, Page: page})
		if e != nil {
			return r.abort(ctx, s, a, "decoder_rejected", failure("decoder_rejected", false))
		}
		if len(body)+c.ByteCount > 2<<20 || len(page.Added)+len(page.Modified)+len(page.Removed)+c.ChangeCount > 1000 {
			return r.abort(ctx, s, a, "decoder_rejected", failure("limit_exceeded", false))
		}
		if e = r.refresh(ctx, s, a, c); e != nil {
			return Outcome{}, false, e
		}
		if !found {
			if e = r.evidence.Put(ctx, key, raw); e != nil {
				return Outcome{}, false, e
			}
		}
		// SQL independently reauthorizes after storage; revocation cannot publish retained bytes.
		if e = r.repo.Stage(ctx, s, a, c.NextPageIndex, body); e != nil {
			return Outcome{}, false, e
		}
	}
	return Outcome{}, false, failure("limit_exceeded", false)
}

// cleanup makes a known-abandoned attempt unreadable before another internal attempt starts.
func (r *Runner) cleanup(ctx context.Context, s Scope, a attempt) error {
	for page := 0; page < 10; page++ {
		if e := r.evidence.Delete(ctx, EvidenceKey{s, a.SetID, page, nil}); e != nil {
			return failure("cleanup_required", true)
		}
	}
	return nil
}

// abort commits abandonment before cleanup; uncertain abandonment never authorizes restart.
func (r *Runner) abort(ctx context.Context, s Scope, a attempt, reason string, cause error, decoderCode ...string) (Outcome, bool, error) {
	if ctx.Err() != nil {
		return Outcome{}, false, failure("cancelled", true)
	}
	var abandonErr error
	if len(decoderCode) == 1 {
		abandonErr = r.repo.RejectDecode(ctx, s, a, decoderCode[0])
	} else {
		abandonErr = r.repo.Abandon(ctx, s, a, reason)
	}
	if abandonErr != nil {
		return Outcome{}, false, abandonErr
	}
	if e := r.cleanup(ctx, s, a); e != nil {
		return Outcome{Status: "abandoned"}, false, e
	}
	if reason == "pagination_mutation" {
		return Outcome{Status: "abandoned"}, true, nil
	}
	var p *ProviderFailure
	if errors.As(cause, &p) {
		code := p.Code
		if !oneOf(code, "provider_transient", "provider_rejected", "retry_deferred") {
			code = "provider_transient"
		}
		delay := p.RetryAfter
		if delay < 0 {
			delay = 0
		}
		if delay > 24*time.Hour {
			code = "retry_deferred"
			delay = 0
		}
		return Outcome{Status: "abandoned", RetryAfter: delay}, false, failure(code, code == "provider_transient")
	}
	var own *Failure
	if errors.As(cause, &own) {
		return Outcome{Status: "abandoned"}, false, own
	}
	return Outcome{Status: "abandoned"}, false, failure("provider_transient", true)
}

// publicError closes the error vocabulary even when an injected dependency returns arbitrary text.
func publicError(err error) error {
	var own *Failure
	if errors.As(err, &own) {
		switch own.Code {
		case "invalid_request", "not_authorized", "evidence_conflict", "evidence_deleted", "invalid_evidence", "evidence_missing", "invalid_metadata", "invalid_update", "stale_attempt", "limit_exceeded", "attempt_terminal", "decoder_rejected", "provider_rejected", "configuration_error", "invalid_provider_request", "retry_deferred":
			return failure(own.Code, false)
		case "busy", "temporarily_unavailable", "commit_unknown", "evidence_unavailable", "cleanup_required", "cancelled", "credentials_unavailable", "provider_transient", "pagination_mutation":
			return failure(own.Code, true)
		}
	}
	return failure("temporarily_unavailable", true)
}
