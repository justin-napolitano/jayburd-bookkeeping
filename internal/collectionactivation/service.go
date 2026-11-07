package collectionactivation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/ledger"
)

// Service executes only the immutable approved tuple on a restricted API pool.
type Service struct {
	db   ledger.Beginner
	plan *Plan
	now  func() time.Time
}

// NewService requires an unexpired validated plan; no caller-supplied SQL exists.
func NewService(db ledger.Beginner, p *Plan) (*Service, error) {
	if db == nil || p == nil || !p.deadline.After(time.Now()) {
		return nil, rejected
	}
	return &Service{db: db, plan: p, now: time.Now}, nil
}

func failure(status int, code string) *ledger.Failure {
	return &ledger.Failure{Status: status, Code: code}
}
func unavailable() *ledger.Failure { return failure(503, "temporarily_unavailable") }

// adopt compares the verified actor, then retries only known whole aborts. An
// uncertain commit requires explicit identical-plan replay, never automatic retry.
func (s *Service) adopt(parent context.Context, actor, id string) (json.RawMessage, error) {
	if s == nil || s.db == nil || s.plan == nil || actor != s.plan.value.Actor || id != s.plan.value.ID || !s.plan.deadline.After(s.now()) {
		return nil, failure(404, "activation_denied")
	}
	deadline := s.now().Add(4 * time.Second)
	if s.plan.deadline.Before(deadline) {
		deadline = s.plan.deadline
	}
	ctx, cancel := context.WithDeadline(parent, deadline)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		if !s.plan.deadline.After(s.now()) || ctx.Err() != nil {
			return nil, unavailable()
		}
		body, err, unknown := s.attempt(ctx)
		if err == nil {
			return body, nil
		}
		if unknown {
			return nil, unavailable()
		}
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			if pg.Code == "40001" || pg.Code == "40P01" {
				continue
			}
			switch pg.Code {
			case "P2800":
				return nil, failure(400, "invalid_activation")
			case "P2801":
				return nil, failure(404, "activation_denied")
			case "P2802", "P2803", "P2804":
				return nil, failure(409, "activation_conflict")
			case "P2805":
				return nil, failure(422, "activation_limit")
			}
		}
		return nil, unavailable()
	}
	return nil, unavailable()
}

func (s *Service) attempt(ctx context.Context) (json.RawMessage, error, bool) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err, false
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	if _, err = tx.Exec(ctx, "select set_config('statement_timeout','3500ms',true),set_config('lock_timeout','1s',true)"); err != nil {
		return nil, err, false
	}
	p := s.plan.value
	var raw json.RawMessage
	err = tx.QueryRow(ctx, "select banking.adopt_collection_item($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::text,$7::text)", p.Actor, p.Workspace, p.Entity, p.Book, p.Item, p.Key, s.plan.body).Scan(&raw)
	if err != nil {
		return nil, err, false
	}
	if !validResult(raw, len(p.Sources)) {
		return nil, unavailable(), false
	}
	if !s.plan.deadline.After(s.now()) {
		return nil, unavailable(), false
	}
	if err = tx.Commit(ctx); err != nil {
		var pg *pgconn.PgError
		unknown := !errors.As(err, &pg) && !errors.Is(err, pgx.ErrTxCommitRollback)
		if pg != nil && (pg.Code == "40003" || strings.HasPrefix(pg.Code, "08")) {
			unknown = true
		}
		return nil, err, unknown
	}
	out, _ := json.Marshal(struct {
		Profile string          `json:"profile"`
		Plan    string          `json:"plan_id"`
		Result  json.RawMessage `json:"result"`
	}{"private-sandbox-activation-result-v1", p.ID, raw})
	return out, nil, false
}

func validResult(raw []byte, n int) bool {
	if len(raw) > 32768 {
		return false
	}
	var result struct {
		Item       string `json:"collection_item_id"`
		Generation string `json:"item_generation"`
		Sources    []struct {
			ID             string `json:"source_account_id"`
			SourceRevision string `json:"source_revision"`
			Revision       string `json:"revision"`
		} `json:"sources"`
	}
	if decode(raw, &result) != nil || !validID(result.Item) || !revision.MatchString(result.Generation) || len(result.Sources) != n {
		return false
	}
	seen := map[string]bool{}
	for _, v := range result.Sources {
		if !validID(v.ID) || seen[v.ID] || v.SourceRevision != "0" || v.Revision != "1" {
			return false
		}
		seen[v.ID] = true
	}
	return true
}
