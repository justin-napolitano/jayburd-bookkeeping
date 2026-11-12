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
	"time"

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
	BankKey, LinkKey                                                                                  string
}

// parse rejects unknown flags and suppresses argument contents in errors.
func parse(args []string) (options, error) {
	var o options
	f := flag.NewFlagSet("worker", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&o.Worker.Environment, "provider-environment", "", "explicit sandbox or production; legacy omission is Sandbox")
	f.StringVar(&o.Worker.ApprovalExpiresAt, "provider-approval-expires-at", "", "finite Production runtime approval deadline")
	f.StringVar(&o.BankKey, "bank-credential-key", "", "approved Production bank cryptoKey resource")
	f.StringVar(&o.LinkKey, "link-handoff-key", "", "approved Production Link cryptoKey resource for scope matching")
	f.StringVar(&o.Worker.Mode, "mode", "", "publisher or consumer")
	f.BoolVar(&o.Worker.Categories, "automatic-categories", false, "explicit category-aware consumer publication; publisher cannot enable")
	f.StringVar(&o.Worker.Listen, "listen", "127.0.0.1:8080", "explicit TCP listener")
	f.StringVar(&o.Worker.Instance, "cloud-sql-instance", "", "project:us-central1:instance")
	f.StringVar(&o.Worker.Caller.Audience, "caller-audience", "", "exact service audience")
	f.StringVar(&o.Worker.Caller.Email, "caller-email", "", "authorized workload service account")
	f.StringVar(&o.Worker.Caller.Subject, "caller-subject", "", "immutable numeric workload subject")
	f.StringVar(&o.Worker.BindingID, "binding-id", "", "enrolled publisher book binding")
	f.StringVar(&o.Worker.PublisherPolicy, "publisher-policy", "", "binding-v1 or shared-v1")
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
		if o.MappingBucket != "" || o.MappingPrefix != "" || o.EvidenceBucket != "" || o.ClientIDVersion != "" || o.ClientSecretVersion != "" || o.ProjectNumber != "" || o.BankKey != "" || o.LinkKey != "" {
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
	prefix := "jayburd-dev"
	if o.Worker.ProviderEnvironment() == "production" {
		prefix = "jayburd-prod"
	}
	expectedCaller := prefix + "-task-invoker@" + p + ".iam.gserviceaccount.com"
	if o.Worker.Mode == "publisher" {
		expectedCaller = prefix + "-scheduler@" + p + ".iam.gserviceaccount.com"
		if o.Worker.Queue.Queue != prefix+"-acquisition" || o.Worker.Queue.Location != "us-central1" || o.Worker.Queue.ServiceAccount != prefix+"-task-invoker@"+p+".iam.gserviceaccount.com" {
			return options{}, workerruntime.ErrConfiguration
		}
	} else if o.MappingBucket != p+"-"+prefix+"-credential-registry" || o.MappingPrefix != "provider-credentials-v1" || o.EvidenceBucket != p+"-"+prefix+"-acquisition-evidence" || !secretVersion(o.ClientIDVersion, o.ProjectNumber, prefix+"-bank-sync-client-id") || !secretVersion(o.ClientSecretVersion, o.ProjectNumber, prefix+"-bank-sync-client-secret") {
		return options{}, workerruntime.ErrConfiguration
	}
	if o.Worker.Mode == "consumer" {
		if o.Worker.ProviderEnvironment() == "production" {
			policy := bankcredential.ProductionPolicy{BankKey: o.BankKey, LinkKey: o.LinkKey}
			// Shared hosting requires its exact reviewed project pair, while instance isolation remains independently validated.
			sharedMismatch := (p == "jayburd-bookkeeping-dev") != (o.ProjectNumber == "9088794002")
			if !policy.Valid() || sharedMismatch || !strings.HasPrefix(o.BankKey, "projects/"+p+"/") || !strings.HasPrefix(o.LinkKey, "projects/"+p+"/") {
				return options{}, workerruntime.ErrConfiguration
			}
		} else if o.BankKey != "" || o.LinkKey != "" {
			return options{}, workerruntime.ErrConfiguration
		}
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
	if !o.Worker.ApprovedAt(time.Now()) {
		return nil, nil, workerruntime.ErrUnavailable
	}
	if o.Worker.Mode == "publisher" {
		r, e := workerruntime.Managed(ctx, o.Worker, workerruntime.ConsumerAdapters{})
		return r, func() {}, e
	}
	t := workerruntime.DirectTransport()
	dependency := o.Worker.ApprovedTransport(t)
	cleanup := func() { t.CloseIdleConnections() }
	tokens, e := workerruntime.NewMetadataTokens(dependency)
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	reader, e := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: o.ProjectNumber, Tokens: tokens, Transport: dependency})
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	wrapper, e := bankcredential.NewKMS("projects/"+project(o.Worker.Instance)+"/locations/us-central1/keyRings/jayburd-dev/cryptoKeys/bank-credentials", tokens, t)
	if o.Worker.ProviderEnvironment() == "production" {
		wrapper, e = bankcredential.NewProductionKMS(o.BankKey, tokens, dependency)
	}
	if e != nil {
		cleanup()
		return nil, nil, workerruntime.ErrConfiguration
	}
	factory := func(db acquisition.Beginner) (acquisition.CredentialResolver, error) {
		static := func(ctx context.Context) (acquisition.ProviderCredentials, error) {
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
		}
		if o.Worker.ProviderEnvironment() == "production" {
			return bankcredential.NewProductionResolver(db, wrapper, bankcredential.ProductionPolicy{BankKey: o.BankKey, LinkKey: o.LinkKey}, static)
		}
		return bankcredential.NewResolver(db, wrapper, static)
	}
	evidence, e := cloudevidence.New(o.EvidenceBucket, tokens, dependency)
	if o.Worker.ProviderEnvironment() == "production" {
		evidence, e = cloudevidence.NewProduction(project(o.Worker.Instance), o.EvidenceBucket, tokens, dependency)
	}
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
