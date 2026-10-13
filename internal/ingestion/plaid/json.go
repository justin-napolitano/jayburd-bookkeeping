package plaid

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// strictJSON checks every nested object before ordinary RawMessage decoding can collapse duplicate keys.
func strictJSON(raw []byte) error {
	if !utf8.Valid(raw) || !validSurrogates(raw) {
		return fail("invalid_json")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := scanValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fail("invalid_json")
	}
	return nil
}

// scanValue imposes bounded depth and numeric lexical work even inside inert allowlisted metadata.
func scanValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return fail("limit_exceeded")
	}
	token, err := d.Token()
	if err != nil {
		return fail("invalid_json")
	}
	switch t := token.(type) {
	case json.Delim:
		switch t {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				key, ok := k.(string)
				if e != nil || !ok || strings.ContainsRune(key, 0) {
					return fail("invalid_json")
				}
				if seen[key] {
					return fail("duplicate_key")
				}
				seen[key] = true
				if e = scanValue(d, depth+1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return fail("invalid_json")
			}
		case '[':
			for d.More() {
				if e := scanValue(d, depth+1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return fail("invalid_json")
			}
		default:
			return fail("invalid_json")
		}
	case json.Number:
		if len(t) > 128 {
			return fail("limit_exceeded")
		}
		if i := strings.IndexAny(string(t), "eE"); i >= 0 {
			exponent, e := strconv.ParseInt(string(t)[i+1:], 10, 32)
			if e != nil || exponent < -100 || exponent > 100 {
				return fail("invalid_number")
			}
		}
	case string:
		if strings.ContainsRune(t, 0) {
			return fail("invalid_json")
		}
	}
	return nil
}

// validSurrogates rejects JSON's otherwise replacement-decoded lone UTF-16 surrogates.
func validSurrogates(raw []byte) bool {
	inside := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		n, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if n >= 0xDC00 && n <= 0xDFFF {
			return false
		}
		if n < 0xD800 || n > 0xDBFF {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		low, e := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if e != nil || low < 0xDC00 || low > 0xDFFF {
			return false
		}
		i += 6
	}
	return !inside
}
