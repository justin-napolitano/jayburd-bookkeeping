package ledger

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"unicode/utf8"
)

// reportLayout fixes presentation semantics independently of pagination.
func reportLayout(kind string) string {
	switch kind {
	case "balance-sheet":
		return "basic-unclassified-v1"
	case "general-ledger":
		return "posting-v1"
	default:
		return "account-v1"
	}
}

// reportCursor is untrusted continuity metadata, never an authorization token.
type reportCursor struct {
	Version    string   `json:"version"`
	Report     string   `json:"report"`
	Layout     string   `json:"layout"`
	Revision   string   `json:"revision"`
	FilterHash string   `json:"filter_hash"`
	LastKey    []string `json:"last_key"`
}

// decodeReportCursor rejects alternate encodings and duplicate JSON object keys.
func decodeReportCursor(encoded, kind, revision string) (map[string]any, error) {
	bad := errors.New("invalid cursor")
	if len(encoded) == 0 || len(encoded) > 2048 {
		return nil, bad
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil || !utf8.Valid(raw) || base64.RawURLEncoding.EncodeToString(raw) != encoded {
		return nil, bad
	}
	// encoding/json replaces lone UTF-16 escapes; reject them before decoding.
	for i := 0; i+1 < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		if raw[i+1] != 'u' {
			i++
			continue
		}
		if i+6 > len(raw) {
			return nil, bad
		}
		n, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
		if err != nil {
			return nil, bad
		}
		if n >= 0xD800 && n <= 0xDBFF {
			if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
				return nil, bad
			}
			low, err := strconv.ParseUint(string(raw[i+8:i+12]), 16, 16)
			if err != nil || low < 0xDC00 || low > 0xDFFF {
				return nil, bad
			}
			i += 11
		} else {
			if n >= 0xDC00 && n <= 0xDFFF {
				return nil, bad
			}
			i += 5
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return nil, bad
	}
	seen := map[string]bool{}
	for dec.More() {
		token, err = dec.Token()
		if err != nil {
			return nil, bad
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return nil, bad
		}
		switch key {
		case "version", "report", "layout", "revision", "filter_hash", "last_key":
		default:
			return nil, bad
		}
		seen[key] = true
		var v json.RawMessage
		if dec.Decode(&v) != nil {
			return nil, bad
		}
	}
	if _, err = dec.Token(); err != nil {
		return nil, bad
	}
	if _, err = dec.Token(); err != io.EOF {
		return nil, bad
	}
	var c reportCursor
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&c) != nil || len(seen) != 6 || c.Version != "financial-reporting-v1" || c.Report != kind || c.Layout != reportLayout(kind) || revision == "" || c.Revision != revision || c.FilterHash == "" || c.LastKey == nil {
		return nil, bad
	}
	return map[string]any{"revision": c.Revision, "filter_hash": c.FilterHash, "last_key": c.LastKey}, nil
}

// reportRequest translates only transport syntax; SQL validates financial filters.
func reportRequest(r *http.Request, kind string) (string, error) {
	bad := errors.New("invalid report request")
	if r.Method != "GET" || r.ContentLength != 0 || len(r.TransferEncoding) > 0 || len(r.URL.RawQuery) > 4096 || len(r.RequestURI) > 4096 {
		return "", bad
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", bad
	}
	body := map[string]any{"report": kind}
	for key, values := range q {
		if len(values) != 1 || values[0] == "" {
			return "", bad
		}
		switch key {
		case "through_date", "starts_on", "ends_before", "account_id", "compare_through_date", "compare_starts_on", "compare_ends_before", "snapshot_revision":
			body[key] = values[0]
		case "limit":
			n, err := strconv.Atoi(values[0])
			if err != nil || n < 1 || n > 500 || strconv.Itoa(n) != values[0] {
				return "", bad
			}
			body[key] = n
		case "cursor":
		default:
			return "", bad
		}
	}
	if c := q.Get("cursor"); c != "" {
		cursor, err := decodeReportCursor(c, kind, q.Get("snapshot_revision"))
		if err != nil {
			return "", bad
		}
		body["cursor"] = cursor
	}
	raw, err := json.Marshal(body)
	if len(raw) > 8192 {
		return "", bad
	}
	return string(raw), err
}

// reportResponse preserves decimal JSON strings and buffers the whole wire envelope.
func reportResponse(body []byte, kind string) ([]byte, error) {
	bad := errors.New("invalid report response")
	if len(body) > 2<<20 {
		return nil, &Failure{422, "result_out_of_range"}
	}
	var result map[string]json.RawMessage
	if json.Unmarshal(body, &result) != nil || result == nil {
		return nil, bad
	}
	var version, report string
	if json.Unmarshal(result["contract_version"], &version) != nil || version != "financial-reporting-sql-v1" || json.Unmarshal(result["report"], &report) != nil || report != kind {
		return nil, bad
	}
	cursor, ok := result["next_cursor"]
	if !ok {
		return nil, bad
	}
	if string(cursor) != "null" {
		var c reportCursor
		if json.Unmarshal(cursor, &c) != nil {
			return nil, bad
		}
		c.Version = "financial-reporting-v1"
		c.Report = kind
		c.Layout = reportLayout(kind)
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, bad
		}
		encoded := base64.RawURLEncoding.EncodeToString(raw)
		if len(encoded) > 2048 {
			return nil, &Failure{422, "result_out_of_range"}
		}
		result["next_cursor"], _ = json.Marshal(encoded)
	}
	result["contract_version"], _ = json.Marshal("financial-reporting-v1")
	result["layout"], _ = json.Marshal(reportLayout(kind))
	raw, err := json.Marshal(result)
	if len(raw) > 2<<20 {
		return nil, &Failure{422, "result_out_of_range"}
	}
	return raw, err
}
