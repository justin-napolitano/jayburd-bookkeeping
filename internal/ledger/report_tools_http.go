package ledger

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

// reportExportRequest pins the entire selection, never a client-selected page.
func reportExportRequest(r *http.Request, kind string) (string, error) {
	raw, err := reportRequest(r, kind)
	if err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &fields) != nil || fields["snapshot_revision"] == nil || fields["cursor"] != nil || fields["limit"] != nil {
		return "", errors.New("invalid export request")
	}
	return raw, nil
}

// reportJournalRequest transports a pinned GL selector; SQL proves line membership.
func reportJournalRequest(r *http.Request) (string, error) {
	bad := errors.New("invalid report journal request")
	if r.Method != "GET" || r.ContentLength != 0 || len(r.TransferEncoding) > 0 || len(r.RequestURI) > 4096 || len(r.URL.RawQuery) > 4096 {
		return "", bad
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 5 {
		return "", bad
	}
	fields := map[string]any{"journal_id": r.PathValue("journal")}
	for key, values := range q {
		if len(values) != 1 || values[0] == "" {
			return "", bad
		}
		switch key {
		case "account_id", "starts_on", "ends_before", "snapshot_revision":
			fields[key] = values[0]
		case "line_number":
			n, err := strconv.Atoi(values[0])
			if err != nil || n < 1 || n > 1000 || strconv.Itoa(n) != values[0] {
				return "", bad
			}
			fields[key] = n
		default:
			return "", bad
		}
	}
	raw, err := json.Marshal(fields)
	return string(raw), err
}
