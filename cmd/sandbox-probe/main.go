// Command sandbox-probe is an explicitly enabled, managed-identity-only Sandbox smoke test.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxqualification"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
	"io"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

// run validates public numeric pins before composing identity or reading credentials.
func run(ctx context.Context, args []string, out io.Writer, probe func(context.Context, string, string, string) sandboxqualification.Receipt) int {
	f := flag.NewFlagSet("sandbox-probe", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	allow := f.Bool("allow-sandbox", false, "")
	project := f.String("project-number", "", "")
	id := f.String("client-id-version", "", "")
	secret := f.String("client-secret-version", "", "")
	numeric := regexp.MustCompile(`^[1-9][0-9]{0,18}$`)
	r := sandboxqualification.Blocked("invalid_invocation")
	if f.Parse(args) == nil && f.NArg() == 0 && *allow && numeric.MatchString(*project) && numeric.MatchString(*id) && numeric.MatchString(*secret) && probe != nil {
		r = probe(ctx, *project, *id, *secret)
	}
	_ = json.NewEncoder(out).Encode(r)
	if r.Status == "blocked" {
		return 2
	}
	return 0
}

// managed reads exact two GCP versions in runtime memory; no environment values or ADC are used.
func managed(ctx context.Context, project, id, secret string) sandboxqualification.Receipt {
	t := workerruntime.DirectTransport()
	defer t.CloseIdleConnections()
	tokens, e := workerruntime.NewMetadataTokens(t)
	if e != nil {
		return sandboxqualification.Blocked("input_unavailable")
	}
	reader, e := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: project, Tokens: tokens, Transport: t})
	if e != nil {
		return sandboxqualification.Blocked("input_invalid")
	}
	readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	clientID, e := reader.Read(readCtx, "projects/"+project+"/secrets/jayburd-dev-bank-sync-client-id/versions/"+id)
	if e != nil {
		return sandboxqualification.Blocked("input_unavailable")
	}
	credential, e := reader.Read(readCtx, "projects/"+project+"/secrets/jayburd-dev-bank-sync-client-secret/versions/"+secret)
	if e != nil {
		return sandboxqualification.Blocked("input_unavailable")
	}
	return sandboxqualification.RunHosted(ctx, true, clientID, credential, t)
}

// main propagates platform termination without printing dependency errors.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, managed))
}
