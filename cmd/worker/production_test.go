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

// sharedProductionArgs pins the reviewed shared instance with fictional caller/version metadata only.
func sharedProductionArgs() []string {
	args := productionArgs()
	for i, value := range args {
		value = strings.ReplaceAll(value, "synthetic-project", "jayburd-bookkeeping-dev")
		value = strings.ReplaceAll(value, "projects/123456789012/", "projects/9088794002/")
		if strings.HasPrefix(value, "-cloud-sql-instance=") {
			value = "-cloud-sql-instance=jayburd-bookkeeping-dev:us-central1:jayburd-dev"
		}
		if strings.HasPrefix(value, "-project-number=") {
			value = "-project-number=9088794002"
		}
		if strings.HasPrefix(value, "-bank-credential-key=") {
			value = "-bank-credential-key=projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/bank-credentials"
		}
		if strings.HasPrefix(value, "-link-handoff-key=") {
			value = "-link-handoff-key=projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-prod/cryptoKeys/hosted-link"
		}
		args[i] = value
	}
	return args
}

// TestSharedProductionWorkerArguments covers the actual profile regression and consistent mixed-resource denials.
func TestSharedProductionWorkerArguments(t *testing.T) {
	if _, err := parse(sharedProductionArgs()); err != nil {
		t.Error("reviewed shared Production consumer rejected")
	}
	for _, mismatch := range []string{"shared-id-foreign-number", "foreign-id-shared-number", "wrong-shared-instance", "sandbox-secret", "sandbox-key"} {
		t.Run(mismatch, func(t *testing.T) {
			args := sharedProductionArgs()
			for i, value := range args {
				switch mismatch {
				case "shared-id-foreign-number":
					value = strings.ReplaceAll(value, "projects/9088794002/", "projects/123456789012/")
					if value == "-project-number=9088794002" {
						value = "-project-number=123456789012"
					}
				case "foreign-id-shared-number":
					value = strings.ReplaceAll(value, "jayburd-bookkeeping-dev", "synthetic-project")
					if strings.HasPrefix(value, "-cloud-sql-instance=") {
						value = "-cloud-sql-instance=synthetic-project:us-central1:pilot-db"
					}
				case "wrong-shared-instance":
					if strings.HasPrefix(value, "-cloud-sql-instance=") {
						value = "-cloud-sql-instance=jayburd-bookkeeping-dev:us-central1:pilot-db"
					}
				case "sandbox-secret":
					value = strings.ReplaceAll(value, "jayburd-prod-bank-sync-client-secret", "jayburd-dev-bank-sync-client-secret")
				case "sandbox-key":
					value = strings.ReplaceAll(value, "keyRings/jayburd-prod", "keyRings/jayburd-dev")
				}
				args[i] = value
			}
			if _, err := parse(args); err == nil {
				t.Fatal("mixed or unreviewed shared profile accepted")
			}
		})
	}
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
