package webhook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// Handler owns a separately composed public notification boundary, never bookkeeping routes.
type Handler struct {
	verifier *Verifier
	receiver Receiver
	slots    chan struct{}
}

// NewHandler requires explicit verifier and durable receiver dependencies.
func NewHandler(v *Verifier, r Receiver) (*Handler, error) {
	if v == nil || r == nil {
		return nil, errUnavailable
	}
	return &Handler{v, r, make(chan struct{}, 16)}, nil
}

// rejected emits only fixed classifications and never signature, body, key or SQL diagnostics.
func rejected(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"status":"not_accepted"}`))
}

// ServeHTTP acknowledges only a confirmed receipt after exact signature and body verification.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != "POST" || r.URL.Path != "/webhooks/plaid" || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		rejected(w, 404)
		return
	}
	headers := r.Header.Values("Plaid-Verification")
	if len(headers) != 1 || len(headers[0]) == 0 || len(headers[0]) > 8192 {
		rejected(w, 401)
		return
	}
	if len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != "application/json" || len(r.Header.Values("Content-Encoding")) != 0 {
		rejected(w, 400)
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		rejected(w, 503)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		rejected(w, 503)
		return
	}
	body, e := io.ReadAll(http.MaxBytesReader(w, r.Body, bodyLimit))
	if e != nil {
		var maximum *http.MaxBytesError
		if errors.As(e, &maximum) {
			rejected(w, 413)
		} else {
			rejected(w, 400)
		}
		return
	}
	verified, e := h.verifier.Verify(ctx, headers[0], body)
	if e != nil {
		if errors.Is(e, errUnavailable) {
			rejected(w, 503)
		} else {
			rejected(w, 401)
		}
		return
	}
	receipt, e := h.receiver.Receive(ctx, verified)
	if e != nil {
		if errors.Is(e, errDenied) {
			rejected(w, 403)
		} else {
			rejected(w, 503)
		}
		return
	}
	if ctx.Err() != nil || !validReceipt(receipt) {
		rejected(w, 503)
		return
	}
	w.WriteHeader(204)
}

// validReceipt prevents a malformed injected receiver from acknowledging uncommitted work.
func validReceipt(r Receipt) bool {
	return (r.State == "accepted" || r.State == "duplicate") && validID(r.ReceiptID) && r.StreamCount >= 1 && r.StreamCount <= 25
}
