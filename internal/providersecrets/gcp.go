package providersecrets

import (
	"context"
	"encoding/base64"
	"hash/crc32"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var projectName = regexp.MustCompile(`^[a-z][a-z0-9-]{4,61}[a-z0-9]$`)
var pathPart = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// GCPReader reads only numeric versions in a configured project, with integrity checks.
type GCPReader struct {
	config GCPConfig
	http   *http.Client
}

// NewGCPReader validates identifiers without reading credentials or making requests.
func NewGCPReader(c GCPConfig) (*GCPReader, error) {
	if (!projectName.MatchString(c.ProjectID) && !positive(c.ProjectID)) || c.Tokens == nil || c.Transport == nil {
		return nil, fail("secret_configuration")
	}
	return &GCPReader{c, client(c.Transport)}, nil
}

// resource validates global resource names with no aliases, alternate projects or path escaping.
func (r *GCPReader) resource(s string, version bool) bool {
	p := strings.Split(s, "/")
	n := 4
	if version {
		n = 6
	}
	if len(p) != n || p[0] != "projects" || p[1] != r.config.ProjectID || p[2] != "secrets" || !pathPart.MatchString(p[3]) {
		return false
	}
	return !version || (p[4] == "versions" && positive(p[5]))
}

// Read returns an in-memory exact payload only after CRC32C and response identity validation.
func (r *GCPReader) Read(ctx context.Context, name string) ([]byte, error) {
	if !r.resource(name, true) {
		return nil, fail("secret_reference_invalid")
	}
	token, e := r.config.Tokens.AccessToken(ctx)
	if e != nil {
		return nil, fail("secret_auth_failed")
	}
	b, _, e := request(ctx, r.http, "GET", "https://secretmanager.googleapis.com/v1/"+name+":access", token, "", nil)
	if e != nil {
		return nil, e
	}
	var v struct {
		Name    string `json:"name"`
		Payload struct {
			Data string `json:"data"`
			CRC  string `json:"dataCrc32c"`
		} `json:"payload"`
	}
	if e = strict(b, &v); e != nil {
		return nil, e
	}
	data, e := base64.StdEncoding.Strict().DecodeString(v.Payload.Data)
	crc, ce := strconv.ParseUint(v.Payload.CRC, 10, 32)
	if e != nil || ce != nil || strconv.FormatUint(crc, 10) != v.Payload.CRC || v.Name != name || len(data) == 0 || len(data) > 65536 || uint32(crc) != crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli)) {
		return nil, fail("secret_integrity_failed")
	}
	return data, nil
}

// versions is writer-only bounded propagation discovery, never acquisition runtime fallback.
func (r *GCPReader) versions(ctx context.Context, name string) ([]string, error) {
	if !r.resource(name, false) {
		return nil, fail("secret_reference_invalid")
	}
	token, e := r.config.Tokens.AccessToken(ctx)
	if e != nil {
		return nil, fail("secret_auth_failed")
	}
	b, _, e := request(ctx, r.http, "GET", "https://secretmanager.googleapis.com/v1/"+name+"/versions?pageSize=16", token, "", nil)
	if e != nil {
		return nil, e
	}
	var v struct {
		Versions []struct{ Name, State string } `json:"versions"`
		Next     string                         `json:"nextPageToken"`
	}
	if e = strict(b, &v); e != nil {
		return nil, e
	}
	if v.Next != "" || len(v.Versions) > 16 {
		return nil, fail("secret_propagation_bounds")
	}
	out := []string{}
	seen := map[string]bool{}
	for _, x := range v.Versions {
		if !r.resource(x.Name, true) || !strings.HasPrefix(x.Name, name+"/versions/") || seen[x.Name] {
			return nil, fail("secret_response_invalid")
		}
		seen[x.Name] = true
		if x.State != "ENABLED" {
			return nil, fail("secret_version_unavailable")
		}
		out = append(out, x.Name)
	}
	return out, nil
}

// escaped encodes a whole object name as a single API path segment.
func escaped(s string) string { return url.PathEscape(s) }
