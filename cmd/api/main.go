// Command api is health-only unless the owner explicitly selects managed mode.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connectionruntime"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/server"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
)

// main emits fixed operational status only; never print flags, tokens or DSNs.
func main() {
	flag.CommandLine.SetOutput(io.Discard)
	var enrollment connectionruntime.Config
	flag.BoolVar(&enrollment.Enabled, "connections-enabled", false, "explicitly enable Sandbox connection composition")
	flag.BoolVar(&enrollment.LinkEventsQualified, "plaid-link-events-qualified", false, "ordinary SDK account metadata; not required by Sandbox Hosted Link")
	flag.StringVar(&enrollment.ProjectNumber, "provider-project-number", "", "canonical Secret Manager project number")
	flag.StringVar(&enrollment.InfisicalOrigin, "infisical-origin", "", "approved Infisical HTTPS origin")
	flag.StringVar(&enrollment.InfisicalProject, "infisical-project", "", "explicit Infisical project UUID")
	flag.StringVar(&enrollment.InfisicalEnvironment, "infisical-environment", "", "explicit dev namespace")
	flag.StringVar(&enrollment.InfisicalIdentity, "infisical-identity", "", "GCP-auth writer identity UUID")
	flag.StringVar(&enrollment.RegistryBucket, "provider-registry-bucket", "", "private metadata registry bucket")
	flag.StringVar(&enrollment.RegistryPrefix, "provider-registry-prefix", "", "registry object prefix")
	flag.StringVar(&enrollment.ClientIDVersion, "plaid-client-id-version", "", "numeric immutable GSM client ID version")
	flag.StringVar(&enrollment.ClientSecretVersion, "plaid-client-secret-version", "", "numeric immutable GSM client secret version")
	flag.StringVar(&enrollment.ConsentExpiresAt, "plaid-consent-expires-at", "", "fixed RFC3339 local approval expiry within30days")
	flag.StringVar(&enrollment.WebhookURL, "plaid-webhook-url", "", "optional exact reviewed Sandbox Transactions receiver")

	mode := flag.String("mode", "health", "health or explicitly provisioned managed mode")
	listen := flag.String("listen", "127.0.0.1:8080", "explicit listening address")
	origin := flag.String("origin", "", "exact HTTPS frontend origin; loopback HTTP allowed for local development")
	project := flag.String("project", "", "explicit Identity Platform project")
	databaseMode := flag.String("database-mode", "url", "url or cloudsql IAM connector")
	instance := flag.String("cloud-sql-instance", "", "explicit project:region:instance identifier")
	flag.Parse()
	if enrollment.Enabled {
		enrollment.ProjectID = *project
	}
	if flag.NArg() != 0 || enrollment.ManagedScope(*mode, *databaseMode, *project) != nil {
		fmt.Fprintln(os.Stderr, "server configuration rejected")
		os.Exit(1)
	}
	c := server.Config{Mode: *mode, Listen: *listen, Origin: *origin, Project: *project, DatabaseMode: *databaseMode, CloudSQLInstance: *instance}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var d server.Dependencies
	if c.Mode == "managed" {
		if c.DatabaseMode != "cloudsql" {
			c.DatabaseURL = os.Getenv("JAYBURD_DATABASE_URL")
		}
		var err error
		d, err = server.Managed(ctx, c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "server initialization failed")
			os.Exit(1)
		}
	}
	if enrollment.Enabled {
		transport := workerruntime.DirectTransport()
		tokens, e := workerruntime.NewMetadataTokens(transport)
		operations, ok := d.Operations.(*ledger.Service)
		if e != nil || !ok || operations == nil || operations.DB == nil {
			transport.CloseIdleConnections()
			if d.Close != nil {
				d.Close()
			}
			fmt.Fprintln(os.Stderr, "connection initialization failed")
			os.Exit(1)
		}
		service, e := connectionruntime.New(ctx, enrollment, operations.DB, transport, tokens)
		if e != nil {
			transport.CloseIdleConnections()
			if d.Close != nil {
				d.Close()
			}
			fmt.Fprintln(os.Stderr, "connection initialization failed")
			os.Exit(1)
		}
		d.Connections = service
		previousClose := d.Close
		d.Close = func() {
			transport.CloseIdleConnections()
			if previousClose != nil {
				previousClose()
			}
		}
	}
	s, err := server.New(c, d)
	if err != nil {
		if d.Close != nil {
			d.Close()
		}
		fmt.Fprintln(os.Stderr, "server configuration rejected")
		os.Exit(1)
	}
	if err = s.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "server stopped")
		os.Exit(1)
	}
}
