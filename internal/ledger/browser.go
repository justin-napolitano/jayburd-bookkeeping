package ledger

import (
	"errors"
	"net/http"
	"net/url"
)

// BrowserBoundary permits one exact HTTPS origin without granting cookie authority
// or CORS. The same-origin frontend/proxy must forward the bearer header explicitly.
func BrowserBoundary(origin string, next http.Handler) (http.Handler, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || next == nil {
		return nil, errors.New("invalid browser boundary")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origins := r.Header.Values("Origin")
		sites := r.Header.Values("Sec-Fetch-Site")
		if len(origins) > 1 || (len(origins) == 1 && origins[0] != origin) || len(sites) > 1 || (len(sites) == 1 && sites[0] != "same-origin" && sites[0] != "none") {
			w.Header().Set("Cache-Control", "no-store")
			problem(w, requestID(), &Failure{403, "permission_denied"})
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}
