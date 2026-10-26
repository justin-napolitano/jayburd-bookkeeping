package main

import (
	"context"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
	"strings"
	"testing"
	"time"
)

// productionArgs is an independent fictional destination/version/key profile, never cloud discovery.
func productionArgs() []string {
	args := consumerArgs()
	for i, v := range args {
		v = strings.ReplaceAll(v, "jayburd-dev", "jayburd-prod")
		v = strings.ReplaceAll(v, ":jayburd-prod", ":pilot-db")
		args[i] = v
	}
	return append(args, "-provider-environment=production", "-provider-approval-expires-at="+time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "-bank-credential-key=projects/synthetic-project/locations/us-central1/keyRings/pilot/cryptoKeys/bank", "-link-handoff-key=projects/synthetic-project/locations/us-central1/keyRings/pilot/cryptoKeys/link")
}

// TestProductionWorkerArguments rejects cross-profile identities, evidence, keys and secret aliases before startup.
func TestProductionWorkerArguments(t *testing.T) {
	if _, e := parse(productionArgs()); e != nil {
		t.Fatal("valid fictional Production rejected")
	}
	for _, pair := range [][2]string{{"jayburd-prod-bank-sync-client-secret", "jayburd-dev-bank-sync-client-secret"}, {"jayburd-prod-acquisition-evidence", "jayburd-dev-acquisition-evidence"}, {"keyRings/pilot", "keyRings/jayburd-dev"}, {"/versions/1", "/versions/latest"}, {"-provider-environment=production", "-provider-environment=Production"}, {"jayburd-prod-task-invoker", "jayburd-dev-task-invoker"}} {
		args := productionArgs()
		for i, v := range args {
			args[i] = strings.ReplaceAll(v, pair[0], pair[1])
		}
		if _, e := parse(args); e == nil {
			t.Fatal("mixed worker profile accepted")
		}
	}
}

// TestProductionWorkerColdExpired performs no metadata/SQL/provider discovery after approval expiry.
func TestProductionWorkerColdExpired(t *testing.T) {
	args := productionArgs()
	for i, v := range args {
		if strings.HasPrefix(v, "-provider-approval-expires-at=") {
			args[i] = "-provider-approval-expires-at=2020-01-01T00:00:00Z"
		}
	}
	o, e := parse(args)
	if e != nil {
		t.Fatal(e)
	}
	r, close, e := managed(context.Background(), o)
	if e != workerruntime.ErrUnavailable || r != nil || close != nil {
		t.Fatal("expired worker reached startup")
	}
}
