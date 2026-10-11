package ledger

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBrowserBoundary exercises origin rejection before identity/database access.
func TestBrowserBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, origin, site    string
		duplicate, cookieOnly bool
		status                int
	}{
		{name: "same_origin", origin: "https://books.example.invalid", site: "same-origin", status: 200},
		{name: "nonbrowser_bearer", status: 200},
		{name: "foreign", origin: "https://evil.invalid", status: 403},
		{name: "null", origin: "null", status: 403},
		{name: "duplicate", origin: "https://books.example.invalid", duplicate: true, status: 403},
		{name: "cross_site", site: "cross-site", status: 403},
		{name: "same_site_not_origin", site: "same-site", status: 403},
		{name: "cookie_only", origin: "https://books.example.invalid", cookieOnly: true, status: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ops := &operationsStub{result: Result{Status: 200, Body: []byte(`{}`)}}
			h, err := BrowserBoundary("https://books.example.invalid", Handler(authStub{}, ops))
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("GET", "/v1/entities/e/books/b/journals/j", nil)
			r.AddCookie(&http.Cookie{Name: "session", Value: "synthetic"})
			if !tc.cookieOnly {
				r.Header.Set("Authorization", "Bearer synthetic")
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.site != "" {
				r.Header.Set("Sec-Fetch-Site", tc.site)
			}
			if tc.duplicate {
				r.Header.Add("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d", w.Code)
			}
			if tc.status != 200 && ops.calls != 0 {
				t.Fatal("denied request reached operations")
			}
			if w.Header().Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("unexpected CORS grant")
			}
		})
	}
	for _, origin := range []string{"http://books.invalid", "https://books.invalid/", "https://user@books.invalid", "*", "null"} {
		if _, err := BrowserBoundary(origin, http.NotFoundHandler()); err == nil {
			t.Fatal("invalid origin accepted")
		}
	}
}
