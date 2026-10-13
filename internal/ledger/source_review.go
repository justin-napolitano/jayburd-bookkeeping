package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

// SourceReview permits only scoped reads and draft creation; posting uses durable command recovery.
func (s *Service) SourceReview(ctx context.Context, c Command, action string) (Result, error) {
	switch action {
	case "sources", "list", "detail", "categories", "draft":
	default:
		return Result{}, &Failure{400, "invalid_request"}
	}
	return s.execute(ctx, c, "review_"+action)
}

// sourceReviewRequest bounds named query values; SQL owns authorization and snapshot semantics.
func sourceReviewRequest(r *http.Request) (string, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", errors.New("invalid query")
	}
	v := map[string]any{"source_account_id": "", "state": nil, "limit": 50, "cursor": nil}
	for key, values := range q {
		if len(values) != 1 || values[0] == "" {
			return "", errors.New("invalid query")
		}
		value := values[0]
		switch key {
		case "source_account_id":
			if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(value) {
				return "", errors.New("invalid source")
			}
			v[key] = value
		case "state":
			switch value {
			case "needs_review", "overlap_hold", "draft_ready", "posted":
				v[key] = value
			default:
				return "", errors.New("invalid state")
			}
		case "limit":
			n, e := strconv.Atoi(value)
			if e != nil || n < 1 || n > 100 || strconv.Itoa(n) != value {
				return "", errors.New("invalid limit")
			}
			v[key] = n
		case "cursor":
			if len(value) > 2048 {
				return "", errors.New("invalid cursor")
			}
			v[key] = value
		default:
			return "", errors.New("invalid query")
		}
	}
	if v["source_account_id"] == "" {
		return "", errors.New("missing source")
	}
	raw, err := json.Marshal(v)
	return string(raw), err
}
