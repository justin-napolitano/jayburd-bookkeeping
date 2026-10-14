package providersecrets

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxResponse = 128 << 10

// client disables redirects; the caller cannot supply a destination through request metadata.
func client(t http.RoundTripper) *http.Client {
	return &http.Client{Transport: t, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// request performs one bounded attempt and never includes response/error text in failures.
func request(ctx context.Context, c *http.Client, method, endpoint, token, contentType string, body []byte) ([]byte, int, error) {
	if !opaque.MatchString(token) {
		return nil, 0, fail("secret_auth_failed")
	}
	q, e := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if e != nil {
		return nil, 0, fail("secret_configuration")
	}
	q.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		q.Header.Set("Content-Type", contentType)
	}
	r, e := c.Do(q)
	if e != nil {
		return nil, 0, fail("secret_transport_unknown")
	}
	defer r.Body.Close()
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		if r.StatusCode == 404 {
			return nil, 404, fail("secret_absent")
		}
		if r.StatusCode == 412 || r.StatusCode == 409 {
			return nil, r.StatusCode, fail("secret_conflict")
		}
		return nil, r.StatusCode, fail("secret_http_rejected")
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, maxResponse+1))
	if e != nil || len(b) > maxResponse {
		return nil, r.StatusCode, fail("secret_response_invalid")
	}
	return b, r.StatusCode, nil
}

// strict checks duplicate keys, nesting, trailing values and null before typed decoding.
func strict(raw []byte, out any) error {
	if len(raw) == 0 || len(raw) > maxResponse {
		return fail("secret_response_invalid")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e := walk(d, 0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return fail("secret_response_invalid")
	}
	if string(bytes.TrimSpace(raw)) == "null" {
		return fail("secret_response_invalid")
	}
	if e := json.Unmarshal(raw, out); e != nil {
		return fail("secret_response_invalid")
	}
	return nil
}

// walk validates material JSON syntax without ever rendering rejected data.
func walk(d *json.Decoder, depth int) error {
	if depth > 16 {
		return fail("secret_response_invalid")
	}
	v, e := d.Token()
	if e != nil {
		return fail("secret_response_invalid")
	}
	delim, ok := v.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			s, ok := k.(string)
			if e != nil || !ok || seen[strings.ToLower(s)] {
				return fail("secret_response_invalid")
			}
			seen[strings.ToLower(s)] = true
			if e = walk(d, depth+1); e != nil {
				return e
			}
		}
	case '[':
		for d.More() {
			if e = walk(d, depth+1); e != nil {
				return e
			}
		}
	default:
		return fail("secret_response_invalid")
	}
	_, e = d.Token()
	if e != nil {
		return fail("secret_response_invalid")
	}
	return nil
}

// safePath rejects traversal and URL syntax in configured registry namespaces.
func safePath(s string) bool {
	if s == "" || len(s) > 128 || strings.HasPrefix(s, "/") || strings.HasSuffix(s, "/") {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." || !pathPart.MatchString(p) {
			return false
		}
	}
	return true
}

// stringsReader keeps login form construction inside the trusted adapter.
func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }

// loginResponse applies the same fixed response/error boundary to the non-bearer login endpoint.
func loginResponse(c *http.Client, q *http.Request) ([]byte, error) {
	r, e := c.Do(q)
	if e != nil {
		return nil, fail("secret_auth_failed")
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, fail("secret_auth_failed")
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, maxResponse+1))
	if e != nil || len(b) > maxResponse {
		return nil, fail("secret_auth_failed")
	}
	return b, nil
}

// versionNumber compares validated positive numeric destination versions without lexical ordering.
func versionNumber(s string) int64 {
	p := strings.LastIndex(s, "/")
	n, _ := strconv.ParseInt(s[p+1:], 10, 64)
	return n
}
