package observer

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
)

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Wakeup is the existing identifier-only work-wakeup-v1 message; it conveys no authority.
type Wakeup struct {
	OperationID        string
	DispatchGeneration int64
}

// ParseWakeup rejects coercion, duplicate/unknown keys and trailing input before database access.
func ParseWakeup(raw []byte) (Wakeup, error) {
	var out Wakeup
	if len(raw) == 0 || len(raw) > 1024 {
		return out, reject()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return out, reject()
	}
	seen := map[string]bool{}
	for d.More() {
		key, e := d.Token()
		if e != nil {
			return Wakeup{}, reject()
		}
		k, ok := key.(string)
		if !ok || seen[k] {
			return Wakeup{}, reject()
		}
		seen[k] = true
		var v any
		if d.Decode(&v) != nil {
			return Wakeup{}, reject()
		}
		switch k {
		case "schema_version":
			n, ok := v.(json.Number)
			if !ok || n.String() != "1" {
				return Wakeup{}, reject()
			}
		case "operation_id":
			s, ok := v.(string)
			if !ok || !canonicalUUID.MatchString(s) {
				return Wakeup{}, reject()
			}
			out.OperationID = s
		case "dispatch_generation":
			s, ok := v.(string)
			if !ok {
				return Wakeup{}, reject()
			}
			n, e := strconv.ParseInt(s, 10, 64)
			if e != nil || n < 1 || strconv.FormatInt(n, 10) != s {
				return Wakeup{}, reject()
			}
			out.DispatchGeneration = n
		default:
			return Wakeup{}, reject()
		}
	}
	if _, err = d.Token(); err != nil || len(seen) != 3 {
		return Wakeup{}, reject()
	}
	if _, err = d.Token(); err != io.EOF {
		return Wakeup{}, reject()
	}
	return out, nil
}

// encodeWakeup emits only validated canonical identifiers from committed dispatch metadata.
func encodeWakeup(w Wakeup) ([]byte, error) {
	if !canonicalUUID.MatchString(w.OperationID) || w.DispatchGeneration < 1 {
		return nil, reject()
	}
	return json.Marshal(struct {
		SchemaVersion      int    `json:"schema_version"`
		OperationID        string `json:"operation_id"`
		DispatchGeneration string `json:"dispatch_generation"`
	}{1, w.OperationID, strconv.FormatInt(w.DispatchGeneration, 10)})
}
