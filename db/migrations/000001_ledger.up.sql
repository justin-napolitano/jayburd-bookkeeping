-- ADR 0005 / posting-boundary.md. Synthetic proof; runtime has no table DML.
BEGIN;
DO $$ BEGIN
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='bookkeeping_owner') THEN CREATE ROLE bookkeeping_owner NOLOGIN; END IF;
 IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='bookkeeping_api') THEN CREATE ROLE bookkeeping_api NOLOGIN; END IF;
 IF EXISTS (SELECT FROM pg_roles WHERE rolname IN ('bookkeeping_owner','bookkeeping_api') AND (rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb OR rolcanlogin))
 OR pg_has_role('bookkeeping_api','bookkeeping_owner','MEMBER') THEN RAISE EXCEPTION 'unsafe ledger roles'; END IF;
END $$;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SET LOCAL ROLE bookkeeping_owner;
-- Schemas are created by migration identity below because owner cannot CREATE database objects yet.
RESET ROLE;
CREATE SCHEMA ledger AUTHORIZATION bookkeeping_owner;
CREATE SCHEMA reference AUTHORIZATION bookkeeping_owner;
CREATE SCHEMA audit AUTHORIZATION bookkeeping_owner;
CREATE SCHEMA integration AUTHORIZATION bookkeeping_owner;
CREATE SCHEMA reporting AUTHORIZATION bookkeeping_owner;
CREATE SCHEMA banking AUTHORIZATION bookkeeping_owner;
SET LOCAL ROLE bookkeeping_owner;
ALTER DEFAULT PRIVILEGES REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;
CREATE DOMAIN ledger.amount AS numeric(38,9) CHECK (VALUE >= 0 AND VALUE < 1e29 AND VALUE <> 'NaN'::numeric);
CREATE DOMAIN ledger.rate AS numeric(38,18) CHECK (VALUE > 0 AND VALUE < 1e20 AND VALUE <> 'NaN'::numeric);
CREATE TABLE ledger.entity (
 entity_id uuid PRIMARY KEY, name text NOT NULL, kind text NOT NULL CHECK(kind IN ('legal','personal')),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), disabled_at timestamptz
);
CREATE TABLE ledger.principal (
 principal_id uuid PRIMARY KEY, issuer text NOT NULL, subject text NOT NULL,
 disabled_at timestamptz, created_at timestamptz NOT NULL DEFAULT clock_timestamp(), UNIQUE(issuer,subject)
);
CREATE TABLE ledger.membership (
 entity_id uuid NOT NULL REFERENCES ledger.entity, principal_id uuid NOT NULL REFERENCES ledger.principal,
 permission text NOT NULL CHECK(permission IN ('read','prepare','post','close')), revoked_at timestamptz,
 PRIMARY KEY(entity_id,principal_id,permission)
);
CREATE TABLE ledger.book (
 entity_id uuid NOT NULL REFERENCES ledger.entity, book_id uuid NOT NULL, name text NOT NULL,
 revision bigint NOT NULL DEFAULT 0 CHECK(revision>=0), created_at timestamptz NOT NULL DEFAULT clock_timestamp(), disabled_at timestamptz,
 PRIMARY KEY(entity_id,book_id), UNIQUE(entity_id,name)
);
CREATE TABLE reference.currency (
 currency_version_id uuid PRIMARY KEY, alphabetic_code text NOT NULL CHECK(alphabetic_code ~ '^[A-Z]{3}$'),
 numeric_code text CHECK(numeric_code ~ '^[0-9]{3}$'), minor_unit smallint CHECK(minor_unit BETWEEN 0 AND 9),
 valid_from date NOT NULL, valid_to date, source_release text NOT NULL,
 CHECK(isfinite(valid_from) AND (valid_to IS NULL OR (isfinite(valid_to) AND valid_to>valid_from))), UNIQUE(alphabetic_code,valid_from)
);
-- Version intervals serialize per currency code; RR setup is rejected because
-- its stale snapshot could miss a version committed while waiting on the lock.
CREATE FUNCTION reference.currency_interval() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN
 IF current_setting('transaction_isolation')='repeatable read' THEN RAISE EXCEPTION 'unsupported reference setup isolation'; END IF;
 PERFORM pg_advisory_xact_lock(77421,hashtext(NEW.alphabetic_code));
 IF EXISTS(SELECT FROM reference.currency c WHERE c.alphabetic_code=NEW.alphabetic_code AND daterange(c.valid_from,c.valid_to)&&daterange(NEW.valid_from,NEW.valid_to)) THEN RAISE EXCEPTION 'overlapping currency versions'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER intervals BEFORE INSERT ON reference.currency FOR EACH ROW EXECUTE FUNCTION reference.currency_interval();
CREATE TABLE ledger.book_policy (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, policy_id uuid NOT NULL,
 effective_from date NOT NULL, effective_to date, functional_currency_version_id uuid NOT NULL REFERENCES reference.currency,
 basis text NOT NULL, timezone text NOT NULL, posting_scale smallint NOT NULL CHECK(posting_scale BETWEEN 0 AND 9),
 rounding_mode text NOT NULL, rate_policy_code text NOT NULL,
 approved_by uuid NOT NULL REFERENCES ledger.principal, approved_at timestamptz NOT NULL,
 PRIMARY KEY(entity_id,book_id,policy_id), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 UNIQUE(entity_id,book_id,effective_from), CHECK(isfinite(effective_from) AND (effective_to IS NULL OR (isfinite(effective_to) AND effective_to>effective_from)))
);
CREATE TABLE ledger.account (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, account_id uuid NOT NULL, code text NOT NULL, name text NOT NULL,
 category text NOT NULL CHECK(category IN ('asset','liability','equity','revenue','expense')),
 normal_side text NOT NULL CHECK(normal_side IN ('debit','credit')), monetary boolean NOT NULL,
 active_from date NOT NULL, active_to date,
 PRIMARY KEY(entity_id,book_id,account_id), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 UNIQUE(entity_id,book_id,code), CHECK(isfinite(active_from) AND (active_to IS NULL OR (isfinite(active_to) AND active_to>active_from)))
);
CREATE TABLE ledger.period (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, period_id uuid NOT NULL, starts_on date NOT NULL, ends_before date NOT NULL,
 state text NOT NULL DEFAULT 'open' CHECK(state IN ('open','closed')),
 closed_at timestamptz, closed_by uuid REFERENCES ledger.principal, close_revision bigint,
 PRIMARY KEY(entity_id,book_id,period_id), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 CHECK(isfinite(starts_on) AND isfinite(ends_before) AND starts_on<ends_before),
 CHECK((state='open' AND closed_at IS NULL AND closed_by IS NULL AND close_revision IS NULL) OR
       (state='closed' AND closed_at IS NOT NULL AND closed_by IS NOT NULL AND close_revision IS NOT NULL))
);
CREATE TABLE reference.rate_observation (
 rate_observation_id uuid PRIMARY KEY, base_currency_version_id uuid NOT NULL REFERENCES reference.currency,
 quote_currency_version_id uuid NOT NULL REFERENCES reference.currency, quote_per_base ledger.rate NOT NULL,
 effective_at timestamptz NOT NULL, observed_at timestamptz NOT NULL, source text NOT NULL, source_reference text NOT NULL,
 purpose text NOT NULL CHECK(purpose IN ('spot','closing','valuation')), policy_code text NOT NULL,
 supersedes_id uuid REFERENCES reference.rate_observation, UNIQUE(source,source_reference),
 CHECK(base_currency_version_id<>quote_currency_version_id AND isfinite(effective_at) AND isfinite(observed_at))
);
CREATE TABLE ledger.journal_entry (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, journal_id uuid NOT NULL,
 period_id uuid NOT NULL, policy_id uuid NOT NULL, accounting_date date NOT NULL CHECK(isfinite(accounting_date)),
 kind text NOT NULL CHECK(kind IN ('manual','reversal')), description text NOT NULL CHECK(length(description) BETWEEN 1 AND 500),
 posted_at timestamptz NOT NULL, posted_by uuid NOT NULL REFERENCES ledger.principal,
 book_revision bigint NOT NULL CHECK(book_revision>0), reverses_journal_id uuid, reason text,
 PRIMARY KEY(entity_id,book_id,journal_id), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book,
 FOREIGN KEY(entity_id,book_id,period_id) REFERENCES ledger.period,
 FOREIGN KEY(entity_id,book_id,policy_id) REFERENCES ledger.book_policy,
 FOREIGN KEY(entity_id,book_id,reverses_journal_id) REFERENCES ledger.journal_entry,
 UNIQUE(entity_id,book_id,reverses_journal_id), UNIQUE(entity_id,book_id,book_revision),
 CHECK((kind='manual' AND reverses_journal_id IS NULL AND reason IS NULL) OR (kind='reversal' AND reverses_journal_id IS NOT NULL AND length(reason) BETWEEN 1 AND 500))
);
CREATE TABLE ledger.journal_line (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, journal_id uuid NOT NULL, line_number integer NOT NULL CHECK(line_number BETWEEN 1 AND 1000),
 account_id uuid NOT NULL, side text NOT NULL CHECK(side IN ('debit','credit')),
 transaction_currency_version_id uuid NOT NULL REFERENCES reference.currency,
 transaction_amount ledger.amount NOT NULL CHECK(transaction_amount>0), functional_amount ledger.amount NOT NULL CHECK(functional_amount>0),
 valuation_basis text NOT NULL CHECK(valuation_basis IN ('identity','spot')), rate_observation_id uuid REFERENCES reference.rate_observation,
 PRIMARY KEY(entity_id,book_id,journal_id,line_number),
 FOREIGN KEY(entity_id,book_id,journal_id) REFERENCES ledger.journal_entry,
 FOREIGN KEY(entity_id,book_id,account_id) REFERENCES ledger.account,
 CHECK((valuation_basis='identity' AND rate_observation_id IS NULL) OR (valuation_basis='spot' AND rate_observation_id IS NOT NULL))
);
CREATE TABLE integration.operation_receipt (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, operation text NOT NULL CHECK(operation IN ('post','reverse')),
 idempotency_key text NOT NULL CHECK(length(idempotency_key) BETWEEN 1 AND 128 AND idempotency_key ~ '^[A-Za-z0-9._:-]+$'),
 principal_id uuid NOT NULL REFERENCES ledger.principal, request_hash bytea NOT NULL CHECK(octet_length(request_hash)=32),
 normalized_request jsonb NOT NULL, canonical_version smallint NOT NULL CHECK(canonical_version=1), result jsonb NOT NULL,
 http_status smallint NOT NULL CHECK(http_status=201), created_at timestamptz NOT NULL,
 PRIMARY KEY(entity_id,book_id,operation,idempotency_key), FOREIGN KEY(entity_id,book_id) REFERENCES ledger.book
);
CREATE TABLE audit.action (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, action_id uuid NOT NULL, principal_id uuid NOT NULL REFERENCES ledger.principal,
 operation text NOT NULL, target_id uuid NOT NULL, book_revision bigint NOT NULL, request_id uuid NOT NULL, recorded_at timestamptz NOT NULL, reason text,
 PRIMARY KEY(entity_id,book_id,action_id), FOREIGN KEY(entity_id,book_id,target_id) REFERENCES ledger.journal_entry
);
CREATE TABLE integration.outbox_message (
 entity_id uuid NOT NULL, book_id uuid NOT NULL, message_id uuid NOT NULL, event_type text NOT NULL CHECK(event_type='journal.posted'),
 event_version integer NOT NULL CHECK(event_version=1), target_id uuid NOT NULL, book_revision bigint NOT NULL, created_at timestamptz NOT NULL,
 PRIMARY KEY(entity_id,book_id,message_id), FOREIGN KEY(entity_id,book_id,target_id) REFERENCES ledger.journal_entry,
 UNIQUE(entity_id,book_id,target_id)
);
CREATE INDEX journal_date ON ledger.journal_entry(entity_id,book_id,accounting_date,journal_id);
CREATE INDEX account_history ON ledger.journal_line(entity_id,book_id,account_id,journal_id);
CREATE INDEX rate_date ON reference.rate_observation(base_currency_version_id,quote_currency_version_id,effective_at);

-- Even privileged ordinary DML must not rewrite posted facts or reference versions.
CREATE FUNCTION ledger.immutable_row() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
BEGIN RAISE EXCEPTION USING ERRCODE='P2108', MESSAGE='immutable_history'; END $$;
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON ledger.journal_entry FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON ledger.journal_line FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON reference.currency FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON reference.rate_observation FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON ledger.book_policy FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.operation_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.outbox_message FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON audit.action FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();

-- Controlled setup changes serialize with posting and invalidate reviewed revisions.
CREATE FUNCTION ledger.setup_change() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE e uuid; b uuid;
BEGIN
 e:=coalesce(NEW.entity_id,OLD.entity_id); b:=coalesce(NEW.book_id,OLD.book_id);
 IF TG_OP='UPDATE' AND (NEW.entity_id<>OLD.entity_id OR NEW.book_id<>OLD.book_id) THEN RAISE EXCEPTION 'scope immutable'; END IF;
 UPDATE ledger.book SET revision=revision+1 WHERE entity_id=e AND book_id=b;
 IF TG_TABLE_NAME='period' AND TG_OP<>'DELETE' THEN
  IF TG_OP='UPDATE' AND (OLD.state='closed' OR NEW.starts_on<>OLD.starts_on OR NEW.ends_before<>OLD.ends_before) THEN RAISE EXCEPTION 'period immutable'; END IF;
  IF EXISTS(SELECT FROM ledger.period p WHERE p.entity_id=e AND p.book_id=b AND p.period_id<>NEW.period_id AND daterange(p.starts_on,p.ends_before)&&daterange(NEW.starts_on,NEW.ends_before)) THEN RAISE EXCEPTION 'overlapping periods'; END IF;
 ELSIF TG_TABLE_NAME='book_policy' AND TG_OP='INSERT' THEN
  IF EXISTS(SELECT FROM ledger.book_policy p WHERE p.entity_id=e AND p.book_id=b AND daterange(p.effective_from,p.effective_to)&&daterange(NEW.effective_from,NEW.effective_to)) THEN RAISE EXCEPTION 'overlapping policies'; END IF;
 ELSIF TG_TABLE_NAME='account' AND TG_OP='UPDATE' THEN
  IF (to_jsonb(NEW)-'active_to')<>(to_jsonb(OLD)-'active_to') THEN RAISE EXCEPTION 'account semantics immutable'; END IF;
 END IF;
 RETURN coalesce(NEW,OLD);
END $$;
CREATE TRIGGER setup BEFORE INSERT OR UPDATE OR DELETE ON ledger.period FOR EACH ROW EXECUTE FUNCTION ledger.setup_change();
CREATE TRIGGER setup BEFORE INSERT ON ledger.book_policy FOR EACH ROW EXECUTE FUNCTION ledger.setup_change();
CREATE TRIGGER setup BEFORE INSERT OR UPDATE OR DELETE ON ledger.account FOR EACH ROW EXECUTE FUNCTION ledger.setup_change();
GRANT USAGE ON SCHEMA ledger,reporting TO bookkeeping_api;
COMMIT;
