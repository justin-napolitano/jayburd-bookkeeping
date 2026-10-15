//go:build integration

package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// durableCommand mirrors only the public recovery contract for synthetic assertions.
type durableCommand struct {
	ID           string          `json:"command_id"`
	Operation    string          `json:"operation"`
	Entity       *string         `json:"entity_id"`
	Book         *string         `json:"book_id"`
	State        string          `json:"state"`
	Body         json.RawMessage `json:"body"`
	Result       json.RawMessage `json:"result"`
	Created      string          `json:"created_at"`
	Acknowledged bool            `json:"acknowledged"`
}

// registration preserves the reviewed financial body inside its immutable envelope.
func registration(op, key string) Command {
	c := command(key)
	if op == "setup" {
		c = setupCommand(key)
		c.Body = `{"operation":"setup","entity_id":null,"book_id":null,"body":` + c.Body + `}`
	} else {
		c.Body = fmt.Sprintf(`{"operation":"post","entity_id":%q,"book_id":%q,"body":%s}`, entity, book, c.Body)
	}
	return c
}

// decodeCommand requires a complete typed command instead of silently accepting an empty object.
func decodeCommand(t *testing.T, r Result) durableCommand {
	t.Helper()
	var c durableCommand
	if json.Unmarshal(r.Body, &c) != nil || c.ID == "" || c.Created == "" || c.State == "" {
		t.Fatal("invalid durable command")
	}
	return c
}

// recoverCommand submits explicit synchronous lifecycle transitions using trusted actor identity.
func recoverCommand(t *testing.T, s *Service, id, action string) durableCommand {
	t.Helper()
	r, err := s.Recovery(context.Background(), Command{Actor: actor, CommandID: id, Request: requestID(), Body: `{}`}, action)
	if err != nil {
		t.Fatal(err)
	}
	return decodeCommand(t, r)
}

// pendingCommand discovers durable work without executing it or synthesizing acknowledgement.
func pendingCommand(t *testing.T, s *Service, who string) *durableCommand {
	t.Helper()
	r, err := s.Recovery(context.Background(), Command{Actor: who}, "pending")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Command *durableCommand `json:"command"`
	}
	if json.Unmarshal(r.Body, &v) != nil {
		t.Fatal("invalid pending response")
	}
	return v.Command
}

// clearInaccessible invokes the payload-free escape hatch and decodes only its fixed boolean receipt.
func clearInaccessible(t *testing.T, s *Service, who string) bool {
	t.Helper()
	r, err := s.Recovery(context.Background(), Command{Actor: who, Request: requestID(), Body: `{}`}, "clear")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Cleared bool `json:"cleared"`
	}
	if json.Unmarshal(r.Body, &v) != nil {
		t.Fatal("invalid clear receipt")
	}
	return v.Cleared
}

// TestCommandLifecycle proves prepare, execute, replay, explicit acknowledgement and immutable history.
func TestCommandLifecycle(t *testing.T) {
	for _, op := range []string{"setup", "post"} {
		t.Run(op, func(t *testing.T) {
			c, s, _ := fixtureDB(t, "command_lifecycle_"+op)
			ctx := context.Background()
			input := registration(op, "reviewed")
			if pendingCommand(t, s, actor) != nil {
				t.Fatal("unexpected pending command")
			}
			r, err := s.Recovery(ctx, input, "register")
			if err != nil {
				t.Fatal(err)
			}
			v := decodeCommand(t, r)
			if r.Status != 201 || v.State != "prepared" || v.Acknowledged || string(v.Result) != "null" {
				t.Fatal("registration executed")
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "0" || scalar(t, c, "SELECT count(*)::text FROM ledger.entity") != "1" {
				t.Fatal("preparation changed accounting")
			}
			replay, err := s.Recovery(ctx, input, "register")
			if err != nil || decodeCommand(t, replay).ID != v.ID {
				t.Fatal("registration replay diverged", err)
			}
			changed := input
			changed.Key = "new-key"
			_, err = s.Recovery(ctx, changed, "register")
			requireFailure(t, err, "pending_command")
			changed = input
			changed.Body = strings.Replace(changed.Body, `"body":{`, `"body":{"unexpected":true,`, 1)
			_, err = s.Recovery(ctx, changed, "register")
			requireFailure(t, err, "invalid_request")
			changed = input
			changed.Body = strings.Replace(changed.Body, `2026-01-01`, `2026-01-02`, 1)
			_, err = s.Recovery(ctx, changed, "register")
			requireFailure(t, err, "idempotency_conflict")
			_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, "acknowledge")
			requireFailure(t, err, "invalid_command_state")
			executed := recoverCommand(t, s, v.ID, "execute")
			if executed.State != "committed" || executed.Acknowledged || string(executed.Result) == "null" {
				t.Fatal("execution result lost")
			}
			if again := recoverCommand(t, s, v.ID, "execute"); string(again.Result) != string(executed.Result) {
				t.Fatal("replay changed result")
			}
			_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, "cancel")
			requireFailure(t, err, "invalid_command_state")
			if found := pendingCommand(t, s, actor); found == nil || found.ID != v.ID || found.Acknowledged {
				t.Fatal("discovery acknowledged result")
			}
			if ack := recoverCommand(t, s, v.ID, "acknowledge"); !ack.Acknowledged {
				t.Fatal("acknowledgement lost")
			}
			if pendingCommand(t, s, actor) != nil {
				t.Fatal("acknowledged result still pending")
			}
			if ack := recoverCommand(t, s, v.ID, "acknowledge"); !ack.Acknowledged {
				t.Fatal("acknowledgement replay lost")
			}
			replay, err = s.Recovery(ctx, input, "register")
			if err != nil || !decodeCommand(t, replay).Acknowledged {
				t.Fatal("historical replay lost")
			}
			input.Key = "second"
			r, err = s.Recovery(ctx, input, "register")
			if err != nil {
				t.Fatal(err)
			}
			v2 := decodeCommand(t, r)
			if cancelled := recoverCommand(t, s, v2.ID, "cancel"); cancelled.State != "cancelled" || cancelled.Acknowledged {
				t.Fatal("cancellation not explicit")
			}
			if again := recoverCommand(t, s, v2.ID, "cancel"); again.State != "cancelled" {
				t.Fatal("cancel replay lost")
			}
			_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v2.ID, Request: requestID(), Body: `{}`}, "execute")
			requireFailure(t, err, "invalid_command_state")
			recoverCommand(t, s, v2.ID, "acknowledge")
			if scalar(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != "2" || scalar(t, c, "SELECT count(*)::text FROM audit.command_action") != "6" {
				t.Fatal("command history duplicated or removed")
			}
			if op == "post" && scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
				t.Fatal("duplicate posting")
			}
			if op == "setup" && scalar(t, c, "SELECT count(*)::text FROM ledger.setup_receipt") != "1" {
				t.Fatal("duplicate setup")
			}
			for _, q := range []string{"DELETE FROM ledger.interactive_command", "UPDATE ledger.interactive_command SET request_body='{}'", "UPDATE ledger.interactive_command SET result='{}'", "DELETE FROM audit.command_action"} {
				if _, err = c.Exec(ctx, q); err == nil {
					t.Fatal("immutable history changed")
				}
			}
		})
	}
}

// TestCommandAuthorization rechecks actor ownership, current read/post grants and active identity.
func TestCommandAuthorization(t *testing.T) {
	c, s, db := fixtureDB(t, "command_authority")
	ctx := context.Background()
	other := requestID()
	sql(t, c, "INSERT INTO ledger.principal VALUES($1,'synthetic','other',NULL)", other)
	sql(t, c, "INSERT INTO ledger.membership VALUES($1,$2,'read',NULL),($1,$2,'post',NULL)", entity, other)
	r, err := s.Recovery(ctx, registration("post", "scope"), "register")
	if err != nil {
		t.Fatal(err)
	}
	v := decodeCommand(t, r)
	if pendingCommand(t, s, other) != nil {
		t.Fatal("another actor discovered request")
	}
	for _, action := range []string{"execute", "cancel", "acknowledge"} {
		_, err = s.Recovery(ctx, Command{Actor: other, CommandID: v.ID, Request: requestID(), Body: `{}`}, action)
		requireFailure(t, err, "not_found")
	}
	for _, permission := range []string{"read", "post"} {
		sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE principal_id=$1 AND permission=$2", actor, permission)
		for _, action := range []string{"pending", "execute", "cancel", "acknowledge"} {
			result, err := s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, action)
			requireFailure(t, err, "permission_denied")
			if len(result.Body) != 0 {
				t.Fatal("revoked financial body leaked")
			}
		}
		blocked := registration("setup", "blocked")
		_, err = s.Recovery(ctx, blocked, "register")
		requireFailure(t, err, "pending_command")
		sql(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE principal_id=$1 AND permission=$2", actor, permission)
	}
	recoverCommand(t, s, v.ID, "execute")
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE principal_id=$1", actor)
	_, err = s.Recovery(ctx, Command{Actor: actor}, "pending")
	requireFailure(t, err, "not_found")
	sql(t, c, "UPDATE ledger.principal SET disabled_at=now() WHERE principal_id=$1", actor)
	_, err = s.Recovery(ctx, registration("setup", "disabled"), "register")
	requireFailure(t, err, "not_found")
	runtime := connection(t, db, "synthetic_runtime")
	defer runtime.Close(ctx)
	for _, q := range []string{"SELECT * FROM ledger.interactive_command", "SELECT * FROM audit.command_action", "UPDATE ledger.interactive_command SET acknowledged_at=now()", "SELECT ledger.render_command(NULL::ledger.interactive_command)"} {
		if _, err = runtime.Exec(ctx, q); err == nil {
			t.Fatal("runtime crossed command table boundary")
		}
	}
}

// TestInaccessibleCommandClear proves explicit release never exposes or executes saved financial work.
func TestInaccessibleCommandClear(t *testing.T) {
	for _, terminal := range []string{"prepared", "committed", "cancelled"} {
		t.Run(terminal, func(t *testing.T) {
			c, s, _ := fixtureDB(t, "command_clear_"+terminal)
			ctx := context.Background()
			r, err := s.Recovery(ctx, registration("post", "blocked"), "register")
			if err != nil {
				t.Fatal(err)
			}
			v := decodeCommand(t, r)
			if terminal == "committed" {
				recoverCommand(t, s, v.ID, "execute")
			} else if terminal == "cancelled" {
				recoverCommand(t, s, v.ID, "cancel")
			}
			sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE entity_id=$1 AND principal_id=$2 AND permission='read'", entity, actor)
			_, err = s.Recovery(ctx, Command{Actor: actor}, "pending")
			requireFailure(t, err, "permission_denied")
			journals := scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry")
			if !clearInaccessible(t, s, actor) || clearInaccessible(t, s, actor) {
				t.Fatal("clear receipt did not distinguish initial action from replay")
			}
			if scalar(t, c, "SELECT state||':'||(acknowledged_at IS NOT NULL)::text FROM ledger.interactive_command WHERE command_id=$1", v.ID) != map[string]string{"prepared": "cancelled:true", "committed": "committed:true", "cancelled": "cancelled:true"}[terminal] {
				t.Fatal("unexpected terminal recovery state")
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != journals {
				t.Fatal("clear changed accounting")
			}
			expectedActions := map[string]string{"prepared": "3", "committed": "3", "cancelled": "3"}[terminal]
			if scalar(t, c, "SELECT count(*)::text FROM audit.command_action WHERE command_id=$1", v.ID) != expectedActions {
				t.Fatal("clear audit evidence incomplete")
			}
		})
	}
	for _, op := range []string{"setup", "post"} {
		t.Run("authorized_"+op, func(t *testing.T) {
			_, s, _ := fixtureDB(t, "command_clear_authorized_"+op)
			r, err := s.Recovery(context.Background(), registration(op, "visible"), "register")
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.Recovery(context.Background(), Command{Actor: actor, Request: requestID(), Body: `{}`}, "clear")
			requireFailure(t, err, "invalid_command_state")
			if found := pendingCommand(t, s, actor); found == nil || found.ID != decodeCommand(t, r).ID {
				t.Fatal("authorized command was cleared")
			}
		})
	}
	t.Run("other_principal", func(t *testing.T) {
		c, s, _ := fixtureDB(t, "command_clear_other")
		other := requestID()
		sql(t, c, "INSERT INTO ledger.principal VALUES($1,'synthetic','clear-other',NULL)", other)
		r, err := s.Recovery(context.Background(), registration("post", "owned"), "register")
		if err != nil {
			t.Fatal(err)
		}
		if clearInaccessible(t, s, other) {
			t.Fatal("other principal cleared foreign work")
		}
		if found := pendingCommand(t, s, actor); found == nil || found.ID != decodeCommand(t, r).ID {
			t.Fatal("foreign clear changed owned command")
		}
	})
	for name, body := range map[string]string{"empty": "", "malformed": "{", "nonempty": `{"unexpected":true}`} {
		t.Run("invalid_body_"+name, func(t *testing.T) {
			_, s, _ := fixtureDB(t, "command_clear_invalid_"+name)
			_, err := s.Recovery(context.Background(), Command{Actor: actor, Request: requestID(), Body: body}, "clear")
			requireFailure(t, err, "invalid_request")
		})
	}
}

// TestCommandConcurrency forces actual principal lock contention for register and execute/cancel.
func TestCommandConcurrency(t *testing.T) {
	for _, same := range []bool{true, false} {
		t.Run(fmt.Sprintf("register_%t", same), func(t *testing.T) {
			c, s, _ := fixtureDB(t, fmt.Sprintf("command_register_%t", same))
			ctx := context.Background()
			tx, err := c.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err = tx.Exec(ctx, "SELECT 1 FROM ledger.principal WHERE principal_id=$1 FOR UPDATE", actor); err != nil {
				t.Fatal(err)
			}
			type outcome struct {
				r   Result
				err error
			}
			done := make(chan outcome, 2)
			for i := range 2 {
				key := "same"
				if !same {
					key = fmt.Sprint(i)
				}
				go func(key string) {
					r, err := s.Recovery(ctx, registration("post", key), "register")
					done <- outcome{r, err}
				}(key)
			}
			waitSetupBlocked(t, c, 2)
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			first, second := <-done, <-done
			if same {
				if first.err != nil || second.err != nil || decodeCommand(t, first.r).ID != decodeCommand(t, second.r).ID {
					t.Fatal("concurrent replay failed", first.err, second.err)
				}
			} else {
				if first.err == nil && second.err == nil {
					t.Fatal("two pending commands")
				}
				if first.err != nil {
					requireFailure(t, first.err, "pending_command")
				}
				if second.err != nil {
					requireFailure(t, second.err, "pending_command")
				}
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != "1" {
				t.Fatal("duplicate command")
			}
		})
	}
	for _, executeFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("execute_cancel_%t", executeFirst), func(t *testing.T) {
			c, s, db := fixtureDB(t, fmt.Sprintf("command_cancel_%t", executeFirst))
			ctx := context.Background()
			r, err := s.Recovery(ctx, registration("post", "race"), "register")
			if err != nil {
				t.Fatal(err)
			}
			v := decodeCommand(t, r)
			runtime := connection(t, db, "synthetic_runtime")
			defer runtime.Close(ctx)
			tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			first, second := "execute", "cancel"
			if !executeFirst {
				first, second = second, first
			}
			var body []byte
			if err = tx.QueryRow(ctx, "SELECT ledger.transition_command($1,$2,$3,$4,'{}')", actor, v.ID, requestID(), first).Scan(&body); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				_, err := s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, second)
				done <- err
			}()
			waitSetupBlocked(t, c, 1)
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			requireFailure(t, <-done, "invalid_command_state")
			count := "0"
			if executeFirst {
				count = "1"
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != count {
				t.Fatal("race duplicated or bypassed cancellation")
			}
		})
	}
}

// TestCommandLostAcknowledgement discards actual committed replies at each durable transition.
func TestCommandLostAcknowledgement(t *testing.T) {
	for _, lost := range []string{"register", "execute", "cancel", "acknowledge"} {
		t.Run(lost, func(t *testing.T) {
			c, s, _ := fixtureDB(t, "command_lost_"+lost)
			ctx := context.Background()
			input := registration("post", "lost")
			var v durableCommand
			if lost == "register" {
				s.DB = &lostCommitDB{Beginner: s.DB}
				_, err := s.Recovery(ctx, input, "register")
				requireFailure(t, err, "temporarily_unavailable")
				v = *pendingCommand(t, s, actor)
			} else {
				r, err := s.Recovery(ctx, input, "register")
				if err != nil {
					t.Fatal(err)
				}
				v = decodeCommand(t, r)
				if lost == "acknowledge" {
					recoverCommand(t, s, v.ID, "execute")
				}
				s.DB = &lostCommitDB{Beginner: s.DB}
				_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, lost)
				requireFailure(t, err, "temporarily_unavailable")
			}
			if lost == "register" {
				r, err := s.Recovery(ctx, input, "register")
				if err != nil || decodeCommand(t, r).ID != v.ID {
					t.Fatal("lost register duplicated")
				}
			} else {
				recoverCommand(t, s, v.ID, lost)
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != "1" {
				t.Fatal("duplicate command")
			}
			count := "0"
			if lost == "execute" || lost == "acknowledge" {
				count = "1"
			}
			if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != count {
				t.Fatal("lost reply changed financial effect")
			}
		})
	}
}

// TestCommandValidationAndRollback rejects malformed review input and rolls back failures after writes.
func TestCommandValidationAndRollback(t *testing.T) {
	c, s, _ := fixtureDB(t, "command_rollback")
	ctx := context.Background()
	for _, raw := range []string{
		strings.Replace(registration("post", "bad").Body, `"side":"debit"`, `"side":"debit","side":"credit"`, 1),
		strings.Replace(registration("post", "bad").Body, `"body":{`, `"body":{"actor":"spoof",`, 1),
		strings.Replace(registration("post", "bad").Body, `"body":{`, `"body":{"lines":[],`, 1),
		strings.Replace(registration("setup", "bad").Body, `"code":`, `"unknown":true,"code":`, 1),
	} {
		input := registration("post", "bad")
		input.Body = raw
		_, err := s.Recovery(ctx, input, "register")
		requireFailure(t, err, "invalid_request")
	}
	oversized := registration("post", "large")
	oversized.Body = strings.Repeat(" ", 1<<20) + oversized.Body
	_, err := s.Recovery(ctx, oversized, "register")
	requireFailure(t, err, "request_too_large")
	if scalar(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != "0" {
		t.Fatal("invalid command stored")
	}
	sql(t, c, `CREATE FUNCTION public.command_fail() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'synthetic command failure'; END$$`)
	sql(t, c, "CREATE TRIGGER fail AFTER INSERT ON audit.command_action FOR EACH ROW EXECUTE FUNCTION public.command_fail()")
	_, err = s.Recovery(ctx, registration("post", "rolledback"), "register")
	requireFailure(t, err, "internal_error")
	if scalar(t, c, "SELECT count(*)::text FROM ledger.interactive_command") != "0" {
		t.Fatal("partial registration")
	}
	sql(t, c, "DROP TRIGGER fail ON audit.command_action")
	r, err := s.Recovery(ctx, registration("post", "execute"), "register")
	if err != nil {
		t.Fatal(err)
	}
	v := decodeCommand(t, r)
	for _, table := range []string{"ledger.interactive_command", "audit.command_action"} {
		sql(t, c, "CREATE TRIGGER fail AFTER INSERT OR UPDATE ON "+table+" FOR EACH ROW EXECUTE FUNCTION public.command_fail()")
		_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, "execute")
		requireFailure(t, err, "internal_error")
		assertEmpty(t, c)
		if scalar(t, c, "SELECT state FROM ledger.interactive_command") != "prepared" {
			t.Fatal("failure acknowledged command")
		}
		sql(t, c, "DROP TRIGGER fail ON "+table)
	}
	sql(t, c, "CREATE CONSTRAINT TRIGGER fail AFTER UPDATE ON ledger.interactive_command DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION public.command_fail()")
	_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, "execute")
	requireFailure(t, err, "internal_error")
	assertEmpty(t, c)
	sql(t, c, "DROP TRIGGER fail ON ledger.interactive_command")
	recoverCommand(t, s, v.ID, "execute")
}

// TestCommandScopeIsolation keeps foreign targets hidden and reauthorizes new setup scopes.
func TestCommandScopeIsolation(t *testing.T) {
	c, s, _ := fixtureDB(t, "command_scope_isolation")
	ctx := context.Background()
	other := requestID()
	sql(t, c, "INSERT INTO ledger.principal VALUES($1,'synthetic','foreign-owner',NULL)", other)
	setup := setupCommand("foreign-setup")
	setup.Actor = other
	result, err := s.Setup(ctx, setup)
	if err != nil {
		t.Fatal(err)
	}
	foreign := decodeSetup(t, result)
	input := registration("post", "foreign-target")
	input.Body = strings.Replace(input.Body, `"entity_id":"`+entity+`"`, `"entity_id":"`+foreign.Entity+`"`, 1)
	input.Body = strings.Replace(input.Body, `"book_id":"`+book+`"`, `"book_id":"`+foreign.Book+`"`, 1)
	_, err = s.Recovery(ctx, input, "register")
	requireFailure(t, err, "not_found")
	if pendingCommand(t, s, actor) != nil {
		t.Fatal("foreign target registered")
	}
	result, err = s.Recovery(ctx, registration("setup", "own-setup"), "register")
	if err != nil {
		t.Fatal(err)
	}
	v := decodeCommand(t, result)
	committed := recoverCommand(t, s, v.ID, "execute")
	var scope struct {
		Entity string `json:"entity_id"`
	}
	if json.Unmarshal(committed.Result, &scope) != nil || scope.Entity == "" {
		t.Fatal("missing new setup scope")
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=now() WHERE entity_id=$1 AND permission='read'", scope.Entity)
	for _, action := range []string{"pending", "execute", "acknowledge"} {
		r, err := s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, action)
		requireFailure(t, err, "permission_denied")
		if len(r.Body) > 0 {
			t.Fatal("revoked setup result leaked")
		}
	}
	sql(t, c, "UPDATE ledger.membership SET revoked_at=NULL WHERE entity_id=$1", scope.Entity)
	recoverCommand(t, s, v.ID, "acknowledge")
}

// TestCommandExecutionValidation leaves reviewed intent recoverable after ordinary ledger rejection.
func TestCommandExecutionValidation(t *testing.T) {
	c, s, _ := fixtureDB(t, "command_semantic_rejection")
	ctx := context.Background()
	input := registration("post", "unbalanced")
	input.Body = strings.Replace(input.Body, `"transaction_amount":"1000","functional_amount":"1000"`, `"transaction_amount":"999","functional_amount":"999"`, 1)
	r, err := s.Recovery(ctx, input, "register")
	if err != nil {
		t.Fatal(err)
	}
	v := decodeCommand(t, r)
	_, err = s.Recovery(ctx, Command{Actor: actor, CommandID: v.ID, Request: requestID(), Body: `{}`}, "execute")
	requireFailure(t, err, "invalid_journal")
	assertEmpty(t, c)
	found := pendingCommand(t, s, actor)
	if found == nil || found.State != "prepared" || found.Acknowledged {
		t.Fatal("rejected command was lost")
	}
	recoverCommand(t, s, v.ID, "cancel")
	recoverCommand(t, s, v.ID, "acknowledge")
}

// TestCommandDatabaseUnknownCommit keeps database-reported uncertainty recoverable after a real commit.
func TestCommandDatabaseUnknownCommit(t *testing.T) {
	for _, code := range []string{"08007", "40003"} {
		t.Run(code, func(t *testing.T) {
			c, s, _ := fixtureDB(t, "command_unknown_"+code)
			input := registration("post", "database-unknown")
			registered, err := s.Recovery(context.Background(), input, "register")
			if err != nil {
				t.Fatal(err)
			}
			id := decodeCommand(t, registered).ID
			s.DB = &lostCommitDB{Beginner: s.DB, replyError: &pgconn.PgError{Code: code, Message: "synthetic unknown completion"}}
			result, err := s.Recovery(context.Background(), Command{Actor: actor, CommandID: id, Request: requestID(), Body: `{}`}, "execute")
			requireFailure(t, err, "temporarily_unavailable")
			if len(result.Body) != 0 {
				t.Fatal("unconfirmed commit result escaped")
			}
			recovered := pendingCommand(t, s, actor)
			if recovered == nil || recovered.ID != id || recovered.State != "committed" {
				t.Fatal("committed work not recoverable")
			}
			recoverCommand(t, s, id, "execute")
			if scalar(t, c, "SELECT count(*)::text FROM ledger.journal_entry") != "1" {
				t.Fatal("unknown completion duplicated effect")
			}
		})
	}
}
