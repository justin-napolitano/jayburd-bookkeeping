// Command migrate checks metadata or applies embedded SQL through an explicit restricted IAM connector.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/migration"
)

// connectorFactory permits synthetic pre-I/O tests without a credential source.
type connectorFactory func(context.Context, clouddb.Config, string) (*pgxpool.Config, func(), error)

// run defaults to offline verification; only explicit bounded profiles reach IAM transport.
func run() error { return runWith(os.Args[1:], time.Now(), clouddb.ConnectorForEnvironment) }

// runWith parses a closed command and checks authority metadata before any connector construction.
func runWith(args []string, now time.Time, connect connectorFactory) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	action := flags.String("action", "verify", "offline verify, read-only status or up")
	instance := flags.String("instance", "", "explicit project:region:instance")
	environment := flags.String("environment", "sandbox", "sandbox or production")
	expiry := flags.String("approval-expires-at", "", "Production execution deadline in UTC")
	if flags.Parse(args) != nil || flags.NArg() != 0 || (*environment != "sandbox" && *environment != "production") {
		return migration.ErrMigration
	}
	if _, err := migration.Verify(migrations.Files); err != nil {
		return err
	}
	if *action == "verify" && *instance == "" && *expiry == "" {
		return nil
	}
	if (*action != "up" && *action != "status") || *instance == "" {
		return migration.ErrMigration
	}
	cfgInput := clouddb.Config{Instance: *instance, Component: "migrator"}
	if cfgInput.ValidateEnvironment(*environment) != nil || (*environment == "sandbox" && (strings.HasSuffix(*instance, ":jayburd-prod") || *expiry != "")) {
		return migration.ErrMigration
	}
	deadline := now.Add(10 * time.Minute)
	if *environment == "production" {
		approvedUntil, err := time.Parse("2006-01-02T15:04:05Z", *expiry)
		if err != nil || approvedUntil.Format("2006-01-02T15:04:05Z") != *expiry || !approvedUntil.After(now) || approvedUntil.After(now.Add(7*24*time.Hour)) {
			return migration.ErrMigration
		}
		if approvedUntil.Before(deadline) {
			deadline = approvedUntil
		}
	}
	if _, present := os.LookupEnv("JAYBURD_DATABASE_URL"); present {
		return migration.ErrMigration
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, timeout := context.WithDeadline(ctx, deadline)
	defer timeout()
	cfg, closeConnector, err := connect(ctx, cfgInput, *environment)
	if err != nil {
		return migration.ErrMigration
	}
	defer closeConnector()
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(30 * time.Minute)
	if *action == "status" {
		return migration.Status(ctx, db, migrations.Files)
	}
	return migration.Up(ctx, db, migrations.Files)
}

// main emits fixed status only; provider/SQL errors may carry protected payloads.
func main() {
	if run() != nil {
		fmt.Fprintln(os.Stderr, "migration rejected or failed")
		os.Exit(1)
	}
	fmt.Println("PASS: requested migration action completed")
}
