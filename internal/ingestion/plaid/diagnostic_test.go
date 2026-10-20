package plaid_test

import (
	"errors"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ingestion/plaid"
	"testing"
)

// TestDecoderDiagnosticRedaction cannot expose arbitrary error text through a code.
func TestDecoderDiagnosticRedaction(t *testing.T) {
	for _, err := range []error{errors.New("private-value"), &plaid.Error{Code: "private-value"}, (*plaid.Error)(nil), nil} {
		if got := plaid.ErrorCode(err); got != "invalid_field" {
			t.Fatal("untrusted code escaped")
		}
	}
	if plaid.ErrorCode(&plaid.Error{Code: "unsupported_currency"}) != "unsupported_currency" {
		t.Fatal("known classification lost")
	}
}
