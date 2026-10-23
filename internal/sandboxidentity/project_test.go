package sandboxidentity

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Independent tags deliberately separate equal-valued purchases; only the comparator owns financial equality.
const fixture = `{"profile":"jayburd-sandbox-verification/1","fixture_id":"fresh-1","rows":[{"logical_id":"a","description":"Fictional A","revision":"1","current":true,"previous_revision":null,"source_state":"booked","amount":"12.34"},{"logical_id":"b","description":"Fictional B","revision":"1","current":true,"previous_revision":null,"source_state":"booked","amount":"12.34"}],"pending_links":[]}`
const page = `{"added":[{"transaction_id":"Provider-B","account_id":"Account-A","name":"Fictional B","amount":12.34,"date":"2026-10-21","pending":false},{"transaction_id":"Provider-A","account_id":"Account-A","name":"Fictional A","amount":12.34,"date":"2026-10-21","pending":false}],"modified":[],"removed":[],"has_more":false,"transactions_update_status":"HISTORICAL_UPDATE_COMPLETE"}`

// TestIdentityProjectionIndependence proves wrong observed finance cannot rewrite either expectations or the map.
func TestIdentityProjectionIndependence(t *testing.T) {
	want := `{"profile":"jayburd-sandbox-identity-map/1","fixture_id":"fresh-1","items":[{"logical_id":"a","transaction_id":"Provider-A"},{"logical_id":"b","transaction_id":"Provider-B"}]}` + "\n"
	for _, input := range []string{page, strings.NewReplacer(`12.34`, `987654.321`, `2026-10-21`, `2000-01-01`, `"pending":false`, `"pending":true`).Replace(page)} {
		got, count, err := Project([]byte(fixture), []byte(input), "Account-A")
		if err != nil || string(got) != want || count != 2 {
			t.Fatal("identity-only projection changed with finance", err)
		}
		for _, marker := range []string{"amount", "date", "pending", "12.34", "987654", "Fictional"} {
			if bytes.Contains(got, []byte(marker)) {
				t.Fatal("unexpected financial/tag output")
			}
		}
	}
}

// TestRejectsIncompleteOrAmbiguousIdentities prevents a projection from concealing unsupported or partial histories.
func TestRejectsIncompleteOrAmbiguousIdentities(t *testing.T) {
	cases := map[string]string{
		"more":                      strings.Replace(page, `"has_more":false`, `"has_more":true`, 1),
		"readiness":                 strings.Replace(page, "HISTORICAL_UPDATE_COMPLETE", "INITIAL_UPDATE_COMPLETE", 1),
		"foreign":                   strings.Replace(page, "Account-A", "Account-B", 1),
		"duplicate_id":              strings.Replace(page, "Provider-B", "Provider-A", 1),
		"duplicate_tag":             strings.Replace(page, "Fictional B", "Fictional A", 1),
		"unknown_tag":               strings.Replace(page, "Fictional B", "Unexpected", 1),
		"duplicate_key":             strings.Replace(page, `"has_more":false`, `"has_more":false,"has_more":false`, 1),
		"duplicate_nested_identity": strings.Replace(page, `"transaction_id":"Provider-B"`, `"transaction_id":"Provider-B","transaction_id":"Provider-B"`, 1),
		"modified":                  strings.Replace(page, `"modified":[]`, `"modified":[{"transaction_id":"Provider-C"}]`, 1),
		"removed":                   strings.Replace(page, `"removed":[]`, `"removed":[{"transaction_id":"Provider-C"}]`, 1),
		"null":                      strings.Replace(page, `"removed":[]`, `"removed":null`, 1),
		"credential_in_id":          strings.Replace(page, "Provider-B", "access-sandbox-rejected", 1),
		"trailing":                  page + ` {}`,
		"oversize":                  page + strings.Repeat(" ", 2<<20),
	}
	var p map[string]any
	_ = json.Unmarshal([]byte(page), &p)
	p["added"] = p["added"].([]any)[:1]
	missing, _ := json.Marshal(p)
	cases["missing"] = string(missing)
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if out, n, err := Project([]byte(fixture), []byte(input), "Account-A"); err != ErrProjection || out != nil || n != 0 {
				t.Fatal("unsafe identity input accepted")
			}
		})
	}
	for _, f := range []string{strings.Replace(fixture, `"revision":"1"`, `"revision":"2"`, 1), strings.Replace(fixture, "Fictional B", "Fictional A", 1), strings.Replace(fixture, `"pending_links":[]`, `"pending_links":[{}]`, 1)} {
		if _, _, err := Project([]byte(f), []byte(page), "Account-A"); err != ErrProjection {
			t.Fatal("unsupported fixture accepted")
		}
	}
	if _, _, err := Project([]byte(fixture), []byte(strings.ReplaceAll(page, "Account-A", "access-sandbox-rejected")), "access-sandbox-rejected"); err != ErrProjection {
		t.Fatal("credential-shaped account identity accepted")
	}
}
