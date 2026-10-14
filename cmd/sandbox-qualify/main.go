// Command sandbox-qualify is offline unless an operator explicitly enables a pipe-fed Sandbox diagnostic.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxqualification"
)

// main emits only the fixed receipt; no flag errors or upstream diagnostics reach stderr.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	receipt := execute(ctx, os.Args[1:], readPipe, func() http.RoundTripper {
		return &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, DisableKeepAlives: true}
	})
	_ = json.NewEncoder(os.Stdout).Encode(receipt)
	if receipt.Status == "partial" {
		os.Exit(3)
	}
	os.Exit(2)
}

// execute checks flags before reading a descriptor or constructing the network transport.
func execute(ctx context.Context, args []string, open func(int) (io.ReadCloser, error), network func() http.RoundTripper) sandboxqualification.Receipt {
	flags := flag.NewFlagSet("sandbox-qualify", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "offline", "")
	allow := flags.Bool("allow-sandbox", false, "")
	fd := flags.Int("input-fd", -1, "")
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return sandboxqualification.Blocked("invalid_invocation")
	}
	if *mode == "offline" {
		return sandboxqualification.Blocked("offline")
	}
	if *mode != "sandbox-sync" {
		return sandboxqualification.Blocked("invalid_invocation")
	}
	if !*allow {
		return sandboxqualification.Blocked("opt_in_required")
	}
	if *fd < 3 || *fd > 1024 || open == nil || network == nil {
		return sandboxqualification.Blocked("input_unavailable")
	}
	r, e := open(*fd)
	if e != nil || r == nil {
		return sandboxqualification.Blocked("input_unavailable")
	}
	defer r.Close()
	input, e := sandboxqualification.ReadInput(r)
	if e != nil {
		return sandboxqualification.Blocked("input_invalid")
	}
	return sandboxqualification.Run(ctx, true, input, network())
}

// readPipe accepts only inherited anonymous/named pipes and makes reads deadline-aware.
func readPipe(fd int) (io.ReadCloser, error) {
	if fd < 3 || fd > 1024 {
		return nil, os.ErrInvalid
	}
	var stat syscall.Stat_t
	if syscall.Fstat(fd, &stat) != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFIFO {
		return nil, os.ErrInvalid
	}
	if syscall.SetNonblock(fd, true) != nil {
		return nil, os.ErrInvalid
	}
	f := os.NewFile(uintptr(fd), "qualification-input")
	if f == nil {
		return nil, os.ErrInvalid
	}
	if f.SetReadDeadline(time.Now().Add(5*time.Second)) != nil {
		f.Close()
		return nil, os.ErrInvalid
	}
	return f, nil
}
