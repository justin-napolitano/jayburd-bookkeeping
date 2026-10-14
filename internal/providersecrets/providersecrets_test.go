package providersecrets

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/justin-napolitano/jayburd-bookkeeping/internal/acquisition"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/connections"
)

const project = "synthetic-secrets"
const sourceProject = "00000000-0000-4000-8000-000000000001"
const identity = "00000000-0000-4000-8000-000000000002"

// tokens supplies deliberately fictional managed identity material with no external discovery.
type tokens struct{}

// AccessToken returns a fictional OAuth token without external discovery.
func (tokens) AccessToken(context.Context) (string, error) { return "synthetic-oauth", nil }

// IdentityToken verifies the configured fictional audience before returning a test assertion.
func (tokens) IdentityToken(_ context.Context, a string) (string, error) {
	if a != identity {
		return "", errors.New("wrong audience")
	}
	return "synthetic-id-token", nil
}

// fakeCloud emulates immutable objects, source creates and destination propagation only in memory.
type fakeCloud struct {
	mu                             sync.Mutex
	objects                        map[string][]byte
	source                         map[string]string
	versions                       map[string][]byte
	posts                          int
	lostCreate, lostMapping, delay bool
	corruptCRC                     bool
	badName                        bool
	requests                       []string
}

// response returns bounded fictional HTTP data, never the network.
func response(code int, v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}
}

// RoundTrip implements only the exact pinned endpoints needed by the adapter proof.
func (f *fakeCloud) RoundTrip(q *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, q.Method+" "+q.URL.Host+q.URL.Path)
	if q.URL.Host == "storage.googleapis.com" {
		if q.Method == "POST" {
			key := q.URL.Query().Get("name")
			if q.URL.Query().Get("ifGenerationMatch") != "0" {
				return response(400, nil), nil
			}
			if _, ok := f.objects[key]; ok {
				return response(412, nil), nil
			}
			b, _ := io.ReadAll(q.Body)
			f.objects[key] = b
			if f.lostMapping && strings.Contains(key, "/mappings/") {
				f.lostMapping = false
				return nil, errors.New("synthetic lost ack")
			}
			return response(200, map[string]string{"name": key, "generation": "17"}), nil
		}
		parts := strings.SplitN(q.URL.Path, "/o/", 2)
		key, _ := url.PathUnescape(parts[1])
		b, ok := f.objects[key]
		if !ok {
			return response(404, nil), nil
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(b)))}, nil
	}
	if q.URL.Host == "us.infisical.com" {
		if strings.HasSuffix(q.URL.Path, "/login") {
			return response(200, map[string]any{"accessToken": "synthetic-inf-token", "expiresIn": 900, "tokenType": "Bearer"}), nil
		}
		key := strings.TrimPrefix(q.URL.Path, "/api/v4/secrets/")
		if q.Method == "POST" {
			f.posts++
			if _, ok := f.source[key]; ok {
				return response(409, nil), nil
			}
			var body struct{ SecretValue string }
			_ = json.NewDecoder(q.Body).Decode(&body)
			f.source[key] = body.SecretValue
			if !f.delay {
				f.versions["projects/"+project+"/secrets/jayburd-dev-bank-sync-"+key+"/versions/23"] = []byte(body.SecretValue)
			}
			if f.lostCreate {
				f.lostCreate = false
				return nil, errors.New("synthetic lost source ack")
			}
		}
		value, ok := f.source[key]
		if !ok {
			return response(404, nil), nil
		}
		return response(200, map[string]any{"secret": map[string]any{"id": "00000000-0000-4000-8000-000000000009", "workspace": sourceProject, "environment": "dev", "secretKey": key, "secretValue": value, "version": 7}}), nil
	}
	if q.URL.Host == "secretmanager.googleapis.com" {
		name := strings.TrimPrefix(q.URL.Path, "/v1/")
		if strings.HasSuffix(name, "/versions") {
			list := []map[string]string{}
			for k := range f.versions {
				if strings.HasPrefix(k, name+"/") {
					list = append(list, map[string]string{"name": k, "state": "ENABLED"})
				}
			}
			return response(200, map[string]any{"versions": list}), nil
		}
		name = strings.TrimSuffix(name, ":access")
		b, ok := f.versions[name]
		if !ok {
			return response(404, nil), nil
		}
		crc := crc32.Checksum(b, crc32.MakeTable(crc32.Castagnoli))
		if f.corruptCRC {
			crc++
		}
		returned := name
		if f.badName {
			returned += "9"
		}
		return response(200, map[string]any{"name": returned, "payload": map[string]string{"data": base64.StdEncoding.EncodeToString(b), "dataCrc32c": fmt.Sprint(crc)}}), nil
	}
	return nil, errors.New("unexpected synthetic host")
}

// setup configures all finite slots without any actual credential or endpoint call.
func setup(t *testing.T) (*Store, *Resolver, *fakeCloud) {
	t.Helper()
	f := &fakeCloud{objects: map[string][]byte{}, source: map[string]string{}, versions: map[string][]byte{}}
	g, e := NewGCSRegistry(GCSConfig{"synthetic-registry", "credentials", tokens{}, f})
	if e != nil {
		t.Fatal(e)
	}
	r, e := NewGCPReader(GCPConfig{project, tokens{}, f})
	if e != nil {
		t.Fatal(e)
	}
	slots := []Slot{}
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("slot-%02d", i)
		slots = append(slots, Slot{id, id, "projects/" + project + "/secrets/jayburd-dev-bank-sync-" + id})
	}
	s, e := NewStore(StoreConfig{"https://us.infisical.com", sourceProject, "dev", identity, tokens{}, f, slots}, g, r)
	if e != nil {
		t.Fatal(e)
	}
	id := "projects/" + project + "/secrets/plaid-client-id/versions/11"
	secret := "projects/" + project + "/secrets/plaid-client-secret/versions/19"
	f.versions[id] = []byte("synthetic-client")
	f.versions[secret] = []byte("synthetic-secret")
	resolver, e := NewResolver(r, g, ResolverConfig{id, secret})
	if e != nil {
		t.Fatal(e)
	}
	return s, resolver, f
}

// reference uses fixed fictional SQL metadata, independent of physical version ordinals.
func reference(n int) connections.CredentialReference {
	return connections.CredentialReference{ReferenceID: fmt.Sprintf("00000000-0000-4000-8000-%012d", n), Version: "4", ItemID: "synthetic-item", Environment: "sandbox"}
}

// acq converts only the identical reference metadata between the established interfaces.
func acq(r connections.CredentialReference) acquisition.CredentialReference {
	return acquisition.CredentialReference{ReferenceID: r.ReferenceID, Version: r.Version, ItemID: r.ItemID, Environment: r.Environment}
}

// TestExactPropagationAndRuntime proves independent version ordinals and immutable runtime readback.
func TestExactPropagationAndRuntime(t *testing.T) {
	s, r, f := setup(t)
	ref := reference(40)
	if e := s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); e != nil {
		t.Fatal(e)
	}
	m, e := s.registry.Lookup(context.Background(), ref)
	if e != nil {
		t.Fatal(e)
	}
	if m.InfisicalVersion != "7" || !strings.HasSuffix(m.GCPVersion, "/23") {
		t.Fatal("version ordinals assumed")
	}
	v, e := r.Resolve(context.Background(), acq(ref))
	if e != nil {
		t.Fatal(e)
	}
	if v.AccessToken != "synthetic-access" || v.ClientID != "synthetic-client" {
		t.Fatal("wrong synthetic values")
	}
	if e = s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); e != nil {
		t.Fatal(e)
	}
	if f.posts != 1 {
		t.Fatal("source write repeated")
	}
	for _, b := range f.objects {
		if strings.Contains(string(b), "synthetic-access") {
			t.Fatal("credential leaked to registry")
		}
	}
}

// TestUnknownSourceAckRecoversWithoutRepeat recovers an acknowledged-lost source write without another POST.
func TestUnknownSourceAckRecoversWithoutRepeat(t *testing.T) {
	s, _, f := setup(t)
	f.lostCreate = true
	ref := reference(41)
	if e := s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); code(e) != "secret_create_unknown" {
		t.Fatal(e)
	}
	s2 := *s
	s2.registry = &GCSRegistry{s.registry.config, client(f)}
	if _, e := s2.Read(context.Background(), ref); e != nil {
		t.Fatal(e)
	}
	if f.posts != 1 {
		t.Fatal("source request repeated after process restart")
	}
}

// TestUnknownIntentAndMissingSourceNeverRecreate preserves uncertainty when durable intent exists without source evidence.
func TestUnknownIntentAndMissingSourceNeverRecreate(t *testing.T) {
	s, _, f := setup(t)
	ref := reference(42)
	raw, _ := json.Marshal(envelope{"jayburd-provider-credential-v1", ref, "synthetic-access"})
	sum := sha256.Sum256(raw)
	if _, _, e := s.reserve(context.Background(), ref, hex.EncodeToString(sum[:])); e != nil {
		t.Fatal(e)
	}
	if e := s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); e == nil {
		t.Fatal("missing source treated as completed")
	}
	if f.posts != 0 {
		t.Fatal("existing create intent replayed")
	}
}

// TestPropagationPendingAndMappingLostAck reconciles delayed sync and a lost mapping acknowledgement.
func TestPropagationPendingAndMappingLostAck(t *testing.T) {
	s, _, f := setup(t)
	f.delay = true
	ref := reference(43)
	if e := s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); code(e) != "secret_propagation_pending" {
		t.Fatal(e)
	}
	f.versions["projects/"+project+"/secrets/jayburd-dev-bank-sync-slot-00/versions/9"] = []byte(f.source["slot-00"])
	f.lostMapping = true
	if _, e := s.Read(context.Background(), ref); e != nil {
		t.Fatal(e)
	}
	if f.posts != 1 {
		t.Fatal("propagation repeated source mutation")
	}
}

// TestPoolExhaustionAndConflictingPayload prevents slot reuse and credential replacement.
func TestPoolExhaustionAndConflictingPayload(t *testing.T) {
	s, _, f := setup(t)
	for i := 0; i < 8; i++ {
		if e := s.Put(context.Background(), reference(50+i), connections.NewSecret("synthetic-access")); e != nil {
			t.Fatal(e)
		}
	}
	if e := s.Put(context.Background(), reference(90), connections.NewSecret("synthetic-access")); code(e) != "secret_pool_exhausted" {
		t.Fatal(e)
	}
	if e := s.Put(context.Background(), reference(50), connections.NewSecret("different-synthetic-access")); code(e) != "secret_reservation_conflict" {
		t.Fatal(e)
	}
	if f.posts != 8 {
		t.Fatal("capacity or immutability escaped")
	}
}

// TestIntegrityAndReferenceDenials rejects corrupted payloads, wrong identity and resource aliases.
func TestIntegrityAndReferenceDenials(t *testing.T) {
	s, r, f := setup(t)
	ref := reference(60)
	if e := s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); e != nil {
		t.Fatal(e)
	}
	f.corruptCRC = true
	if _, e := r.Resolve(context.Background(), acq(ref)); code(e) != "secret_integrity_failed" {
		t.Fatal(e)
	}
	f.corruptCRC = false
	f.badName = true
	if _, e := r.Resolve(context.Background(), acq(ref)); code(e) != "secret_integrity_failed" {
		t.Fatal(e)
	}
	f.badName = false
	wrong := ref
	wrong.ItemID = "different-item"
	if _, e := r.Resolve(context.Background(), acq(wrong)); e == nil {
		t.Fatal("wrong Item accepted")
	}
	if _, e := r.reader.Read(context.Background(), "projects/"+project+"/secrets/x/versions/latest"); code(e) != "secret_reference_invalid" {
		t.Fatal(e)
	}
	if _, e := r.reader.Read(context.Background(), "projects/other-project/secrets/x/versions/1"); code(e) != "secret_reference_invalid" {
		t.Fatal(e)
	}
}

// TestRedactedPreflightAndStrictJSON keeps reports redacted and rejects ambiguous JSON.
func TestRedactedPreflightAndStrictJSON(t *testing.T) {
	_, r, _ := setup(t)
	v := r.Preflight(context.Background(), acq(reference(70)))
	b, _ := json.Marshal(v)
	if string(b) != "{\"status\":\"PARTIAL\",\"code\":\"runtime_reference_unavailable\"}" {
		t.Fatal("unexpected preflight")
	}
	var x any
	for _, b := range []string{`{"secret":"one","secret":"two"}`, `{} {}`, `null`} {
		if strict([]byte(b), &x) == nil {
			t.Fatal("ambiguous JSON accepted")
		}
	}
}

// transportFunc isolates response corruption and endpoint assertions without listening sockets.
type transportFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes only the synthetic test closure.
func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestHTTPBoundsRedirectAndRedaction rejects redirects, excessive bodies and vendor diagnostics.
func TestHTTPBoundsRedirectAndRedaction(t *testing.T) {
	for _, status := range []int{302, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			_, _, e := request(context.Background(), client(transportFunc(func(*http.Request) (*http.Response, error) {
				return response(status, map[string]string{"secret": "fictional-never-echo"}), nil
			})), "GET", "https://secretmanager.googleapis.com/v1/test", "synthetic-token", "", nil)
			if e == nil || strings.Contains(fmt.Sprint(e), "fictional-never-echo") {
				t.Fatal("unredacted or accepted response")
			}
		})
	}
	_, _, e := request(context.Background(), client(transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat("x", maxResponse+1)))}, nil
	})), "GET", "https://secretmanager.googleapis.com/v1/test", "synthetic-token", "", nil)
	if code(e) != "secret_response_invalid" {
		t.Fatal(e)
	}
}

// TestDestinationMustStartEmptyAndChangedSourceRejected blocks unrelated destination history and changed source bytes.
func TestDestinationMustStartEmptyAndChangedSourceRejected(t *testing.T) {
	s, _, f := setup(t)
	ref := reference(71)
	f.versions["projects/"+project+"/secrets/jayburd-dev-bank-sync-slot-00/versions/1"] = []byte("prior-unrelated-synthetic")
	if e := s.Put(context.Background(), ref, connections.NewSecret("synthetic-access")); code(e) != "secret_destination_not_empty" {
		t.Fatal(e)
	}
	if f.posts != 0 {
		t.Fatal("nonempty destination triggered source write")
	}
	s, _, f = setup(t)
	f.delay = true
	ref = reference(72)
	_ = s.Put(context.Background(), ref, connections.NewSecret("synthetic-access"))
	f.source["slot-00"] = strings.ReplaceAll(f.source["slot-00"], "synthetic-access", "changed-access")
	if _, e := s.Read(context.Background(), ref); code(e) != "secret_integrity_failed" {
		t.Fatal(e)
	}
}

// TestConcurrentSameReferenceCreatesOnce proves concurrent reservation grants one source creation only.
func TestConcurrentSameReferenceCreatesOnce(t *testing.T) {
	s, _, f := setup(t)
	ref := reference(73)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Put(context.Background(), ref, connections.NewSecret("synthetic-access"))
		}()
	}
	wg.Wait()
	if _, e := s.Read(context.Background(), ref); e != nil {
		t.Fatal(e)
	}
	if f.posts != 1 {
		t.Fatal("concurrent replay repeated source create")
	}
}

// TestCanonicalNumericProjectConfiguration binds numeric project metadata without accepting another alias.
func TestCanonicalNumericProjectConfiguration(t *testing.T) {
	r, e := NewGCPReader(GCPConfig{"123456789012", tokens{}, transportFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("must not call") })})
	if e != nil {
		t.Fatal(e)
	}
	if !r.resource("projects/123456789012/secrets/slot/versions/3", true) || r.resource("projects/synthetic-secrets/secrets/slot/versions/3", true) {
		t.Fatal("project alias escaped exact scope")
	}
}
