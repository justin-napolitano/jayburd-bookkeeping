package collectionactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const a = "00000000-0000-4000-8000-000000000001"
const b = "00000000-0000-4000-8000-000000000002"

func authored(now time.Time) intent {
	return intent{Profile: "private-sandbox-activation-v1", Project: "jayburd-bookkeeping-dev", ID: a, Actor: a, Workspace: b, Entity: a, Book: b, Item: a, Generation: "1", Key: "activation-453", Evidence: b, Sources: []source{{ID: a, Name: "Reviewed fictional checking"}}, Expires: now.Add(30 * time.Minute).UTC().Format("2006-01-02T15:04:05Z")}
}
func envelope(raw []byte) []byte {
	sum := sha256.Sum256(raw)
	out, _ := json.Marshal(struct {
		Hash string          `json:"sha256"`
		Plan json.RawMessage `json:"plan"`
	}{hex.EncodeToString(sum[:]), raw})
	return out
}
func loadBytes(t *testing.T, raw []byte, now time.Time) (*Plan, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(p, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return Load(p, now)
}
func goodPlan(t *testing.T) *Plan {
	t.Helper()
	now := time.Now()
	raw, _ := json.Marshal(authored(now))
	p, e := loadBytes(t, envelope(raw), now)
	if e != nil {
		t.Fatal(e)
	}
	return p
}

// TestActivationPlanClosed exercises the approved-file trust boundary separately
// from authentication; a self-consistent hash cannot confer any actor authority.
func TestActivationPlanClosed(t *testing.T) {
	now := time.Now()
	v := authored(now)
	raw, _ := json.Marshal(v)
	if p, e := loadBytes(t, envelope(raw), now); e != nil || len(p.Digest()) != 64 {
		t.Fatal("valid plan rejected")
	}
	for _, mutate := range []func(*intent){
		func(v *intent) { v.Project = "production" }, func(v *intent) { v.Actor = "" }, func(v *intent) { v.Generation = "01" }, func(v *intent) { v.Generation = "9223372036854775808" },
		func(v *intent) { v.Expires = now.Add(-time.Second).UTC().Format(time.RFC3339) }, func(v *intent) { v.Expires = now.Add(2 * time.Hour).UTC().Format(time.RFC3339) },
		func(v *intent) { v.Sources = append(v.Sources, v.Sources[0]) }, func(v *intent) { v.Sources[0].Name = "\nprivate" }, func(v *intent) { v.Sources[0].Name = strings.Repeat("界", 201) },
	} {
		test := authored(now)
		mutate(&test)
		r, _ := json.Marshal(test)
		if _, e := loadBytes(t, envelope(r), now); e == nil {
			t.Fatal("unsafe plan accepted")
		}
	}
	for _, r := range [][]byte{
		[]byte(strings.Replace(string(raw), `"plan_id":`, `"actor_id":"`+b+`","plan_id":`, 1)),
		[]byte(strings.Replace(string(raw), `"actor_id"`, `"Actor_Id"`, 1)), []byte(strings.Replace(string(raw), `"actor_id"`, `"actor"`, 1)),
		[]byte(strings.TrimSuffix(string(raw), "}") + `,"sql":"SELECT private"}`), append(raw, []byte(` {}`)...), []byte(`null`),
	} {
		if _, e := loadBytes(t, envelope(r), now); e == nil {
			t.Fatal("ambiguous plan accepted")
		}
	}
	bad := envelope(raw)
	bad[12] ^= 1
	if _, e := loadBytes(t, bad, now); e == nil {
		t.Fatal("hash mismatch accepted")
	}
	path := filepath.Join(t.TempDir(), "actual")
	_ = os.WriteFile(path, envelope(raw), 0600)
	link := path + ".link"
	_ = os.Symlink(path, link)
	if _, e := Load(link, now); e == nil {
		t.Fatal("symlink accepted")
	}
	if _, e := Load(filepath.Dir(path), now); e == nil {
		t.Fatal("directory accepted")
	}
	if _, e := loadBytes(t, []byte(strings.Repeat("x", 32769)), now); e == nil {
		t.Fatal("oversized plan accepted")
	}
}

type fakeDB struct {
	calls, commits, rollbacks int
	rowErr, commitErr         error
	query                     string
	args                      []any
	iso                       pgx.TxIsoLevel
}

func (d *fakeDB) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	d.calls++
	d.iso = o.IsoLevel
	return &fakeTx{d: d}, nil
}

type fakeTx struct {
	pgx.Tx
	d *fakeDB
}

func (t *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (t *fakeTx) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.d.query = q
	t.d.args = args
	return fakeRow{d: t.d}
}
func (t *fakeTx) Commit(context.Context) error   { t.d.commits++; return t.d.commitErr }
func (t *fakeTx) Rollback(context.Context) error { t.d.rollbacks++; return nil }

type fakeRow struct{ d *fakeDB }

func (r fakeRow) Scan(dst ...any) error {
	*dst[0].(*json.RawMessage) = json.RawMessage(`{"collection_item_id":"` + a + `","item_generation":"2","sources":[{"source_account_id":"` + b + `","source_revision":"0","revision":"1"}]}`)
	return r.d.rowErr
}

// TestActivationAdapterUnknownCommit preserves exact approved SQL intent across
// explicit retry; uncertain acknowledgement is never automatically repeated.
func TestActivationAdapterUnknownCommit(t *testing.T) {
	p := goodPlan(t)
	d := &fakeDB{commitErr: errors.New("private transport error")}
	s, _ := NewService(d, p)
	body, e := s.adopt(context.Background(), a, a)
	if e == nil || len(body) != 0 || d.calls != 1 || d.commits != 1 || d.rollbacks != 1 {
		t.Fatal("unknown commit retried/disclosed")
	}
	want := []any{a, b, a, b, a, p.value.Key, p.body}
	if !reflect.DeepEqual(d.args, want) || d.iso != pgx.Serializable || d.query != "select banking.adopt_collection_item($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::text,$7::text)" {
		t.Fatal("approved intent changed")
	}
	d.commitErr = nil
	body, e = s.adopt(context.Background(), a, a)
	if e != nil || d.calls != 2 || !strings.Contains(string(body), `"profile":"private-sandbox-activation-result-v1"`) || !reflect.DeepEqual(d.args, want) {
		t.Fatal("exact explicit replay failed")
	}
	for _, tc := range []struct{ actor, id string }{{b, a}, {a, b}} {
		if _, e = s.adopt(context.Background(), tc.actor, tc.id); e == nil || d.calls != 2 {
			t.Fatal("foreign plan reached SQL")
		}
	}
	s.now = func() time.Time { return p.deadline }
	if _, e = s.adopt(context.Background(), a, a); e == nil || d.calls != 2 {
		t.Fatal("expired replay reached SQL")
	}
}

func TestActivationAbortAndSafeFailure(t *testing.T) {
	for _, code := range []string{"40001", "40P01", "P2800", "P2801", "P2802", "P2803", "P2804", "P2805", "55P03"} {
		t.Run(code, func(t *testing.T) {
			d := &fakeDB{rowErr: &pgconn.PgError{Code: code, Message: "private SQL value"}}
			s, _ := NewService(d, goodPlan(t))
			body, e := s.adopt(context.Background(), a, a)
			wanted := 1
			if code == "40001" || code == "40P01" {
				wanted = 3
			}
			if e == nil || len(body) != 0 || d.calls != wanted || strings.Contains(e.Error(), "private") {
				t.Fatal("unsafe failure/retry")
			}
		})
	}
}

type authenticator struct {
	actor string
	err   error
	calls int
}

func (a *authenticator) Authenticate(context.Context, string) (string, error) {
	a.calls++
	return a.actor, a.err
}

// TestActivationHTTPIdentity authenticates before parsing and never accepts
// caller actor/tuple/body substitutions or persistence-oriented retry headers.
func TestActivationHTTPIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, body, actor, auth string
		status                  int
	}{
		{"success", `{"plan_id":"` + a + `"}`, a, "Bearer synthetic", 200},
		{"unauthenticated", `broken`, a, "", 401}, {"foreign-before-parse", `broken`, b, "Bearer synthetic", 404},
		{"actor-override", `{"plan_id":"` + a + `","actor_id":"` + a + `"}`, a, "Bearer synthetic", 400},
		{"duplicate", `{"plan_id":"` + a + `","plan_id":"` + a + `"}`, a, "Bearer synthetic", 400},
		{"case-alias", `{"Plan_Id":"` + a + `"}`, a, "Bearer synthetic", 400},
		{"foreign-plan", `{"plan_id":"` + b + `"}`, a, "Bearer synthetic", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &fakeDB{}
			s, _ := NewService(d, goodPlan(t))
			auth := &authenticator{actor: tc.actor}
			req := httptest.NewRequest(http.MethodPost, "/v1/collection-activation/adopt", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			w := httptest.NewRecorder()
			Handler(auth, s).ServeHTTP(w, req)
			if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Body.String())
			}
			if tc.status != 200 && d.calls != 0 {
				t.Fatal("rejected input reached SQL")
			}
		})
	}
}
