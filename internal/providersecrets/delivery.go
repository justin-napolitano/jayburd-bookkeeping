package providersecrets

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"hash/crc32"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DeliveryConfig selects a read-only source identity and a separate destination writer.
type DeliveryConfig struct {
	Source      StoreConfig
	Destination GCPConfig
}

// Delivery is an explicit one-shot adapter; it is not wired to application startup.
type Delivery struct {
	source      *Store
	destination *GCPReader
	registry    *GCSRegistry
}

// DeliveryRequest selects metadata only, never caller-supplied credential bytes or URLs.
type DeliveryRequest struct{ Key, SourceID, SourceVersion string }

// DeliveryReceipt proves acknowledgement only; runtime verification is separate.
type DeliveryReceipt struct {
	Request                                               DeliveryRequest
	Origin, Project, Environment, Destination, GCPVersion string
}

// NewDelivery validates fixed namespaces without token discovery or network calls.
func NewDelivery(c DeliveryConfig, registry *GCSRegistry) (*Delivery, error) {
	s := c.Source
	if registry == nil || s.IdentityTokens == nil || s.Transport == nil || !uuid.MatchString(s.ProjectID) || !uuid.MatchString(s.IdentityID) || s.Environment != "dev" || len(s.Slots) != 0 {
		return nil, fail("secret_configuration")
	}
	if s.InfisicalOrigin != "https://app.infisical.com" && s.InfisicalOrigin != "https://us.infisical.com" && s.InfisicalOrigin != "https://eu.infisical.com" {
		return nil, fail("secret_configuration")
	}
	r, e := NewGCPReader(c.Destination)
	if e != nil {
		return nil, e
	}
	return &Delivery{&Store{config: s, http: client(s.Transport)}, r, registry}, nil
}

// target defines the complete allowlist; folder paths and destination names cannot be supplied.
func (d *Delivery) target(q DeliveryRequest) (string, string, error) {
	if !uuid.MatchString(q.SourceID) || !positive(q.SourceVersion) {
		return "", "", fail("secret_reference_invalid")
	}
	folder, suffix := "/", ""
	switch q.Key {
	case "JAYBURD_DELIVERY_CANARY":
		suffix = "canary"
	case "PLAID_CLIENT_ID":
		suffix = "client-id"
	case "PLAID_SECRET":
		suffix = "client-secret"
	default:
		if len(q.Key) != 7 || !strings.HasPrefix(q.Key, "slot-0") || q.Key[6] < '0' || q.Key[6] > '7' {
			return "", "", fail("secret_reference_invalid")
		}
		folder, suffix = "/bank-sync", q.Key
	}
	return folder, "projects/" + d.destination.config.ProjectID + "/secrets/jayburd-dev-bank-sync-" + suffix, nil
}

// Deliver reserves durably before one addVersion attempt and refuses mutation on replay ambiguity.
func (d *Delivery) Deliver(ctx context.Context, q DeliveryRequest) (DeliveryReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	folder, parent, e := d.target(q)
	if e != nil {
		return DeliveryReceipt{}, e
	}
	expected := DeliveryReceipt{Request: q, Origin: d.source.config.InfisicalOrigin, Project: d.source.config.ProjectID, Environment: "dev", Destination: parent}
	// One source version per destination gets one durable intent, even if source IDs change.
	key := "delivery/" + strings.TrimPrefix(parent, "projects/") + "/" + q.SourceVersion
	if folder == "/bank-sync" {
		key = "delivery/" + strings.TrimPrefix(parent, "projects/") + "/once"
	}
	var receipt DeliveryReceipt
	e = d.registry.get(ctx, key+"/receipt.json", &receipt)
	if e == nil {
		return d.receipt(expected, receipt)
	}
	if code(e) != "secret_absent" {
		return DeliveryReceipt{}, e
	}
	token, e := d.source.auth(ctx)
	if e != nil {
		return DeliveryReceipt{}, e
	}
	query := url.Values{"projectId": {expected.Project}, "environment": {"dev"}, "secretPath": {folder}, "version": {q.SourceVersion}, "type": {"shared"}, "expandSecretReferences": {"false"}, "includeImports": {"false"}, "viewSecretValue": {"true"}}
	raw, _, e := request(ctx, d.source.http, "GET", expected.Origin+"/api/v4/secrets/"+q.Key+"?"+query.Encode(), token, "", nil)
	if e != nil {
		return DeliveryReceipt{}, e
	}
	var source struct {
		Secret sourceValue `json:"secret"`
	}
	if e = strict(raw, &source); e != nil {
		return DeliveryReceipt{}, e
	}
	v := source.Secret
	if v.ID != q.SourceID || v.Workspace != expected.Project || v.Environment != "dev" || v.SecretKey != q.Key || string(v.Version) != q.SourceVersion || len(v.SecretValue) == 0 || len(v.SecretValue) > 65536 || strings.Contains(v.SecretValue, "${") {
		return DeliveryReceipt{}, fail("secret_source_mismatch")
	}
	if folder == "/" && !opaque.MatchString(v.SecretValue) {
		return DeliveryReceipt{}, fail("secret_source_mismatch")
	}
	if e = d.registry.create(ctx, key+"/intent.json", expected); e != nil {
		return DeliveryReceipt{}, fail("secret_delivery_pending")
	}
	access, e := d.destination.config.Tokens.AccessToken(ctx)
	if e != nil {
		return DeliveryReceipt{}, fail("secret_delivery_pending")
	}
	payload := []byte(v.SecretValue)
	body, _ := json.Marshal(map[string]any{"payload": map[string]string{"data": base64.StdEncoding.EncodeToString(payload), "dataCrc32c": strconv.FormatUint(uint64(crc32.Checksum(payload, crc32.MakeTable(crc32.Castagnoli))), 10)}})
	raw, _, e = request(ctx, d.destination.http, "POST", "https://secretmanager.googleapis.com/v1/"+parent+":addVersion", access, "application/json", body)
	if e != nil {
		return DeliveryReceipt{}, fail("secret_delivery_pending")
	}
	var added struct{ Name, State string }
	if e = strict(raw, &added); e != nil || added.State != "ENABLED" || !d.destination.resource(added.Name, true) || !strings.HasPrefix(added.Name, parent+"/versions/") {
		return DeliveryReceipt{}, fail("secret_delivery_pending")
	}
	expected.GCPVersion = added.Name
	if e = d.registry.create(ctx, key+"/receipt.json", expected); e != nil {
		return DeliveryReceipt{}, fail("secret_delivery_pending")
	}
	return expected, nil
}

// receipt rejects cross-source or malformed historical acknowledgements without retrieving payloads.
func (d *Delivery) receipt(expected, actual DeliveryReceipt) (DeliveryReceipt, error) {
	version := actual.GCPVersion
	actual.GCPVersion = ""
	if actual != expected || !d.destination.resource(version, true) || !strings.HasPrefix(version, expected.Destination+"/versions/") {
		return DeliveryReceipt{}, fail("secret_delivery_conflict")
	}
	actual.GCPVersion = version
	return actual, nil
}
