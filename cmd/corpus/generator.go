package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
)

// auditGenerator checks exact EUR balance in the pinned generator's synthetic CSV;
// it is not a general import parser and never repairs rounding differences.
func auditGenerator(r io.Reader, out io.Writer) error {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return err
	}
	columns := map[string]int{}
	for i, name := range header {
		if _, exists := columns[name]; exists {
			return fmt.Errorf("duplicate CSV column")
		}
		columns[name] = i
	}
	for _, name := range []string{"journal_entry_id", "debit_amount_eur", "credit_amount_eur"} {
		if _, ok := columns[name]; !ok {
			return fmt.Errorf("missing column %s", name)
		}
	}
	balances := map[string]*big.Rat{}
	lines := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		id := row[columns["journal_entry_id"]]
		if id == "" {
			return fmt.Errorf("empty journal identity")
		}
		debit, err := decimal(row[columns["debit_amount_eur"]], false)
		if err != nil {
			return fmt.Errorf("invalid debit at row %d", lines+2)
		}
		credit, err := decimal(row[columns["credit_amount_eur"]], false)
		if err != nil {
			return fmt.Errorf("invalid credit at row %d", lines+2)
		}
		add(balances, id, new(big.Rat).Sub(debit, credit))
		lines++
	}
	if lines == 0 {
		return fmt.Errorf("empty generator sample")
	}
	failures := 0
	for _, balance := range balances {
		if balance.Sign() != 0 {
			failures++
		}
	}
	fmt.Fprintf(out, "journals=%d lines=%d unbalanced=%d\n", len(balances), lines, failures)
	if failures != 0 {
		return fmt.Errorf("sample rejected: exact journal balance required")
	}
	return nil
}
