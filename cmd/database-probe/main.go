// Command database-probe explicitly verifies the API IAM boundary; default execution is offline.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/databaseprobe"
)

// run refuses ambiguous actions and never prints provider errors or input values.
func run(args []string) error {
	f := flag.NewFlagSet("database-probe", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	action := f.String("action", "verify", "offline verify or explicit probe")
	instance := f.String("instance", "", "explicit Cloud SQL instance")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return databaseprobe.ErrProbe
	}
	if *action == "verify" && *instance == "" {
		return nil
	}
	c := clouddb.Config{Instance: *instance, Component: "api"}
	if *action != "probe" || c.Validate() != nil {
		return databaseprobe.ErrProbe
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	cfg, closeConnector, err := clouddb.Connector(ctx, c)
	if err != nil {
		return databaseprobe.ErrProbe
	}
	defer closeConnector()
	db := stdlib.OpenDB(*cfg.ConnConfig)
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	return databaseprobe.Run(ctx, db, "jayburd-dev-api@"+c.Project()+".iam")
}

// main publishes fixed outcomes only; successful offline verification is not hosted evidence.
func main() {
	if run(os.Args[1:]) != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED: database permission probe")
		os.Exit(1)
	}
	fmt.Println("PASS: requested database probe action completed")
}
