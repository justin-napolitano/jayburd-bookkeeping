package providersecrets

import (
	"context"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
)

// PreflightResult contains fixed, redacted evidence classifications only.
type PreflightResult struct {
	Status string `json:"status"`
	Code   string `json:"code"`
}

// Preflight verifies one authorized runtime reference, never source writes or provider calls.
// PASS is only exact credential readback, not live-provider or whole-system qualification.
func (r *Resolver) Preflight(ctx context.Context, ref acquisition.CredentialReference) PreflightResult {
	_, e := r.Resolve(ctx, ref)
	if e == nil {
		return PreflightResult{"PASS", "exact_runtime_reference_verified"}
	}
	switch code(e) {
	case "secret_mapping_pending", "secret_absent", "secret_transport_unknown":
		return PreflightResult{"PARTIAL", "runtime_reference_unavailable"}
	default:
		return PreflightResult{"BLOCKED", "runtime_reference_rejected"}
	}
}
