package providersecrets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// deliveryCloud accepts only the narrow delivery protocol, with in-memory durable intents.
type deliveryCloud struct {
	storage fakeCloud
	mu      sync.Mutex
	writes  int
	failure string
}

// RoundTrip rejects destination reads/listing and verifies source selectors and payload integrity.
func (f *deliveryCloud) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "storage.googleapis.com" {
		return f.storage.RoundTrip(r)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Host == "us.infisical.com" {
		if strings.HasSuffix(r.URL.Path, "/login") {
			return response(200, map[string]any{"accessToken": "synthetic-auth", "tokenType": "Bearer", "expiresIn": 900}), nil
		}
		q := r.URL.Query()
		folder := "/"
		if strings.HasPrefix(r.URL.Path, "/api/v4/secrets/slot-0") {
			folder = "/bank-sync"
		}
		if r.Method != "GET" || q.Get("includeImports") != "false" || q.Get("expandSecretReferences") != "false" || q.Get("version") != "7" || q.Get("secretPath") != folder || q.Get("projectId") != sourceProject || q.Get("environment") != "dev" {
			return response(400, nil), nil
		}
		if f.failure == "denied" {
			return response(403, map[string]string{"message": "private-vendor-diagnostic"}), nil
		}
		key := strings.TrimPrefix(r.URL.Path, "/api/v4/secrets/")
		if f.failure == "source" {
			key = "WRONG"
		}
		fields := map[string]any{"id": "00000000-0000-4000-8000-000000000009", "workspace": sourceProject, "environment": "dev", "secretKey": key, "secretPath": folder, "version": 7, "secretValue": "fictional-canary"}
		switch f.failure {
		case "wrong-path":
			fields["secretPath"] = "/other"
		case "wrong-project":
			fields["workspace"] = identity
		case "wrong-version":
			fields["version"] = 8
		}
		return response(200, map[string]any{"secret": fields}), nil
	}
	if r.URL.Host != "secretmanager.googleapis.com" || r.Method != "POST" || (!strings.HasSuffix(r.URL.Path, "/secrets/jayburd-dev-bank-sync-client-id:addVersion") && !strings.HasSuffix(r.URL.Path, "/secrets/jayburd-dev-bank-sync-canary:addVersion") && !strings.Contains(r.URL.Path, "/secrets/jayburd-dev-bank-sync-slot-0")) {
		return nil, errors.New("forbidden endpoint")
	}
	var body struct {
		Payload struct{ Data, DataCrc32c string }
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		return response(400, nil), nil
	}
	b, e := base64.StdEncoding.DecodeString(body.Payload.Data)
	if e != nil || string(b) != "fictional-canary" || body.Payload.DataCrc32c != strconv.FormatUint(uint64(crc32.Checksum(b, crc32.MakeTable(crc32.Castagnoli))), 10) {
		return response(400, nil), nil
	}
	f.writes++
	if f.failure == "lost" {
		return nil, errors.New("private-vendor-diagnostic")
	}
	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/"), ":addVersion") + "/versions/23"
	if f.failure == "response" {
		name = "projects/wrong-project/secrets/other/versions/23"
	}
	return response(200, map[string]string{"name": name, "state": "ENABLED"}), nil
}

// newDeliveryProof constructs exclusively fictional transports; no default network is available.
func newDeliveryProof(t *testing.T, failure string) (*Delivery, *deliveryCloud, DeliveryRequest) {
	t.Helper()
	f := &deliveryCloud{failure: failure, storage: fakeCloud{objects: map[string][]byte{}}}
	g, e := NewGCSRegistry(GCSConfig{Bucket: "synthetic-registry", Prefix: "delivery-v1", Tokens: tokens{}, Transport: f})
	if e != nil {
		t.Fatal(e)
	}
	d, e := NewDelivery(DeliveryConfig{Source: StoreConfig{InfisicalOrigin: "https://us.infisical.com", ProjectID: sourceProject, Environment: "dev", IdentityID: identity, IdentityTokens: tokens{}, Transport: f}, Destination: GCPConfig{ProjectID: project, Tokens: tokens{}, Transport: f}}, g)
	if e != nil {
		t.Fatal(e)
	}
	return d, f, DeliveryRequest{"PLAID_CLIENT_ID", "00000000-0000-4000-8000-000000000009", "7"}
}

// TestDeliveryCanary proves exact source selection, CRC-protected write and metadata-only replay.
func TestDeliveryCanary(t *testing.T) {
	d, f, q := newDeliveryProof(t, "")
	first, e := d.Deliver(context.Background(), q)
	if e != nil || !strings.HasSuffix(first.GCPVersion, "/23") {
		t.Fatalf("receipt: %v", e)
	}
	second, e := d.Deliver(context.Background(), q)
	if e != nil || first != second || f.writes != 1 {
		t.Fatal("replay changed delivery")
	}
	q.SourceID = "00000000-0000-4000-8000-000000000008"
	if _, e = d.Deliver(context.Background(), q); code(e) != "secret_delivery_conflict" {
		t.Fatal("conflicting source accepted")
	}
}

// TestDeliveryFailures proves denial, mismatches and ambiguous acknowledgements never cause retries.
func TestDeliveryFailures(t *testing.T) {
	for _, mode := range []string{"denied", "source", "lost", "response"} {
		t.Run(mode, func(t *testing.T) {
			d, f, q := newDeliveryProof(t, mode)
			for i := 0; i < 2; i++ {
				_, e := d.Deliver(context.Background(), q)
				if e == nil || strings.Contains(e.Error(), "private") {
					t.Fatal("failure not redacted")
				}
			}
			want := 0
			if mode == "lost" || mode == "response" {
				want = 1
			}
			if f.writes != want {
				t.Fatal("unexpected write count")
			}
		})
	}
}

// TestDeliveryConcurrent reserves the destination once across overlapping invocations.
func TestDeliveryConcurrent(t *testing.T) {
	d, f, q := newDeliveryProof(t, "")
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = d.Deliver(context.Background(), q) }()
	}
	wg.Wait()
	if f.writes != 1 {
		t.Fatal("concurrent duplicate")
	}
}

// TestDeliveryAllowlist rejects traversal, arbitrary destinations and nonnumeric source versions.
func TestDeliveryAllowlist(t *testing.T) {
	d, _, q := newDeliveryProof(t, "")
	for _, key := range []string{"OTHER", "../PLAID_SECRET", "slot-08", "slot-000"} {
		q.Key = key
		if _, _, e := d.target(q); e == nil {
			t.Fatal("invalid key accepted")
		}
	}
	for _, key := range []string{"PLAID_CLIENT_ID", "PLAID_SECRET", "JAYBURD_DELIVERY_CANARY", "slot-00", "slot-07"} {
		q.Key = key
		if _, _, e := d.target(q); e != nil {
			t.Fatal(e)
		}
	}
	q.SourceVersion = "latest"
	if _, _, e := d.target(q); e == nil {
		t.Fatal("alias accepted")
	}
}

// TestDeliveryLostRegistryAcknowledgement distinguishes an uncertain intent from a recoverable receipt.
func TestDeliveryLostRegistryAcknowledgement(t *testing.T) {
	for _, part := range []string{"intent.json", "receipt.json"} {
		t.Run(part, func(t *testing.T) {
			d, f, q := newDeliveryProof(t, "")
			lost := false
			d.registry.http = client(transportFunc(func(r *http.Request) (*http.Response, error) {
				result, e := f.RoundTrip(r)
				if !lost && r.Method == "POST" && strings.HasSuffix(r.URL.Query().Get("name"), part) {
					lost = true
					return nil, errors.New("synthetic lost acknowledgement")
				}
				return result, e
			}))
			if _, e := d.Deliver(context.Background(), q); code(e) != "secret_delivery_pending" {
				t.Fatal("unknown acknowledgement accepted")
			}
			_, e := d.Deliver(context.Background(), q)
			if part == "intent.json" {
				if code(e) != "secret_delivery_pending" || f.writes != 0 {
					t.Fatal("unknown intent authorized write")
				}
			} else if e != nil || f.writes != 1 {
				t.Fatal("durable receipt recovery failed")
			}
		})
	}
}

// TestDeliveryTamperedReceipt denies a wrong destination even when the object key is correct.
func TestDeliveryTamperedReceipt(t *testing.T) {
	d, f, q := newDeliveryProof(t, "")
	if _, e := d.Deliver(context.Background(), q); e != nil {
		t.Fatal(e)
	}
	for k, b := range f.storage.objects {
		if strings.HasSuffix(k, "receipt.json") {
			var receipt DeliveryReceipt
			_ = json.Unmarshal(b, &receipt)
			receipt.GCPVersion = "projects/" + project + "/secrets/other/versions/23"
			f.storage.objects[k], _ = json.Marshal(receipt)
		}
	}
	if _, e := d.Deliver(context.Background(), q); code(e) != "secret_delivery_conflict" || f.writes != 1 {
		t.Fatal("tampered receipt accepted")
	}
}

// TestDeliverySlotNeverReused prevents another source version from overwriting a reserved dynamic slot.
func TestDeliverySlotNeverReused(t *testing.T) {
	d, f, q := newDeliveryProof(t, "")
	q.Key = "slot-00"
	writes := 0
	transport := transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "us.infisical.com" && !strings.HasSuffix(r.URL.Path, "/login") {
			if r.URL.Query().Get("secretPath") != "/bank-sync" {
				t.Fatal("wrong dynamic folder")
			}
			return response(200, map[string]any{"secret": map[string]any{"id": q.SourceID, "workspace": sourceProject, "environment": "dev", "secretKey": "slot-00", "version": 7, "secretValue": "fictional-envelope"}}), nil
		}
		if r.URL.Host == "secretmanager.googleapis.com" {
			if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/jayburd-dev-bank-sync-slot-00:addVersion") {
				t.Fatal("unexpected dynamic operation")
			}
			writes++
			return response(200, map[string]string{"name": "projects/" + project + "/secrets/jayburd-dev-bank-sync-slot-00/versions/31", "state": "ENABLED"}), nil
		}
		return f.RoundTrip(r)
	})
	d.source.http = client(transport)
	d.destination.http = client(transport)
	if _, e := d.Deliver(context.Background(), q); e != nil {
		t.Fatal(e)
	}
	q.SourceVersion = "8"
	if _, e := d.Deliver(context.Background(), q); code(e) != "secret_delivery_conflict" || writes != 1 {
		t.Fatal("dynamic slot reused")
	}
}

// TestDeliveryCanaryDestinationIsSeparate proves the job key cannot resolve to a Plaid destination.
func TestDeliveryCanaryDestinationIsSeparate(t *testing.T) {
	d, f, q := newDeliveryProof(t, "")
	q.Key = "JAYBURD_DELIVERY_CANARY"
	receipt, e := d.Deliver(context.Background(), q)
	if e != nil || !strings.HasSuffix(receipt.Destination, "/jayburd-dev-bank-sync-canary") || f.writes != 1 {
		t.Fatal("canary destination mismatch", e)
	}
}

// TestCanaryMetadataLookup proves exact-key, value-disabled lookup has no destination/storage calls.
func TestCanaryMetadataLookup(t *testing.T) {
	for _, mode := range []string{"ok", "denied", "wrong-key", "wrong-version", "wrong-project", "wrong-folder", "bad-id"} {
		t.Run(mode, func(t *testing.T) {
			d, _, _ := newDeliveryProof(t, "")
			calls := 0
			d.source.http = client(transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host != "us.infisical.com" {
					t.Fatal("unexpected destination")
				}
				if strings.HasSuffix(r.URL.Path, "/login") {
					return response(200, map[string]any{"accessToken": "synthetic-auth", "tokenType": "Bearer", "expiresIn": 900}), nil
				}
				q := r.URL.Query()
				if r.Method != "GET" || r.URL.Path != "/api/v4/secrets/JAYBURD_DELIVERY_CANARY" || q.Get("viewSecretValue") != "false" || q.Get("expandSecretReferences") != "false" || q.Get("includeImports") != "false" || q.Get("version") != "1" || q.Get("secretPath") != "/" || q.Get("projectId") != sourceProject || q.Get("environment") != "dev" {
					t.Fatal("unsafe lookup")
				}
				if mode == "denied" {
					return response(403, map[string]string{"message": "private-diagnostic"}), nil
				}
				fields := map[string]any{"id": "00000000-0000-4000-8000-000000000009", "workspace": sourceProject, "environment": "dev", "secretKey": "JAYBURD_DELIVERY_CANARY", "secretPath": "/", "version": 1, "secretComment": "private-comment", "secretValue": "unexpected-private-field"}
				switch mode {
				case "wrong-key":
					fields["secretKey"] = "PLAID_SECRET"
				case "wrong-version":
					fields["version"] = 2
				case "wrong-project":
					fields["workspace"] = identity
				case "wrong-folder":
					fields["secretPath"] = "/other"
				case "bad-id":
					fields["id"] = "not-an-id"
				}
				return response(200, map[string]any{"secret": fields}), nil
			}))
			m, e := d.LookupCanary(context.Background(), "1")
			if mode == "ok" {
				if e != nil || m.SourceVersion != "1" || calls != 2 {
					t.Fatal("lookup failed", e)
				}
			} else if e == nil {
				t.Fatal("mismatch accepted")
			}
			encoded, _ := json.Marshal(m)
			if strings.Contains(string(encoded), "private") || (e != nil && strings.Contains(e.Error(), "private")) {
				t.Fatal("unsafe output")
			}
		})
	}
}

// TestRootLookupAllowlist prevents caller-controlled paths or dynamic slots from becoming credential discovery.
func TestRootLookupAllowlist(t *testing.T) {
	for _, key := range []string{"PLAID_CLIENT_ID", "PLAID_SECRET", "slot-00", "../PLAID_SECRET", "OTHER"} {
		d, _, _ := newDeliveryProof(t, "")
		calls := 0
		d.source.http = client(transportFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if strings.HasSuffix(r.URL.Path, "/login") {
				return response(200, map[string]any{"accessToken": "synthetic-auth", "tokenType": "Bearer", "expiresIn": 900}), nil
			}
			if r.URL.Path != "/api/v4/secrets/"+key || r.URL.Query().Get("viewSecretValue") != "false" {
				t.Fatal("unsafe query")
			}
			return response(200, map[string]any{"secret": map[string]any{"id": "00000000-0000-4000-8000-000000000009", "workspace": sourceProject, "environment": "dev", "secretKey": key, "version": 1}}), nil
		}))
		_, err := d.LookupRoot(context.Background(), key, "1")
		allowed := key == "PLAID_CLIENT_ID" || key == "PLAID_SECRET"
		if allowed && (err != nil || calls != 2) {
			t.Fatal("valid root key rejected")
		}
		if !allowed && (err == nil || calls != 0) {
			t.Fatal("unapproved key contacted source")
		}
	}
}

// TestDynamicSlots proves the complete fixed pool uses its folder and destination once, including replay ambiguity.
func TestDynamicSlots(t *testing.T) {
	for i := 0; i < 8; i++ {
		for _, failure := range []string{"", "lost", "response", "denied", "source", "wrong-path", "wrong-project", "wrong-version"} {
			d, f, q := newDeliveryProof(t, failure)
			q.Key = "slot-0" + strconv.Itoa(i)
			first, err := d.Deliver(context.Background(), q)
			if failure == "" && (err != nil || !strings.HasSuffix(first.Destination, "/jayburd-dev-bank-sync-"+q.Key)) {
				t.Fatal("slot delivery rejected", err)
			}
			if failure != "" && err == nil {
				t.Fatal("failure accepted")
			}
			_, second := d.Deliver(context.Background(), q)
			if failure == "" && second != nil {
				t.Fatal("receipt replay failed")
			}
			if failure != "" && second == nil {
				t.Fatal("ambiguous or denied delivery retried")
			}
			expected := 1
			if failure == "denied" || failure == "source" || strings.HasPrefix(failure, "wrong-") {
				expected = 0
			}
			if f.writes != expected {
				t.Fatal("duplicate destination creation")
			}
		}
	}
}

// TestSlotMetadataLookup proves value-disabled fixed namespace discovery and rejects mismatched source metadata.
func TestSlotMetadataLookup(t *testing.T) {
	for i := 0; i < 8; i++ {
		key := "slot-0" + strconv.Itoa(i)
		for _, mode := range []string{"ok", "wrong-path", "wrong-project", "wrong-version", "wrong-key"} {
			d, f, _ := newDeliveryProof(t, "")
			d.source.http = client(transportFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/login") {
					return response(200, map[string]any{"accessToken": "synthetic-auth", "tokenType": "Bearer", "expiresIn": 900}), nil
				}
				q := r.URL.Query()
				if r.Method != "GET" || r.URL.Path != "/api/v4/secrets/"+key || q.Get("secretPath") != "/bank-sync" || q.Get("viewSecretValue") != "false" || q.Get("version") != "1" || q.Get("projectId") != sourceProject || q.Get("includeImports") != "false" || q.Get("expandSecretReferences") != "false" {
					t.Fatal("unsafe lookup")
				}
				fields := map[string]any{"id": "00000000-0000-4000-8000-000000000009", "workspace": sourceProject, "environment": "dev", "secretKey": key, "secretPath": "/bank-sync", "version": 1, "secretValue": "private-unexpected"}
				switch mode {
				case "wrong-path":
					fields["secretPath"] = "/"
				case "wrong-project":
					fields["workspace"] = identity
				case "wrong-version":
					fields["version"] = 2
				case "wrong-key":
					fields["secretKey"] = "PLAID_SECRET"
				}
				return response(200, map[string]any{"secret": fields}), nil
			}))
			metadata, err := d.LookupSelected(context.Background(), key, "1")
			if (err == nil) != (mode == "ok") || f.writes != 0 {
				t.Fatal("lookup result/side effect mismatch")
			}
			encoded, _ := json.Marshal(metadata)
			if strings.Contains(string(encoded), "private") {
				t.Fatal("value leaked")
			}
		}
	}
	for _, key := range []string{"slot-08", "slot-00/other", "../slot-00", "/bank-sync/slot-00", "OTHER"} {
		d, _, _ := newDeliveryProof(t, "")
		d.source.http = client(transportFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid lookup contacted source")
			return nil, nil
		}))
		if _, err := d.LookupSelected(context.Background(), key, "1"); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
}
