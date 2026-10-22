// Command webhook explicitly composes a dedicated Sandbox notification listener.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhookhost"
)

// parse accepts public identifiers only and never echoes malformed arguments.
func parse(args []string) (webhookhost.Config, error) {
	var c webhookhost.Config
	f := flag.NewFlagSet("webhook", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&c.Listen, "listen", "127.0.0.1:8080", "explicit TCP listener")
	f.StringVar(&c.Instance, "cloud-sql-instance", "", "project:us-central1:instance")
	f.StringVar(&c.ProjectNumber, "project-number", "", "numeric secret project identifier")
	f.StringVar(&c.ClientIDVersion, "plaid-client-id-version", "", "exact numeric client ID secret version resource")
	f.StringVar(&c.ClientSecretVersion, "plaid-client-secret-version", "", "exact numeric client secret version resource")
	if f.Parse(args) != nil || f.NArg() != 0 || c.Validate() != nil {
		return webhookhost.Config{}, webhookhost.ErrConfiguration
	}
	return c, nil
}

// runtime supplies only controlled lifecycle methods; tests can prove rejected flags never initialize it.
type runtime interface {
	Run(context.Context) error
	Close()
}

// execute validates before construction and cleans resources before main selects an exit code.
func execute(ctx context.Context, args []string, construct func(context.Context, webhookhost.Config) (runtime, error)) error {
	c, err := parse(args)
	if err != nil || construct == nil {
		return webhookhost.ErrConfiguration
	}
	if ctx == nil {
		return webhookhost.ErrConfiguration
	}
	if ctx.Err() != nil {
		return webhookhost.ErrServe
	}
	r, err := construct(ctx, c)
	if err != nil {
		if r != nil {
			r.Close()
		}
		return webhookhost.ErrServe
	}
	if r == nil {
		return webhookhost.ErrConfiguration
	}
	defer r.Close()
	return r.Run(ctx)
}

// main reports a fixed outcome only, never configuration, payloads or underlying diagnostics.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	err := execute(ctx, os.Args[1:], func(ctx context.Context, c webhookhost.Config) (runtime, error) { return webhookhost.Managed(ctx, c) })
	if err != nil {
		fmt.Fprintln(os.Stderr, "webhook initialization or service failed")
		os.Exit(1)
	}
}
