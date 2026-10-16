// Command worker explicitly composes one private managed acquisition worker.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/bankcredential"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/cloudevidence"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
)

// options contains public deployment identifiers only; there are no token/password flags.
type options struct {
	Worker                                                                                            workerruntime.Config
	MappingBucket, MappingPrefix, EvidenceBucket, ClientIDVersion, ClientSecretVersion, ProjectNumber string
}

// parse rejects unknown flags and suppresses argument contents in errors.
func parse(args []string) (options, error) {
	var o options
	f := flag.NewFlagSet("worker", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&o.Worker.Mode, "mode", "", "publisher or consumer")
	f.StringVar(&o.Worker.Listen, "listen", "127.0.0.1:8080", "explicit TCP listener")
	f.StringVar(&o.Worker.Instance, "cloud-sql-instance", "", "project:us-central1:instance")
	f.StringVar(&o.Worker.Caller.Audience, "caller-audience", "", "exact service audience")
	f.StringVar(&o.Worker.Caller.Email, "caller-email", "", "authorized workload service account")
	f.StringVar(&o.Worker.Caller.Subject, "caller-subject", "", "immutable numeric workload subject")
	f.StringVar(&o.Worker.BindingID, "binding-id", "", "enrolled publisher book binding")
	f.StringVar(&o.Worker.Queue.Location, "queue-location", "", "explicit queue location")
	f.StringVar(&o.Worker.Queue.Queue, "queue-name", "", "fixed queue name")
	f.StringVar(&o.Worker.Queue.ConsumerURL, "consumer-url", "", "fixed acquisition consumer URL")
	f.StringVar(&o.Worker.Queue.Audience, "consumer-audience", "", "exact consumer service audience")
	f.StringVar(&o.Worker.Queue.ServiceAccount, "task-service-account", "", "queue invocation service account")
	f.StringVar(&o.ProjectNumber, "project-number", "", "exact numeric Google project identifier for secret resources")
	f.StringVar(&o.MappingBucket, "credential-mapping-bucket", "", "private immutable credential mapping bucket")
	f.StringVar(&o.MappingPrefix, "credential-mapping-prefix", "", "reviewed credential mapping prefix")
	f.StringVar(&o.EvidenceBucket, "evidence-bucket", "", "private durable acquisition evidence bucket")
	f.StringVar(&o.ClientIDVersion, "plaid-client-id-version", "", "exact numeric GSM secret version resource")
	f.StringVar(&o.ClientSecretVersion, "plaid-client-secret-version", "", "exact numeric GSM secret version resource")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return options{}, workerruntime.ErrConfiguration
	}
	if o.Worker.Mode == "publisher" {
		o.Worker.Queue.Project = project(o.Worker.Instance)
	}
	if o.Worker.Validate() != nil {
		return options{}, workerruntime.ErrConfiguration
	}
	if o.Worker.Mode == "publisher" {
		if o.MappingBucket != "" || o.MappingPrefix != "" || o.EvidenceBucket != "" || o.ClientIDVersion != "" || o.ClientSecretVersion != "" || o.ProjectNumber != "" {
			return options{}, workerruntime.ErrConfiguration
		}
	} else if o.MappingBucket == "" || o.MappingPrefix == "" || o.EvidenceBucket == "" || o.ClientIDVersion == "" || o.ClientSecretVersion == "" || o.ProjectNumber == "" {
		return options{}, workerruntime.ErrConfiguration
	}
	if o.Worker.Mode == "consumer" {
		n, e := strconv.ParseUint(o.ProjectNumber, 10, 64)
		if e != nil || n == 0 || strconv.FormatUint(n, 10) != o.ProjectNumber {
			return options{}, workerruntime.ErrConfiguration
		}
	}
	p := project(o.Worker.Instance)
	expectedCaller := "jayburd-dev-task-invoker@" + p + ".iam.gserviceaccount.com"
	if o.Worker.Mode == "publisher" {
		expectedCaller = "jayburd-dev-scheduler@" + p + ".iam.gserviceaccount.com"
		if o.Worker.Queue.Queue != "jayburd-dev-acquisition" || o.Worker.Queue.Location != "us-central1" || o.Worker.Queue.ServiceAccount != "jayburd-dev-task-invoker@"+p+".iam.gserviceaccount.com" {
			return options{}, workerruntime.ErrConfiguration
		}
	} else if o.MappingBucket != p+"-jayburd-dev-credential-registry" || o.MappingPrefix != "provider-credentials-v1" || o.EvidenceBucket != p+"-jayburd-dev-acquisition-evidence" || !secretVersion(o.ClientIDVersion, o.ProjectNumber, "jayburd-dev-bank-sync-client-id") || !secretVersion(o.ClientSecretVersion, o.ProjectNumber, "jayburd-dev-bank-sync-client-secret") {
		return options{}, workerruntime.ErrConfiguration
	}
	if o.Worker.Caller.Email != expectedCaller {
		return options{}, workerruntime.ErrConfiguration
	}
	return o, nil
}

// secretVersion pins the reviewed secret name and an exact positive numeric version, never an alias.
func secretVersion(resource, projectID, name string) bool {
	prefix := "projects/" + projectID + "/secrets/" + name + "/versions/"
	if !strings.HasPrefix(resource, prefix) {
		return false
	}
	v := strings.TrimPrefix(resource, prefix)
	n, e := strconv.ParseUint(v, 10, 63)
	return e == nil && n > 0 && strconv.FormatUint(n, 10) == v
}

// project extracts only an already-validated Cloud SQL project identifier.
func project(instance string) string {
	for i, c := range instance {
		if c == ':' {
			return instance[:i]
		}
	}
	return ""
}

// managed owns resolver/evidence HTTP resources and composes only concrete reviewed adapters.
func managed(ctx context.Context, o options) (*workerruntime.Runtime, func(), error) {
	if o.Worker.Mode == "publisher" {
		r, e := workerruntime.Managed(ctx, o.Worker, workerruntime.ConsumerAdapters{})
		return r, func() {}, e
	}
	t := workerruntime.DirectTransport()
	cleanup := func() { t.CloseIdleConnections() }
	tokens, e := workerruntime.NewMetadataTokens(t)
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	reader, e := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: o.ProjectNumber, Tokens: tokens, Transport: t})
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	wrapper, e := bankcredential.NewKMS("projects/"+o.ProjectNumber+"/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials", tokens, t)
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	factory := func(db acquisition.Beginner) (acquisition.CredentialResolver, error) {
		return bankcredential.NewResolver(db, wrapper, func(ctx context.Context) (acquisition.ProviderCredentials, error) {
			var result acquisition.ProviderCredentials
			id, err := reader.Read(ctx, o.ClientIDVersion)
			if err != nil {
				return result, err
			}
			secret, err := reader.Read(ctx, o.ClientSecretVersion)
			if err != nil {
				return result, err
			}
			result.ClientID = string(id)
			result.Secret = string(secret)
			return result, nil
		})
	}
	evidence, e := cloudevidence.New(o.EvidenceBucket, tokens, t)
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	r, e := workerruntime.Managed(ctx, o.Worker, workerruntime.ConsumerAdapters{CredentialFactory: factory, Evidence: evidence})
	if e != nil {
		cleanup()
		return nil, nil, e
	}
	return r, cleanup, nil
}

// execute guarantees cleanup before main chooses a process exit code.
func execute(ctx context.Context, args []string) error {
	o, e := parse(args)
	if e != nil {
		return e
	}
	r, close, e := managed(ctx, o)
	if e != nil {
		return e
	}
	defer close()
	defer r.Close()
	return r.Run(ctx)
}

// main logs only fixed outcomes and never arguments, tokens, payloads or provider diagnostics.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if execute(ctx, os.Args[1:]) != nil {
		fmt.Fprintln(os.Stderr, "worker initialization or service failed")
		os.Exit(1)
	}
}
