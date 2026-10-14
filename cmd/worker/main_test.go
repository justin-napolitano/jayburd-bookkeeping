package main

import (
	"strings"
	"testing"
)

// baseArgs supplies only fictional public deployment identifiers.
func baseArgs(mode string) []string {
	caller := "jayburd-dev-task-invoker"
	if mode == "publisher" {
		caller = "jayburd-dev-scheduler"
	}
	return []string{"-mode=" + mode, "-cloud-sql-instance=synthetic-project:us-central1:jayburd-dev", "-caller-audience=https://worker-example.run.app", "-caller-email=" + caller + "@synthetic-project.iam.gserviceaccount.com", "-caller-subject=123456789012345678901"}
}

// publisherArgs pins one book mandate and one private task destination.
func publisherArgs() []string {
	return append(baseArgs("publisher"), "-binding-id=11111111-1111-4111-8111-111111111111", "-queue-location=us-central1", "-queue-name=jayburd-dev-acquisition", "-consumer-url=https://consumer-example.run.app/internal/acquisition/consume", "-consumer-audience=https://consumer-example.run.app", "-task-service-account=jayburd-dev-task-invoker@synthetic-project.iam.gserviceaccount.com")
}

// consumerArgs requires all durable dependency identifiers without plaintext credentials.
func consumerArgs() []string {
	return append(baseArgs("consumer"), "-project-number=123456789012", "-credential-mapping-bucket=synthetic-project-jayburd-dev-credential-registry", "-credential-mapping-prefix=provider-credentials-v1", "-evidence-bucket=synthetic-project-jayburd-dev-acquisition-evidence", "-plaid-client-id-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-id/versions/1", "-plaid-client-secret-version=projects/123456789012/secrets/jayburd-dev-bank-sync-client-secret/versions/1")
}

// TestWorkerArguments requires explicit operational modes and keeps consumer durable dependencies mandatory.
func TestWorkerArguments(t *testing.T) {
	for _, args := range [][]string{publisherArgs(), consumerArgs()} {
		o, e := parse(args)
		if e != nil || o.Worker.Listen != "127.0.0.1:8080" {
			t.Fatal("valid explicit configuration rejected")
		}
	}
	for _, args := range [][]string{nil, baseArgs("consumer"), append(publisherArgs(), "-evidence-bucket=synthetic-project-jayburd-dev-acquisition-evidence"), append(consumerArgs(), "-binding-id=11111111-1111-4111-8111-111111111111"), append(publisherArgs(), "-password=synthetic-sensitive"), append(consumerArgs(), "unexpected-sensitive-argument")} {
		_, e := parse(args)
		if e == nil || strings.Contains(e.Error(), "sensitive") {
			t.Fatal("unsafe flag acceptance or diagnostics")
		}
	}
}

// TestWorkerIdentifierPins rejects cross-project storage, mutable versions and alternate caller capabilities.
func TestWorkerIdentifierPins(t *testing.T) {
	for _, replace := range []struct{ from, to string }{{"provider-credentials-v1", "other-prefix"}, {"synthetic-project-jayburd-dev-credential-registry", "foreign-registry"}, {"synthetic-project-jayburd-dev-acquisition-evidence", "foreign-evidence"}, {"/versions/1", "/versions/latest"}, {"jayburd-dev-task-invoker@", "jayburd-dev-api@"}, {"/secrets/jayburd-dev-bank-sync-client-id/", "/secrets/other/"}} {
		args := consumerArgs()
		for i, a := range args {
			args[i] = strings.ReplaceAll(a, replace.from, replace.to)
		}
		if _, e := parse(args); e == nil {
			t.Fatal("identifier pin bypass")
		}
	}
	for _, v := range []string{"0", "01", "latest", "9223372036854775808", "1/extra"} {
		if secretVersion("projects/synthetic-project/secrets/client/versions/"+v, "synthetic-project", "client") {
			t.Fatal("noncanonical secret version")
		}
	}
}
