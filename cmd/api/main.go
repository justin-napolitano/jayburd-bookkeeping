// Command api is health-only unless the owner explicitly selects managed mode.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/server"
)

// main emits fixed operational status only; never print flags, tokens or DSNs.
func main() {
	mode := flag.String("mode", "health", "health or explicitly provisioned managed mode")
	listen := flag.String("listen", "127.0.0.1:8080", "explicit listening address")
	origin := flag.String("origin", "", "exact HTTPS frontend origin; loopback HTTP allowed for local development")
	project := flag.String("project", "", "explicit Identity Platform project")
	flag.Parse()
	c := server.Config{Mode: *mode, Listen: *listen, Origin: *origin, Project: *project}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var d server.Dependencies
	if c.Mode == "managed" {
		c.DatabaseURL = os.Getenv("JAYBURD_DATABASE_URL")
		var err error
		d, err = server.Managed(ctx, c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "server initialization failed")
			os.Exit(1)
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
