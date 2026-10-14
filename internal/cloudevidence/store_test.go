package cloudevidence

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
)

// tokenStub supplies a fictional token without external identity discovery.
type tokenStub struct{ calls int }

func (t *tokenStub) AccessToken(ctx context.Context) (string, error) {
	t.calls++
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "synthetic", nil
}

// fakeObject models only immutable content and monotonically increasing generation.
type fakeObject struct {
	raw        []byte
	generation string
}

// fakeGCS intercepts every request and implements the reviewed generation preconditions.
type fakeGCS struct {
	mu                        sync.Mutex
	objects                   map[string]fakeObject
	sequence                  int
	lost, corrupt, raceDelete bool
	calls                     int
}

// response owns a bounded in-memory fixture body, never real provider data.
func response(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

// checksum mirrors GCS's wire encoding using an independently computed CRC32C.
func checksum(raw []byte) string {
	var out [4]byte
	binary.BigEndian.PutUint32(out[:], crc32.Checksum(raw, crc32.MakeTable(crc32.Castagnoli)))
	return base64.StdEncoding.EncodeToString(out[:])
}

// RoundTrip enforces fixed host/auth and storage query semantics in the synthetic service.
func (g *fakeGCS) RoundTrip(r *http.Request) (*http.Response, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if r.URL.Scheme != "https" || r.URL.Host != "storage.googleapis.com" || r.Header.Get("Authorization") != "Bearer synthetic" {
		return nil, errors.New("unexpected transport")
	}
	if r.Context().Err() != nil {
		return nil, r.Context().Err()
	}
	var name string
	if r.Method == http.MethodPost {
		name = r.URL.Query().Get("name")
		if r.URL.Query().Get("ifGenerationMatch") != "0" || r.URL.Query().Get("uploadType") != "media" {
			return nil, errors.New("missing create precondition")
		}
		if _, ok := g.objects[name]; ok {
			return response(412, "private conflict"), nil
		}
		raw, _ := io.ReadAll(r.Body)
		g.sequence++
		g.objects[name] = fakeObject{raw, strconv.Itoa(g.sequence)}
		if g.raceDelete && strings.HasPrefix(name, "raw/") {
			g.objects["deleted/"+strings.TrimPrefix(name, "raw/")] = fakeObject{[]byte("deleted-v1"), "99"}
			g.raceDelete = false
		}
		if g.lost {
			g.lost = false
			return nil, errors.New("private lost upload response")
		}
		return response(200, `{}`), nil
	}
	parts := strings.SplitN(r.URL.EscapedPath(), "/o/", 2)
	if len(parts) != 2 {
		return response(400, "invalid"), nil
	}
	name, _ = url.PathUnescape(parts[1])
	o, ok := g.objects[name]
	if !ok {
		return response(404, "private missing"), nil
	}
	if r.Method == http.MethodDelete {
		if r.URL.Query().Get("ifGenerationMatch") != o.generation {
			return response(412, "private conflict"), nil
		}
		delete(g.objects, name)
		return response(204, ""), nil
	}
	if r.URL.Query().Get("alt") == "media" {
		if r.URL.Query().Get("generation") != o.generation {
			return response(404, "private old generation"), nil
		}
		if g.corrupt {
			return response(200, "corrupt"), nil
		}
		return response(200, string(o.raw)), nil
	}
	m, _ := json.Marshal(metadata{Name: name, Generation: o.generation, Size: strconv.Itoa(len(o.raw)), CRC32C: checksum(o.raw)})
	return response(200, string(m)), nil
}

// fixture pins a project-owned bucket and a fictional full evidence scope.
func fixture(t *testing.T) (*Store, *fakeGCS, acquisition.EvidenceKey) {
	t.Helper()
	g := &fakeGCS{objects: map[string]fakeObject{}}
	s, e := New("synthetic-project-jayburd-dev-acquisition-evidence", &tokenStub{}, g)
	if e != nil {
		t.Fatal(e)
	}
	return s, g, acquisition.EvidenceKey{Scope: acquisition.Scope{EntityID: "00000000-0000-4000-8000-000000000001", BookID: "00000000-0000-4000-8000-000000000002", StreamID: "00000000-0000-4000-8000-000000000003"}, SetID: "00000000-0000-4000-8000-000000000004"}
}

// requireCode asserts fixed classifications rather than remote diagnostic content.
func requireCode(t *testing.T, e error, want string) {
	t.Helper()
	var f *acquisition.Failure
	if !errors.As(e, &f) || f.Code != want {
		t.Fatalf("want %s, got %v", want, e)
	}
}

// TestReplayAndScope proves exact immutable bytes and complete-key separation.
func TestReplayAndScope(t *testing.T) {
	s, g, k := fixture(t)
	ctx := context.Background()
	raw := []byte(`{"amount":9007199254740993.01}`)
	for i := 0; i < 2; i++ {
		if e := s.Put(ctx, k, raw); e != nil {
			t.Fatal(e)
		}
	}
	if g.sequence != 1 {
		t.Fatal("replay created another object")
	}
	got, found, e := s.Get(ctx, k)
	if e != nil || !found || string(got) != string(raw) {
		t.Fatal("exact bytes lost")
	}
	requireCode(t, s.Put(ctx, k, []byte("different")), "evidence_conflict")
	k.Scope.BookID = "00000000-0000-4000-8000-000000000005"
	if _, found, e = s.Get(ctx, k); e != nil || found {
		t.Fatal("cross-book evidence visible")
	}
}

// TestDeletionAndLateUpload requires tombstone denial before and after racing uploads.
func TestDeletionAndLateUpload(t *testing.T) {
	for _, race := range []bool{false, true} {
		t.Run(strconv.FormatBool(race), func(t *testing.T) {
			s, g, k := fixture(t)
			ctx := context.Background()
			g.raceDelete = race
			e := s.Put(ctx, k, []byte("fictional page"))
			if race {
				requireCode(t, e, "evidence_deleted")
			} else if e != nil {
				t.Fatal(e)
			}
			if e = s.Delete(ctx, k); e != nil {
				t.Fatal(e)
			}
			if e = s.Delete(ctx, k); e != nil {
				t.Fatal(e)
			}
			requireCode(t, s.Put(ctx, k, []byte("late")), "evidence_deleted")
			_, _, e = s.Get(ctx, k)
			requireCode(t, e, "evidence_deleted")
		})
	}
}

// TestUnknownUploadAndChecksum independently proves commit-loss replay and corruption rejection.
func TestUnknownUploadAndChecksum(t *testing.T) {
	s, g, k := fixture(t)
	g.lost = true
	raw := []byte("fictional payload")
	requireCode(t, s.Put(context.Background(), k, raw), "evidence_unavailable")
	if e := s.Put(context.Background(), k, raw); e != nil {
		t.Fatal(e)
	}
	if g.sequence != 1 {
		t.Fatal("unknown upload repeated")
	}
	g.corrupt = true
	_, _, e := s.Get(context.Background(), k)
	requireCode(t, e, "invalid_evidence")
}

// transportFunc makes negative HTTP responses explicit without opening sockets.
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestBoundsCancellationAndRedirect rejects invalid inputs before credentials and follows no redirects.
func TestBoundsCancellationAndRedirect(t *testing.T) {
	s, g, k := fixture(t)
	bad := k
	bad.Page = 10
	requireCode(t, s.Put(context.Background(), bad, []byte("x")), "invalid_evidence")
	requireCode(t, s.Put(context.Background(), k, make([]byte, acquisition.MaxRawBytes+1)), "invalid_evidence")
	if g.calls != 0 {
		t.Fatal("invalid input reached transport")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, e := s.Get(ctx, k)
	requireCode(t, e, "evidence_unavailable")
	if g.calls != 0 {
		t.Fatal("cancelled request reached transport")
	}
	calls := 0
	s.client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		r := response(302, "private")
		r.Header.Set("Location", "https://evil.invalid/private")
		return r, nil
	})
	_, _, e = s.Get(context.Background(), k)
	requireCode(t, e, "evidence_unavailable")
	if calls != 1 {
		t.Fatal("redirect followed")
	}
	if _, e = New("unrelated-bucket", &tokenStub{}, g); e == nil {
		t.Fatal("foreign bucket accepted")
	}
}
