// Command sandbox-identity-readback explicitly obtains only a private synthetic identity map.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxidentity"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
)

var errCommand = errors.New("sandbox_identity_readback_failed")

// privateInput accepts only owner-injected pipes/sockets, never saved credential files or standard streams.
func privateInput(fd int) (io.ReadCloser, error) {
	f, err := sandboxidentity.OpenPrivateInput(fd)
	if err != nil {
		return nil, errCommand
	}
	return f, nil
}

// execute validates independent expectations before private injection and persists only a complete identity map.
func execute(args []string, output io.Writer, open func(int) (io.ReadCloser, error), transport http.RoundTripper) error {
	f := flag.NewFlagSet("sandbox-identity-readback", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	action := f.String("action", "verify", "offline or explicit readback")
	fixturePath := f.String("fixture", "", "absolute reviewed fixture.json")
	fixtureHash := f.String("fixture-sha256", "", "reviewed independent raw-byte hash")
	account := f.String("account-id", "", "exact synthetic account identity")
	expires := f.String("expires-at", "", "explicit approved deadline, at most60minutes")
	fd := f.Int("credential-fd", -1, "owner-private credential pipe descriptor")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return errCommand
	}
	if *action == "verify" && *fixturePath == "" && *fixtureHash == "" && *account == "" && *expires == "" && *fd == -1 {
		_, err := io.WriteString(output, "{\"offline\":true}\n")
		return err
	}
	expiry, err := time.Parse(time.RFC3339Nano, *expires)
	if err != nil || !sandboxidentity.ValidReadbackExpiry(expiry, time.Now()) || *action != "readback" || open == nil || transport == nil || *fd < 3 || *fd > 1024 || !sandboxidentity.ValidIdentity(*account) || !filepath.IsAbs(*fixturePath) || filepath.Clean(*fixturePath) != *fixturePath || filepath.Base(*fixturePath) != "fixture.json" || len(*fixtureHash) != 64 {
		return errCommand
	}
	expected, err := hex.DecodeString(*fixtureHash)
	if err != nil || hex.EncodeToString(expected) != *fixtureHash {
		return errCommand
	}
	file, err := os.OpenFile(*fixturePath, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return errCommand
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > sandboxverification.MaxInputBytes {
		return errCommand
	}
	fixture, err := io.ReadAll(io.LimitReader(file, sandboxverification.MaxInputBytes+1))
	if err != nil || len(fixture) > sandboxverification.MaxInputBytes {
		return errCommand
	}
	hash := sha256.Sum256(fixture)
	if hex.EncodeToString(hash[:]) != *fixtureHash || sandboxidentity.ValidateReadbackFixture(fixture) != nil {
		return errCommand
	}
	mapPath := filepath.Join(filepath.Dir(*fixturePath), "identity-map.json")
	if _, err := os.Lstat(mapPath); !os.IsNotExist(err) {
		return errCommand
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ctx, end := context.WithDeadline(ctx, expiry)
	defer end()
	if ctx.Err() != nil {
		return errCommand
	}
	in, err := open(*fd)
	if err != nil {
		return errCommand
	}
	defer in.Close()
	stop := context.AfterFunc(ctx, func() { _ = in.Close() })
	defer stop()
	credential, err := io.ReadAll(io.LimitReader(in, sandboxidentity.MaxCredentialBytes+1))
	if err != nil || ctx.Err() != nil || len(credential) > sandboxidentity.MaxCredentialBytes {
		return errCommand
	}
	// Close the injection pipe before transport; no raw response or credential file is created.
	_ = in.Close()
	identityMap, count, err := sandboxidentity.Readback(ctx, fixture, credential, *account, expiry, transport)
	if err != nil || ctx.Err() != nil || !expiry.After(time.Now()) {
		return errCommand
	}
	destination, err := os.OpenFile(mapPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errCommand
	}
	_, writeErr := destination.Write(identityMap)
	closeErr := destination.Close()
	if writeErr != nil || closeErr != nil || ctx.Err() != nil || !expiry.After(time.Now()) {
		_ = os.Remove(mapPath)
		return errCommand
	}
	_, err = fmt.Fprintf(output, "{\"identity_map_ready\":true,\"count\":%q}\n", strconv.Itoa(count))
	return err
}

// main uses standard TLS with no proxy/discovery and emits fixed diagnostics only.
func main() {
	transport := &http.Transport{Proxy: nil, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second, MaxResponseHeaderBytes: 16 << 10, MaxConnsPerHost: 1, DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	if execute(os.Args[1:], os.Stdout, privateInput, transport) != nil {
		fmt.Fprintln(os.Stderr, "sandbox_identity_readback_failed")
		os.Exit(1)
	}
}
