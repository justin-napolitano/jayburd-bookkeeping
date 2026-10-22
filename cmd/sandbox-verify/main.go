// Command sandbox-verify defaults offline and executes only an explicitly manifested synthetic comparison.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/clouddb"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
)

var errCommand = errors.New("sandbox_verification_command_failed")
var errMismatch = errors.New("sandbox_verification_mismatch")
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// manifest contains reviewed metadata only; financial expectations occupy fixed sibling files.
type manifest struct {
	Profile           string `json:"profile"`
	Instance          string `json:"instance"`
	MandateID         string `json:"mandate_id"`
	MandateRevision   string `json:"mandate_revision"`
	FixtureSHA256     string `json:"fixture_sha256"`
	IdentityMapSHA256 string `json:"identity_map_sha256"`
}

// parseManifest rejects ambiguity before reading expectations or constructing metadata transport.
func parseManifest(raw []byte) (manifest, error) {
	var m manifest
	if len(raw) > 8192 || !utf8.Valid(raw) {
		return m, errCommand
	}
	fields, e := sandboxverification.StrictObject(raw)
	if e != nil || len(fields) != 6 {
		return m, errCommand
	}
	for _, k := range []string{"profile", "instance", "mandate_id", "mandate_revision", "fixture_sha256", "identity_map_sha256"} {
		if _, ok := fields[k]; !ok {
			return m, errCommand
		}
	}
	if json.Unmarshal(raw, &m) != nil || m.Profile != "jayburd-sandbox-verification-manifest/1" || !hashPattern.MatchString(m.FixtureSHA256) || !hashPattern.MatchString(m.IdentityMapSHA256) {
		return manifest{}, errCommand
	}
	c := clouddb.Config{Instance: m.Instance, Component: "verifier"}
	if c.Validate() != nil || c.Project() != "jayburd-bookkeeping-dev" || !sandboxverification.ValidRequest(sandboxverification.Request{MandateID: m.MandateID, Revision: m.MandateRevision, Fixture: []byte("{}"), IdentityMap: []byte("{}")}) {
		return manifest{}, errCommand
	}
	return m, nil
}

// readBounded opens a regular nonsymlink final path without blocking on a FIFO or reflecting diagnostics.
func readBounded(path string, max int) ([]byte, error) {
	f, e := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, errCommand
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > int64(max) {
		return nil, errCommand
	}
	raw, e := io.ReadAll(io.LimitReader(f, int64(max+1)))
	if e != nil || len(raw) > max {
		return nil, errCommand
	}
	return raw, nil
}

// connection owns one metadata-only verifier pool and its deterministic reverse-order cleanup.
type connection struct {
	db    sandboxverification.Beginner
	close func()
}
type connector func(context.Context, clouddb.Config) (connection, error)

// managed uses only the reviewed derived verifier IAM identity; construction does not accept a DSN.
func managed(ctx context.Context, c clouddb.Config) (connection, error) {
	cfg, closeConnector, e := clouddb.Connector(ctx, c)
	if e != nil {
		return connection{}, errCommand
	}
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		closeConnector()
		return connection{}, errCommand
	}
	return connection{pool, func() { pool.Close(); closeConnector() }}, nil
}

// run validates all bounded inputs before transport, never syncs, and emits only count/Boolean evidence.
func run(parent context.Context, args []string, out io.Writer, connect connector) error {
	f := flag.NewFlagSet("sandbox-verify", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	action := f.String("action", "verify", "offline verify or explicit compare")
	path := f.String("manifest", "", "absolute path to reviewed metadata manifest")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return errCommand
	}
	if *action == "verify" && *path == "" {
		_, e := io.WriteString(out, "{\"offline\":true}\n")
		return e
	}
	if *action != "compare" || !filepath.IsAbs(*path) || filepath.Clean(*path) != *path || connect == nil {
		return errCommand
	}
	raw, e := readBounded(*path, 8192)
	if e != nil {
		return errCommand
	}
	m, e := parseManifest(raw)
	if e != nil {
		return errCommand
	}
	dir := filepath.Dir(*path)
	fixture, e := readBounded(filepath.Join(dir, "fixture.json"), sandboxverification.MaxInputBytes)
	if e != nil {
		return errCommand
	}
	identities, e := readBounded(filepath.Join(dir, "identity-map.json"), sandboxverification.MaxInputBytes)
	if e != nil || len(fixture)+len(identities) > sandboxverification.MaxInputBytes {
		return errCommand
	}
	fh := sha256.Sum256(fixture)
	ih := sha256.Sum256(identities)
	if hex.EncodeToString(fh[:]) != m.FixtureSHA256 || hex.EncodeToString(ih[:]) != m.IdentityMapSHA256 {
		return errCommand
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	connection, e := connect(ctx, clouddb.Config{Instance: m.Instance, Component: "verifier"})
	if e != nil {
		return errCommand
	}
	if connection.close != nil {
		defer connection.close()
	}
	result, e := sandboxverification.Compare(ctx, connection.db, sandboxverification.Request{MandateID: m.MandateID, Revision: m.MandateRevision, Fixture: fixture, IdentityMap: identities})
	if e != nil {
		return errCommand
	}
	if json.NewEncoder(out).Encode(result) != nil {
		return errCommand
	}
	if !result.Equal {
		return errMismatch
	}
	return nil
}

// main never exposes file paths, fixture contents, database diagnostics or an unconfirmed prepared result.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if run(ctx, os.Args[1:], os.Stdout, managed) != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED: Sandbox verification")
		os.Exit(1)
	}
}
