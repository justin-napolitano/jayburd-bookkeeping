package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const authoredFixture = `{"profile":"jayburd-sandbox-verification/1","fixture_id":"fresh-1","rows":[{"logical_id":"one","description":"Fictional one","revision":"1","current":true,"previous_revision":null,"source_state":"booked"}],"pending_links":[]}`
const providerPage = `{"added":[{"transaction_id":"Transaction-A","account_id":"Account-A","name":"Fictional one","amount":987654.321}],"modified":[],"removed":[],"has_more":false,"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`
const injectedCredential = `{"client_id":"private-client","secret":"private-secret","access_token":"access-sandbox-private"}`

// fakeTransport sees only synthetic private inputs; tests never contact a provider.
type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// invocation writes independently fixed synthetic expectations before any input/network work.
func invocation(t *testing.T) ([]string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.json")
	if os.WriteFile(path, []byte(authoredFixture), 0600) != nil {
		t.Fatal("fixture setup")
	}
	h := sha256.Sum256([]byte(authoredFixture))
	return []string{"-action=readback", "-fixture=" + path, "-fixture-sha256=" + hex.EncodeToString(h[:]), "-account-id=Account-A", "-credential-fd=3", "-expires-at=" + time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)}, path
}

// TestDefaultAndInvalidStayOffline rejects malformed scope and altered fixture before credential access.
func TestDefaultAndInvalidStayOffline(t *testing.T) {
	never := func(int) (io.ReadCloser, error) { t.Fatal("private input opened"); return nil, errCommand }
	transport := fakeTransport(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network"); return nil, nil })
	var out bytes.Buffer
	if execute(nil, &out, never, transport) != nil || out.String() != "{\"offline\":true}\n" {
		t.Fatal("default not inert")
	}
	for _, args := range [][]string{{"-action=readback"}, {"-secret=private"}, {"-endpoint=https://production.plaid.com"}, {"-credential-fd=0"}, {"trailing"}} {
		out.Reset()
		if execute(args, &out, never, transport) == nil || out.Len() != 0 {
			t.Fatal("unsafe invocation accepted")
		}
	}
	for _, change := range []string{"expired", "future", "account", "hash", "fixture", "duplicate_tags", "symlink", "existing_map"} {
		t.Run(change, func(t *testing.T) {
			args, path := invocation(t)
			switch change {
			case "expired":
				args[5] = "-expires-at=" + time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
			case "future":
				args[5] = "-expires-at=" + time.Now().Add(61*time.Minute).UTC().Format(time.RFC3339)
			case "account":
				args[3] = "-account-id=access-sandbox-private"
			case "hash":
				args[2] = "-fixture-sha256=" + strings.Repeat("0", 64)
			case "fixture":
				if os.WriteFile(path, []byte(authoredFixture+" "), 0600) != nil {
					t.Fatal("fixture mutation")
				}
			case "duplicate_tags":
				raw := strings.Replace(authoredFixture, `"source_state":"booked"`, `"source_state":"pending"`, 1)
				if os.WriteFile(path, []byte(raw), 0600) != nil {
					t.Fatal("fixture mutation")
				}
				h := sha256.Sum256([]byte(raw))
				args[2] = "-fixture-sha256=" + hex.EncodeToString(h[:])
			case "symlink":
				link := filepath.Join(t.TempDir(), "fixture.json")
				if os.Symlink(path, link) != nil {
					t.Fatal("symlink setup")
				}
				args[1] = "-fixture=" + link
			case "existing_map":
				if os.WriteFile(filepath.Join(filepath.Dir(path), "identity-map.json"), []byte("existing"), 0600) != nil {
					t.Fatal("map setup")
				}
			}
			var output bytes.Buffer
			if execute(args, &output, never, transport) == nil || output.Len() != 0 {
				t.Fatal("invalid local scope acquired authority")
			}
		})
	}
}

// TestPrivatePipeSuccessAndReplay proves private create-only output, closed input and no repeat request.
func TestPrivatePipeSuccessAndReplay(t *testing.T) {
	args, path := invocation(t)
	// Model an inherited blocking descriptor with one Go owner, not a second wrapper of os.Pipe.
	fds := make([]int, 2)
	if syscall.Pipe(fds) != nil {
		t.Fatal("pipe setup")
	}
	if _, e := syscall.Write(fds[1], []byte(injectedCredential)); e != nil {
		_ = syscall.Close(fds[0])
		_ = syscall.Close(fds[1])
		t.Fatal("pipe write")
	}
	_ = syscall.Close(fds[1])
	args[4] = "-credential-fd=" + strconv.Itoa(fds[0])
	opened, calls := 0, 0
	open := func(fd int) (io.ReadCloser, error) {
		opened++
		if fd != fds[0] {
			t.Fatal("wrong fd")
		}
		return privateInput(fd)
	}
	transport := fakeTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(providerPage))}, nil
	})
	var out bytes.Buffer
	if execute(args, &out, open, transport) != nil || out.String() != "{\"identity_map_ready\":true,\"count\":\"1\"}\n" || opened != 1 || calls != 1 {
		t.Fatal("private readback failed")
	}
	var stat syscall.Stat_t
	if syscall.Fstat(fds[0], &stat) == nil {
		t.Fatal("credential pipe left open")
	}
	mapPath := filepath.Join(filepath.Dir(path), "identity-map.json")
	info, e := os.Lstat(mapPath)
	if e != nil || info.Mode().Perm() != 0600 || !info.Mode().IsRegular() {
		t.Fatal("map not private")
	}
	raw, e := os.ReadFile(mapPath)
	want := `{"profile":"jayburd-sandbox-identity-map/1","fixture_id":"fresh-1","items":[{"logical_id":"one","transaction_id":"Transaction-A"}]}` + "\n"
	if e != nil || string(raw) != want {
		t.Fatal("map includes unexpected fields")
	}
	fixtureAfter, _ := os.ReadFile(path)
	if string(fixtureAfter) != authoredFixture {
		t.Fatal("expectations altered")
	}
	out.Reset()
	if execute(args, &out, open, transport) == nil || out.Len() != 0 || opened != 1 || calls != 1 {
		t.Fatal("repeat overwrote or repeated provider work")
	}
}

// TestRejectedTransportAndCredentialInputLeaveNoMap verifies bounded failures do not persist partial results.
func TestRejectedTransportAndCredentialInputLeaveNoMap(t *testing.T) {
	for _, name := range []string{"bad_credentials", "large_credentials", "failed_pipe", "foreign_page", "transport"} {
		t.Run(name, func(t *testing.T) {
			args, path := invocation(t)
			credential := injectedCredential
			if name == "bad_credentials" {
				credential = `{"secret":"private-diagnostic"}`
			}
			if name == "large_credentials" {
				credential = strings.Repeat("x", 17<<10)
			}
			calls := 0
			var out bytes.Buffer
			e := execute(args, &out, func(int) (io.ReadCloser, error) {
				if name == "failed_pipe" {
					return nil, errors.New("private-diagnostic")
				}
				return io.NopCloser(strings.NewReader(credential)), nil
			}, fakeTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if name == "transport" {
					return nil, errors.New("private-diagnostic")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(strings.Replace(providerPage, "Account-A", "Account-B", 1)))}, nil
			}))
			if e != errCommand || out.Len() != 0 {
				t.Fatal("failure leaked diagnostic or success")
			}
			if _, e := os.Lstat(filepath.Join(filepath.Dir(path), "identity-map.json")); !os.IsNotExist(e) {
				t.Fatal("failed request persisted output")
			}
			if name != "foreign_page" && name != "transport" && calls != 0 {
				t.Fatal("bad private input reached provider")
			}
		})
	}
}

// TestBlockedPipeExpiresAndCloses proves the operator deadline includes private injection, not just HTTP.
func TestBlockedPipeExpiresAndCloses(t *testing.T) {
	args, _ := invocation(t)
	expiry := time.Now().Add(time.Second)
	args[5] = "-expires-at=" + expiry.UTC().Format(time.RFC3339Nano)
	fds := make([]int, 2)
	if syscall.Pipe(fds) != nil {
		t.Fatal("pipe setup")
	}
	defer syscall.Close(fds[1])
	args[4] = "-credential-fd=" + strconv.Itoa(fds[0])
	var out bytes.Buffer
	start := time.Now()
	opened := false
	e := execute(args, &out, func(fd int) (io.ReadCloser, error) {
		input, err := privateInput(fd)
		opened = err == nil
		return input, err
	}, fakeTransport(func(*http.Request) (*http.Response, error) { t.Fatal("expired pipe reached provider"); return nil, nil }))
	if e != errCommand || !opened || time.Now().Before(expiry) || out.Len() != 0 || time.Since(start) > 3*time.Second {
		t.Fatal("private injection deadline not enforced")
	}
	var stat syscall.Stat_t
	if syscall.Fstat(fds[0], &stat) == nil {
		t.Fatal("expired pipe left open")
	}
}

// TestSavedCredentialsCannotBeRead proves a regular descriptor is rejected before bytes are consumed.
func TestSavedCredentialsCannotBeRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic")
	if os.WriteFile(path, []byte(injectedCredential), 0600) != nil {
		t.Fatal("input setup")
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal("input setup")
	}
	defer f.Close()
	if _, e := privateInput(int(f.Fd())); e != errCommand {
		t.Fatal("saved credentials accepted")
	}
}

// failedAcknowledgement models a lost local terminal reply after the complete private map is committed.
type failedAcknowledgement struct{}

func (failedAcknowledgement) Write([]byte) (int, error) { return 0, errors.New("synthetic lost reply") }

// TestLostAcknowledgementPreservesMapWithoutRepeatingProviderWork makes output uncertainty recoverable owner-side.
func TestLostAcknowledgementPreservesMapWithoutRepeatingProviderWork(t *testing.T) {
	args, path := invocation(t)
	calls := 0
	open := func(int) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(injectedCredential)), nil }
	transport := fakeTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(providerPage))}, nil
	})
	if execute(args, failedAcknowledgement{}, open, transport) == nil || calls != 1 {
		t.Fatal("lost reply not surfaced")
	}
	mapPath := filepath.Join(filepath.Dir(path), "identity-map.json")
	before, err := os.ReadFile(mapPath)
	if err != nil || len(before) == 0 {
		t.Fatal("complete private map lost")
	}
	var out bytes.Buffer
	if execute(args, &out, func(int) (io.ReadCloser, error) { t.Fatal("repeat reopened credential"); return nil, nil }, transport) == nil || out.Len() != 0 || calls != 1 {
		t.Fatal("repeat made new provider request")
	}
	after, _ := os.ReadFile(mapPath)
	if !bytes.Equal(before, after) {
		t.Fatal("complete private result changed")
	}
}
