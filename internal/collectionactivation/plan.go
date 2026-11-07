// Package collectionactivation applies one operator-approved Sandbox adoption plan.
// Approval restricts a separately authenticated actor; it never supplies identity.
package collectionactivation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var rejected = errors.New("activation rejected")
var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var key = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var revision = regexp.MustCompile(`^[1-9][0-9]{0,18}$`)

type source struct {
	ID   string `json:"source_account_id"`
	Name string `json:"display_name"`
}
type intent struct {
	Profile    string   `json:"profile"`
	Project    string   `json:"project"`
	ID         string   `json:"plan_id"`
	Actor      string   `json:"actor_id"`
	Workspace  string   `json:"workspace_id"`
	Entity     string   `json:"entity_id"`
	Book       string   `json:"book_id"`
	Item       string   `json:"sync_item_id"`
	Generation string   `json:"expected_generation"`
	Key        string   `json:"idempotency_key"`
	Evidence   string   `json:"evidence_ref"`
	Sources    []source `json:"sources"`
	Expires    string   `json:"expires_at"`
}

// Plan holds immutable validated metadata. Digest binds exact approved bytes,
// not the identity or authority of whoever placed them on the operator filesystem.
type Plan struct {
	value    intent
	deadline time.Time
	digest   string
	body     string
}

// Digest returns only the approved metadata integrity hash, never plan contents.
func (p *Plan) Digest() string {
	if p == nil {
		return ""
	}
	return p.digest
}

func validID(s string) bool {
	return uuid.MatchString(s) && s != "00000000-0000-0000-0000-000000000000"
}

// Load reads a bounded regular nonsymlink file, rejects ambiguous JSON and checks
// the envelope hash over the exact nested plan bytes before interpreting intent.
func Load(path string, now time.Time) (*Plan, error) {
	st, err := os.Lstat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() > 32768 {
		return nil, rejected
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, rejected
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(st, actual) {
		return nil, rejected
	}
	raw, err := io.ReadAll(io.LimitReader(f, 32769))
	if err != nil || len(raw) > 32768 {
		return nil, rejected
	}
	var envelope struct {
		Hash string          `json:"sha256"`
		Plan json.RawMessage `json:"plan"`
	}
	if decode(raw, &envelope) != nil {
		return nil, rejected
	}
	sum := sha256.Sum256(envelope.Plan)
	digest := hex.EncodeToString(sum[:])
	if envelope.Hash != digest {
		return nil, rejected
	}
	var v intent
	if decode(envelope.Plan, &v) != nil {
		return nil, rejected
	}
	if v.Profile != "private-sandbox-activation-v1" || v.Project != "jayburd-bookkeeping-dev" || !key.MatchString(v.Key) || !revision.MatchString(v.Generation) {
		return nil, rejected
	}
	if _, err := strconv.ParseInt(v.Generation, 10, 64); err != nil {
		return nil, rejected
	}
	for _, id := range []string{v.ID, v.Actor, v.Workspace, v.Entity, v.Book, v.Item, v.Evidence} {
		if !validID(id) {
			return nil, rejected
		}
	}
	if len(v.Sources) < 1 || len(v.Sources) > 25 {
		return nil, rejected
	}
	seen := map[string]bool{}
	for _, s := range v.Sources {
		if !validID(s.ID) || seen[s.ID] || !utf8.ValidString(s.Name) || utf8.RuneCountInString(s.Name) < 1 || utf8.RuneCountInString(s.Name) > 200 || strings.TrimSpace(s.Name) == "" {
			return nil, rejected
		}
		for _, r := range s.Name {
			if unicode.IsControl(r) {
				return nil, rejected
			}
		}
		seen[s.ID] = true
	}
	end, err := time.Parse(time.RFC3339, v.Expires)
	if err != nil || end.Format("2006-01-02T15:04:05Z") != v.Expires || !end.After(now) || end.After(now.Add(time.Hour)) {
		return nil, rejected
	}
	body, _ := json.Marshal(struct {
		Generation string   `json:"expected_generation"`
		Sources    []source `json:"sources"`
		Evidence   string   `json:"evidence_ref"`
	}{v.Generation, v.Sources, v.Evidence})
	return &Plan{value: v, deadline: end, digest: digest, body: string(body)}, nil
}

// decode rejects duplicate names at every nesting level, case aliases, unknown
// fields, excessive depth and trailing JSON before decoding a closed structure.
func decode(raw []byte, out any) error {
	if !utf8.Valid(raw) {
		return rejected
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 10 {
			return rejected
		}
		tok, e := d.Token()
		if e != nil {
			return rejected
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				n, e := d.Token()
				if e != nil {
					return rejected
				}
				s, ok := n.(string)
				if !ok || seen[s] {
					return rejected
				}
				seen[s] = true
				if walk(depth+1) != nil {
					return rejected
				}
			}
		case '[':
			for d.More() {
				if walk(depth+1) != nil {
					return rejected
				}
			}
		default:
			return rejected
		}
		_, e = d.Token()
		return e
	}
	if walk(0) != nil {
		return rejected
	}
	if _, e := d.Token(); e != io.EOF {
		return rejected
	}
	// encoding/json accepts case-insensitive struct names: require canonical names
	// independently at every object boundary using the target's remarshal shape.
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if strict.Decode(out) != nil {
		return rejected
	}
	normalized, e := json.Marshal(out)
	if e != nil {
		return rejected
	}
	var inKeys, expected any
	if json.Unmarshal(raw, &inKeys) != nil || json.Unmarshal(normalized, &expected) != nil {
		return rejected
	}
	if !sameKeys(inKeys, expected) {
		return rejected
	}
	return nil
}

func sameKeys(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, ok := y[k]
			if !ok || !sameKeys(v, w) {
				return false
			}
		}
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i, v := range x {
			if !sameKeys(v, y[i]) {
				return false
			}
		}
	}
	return true
}
