// Command sandbox-identity-map is offline and projects owner-piped identities without reading credentials or making requests.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxidentity"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/sandboxverification"
)

var errCommand = errors.New("sandbox_identity_command_failed")

// privateInput accepts only an explicit pipe/socket, never stdin, a saved payload file or a credential path.
func privateInput(fd int) (io.ReadCloser, error) {
	f := os.NewFile(uintptr(fd), "private-sandbox-identity-input")
	if f == nil {
		return nil, errCommand
	}
	info, err := f.Stat()
	if err != nil || info.Mode()&(os.ModeNamedPipe|os.ModeSocket) == 0 {
		_ = f.Close()
		return nil, errCommand
	}
	return f, nil
}

// execute checks the pinned independently authored fixture before accepting ephemeral provider readback.
// Only identity-map.json is written, create-only; logs contain neither tags, IDs nor observed fields.
func execute(args []string, output io.Writer, open func(int) (io.ReadCloser, error)) error {
	f := flag.NewFlagSet("sandbox-identity-map", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	action := f.String("action", "verify", "offline or explicit project")
	fixturePath := f.String("fixture", "", "absolute reviewed fixture.json")
	fixtureHash := f.String("fixture-sha256", "", "reviewed raw-byte hash")
	account := f.String("account-id", "", "exact synthetic account identity")
	fd := f.Int("input-fd", -1, "explicit owner-private pipe")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return errCommand
	}
	if *action == "verify" && *fixturePath == "" && *fixtureHash == "" && *account == "" && *fd == -1 {
		_, err := io.WriteString(output, "{\"offline\":true}\n")
		return err
	}
	if *action != "project" || open == nil || *fd < 3 || *fd > 1024 || !sandboxidentity.ValidIdentity(*account) || !filepath.IsAbs(*fixturePath) || filepath.Clean(*fixturePath) != *fixturePath || filepath.Base(*fixturePath) != "fixture.json" || len(*fixtureHash) != 64 {
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
	if hex.EncodeToString(hash[:]) != *fixtureHash {
		return errCommand
	}
	in, err := open(*fd)
	if err != nil {
		return errCommand
	}
	defer in.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = in.Close() })
	defer stop()
	page, err := io.ReadAll(io.LimitReader(in, sandboxverification.MaxInputBytes+1))
	if err != nil || ctx.Err() != nil {
		return errCommand
	}
	identityMap, count, err := sandboxidentity.Project(fixture, page, *account)
	if err != nil || ctx.Err() != nil {
		return errCommand
	}
	mapPath := filepath.Join(filepath.Dir(*fixturePath), "identity-map.json")
	destination, err := os.OpenFile(mapPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errCommand
	}
	_, writeErr := destination.Write(identityMap)
	closeErr := destination.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(mapPath)
		return errCommand
	}
	_, err = fmt.Fprintf(output, "{\"identity_map_ready\":true,\"count\":%q}\n", strconv.Itoa(count))
	return err
}

// main emits one fixed failure and never echoes invocation, credentials, raw responses or map values.
func main() {
	if execute(os.Args[1:], os.Stdout, privateInput) != nil {
		fmt.Fprintln(os.Stderr, "sandbox_identity_command_failed")
		os.Exit(1)
	}
}
