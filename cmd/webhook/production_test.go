package main

import (
	"strings"
	"testing"
	"time"
)

// TestProductionReceiverArguments requires explicit profile, exact static versions and finite approval metadata.
func TestProductionReceiverArguments(t *testing.T) {
	args := arguments()
	for i, v := range args {
		args[i] = strings.ReplaceAll(v, "jayburd-dev", "jayburd-prod")
	}
	args = append(args, "--provider-environment=production", "--provider-approval-expires-at="+time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	c, e := parse(args)
	if e != nil || c.ProviderEnvironment() != "production" {
		t.Fatal("valid receiver profile rejected")
	}
	args = append(args, "--provider-environment=Production")
	if _, e := parse(args); e == nil {
		t.Fatal("unknown profile accepted")
	}
}
