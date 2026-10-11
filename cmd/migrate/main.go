// Command migrate applies only embedded, checksummed SQL through an explicit IAM connector.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/justin-napolitano/jayburd-bookkeeping/db/migrations"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/migration"
)

// run defaults to offline artifact verification; applying requires explicit instance and action.
func run() error {
	action := flag.String("action", "verify", "verify or up; no down, force or arbitrary target")
	instance := flag.String("instance", "", "explicit project:region:instance for IAM migration")
	flag.Parse()
	if flag.NArg() != 0 {
		return migration.ErrMigration
	}
	if _, err := migration.Verify(migrations.Files); err != nil {
		return err
	}
	if *action == "verify" && *instance == "" {
		return nil
	}
	if *action != "up" || *instance == "" {
		return migration.ErrMigration
	}
	if _, present := os.LookupEnv("JAYBURD_DATABASE_URL"); present {
		return migration.ErrMigration
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 10*time.Minute)
	defer timeout()
	cfg, closeConnector, err := clouddb.Connector(ctx, clouddb.Config{Instance: *instance, Component: "migrator"})
	if err != nil {
		return migration.ErrMigration
	}
	defer closeConnector()
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(30 * time.Minute)
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
