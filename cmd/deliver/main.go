// Command deliver runs only explicitly selected development credential delivery.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
)

// config carries public binding identifiers; secret values and URLs are never accepted.
type config struct {
	project, number, sourceProject, identity, sourceID, version, key string
	lookup                                                           bool
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var slotPattern = regexp.MustCompile(`^slot-0[0-7]$`)
var projectPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

// valid rejects missing bindings before any metadata authentication or external operation.
func (c config) valid() bool {
	if c.key != "JAYBURD_DELIVERY_CANARY" && c.key != "PLAID_CLIENT_ID" && c.key != "PLAID_SECRET" && !slotPattern.MatchString(c.key) {
		return false
	}
	n, e := strconv.ParseInt(c.number, 10, 64)
	v, ve := strconv.ParseInt(c.version, 10, 64)
	if e != nil || n <= 0 || strconv.FormatInt(n, 10) != c.number || ve != nil || v <= 0 || strconv.FormatInt(v, 10) != c.version || !projectPattern.MatchString(c.project) {
		return false
	}
	ids := []string{c.sourceProject, c.identity}
	if !c.lookup {
		ids = append(ids, c.sourceID)
	} else if c.sourceID != "" {
		return false
	}
	for _, id := range ids {
		if !uuidPattern.MatchString(id) || id == "00000000-0000-0000-0000-000000000000" {
			return false
		}
	}
	return true
}

// run has no implicit execution mode and never renders flag values or dependency errors.
func run(ctx context.Context, args []string, out io.Writer, execute func(context.Context, config) (providersecrets.CanaryMetadata, error)) int {
	f := flag.NewFlagSet("deliver", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var c config
	var authorized, check bool
	f.StringVar(&c.project, "project-id", "", "")
	f.StringVar(&c.number, "project-number", "", "")
	f.StringVar(&c.sourceProject, "infisical-project", "", "")
	f.StringVar(&c.identity, "infisical-identity", "", "")
	f.StringVar(&c.sourceID, "source-id", "", "")
	f.StringVar(&c.key, "key", "JAYBURD_DELIVERY_CANARY", "")
	f.StringVar(&c.version, "source-version", "", "")
	f.BoolVar(&authorized, "execute", false, "")
	f.BoolVar(&check, "check", false, "")
	f.BoolVar(&c.lookup, "lookup", false, "")
	if f.Parse(args) != nil {
		fmt.Fprintln(out, "BLOCKED: delivery configuration")
		return 1
	}
	modes := 0
	for _, on := range []bool{authorized, check, c.lookup} {
		if on {
			modes++
		}
	}
	if f.NArg() != 0 || modes != 1 || !c.valid() {
		fmt.Fprintln(out, "BLOCKED: delivery configuration")
		return 1
	}
	if check {
		fmt.Fprintln(out, "PASS: configuration only; no connection tested")
		return 0
	}
	if execute == nil {
		fmt.Fprintln(out, "BLOCKED: delivery configuration")
		return 1
	}
	metadata, err := execute(ctx, c)
	if err != nil {
		fmt.Fprintln(out, "PARTIAL: delivery unconfirmed; inspect metadata before recovery")
		return 1
	}
	if c.lookup {
		if !uuidPattern.MatchString(metadata.SourceID) || metadata.SourceVersion != c.version {
			fmt.Fprintln(out, "BLOCKED: invalid lookup metadata")
			return 1
		}
		if json.NewEncoder(out).Encode(metadata) != nil {
			return 1
		}
		return 0
	}
	fmt.Fprintln(out, "PASS: delivery acknowledged; independent verification pending")
	return 0
}

// managed composes fixed endpoints and metadata identity without ADC, proxies, SQL or Plaid access.
func managed(ctx context.Context, c config) (providersecrets.CanaryMetadata, error) {
	transport := workerruntime.DirectTransport()
	defer transport.CloseIdleConnections()
	tokens, e := workerruntime.NewMetadataTokens(transport)
	if e != nil {
		return providersecrets.CanaryMetadata{}, e
	}
	registry, e := providersecrets.NewGCSRegistry(providersecrets.GCSConfig{Bucket: c.project + "-jayburd-dev-delivery-registry", Prefix: "delivery-v1", Tokens: tokens, Transport: transport})
	if e != nil {
		return providersecrets.CanaryMetadata{}, e
	}
	delivery, e := providersecrets.NewDelivery(providersecrets.DeliveryConfig{
		Source:      providersecrets.StoreConfig{InfisicalOrigin: "https://app.infisical.com", ProjectID: c.sourceProject, Environment: "dev", IdentityID: c.identity, IdentityTokens: tokens, Transport: transport},
		Destination: providersecrets.GCPConfig{ProjectID: c.number, Tokens: tokens, Transport: transport},
	}, registry)
	if e != nil {
		return providersecrets.CanaryMetadata{}, e
	}
	if c.lookup {
		return delivery.LookupSelected(ctx, c.key, c.version)
	}
	_, e = delivery.Deliver(ctx, providersecrets.DeliveryRequest{Key: c.key, SourceID: c.sourceID, SourceVersion: c.version})
	return providersecrets.CanaryMetadata{}, e
}

// main propagates termination to the bounded one-shot operation; there is no listener or retry loop.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	result := run(ctx, os.Args[1:], os.Stdout, managed)
	cancel()
	os.Exit(result)
}
