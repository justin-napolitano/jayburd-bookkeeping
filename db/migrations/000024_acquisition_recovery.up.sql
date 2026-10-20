-- Issue #220: immutable decoder diagnostics and explicit known-failure recovery.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
CREATE TABLE integration.sync_decode_diagnostic (
 entity_id uuid NOT NULL,book_id uuid NOT NULL,set_id uuid NOT NULL,code text NOT NULL CHECK(code IN ('invalid_json','duplicate_key','limit_exceeded','invalid_number','invalid_expectation','invalid_field','unknown_field','unknown_enum','invalid_cursor_encoding','invalid_cursor_progress','account_mismatch','unsupported_account','invalid_identity','duplicate_identity','invalid_lineage','invalid_date','unsupported_currency','amount_out_of_range')),recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(entity_id,book_id,set_id),FOREIGN KEY(entity_id,book_id,set_id) REFERENCES integration.sync_set
);
CREATE TABLE integration.sync_delivery_recovery (
 operation_id uuid PRIMARY KEY REFERENCES integration.sync_delivery_operation,
 requested_generation bigint NOT NULL CHECK(requested_generation>0),recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_decode_diagnostic FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_delivery_recovery FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
-- Current worker authority and fence are checked by abandonment in this same transaction.
CREATE FUNCTION integration.reject_sync_decode(e uuid,b uuid,id uuid,set_value uuid,f bigint,code_value text) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE result_value jsonb;
BEGIN
 IF code_value IS NULL OR code_value NOT IN ('invalid_json','duplicate_key','limit_exceeded','invalid_number','invalid_expectation','invalid_field','unknown_field','unknown_enum','invalid_cursor_encoding','invalid_cursor_progress','account_mismatch','unsupported_account','invalid_identity','duplicate_identity','invalid_lineage','invalid_date','unsupported_currency','amount_out_of_range') THEN PERFORM integration.sync_error('P2600');END IF;
 result_value:=integration.abandon_sync(e,b,id,set_value,f,'decoder_rejected');
 INSERT INTO integration.sync_decode_diagnostic(entity_id,book_id,set_id,code) VALUES(e,b,set_value,code_value);
 RETURN result_value;
END $$;
-- Only an explicit, authorized manual signal can release a known failed terminal slot.
CREATE FUNCTION integration.recover_failed_sync(t integration.sync_stream) RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE o integration.sync_delivery_operation;proof text;
BEGIN
 PERFORM integration.signal_authorize(t.entity_id,t.book_id,t.stream_id);
 SELECT * INTO o FROM integration.sync_delivery_operation x WHERE x.entity_id=t.entity_id AND x.book_id=t.book_id AND x.stream_id=t.stream_id AND x.state='quarantined' AND NOT EXISTS(SELECT FROM integration.sync_delivery_recovery r WHERE r.operation_id=x.operation_id) ORDER BY x.created_at DESC,x.operation_id DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND THEN RETURN;END IF;
 proof:=integration.sync_delivery_acquisition(o);
 IF o.terminal_code<>'permanent_failure' OR proof<>'abandoned' OR EXISTS(SELECT FROM integration.sync_set x WHERE x.entity_id=t.entity_id AND x.book_id=t.book_id AND x.stream_id=t.stream_id AND x.claim_key IN(o.claim_id,integration.sync_restart_claim(o.entity_id,o.book_id,o.stream_id,o.claim_id)) AND x.state<>'abandoned') THEN PERFORM integration.connection_ops_error('P2804');END IF;
 INSERT INTO integration.sync_delivery_recovery(operation_id,requested_generation) VALUES(o.operation_id,t.requested_generation+1);
END $$;
DROP INDEX integration.sync_delivery_current;
CREATE UNIQUE INDEX sync_delivery_current ON integration.sync_delivery_operation(entity_id,book_id,stream_id) WHERE state IN ('ready','running','retry_wait');
CREATE OR REPLACE FUNCTION integration.sync_delivery_discover(id uuid,maximum integer) RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE d integration.sync_delivery_binding;t integration.sync_stream;o integration.sync_delivery_operation;n integer:=0;
BEGIN
 d:=integration.sync_delivery_authorize(id,true);IF maximum IS NULL OR maximum NOT BETWEEN 1 AND 25 THEN PERFORM integration.sync_delivery_error('P2900');END IF;
 FOR t IN SELECT x.* FROM integration.sync_stream x WHERE x.entity_id=d.entity_id AND x.book_id=d.book_id AND x.worker_login=d.worker_login AND x.service_principal=d.service_principal AND x.requested_generation>x.completed_generation AND integration.signal_eligible(x) AND NOT EXISTS(SELECT FROM integration.sync_delivery_operation a WHERE a.entity_id=x.entity_id AND a.book_id=x.book_id AND a.stream_id=x.stream_id AND a.state<>'succeeded' AND NOT EXISTS(SELECT FROM integration.sync_delivery_recovery r WHERE r.operation_id=a.operation_id)) ORDER BY x.stream_id LIMIT maximum LOOP
  PERFORM integration.sync_delivery_authorize(id,true,t.stream_id);
  INSERT INTO integration.sync_delivery_operation(entity_id,book_id,binding_id,stream_id,target_generation) VALUES(t.entity_id,t.book_id,id,t.stream_id,t.requested_generation) RETURNING * INTO o;
  PERFORM integration.sync_delivery_new_dispatch(o);n:=n+1;
 END LOOP;RETURN jsonb_build_object('created_count',n);
END $$;
CREATE OR REPLACE FUNCTION integration.signal_sync(t integration.sync_stream,k text,why text) RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE n bigint;
BEGIN
 IF k IS NULL OR k!~'^[A-Za-z0-9._:-]{1,128}$' OR why NOT IN ('worker','manual','resume','webhook','catchup') THEN PERFORM integration.connection_ops_error('P2800');END IF;
 SELECT requested_generation INTO n FROM integration.sync_signal WHERE entity_id=t.entity_id AND book_id=t.book_id AND stream_id=t.stream_id AND idempotency_key=k;
 IF NOT FOUND THEN
  IF t.requested_generation=9223372036854775807 THEN PERFORM integration.connection_ops_error('P2805');END IF;
  IF why='manual' THEN PERFORM integration.recover_failed_sync(t); END IF;
  UPDATE integration.sync_stream SET requested_generation=requested_generation+1 WHERE entity_id=t.entity_id AND book_id=t.book_id AND stream_id=t.stream_id RETURNING requested_generation INTO n;
  INSERT INTO integration.sync_signal VALUES(t.entity_id,t.book_id,t.stream_id,k,n);
  INSERT INTO integration.sync_wakeup(entity_id,book_id,stream_id,requested_generation,reason) VALUES(t.entity_id,t.book_id,t.stream_id,n,why);
 END IF;
 RETURN jsonb_build_object('stream_id',t.stream_id,'requested_generation',n::text);
END $$;
CREATE OR REPLACE FUNCTION integration.render_stream_status(t integration.sync_stream) RETURNS jsonb LANGUAGE plpgsql STABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE i integration.sync_item;s banking.source_account;reason_value text;delivery_value text;decoder_value text;
BEGIN
 SELECT * INTO i FROM integration.sync_item WHERE entity_id=t.entity_id AND book_id=t.book_id AND sync_item_id=t.sync_item_id;
 SELECT * INTO s FROM banking.source_account WHERE entity_id=t.entity_id AND book_id=t.book_id AND source_account_id=t.source_account_id;
 SELECT CASE WHEN x.state='abandoned' AND x.reason IN ('pagination_mutation','provider_failure','decoder_rejected','cancelled','item_changed','lease_expired','account_disconnected','account_paused') THEN x.reason ELSE NULL END INTO reason_value FROM integration.sync_set x WHERE x.entity_id=t.entity_id AND x.book_id=t.book_id AND x.stream_id=t.stream_id ORDER BY x.created_at DESC,x.set_id DESC LIMIT 1;
 SELECT o.state INTO delivery_value FROM integration.sync_delivery_operation o WHERE o.entity_id=t.entity_id AND o.book_id=t.book_id AND o.stream_id=t.stream_id AND NOT EXISTS(SELECT FROM integration.sync_delivery_recovery r WHERE r.operation_id=o.operation_id) ORDER BY o.created_at DESC,o.operation_id DESC LIMIT 1;
 SELECT d.code INTO decoder_value FROM integration.sync_set x LEFT JOIN integration.sync_decode_diagnostic d USING(entity_id,book_id,set_id) WHERE x.entity_id=t.entity_id AND x.book_id=t.book_id AND x.stream_id=t.stream_id ORDER BY x.created_at DESC,x.set_id DESC LIMIT 1;
 RETURN jsonb_build_object('delivery_state',delivery_value,'decoder_error',decoder_value,'stream_id',t.stream_id,'source_account_id',t.source_account_id,'sync_item_id',t.sync_item_id,'state',t.state,'stream_revision',t.stream_revision::text,'item_state',i.state,'item_generation',i.generation::text,'binding_revision',s.binding_revision::text,'requested_generation',t.requested_generation::text,'completed_generation',t.completed_generation::text,'queued',t.requested_generation>t.completed_generation,'lease_active',t.current_set IS NOT NULL AND t.lease_until>statement_timestamp(),'last_completed_at',t.last_completed_at,'readiness',t.readiness,'last_error',reason_value,'provider_updated_at',NULL);
END $$;
REVOKE ALL ON integration.sync_decode_diagnostic,integration.sync_delivery_recovery FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_publisher,bookkeeping_webhook,bookkeeping_observer;
REVOKE ALL ON FUNCTION integration.reject_sync_decode(uuid,uuid,uuid,uuid,bigint,text),integration.recover_failed_sync(integration.sync_stream) FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_publisher,bookkeeping_webhook,bookkeeping_observer;
GRANT EXECUTE ON FUNCTION integration.reject_sync_decode(uuid,uuid,uuid,uuid,bigint,text) TO bookkeeping_sync;
COMMIT;
