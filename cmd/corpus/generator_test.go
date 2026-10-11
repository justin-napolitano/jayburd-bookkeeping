package main

import (
	"io"
	"strings"
	"testing"
)

// TestGeneratorAudit rejects cent differences regardless of an upstream tolerance.
func TestGeneratorAudit(t *testing.T) {
	header := "journal_entry_id,debit_amount_eur,credit_amount_eur\n"
	for _, tc := range []struct {
		name, rows string
		valid      bool
	}{
		{"balanced", "fictional,1.00,0\nfictional,0,1.00\n", true},
		{"cent_difference", "fictional,1.00,0\nfictional,0,0.99\n", false},
		{"cross_journal", "one,1,0\ntwo,0,1\n", false},
		{"invalid_decimal", "one,NaN,0\n", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := auditGenerator(strings.NewReader(header+tc.rows), io.Discard)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}
