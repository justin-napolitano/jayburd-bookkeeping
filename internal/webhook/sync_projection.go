package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// syncProjection discards additive metadata only after original-byte authentication;
// exact routing keys are projected before decoding so aliases cannot overwrite them.
func syncProjection(raw []byte, out any) error {
	if len(raw) > bodyLimit || !utf8.Valid(raw) {
		return errInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if uniqueJSONValue(d, 0) != nil {
		return errInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return errInvalid
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return errInvalid
	}
	projected := map[string]json.RawMessage{}
	for d.More() {
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return errInvalid
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return errInvalid
		}
		switch key {
		case "webhook_type", "webhook_code", "item_id", "environment", "initial_update_complete", "historical_update_complete", "user_id":
			projected[key] = value
		case "error", "new_transactions":
			return errInvalid
		}
	}
	selected, err := json.Marshal(projected)
	if err != nil {
		return errInvalid
	}
	return object(selected, []string{"webhook_type", "webhook_code", "item_id", "environment", "initial_update_complete", "historical_update_complete"}, []string{"user_id"}, out)
}

// uniqueJSONValue bounds ignored metadata and rejects duplicate keys at every depth.
func uniqueJSONValue(d *json.Decoder, depth int) error {
	token, err := d.Token()
	if err != nil {
		return errInvalid
	}
	delim, container := token.(json.Delim)
	if !container {
		return nil
	}
	if depth >= 32 || (delim != '{' && delim != '[') {
		return errInvalid
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			token, err := d.Token()
			key, ok := token.(string)
			if err != nil || !ok || seen[key] {
				return errInvalid
			}
			seen[key] = true
		}
		if uniqueJSONValue(d, depth+1) != nil {
			return errInvalid
		}
	}
	closing, err := d.Token()
	if err != nil || (delim == '{' && closing != json.Delim('}')) || (delim == '[' && closing != json.Delim(']')) {
		return errInvalid
	}
	return nil
}
