package webhook

import (
	"bytes"
	"encoding/json"
)

// Rejection identifies a closed validation stage; it never contains request-derived strings or values.
type Rejection uint8

const (
	Unclassified Rejection = iota
	ProfileRejected
	BoundsRejected
	FramingRejected
	HeaderRejected
	VerificationHeaderRejected
	ClaimsRejected
	IssuedAtTypeRejected
	IssuedAtRejected
	KeyLookupRejected
	KeyMetadataRejected
	SignatureRejected
	BodyHashRejected
	BodyJSONRejected
	LinkSchemaRejected
	SyncSchemaRejected
	SyncUnexpectedError
	SyncUnexpectedCount
	SyncNullUser
	SyncMissingInitial
	SyncMissingHistorical
	SyncUnexpectedField
	EnvironmentRejected
	ItemRejected
	UserRejected
	ReadinessRejected
	ContextRejected
)

// String maps even forged enum values to fixed labels, never provider metadata or error text.
func (r Rejection) String() string {
	switch r {
	case ProfileRejected:
		return "profile"
	case BoundsRejected:
		return "bounds"
	case FramingRejected:
		return "framing"
	case HeaderRejected:
		return "header"
	case VerificationHeaderRejected:
		return "verification_header"
	case ClaimsRejected:
		return "claims"
	case IssuedAtTypeRejected:
		return "issued_at_type"
	case IssuedAtRejected:
		return "issued_at"
	case KeyLookupRejected:
		return "key_lookup"
	case KeyMetadataRejected:
		return "key_metadata"
	case SignatureRejected:
		return "signature"
	case BodyHashRejected:
		return "body_hash"
	case BodyJSONRejected:
		return "body_json"
	case LinkSchemaRejected:
		return "link_schema"
	case SyncSchemaRejected:
		return "sync_schema"
	case SyncUnexpectedError:
		return "sync_unexpected_error"
	case SyncUnexpectedCount:
		return "sync_unexpected_count"
	case SyncNullUser:
		return "sync_null_user"
	case SyncMissingInitial:
		return "sync_missing_initial"
	case SyncMissingHistorical:
		return "sync_missing_historical"
	case SyncUnexpectedField:
		return "sync_unexpected_field"
	case EnvironmentRejected:
		return "environment"
	case ItemRejected:
		return "item_identifier"
	case UserRejected:
		return "user_identifier"
	case ReadinessRejected:
		return "readiness"
	case ContextRejected:
		return "context"
	default:
		return "unclassified"
	}
}

// claimsRejection diagnoses already rejected claims without returning their names, timestamps or digest.
func claimsRejection(raw []byte) Rejection {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) == nil {
		if issued, ok := fields["iat"]; ok {
			var value int64
			if json.Unmarshal(issued, &value) != nil {
				return IssuedAtTypeRejected
			}
		}
	}
	return ClaimsRejected
}

// syncRejection runs only after authentication and rejection; it cannot widen the accepted schema.
func syncRejection(raw []byte) Rejection {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return SyncSchemaRejected
	}
	if _, ok := fields["error"]; ok {
		return SyncUnexpectedError
	}
	if _, ok := fields["new_transactions"]; ok {
		return SyncUnexpectedCount
	}
	if v, ok := fields["user_id"]; ok && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		return SyncNullUser
	}
	if v, ok := fields["initial_update_complete"]; !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		return SyncMissingInitial
	}
	if v, ok := fields["historical_update_complete"]; !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		return SyncMissingHistorical
	}
	for key := range fields {
		switch key {
		case "webhook_type", "webhook_code", "item_id", "environment", "initial_update_complete", "historical_update_complete", "user_id":
		default:
			return SyncUnexpectedField
		}
	}
	return SyncSchemaRejected
}
