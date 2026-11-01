// Command credential-check independently reads exact Production static versions without provider or SQL access.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/providersecrets"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/workerruntime"
)

// config contains only reviewed public metadata; callers cannot supply resource names or credential values.
type config struct{ project, number, idVersion, secretVersion, expiresAt string }

// validAt fences the one reviewed project and finite authority before any metadata authentication.
func (c config) validAt(now time.Time) bool {
	deadline, err := time.Parse("2006-01-02T15:04:05Z", c.expiresAt)
	return c.project == "jayburd-bookkeeping-dev" && c.number == "9088794002" && numeric(c.idVersion) && numeric(c.secretVersion) && err == nil && deadline.Format("2006-01-02T15:04:05Z") == c.expiresAt && deadline.After(now) && !deadline.After(now.Add(7*24*time.Hour))
}

// numeric rejects aliases, noncanonical decimal versions and overflow.
func numeric(value string) bool {
	n, err := strconv.ParseInt(value, 10, 64)
	return err == nil && n > 0 && strconv.FormatInt(n, 10) == value
}

// run keeps invalid invocation and dependency diagnostics fixed, including errors containing private bytes.
func run(ctx context.Context, args []string, out io.Writer, execute func(context.Context, config) error) int {
	var c config
	var check, allowed bool
	f := flag.NewFlagSet("credential-check", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&c.project, "project-id", "", "")
	f.StringVar(&c.number, "project-number", "", "")
	f.StringVar(&c.idVersion, "client-id-version", "", "")
	f.StringVar(&c.secretVersion, "client-secret-version", "", "")
	f.StringVar(&c.expiresAt, "approval-expires-at", "", "")
	f.BoolVar(&check, "check", false, "")
	f.BoolVar(&allowed, "execute", false, "")
	if f.Parse(args) != nil || f.NArg() != 0 || check == allowed || !c.validAt(time.Now()) {
		fmt.Fprintln(out, "BLOCKED: credential readback configuration")
		return 1
	}
	if check {
		fmt.Fprintln(out, "PASS: configuration only; no credential read")
		return 0
	}
	if execute == nil {
		fmt.Fprintln(out, "BLOCKED: credential readback configuration")
		return 1
	}
	deadline, _ := time.Parse("2006-01-02T15:04:05Z", c.expiresAt)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if ctx.Err() != nil || execute(ctx, c) != nil {
		fmt.Fprintln(out, "PARTIAL: credential readback unconfirmed")
		return 1
	}
	fmt.Fprintln(out, "PASS: exact Production credential versions readable; integrity checked; provider authentication untested")
	return 0
}

// payloadReader is the existing integrity-checking GCP adapter, never an owner-supplied value source.
type payloadReader interface {
	Read(context.Context, string) ([]byte, error)
}

// readback constructs only the two approved names and discards payloads without returning diagnostics or fingerprints.
func readback(ctx context.Context, c config, reader payloadReader) error {
	for _, target := range []struct{ name, version string }{{"client-id", c.idVersion}, {"client-secret", c.secretVersion}} {
		data, err := reader.Read(ctx, "projects/"+c.number+"/secrets/jayburd-prod-bank-sync-"+target.name+"/versions/"+target.version)
		clear(data)
		if err != nil {
			return err
		}
	}
	return nil
}

// managed authenticates through metadata only and reuses exact-name/CRC32C validation; no Infisical, provider or SQL client is constructed.
func managed(ctx context.Context, c config) error {
	if !c.validAt(time.Now()) {
		return errors.New("invalid readback configuration")
	}
	deadline, _ := time.Parse("2006-01-02T15:04:05Z", c.expiresAt)
	ctx, stop := context.WithDeadline(ctx, deadline)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	transport := workerruntime.DirectTransport()
	defer transport.CloseIdleConnections()
	tokens, err := workerruntime.NewMetadataTokens(transport)
	if err != nil {
		return err
	}
	reader, err := providersecrets.NewGCPReader(providersecrets.GCPConfig{ProjectID: c.number, Tokens: tokens, Transport: transport})
	if err != nil {
		return err
	}
	return readback(ctx, c, reader)
}

// main owns bounded process lifetime and emits no flags, tokens or SDK errors.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, managed))
}
