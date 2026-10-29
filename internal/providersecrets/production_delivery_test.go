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
	"testing"
)

// productionDeliveryCloud independently enforces the two-key protocol on synthetic transports.
type productionDeliveryCloud struct {
	storage                               fakeCloud
	writes, valueReads                    int
	wrongEnvironment, lostAcknowledgement bool
	requests                              int
}

// RoundTrip rejects source discovery, Sandbox mappings and all destination reads.
func (f *productionDeliveryCloud) RoundTrip(r *http.Request) (*http.Response, error) {
	f.requests++
	if r.URL.Host == "storage.googleapis.com" {
		return f.storage.RoundTrip(r)
	}
	if r.URL.Host == "us.infisical.com" {
		if strings.HasSuffix(r.URL.Path, "/login") {
			return response(200, map[string]any{"accessToken": "synthetic-auth", "tokenType": "Bearer", "expiresIn": 900}), nil
		}
		key := strings.TrimPrefix(r.URL.Path, "/api/v4/secrets/")
		q := r.URL.Query()
		if r.Method != "GET" || (key != "PLAID_CLIENT_ID" && key != "PLAID_PRODUCTION_SECRET") || q.Get("environment") != "prod" || q.Get("secretPath") != "/" || q.Get("projectId") != sourceProject || q.Get("version") != "7" || q.Get("includeImports") != "false" || q.Get("expandSecretReferences") != "false" {
			return nil, errors.New("unexpected source request")
		}
		env := "prod"
		if f.wrongEnvironment {
			env = "dev"
		}
		v := map[string]any{"id": "00000000-0000-4000-8000-000000000009", "workspace": sourceProject, "environment": env, "secretKey": key, "secretPath": "/", "version": 7}
		if q.Get("viewSecretValue") == "true" {
			f.valueReads++
			v["secretValue"] = "fictional-production-value"
		} else if q.Get("viewSecretValue") != "false" {
			return nil, errors.New("missing value fence")
		}
		return response(200, map[string]any{"secret": v}), nil
	}
	parents := map[string]bool{"/v1/projects/" + project + "/secrets/jayburd-prod-bank-sync-client-id:addVersion": true, "/v1/projects/" + project + "/secrets/jayburd-prod-bank-sync-client-secret:addVersion": true}
	if r.URL.Host != "secretmanager.googleapis.com" || r.Method != "POST" || !parents[r.URL.Path] {
		return nil, errors.New("unexpected destination operation")
	}
	var body struct {
		Payload struct{ Data, DataCrc32c string }
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		return nil, errors.New("bad payload")
	}
	value, err := base64.StdEncoding.DecodeString(body.Payload.Data)
	if err != nil || string(value) != "fictional-production-value" || body.Payload.DataCrc32c != strconv.FormatUint(uint64(crc32.Checksum(value, crc32.MakeTable(crc32.Castagnoli))), 10) {
		return nil, errors.New("bad integrity")
	}
	f.writes++
	if f.lostAcknowledgement {
		return nil, errors.New("private-provider-diagnostic")
	}
	return response(200, map[string]string{"name": strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/"), ":addVersion") + "/versions/23", "state": "ENABLED"}), nil
}

// productionDeliveryProof never discovers real identities or opens a default transport.
func productionDeliveryProof(t *testing.T) (*Delivery, *productionDeliveryCloud) {
	t.Helper()
	f := &productionDeliveryCloud{storage: fakeCloud{objects: map[string][]byte{}}}
	g, e := NewGCSRegistry(GCSConfig{Bucket: "synthetic-project-jayburd-prod-credential-registry", Prefix: "production-static-delivery-v1", Tokens: tokens{}, Transport: f})
	if e != nil {
		t.Fatal(e)
	}
	d, e := NewProductionDelivery(DeliveryConfig{Source: StoreConfig{InfisicalOrigin: "https://us.infisical.com", ProjectID: sourceProject, Environment: "prod", IdentityID: identity, IdentityTokens: tokens{}, Transport: f}, Destination: GCPConfig{ProjectID: project, Tokens: tokens{}, Transport: f}}, g)
	if e != nil {
		t.Fatal(e)
	}
	return d, f
}

// TestProductionStaticDelivery proves metadata-only lookup, exact destinations and replay without another value read/write.
func TestProductionStaticDelivery(t *testing.T) {
	destinations := map[string]string{
		"PLAID_CLIENT_ID":         "projects/" + project + "/secrets/jayburd-prod-bank-sync-client-id",
		"PLAID_PRODUCTION_SECRET": "projects/" + project + "/secrets/jayburd-prod-bank-sync-client-secret",
	}
	for _, key := range []string{"PLAID_CLIENT_ID", "PLAID_PRODUCTION_SECRET"} {
		d, f := productionDeliveryProof(t)
		m, e := d.LookupSelected(context.Background(), key, "7")
		if e != nil || m.SourceVersion != "7" || f.valueReads != 0 || f.writes != 0 {
			t.Fatal("lookup crossed value fence")
		}
		q := DeliveryRequest{key, m.SourceID, "7"}
		first, e := d.Deliver(context.Background(), q)
		if e != nil || first.Environment != "prod" || first.Destination != destinations[key] || first.GCPVersion != destinations[key]+"/versions/23" || f.writes != 1 {
			t.Fatal("delivery failed", e)
		}
		second, e := d.Deliver(context.Background(), q)
		if e != nil || first != second || f.writes != 1 || f.valueReads != 1 {
			t.Fatal("replay repeated effect")
		}
		q.SourceID = identity
		if _, e := d.Deliver(context.Background(), q); e == nil || f.writes != 1 || f.valueReads != 1 {
			t.Fatal("conflicting source receipt accepted")
		}
	}
}

// TestProductionDeliveryDenials preserves pre-auth key rejection, source-profile validation and unknown-outcome fencing.
func TestProductionDeliveryDenials(t *testing.T) {
	for _, key := range []string{"PLAID_SECRET", "JAYBURD_DELIVERY_CANARY", "slot-00", "../PLAID_PRODUCTION_SECRET"} {
		d, f := productionDeliveryProof(t)
		if _, e := d.Deliver(context.Background(), DeliveryRequest{key, identity, "7"}); e == nil || f.writes != 0 || f.valueReads != 0 || f.requests != 0 {
			t.Fatal("forbidden key accepted")
		}
	}
	d, f := productionDeliveryProof(t)
	f.wrongEnvironment = true
	if _, e := d.Deliver(context.Background(), DeliveryRequest{"PLAID_PRODUCTION_SECRET", "00000000-0000-4000-8000-000000000009", "7"}); e == nil || f.writes != 0 {
		t.Fatal("mixed environment accepted")
	}
	d, f = productionDeliveryProof(t)
	f.lostAcknowledgement = true
	q := DeliveryRequest{"PLAID_PRODUCTION_SECRET", "00000000-0000-4000-8000-000000000009", "7"}
	if _, e := d.Deliver(context.Background(), q); e == nil {
		t.Fatal("lost acknowledgement accepted")
	}
	if _, e := d.Deliver(context.Background(), q); e == nil || f.writes != 1 {
		t.Fatal("ambiguous delivery repeated write")
	}
}

// TestProductionConstructorIsolation rejects mixed registry/source profiles before authentication.
func TestProductionConstructorIsolation(t *testing.T) {
	d, f := productionDeliveryProof(t)
	c := DeliveryConfig{Source: d.source.config, Destination: d.destination.config}
	badRegistry := *d.registry
	badRegistry.config.Prefix = "delivery-v1"
	if _, err := NewProductionDelivery(c, &badRegistry); err == nil {
		t.Fatal("legacy receipt prefix accepted")
	}
	badRegistry = *d.registry
	badRegistry.config.Bucket = "synthetic-project-jayburd-dev-credential-registry"
	if _, err := NewProductionDelivery(c, &badRegistry); err == nil {
		t.Fatal("Sandbox registry accepted")
	}
	c.Source.Environment = "dev"
	if _, err := NewProductionDelivery(c, d.registry); err == nil {
		t.Fatal("Sandbox source accepted")
	}
	c.Source.Environment = "prod"
	if _, err := NewDelivery(c, d.registry); err == nil {
		t.Fatal("legacy constructor accepted Production")
	}
	if f.requests != 0 {
		t.Fatal("constructor performed I/O")
	}
}
