package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// object rejects duplicate/unknown fields, malformed UTF8 and all trailing JSON before typed decoding.
func object(raw []byte, required, optional []string, out any) error {
	if !utf8.Valid(raw) {
		return errInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	tok, e := d.Token()
	if e != nil || tok != json.Delim('{') {
		return errInvalid
	}
	allowed := map[string]bool{}
	for _, k := range append(append([]string{}, required...), optional...) {
		allowed[k] = true
	}
	seen := map[string]bool{}
	nullable := map[string]bool{}
	for _, k := range required {
		nullable[k] = true
	}
	for d.More() {
		tok, e = d.Token()
		k, ok := tok.(string)
		if e != nil || !ok || !allowed[k] || seen[k] {
			return errInvalid
		}
		seen[k] = true
		var value json.RawMessage
		if d.Decode(&value) != nil || !nullable[k] && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return errInvalid
		}
	}
	if tok, e = d.Token(); e != nil || tok != json.Delim('}') {
		return errInvalid
	}
	if _, e = d.Token(); e != io.EOF {
		return errInvalid
	}
	for _, k := range required {
		if !seen[k] {
			return errInvalid
		}
	}
	if json.Unmarshal(raw, out) != nil {
		return errInvalid
	}
	return nil
}
