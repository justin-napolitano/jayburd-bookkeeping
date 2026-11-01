// Command sandbox-webhook defaults offline and requests one event through normal scoped acquisition.
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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/cloudevidence"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxwebhook"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
)

const instance = "jayburd-bookkeeping-dev:us-central1:jayburd-dev"
const projectNumber = "9088794002"

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// options contains exact public scope/pins; SQL still decides current authority and credential identity.
type options struct {
	scope                                       acquisition.Scope
	claim, clientIDVersion, clientSecretVersion string
	expires                                     time.Time
}

// result carries closed Boolean evidence, not provider/SQL payloads or a signed-delivery claim.
type result struct {
	Requested bool `json:"webhook_requested"`
	Completed bool `json:"sync_completed"`
}

// validID rejects zero and noncanonical scope identifiers before any external I/O.
func validID(s string) bool {
	return uuid.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}

// version pins a positive signed-bigint version without aliases or normalization.
func version(s string) bool {
	n, e := strconv.ParseInt(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatInt(n, 10) == s
}

// run keeps default/invalid/expired invocations offline and suppresses argument diagnostics.
func run(ctx context.Context, args []string, out io.Writer, execute func(context.Context, options) (result, error)) error {
	f := flag.NewFlagSet("sandbox-webhook", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	action := f.String("action", "verify", "")
	allow := f.Bool("allow-sandbox", false, "")
	target := f.String("cloud-sql-instance", "", "")
	project := f.String("project-number", "", "")
	expiry := f.String("expires-at", "", "")
	var o options
	f.StringVar(&o.scope.EntityID, "entity-id", "", "")
	f.StringVar(&o.scope.BookID, "book-id", "", "")
	f.StringVar(&o.scope.StreamID, "stream-id", "", "")
	f.StringVar(&o.claim, "claim-id", "", "")
	f.StringVar(&o.clientIDVersion, "client-id-version", "", "")
	f.StringVar(&o.clientSecretVersion, "client-secret-version", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return sandboxwebhook.ErrProbe
	}
	if *action == "verify" && (f.NFlag() == 0 || f.NFlag() == 1) {
		_, err := io.WriteString(out, "{\"offline\":true}\n")
		return err
	}
	var err error
	o.expires, err = time.Parse(time.RFC3339, *expiry)
	now := time.Now()
	if *action != "run" || !*allow || *target != instance || *project != projectNumber || err != nil || o.expires.Format("2006-01-02T15:04:05Z") != *expiry || !o.expires.After(now) || o.expires.After(now.Add(10*time.Minute)) || !validID(o.scope.EntityID) || !validID(o.scope.BookID) || !validID(o.scope.StreamID) || !validID(o.claim) || !version(o.clientIDVersion) || !version(o.clientSecretVersion) || execute == nil || ctx.Err() != nil {
		return sandboxwebhook.ErrProbe
	}
	bounded, cancel := context.WithDeadline(ctx, o.expires)
	defer cancel()
	r, err := execute(bounded, o)
	if json.NewEncoder(out).Encode(r) != nil || err != nil || !r.Requested || !r.Completed || bounded.Err() != nil {
		return sandboxwebhook.ErrProbe
	}
	return nil
}

// managed uses the existing sync IAM login/KMS/evidence capabilities; it cannot select Production or an actor.
func managed(parent context.Context, o options) (result, error) {
	ctx, cancel := context.WithTimeout(parent, 85*time.Second)
	defer cancel()
	t := workerruntime.DirectTransport()
	defer t.CloseIdleConnections()
	tokens, err := workerruntime.NewMetadataTokens(t)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	reader, err := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: projectNumber, Tokens: tokens, Transport: t})
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	wrapper, err := bankcredential.NewKMS("projects/jayburd-bookkeeping-dev/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials", tokens, t)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	startup, stop := context.WithTimeout(ctx, 20*time.Second)
	defer stop()
	config, closeConnector, err := clouddb.Connector(startup, clouddb.Config{Instance: instance, Component: "sync"})
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	defer closeConnector()
	pool, err := pgxpool.NewWithConfig(startup, config)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	defer pool.Close()
	if pool.Ping(startup) != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	static := func(ctx context.Context) (acquisition.ProviderCredentials, error) {
		id, err := reader.Read(ctx, "projects/"+projectNumber+"/secrets/jayburd-dev-bank-sync-client-id/versions/"+o.clientIDVersion)
		if err != nil {
			return acquisition.ProviderCredentials{}, sandboxwebhook.ErrProbe
		}
		secret, err := reader.Read(ctx, "projects/"+projectNumber+"/secrets/jayburd-dev-bank-sync-client-secret/versions/"+o.clientSecretVersion)
		if err != nil {
			return acquisition.ProviderCredentials{}, sandboxwebhook.ErrProbe
		}
		return acquisition.ProviderCredentials{ClientID: string(id), Secret: string(secret)}, nil
	}
	credentials, err := bankcredential.NewResolver(pool, wrapper, static)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	evidence, err := cloudevidence.New("jayburd-bookkeeping-dev-jayburd-dev-acquisition-evidence", tokens, t)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	provider, err := acquisition.NewPlaidHTTP(t)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	probe, err := sandboxwebhook.New(provider, t, o.expires)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	runner, err := acquisition.New(pool, probe, credentials, evidence)
	if err != nil {
		return result{}, sandboxwebhook.ErrProbe
	}
	completed, err := runner.Run(ctx, o.scope, o.claim)
	return result{Requested: probe.Requested(), Completed: err == nil && completed.Status == "completed"}, err
}

// main emits only fixed errors; a failed/uncertain firing is never automatically retried.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if run(ctx, os.Args[1:], os.Stdout, managed) != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED: Sandbox webhook probe")
		os.Exit(1)
	}
}
