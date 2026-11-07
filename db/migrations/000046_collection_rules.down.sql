-- Empty-feature rollback only. Retained editions, decisions and v3 reviews must never be erased.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE banking.collection_workspace,banking.collection_label_state IN SHARE ROW EXCLUSIVE MODE;
DO $$BEGIN
 IF EXISTS(SELECT FROM banking.collection_rule_edition WHERE ruleset_revision>0)OR EXISTS(SELECT FROM banking.collection_rule_decision)OR EXISTS(SELECT FROM banking.collection_rule_preview_result)OR EXISTS(SELECT FROM banking.collection_label_preview WHERE wire_version=3)THEN PERFORM banking.collection_error('P2802');END IF;
END $$;
DROP FUNCTION banking.transition_collection_label_command_v3(uuid,uuid,uuid,text,text);
DROP FUNCTION banking.pending_collection_label_command_v3(uuid,uuid);
DROP FUNCTION banking.prepare_collection_label_command_v3(uuid,uuid,text);
DROP FUNCTION banking.preview_collection_labels_v3(uuid,uuid,text);
DROP FUNCTION banking.publish_collection_rule_preview(uuid,uuid,text,text);
DROP FUNCTION banking.collection_rule_render_command(banking.collection_label_command);
DROP FUNCTION banking.collection_rule_render_preview(banking.collection_label_preview);
DROP FUNCTION banking.collection_rule_result(uuid,jsonb,jsonb);
DROP FUNCTION banking.collection_rule_context(uuid,uuid,text,boolean);
DROP FUNCTION banking.collection_rule_action(uuid,jsonb);
DROP FUNCTION banking.prepare_collection_label_command_v2(uuid,uuid,text);
DROP FUNCTION banking.pending_collection_label_command_v2(uuid,uuid);
DROP FUNCTION banking.transition_collection_label_command_v2(uuid,uuid,uuid,text,text);
ALTER FUNCTION banking.collection_rule_prepare_v2_base(uuid,uuid,text)RENAME TO prepare_collection_label_command_v2;
ALTER FUNCTION banking.collection_rule_pending_v2_base(uuid,uuid)RENAME TO pending_collection_label_command_v2;
ALTER FUNCTION banking.collection_rule_transition_v2_base(uuid,uuid,uuid,text,text)RENAME TO transition_collection_label_command_v2;
DROP FUNCTION banking.get_collection_taxonomy_v3(uuid,uuid);
ALTER FUNCTION banking.collection_rule_catalog_base(uuid,uuid)RENAME TO get_collection_taxonomy_v3;
CREATE OR REPLACE FUNCTION banking.collection_query_rows(actor uuid,w uuid)
RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,booked_date date,body jsonb)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT r.source_id,r.observation_id,r.observation_revision,r.booked_date,r.body||jsonb_build_object('institution',jsonb_build_object('state','unavailable'),'legal_owner',jsonb_build_object('state','unknown'))FROM banking.collection_tag_rows(actor,w)r
$$;
CREATE OR REPLACE FUNCTION banking.collection_query_snapshot(w uuid)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$SELECT banking.collection_tag_snapshot(w)$$;
CREATE OR REPLACE FUNCTION banking.collection_query_pins(v jsonb,current_value jsonb)RETURNS void LANGUAGE sql SET search_path=pg_catalog,pg_temp AS $$SELECT banking.collection_tag_pins(v,current_value)$$;
DROP FUNCTION banking.collection_label_snapshot(uuid);
ALTER FUNCTION banking.collection_rule_legacy_snapshot_base(uuid)RENAME TO collection_label_snapshot;
CREATE OR REPLACE FUNCTION banking.prepare_collection_label_command(actor uuid,w uuid,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,4096);PERFORM banking.collection_keys(v,ARRAY['preview_id','prepare_key']);
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=ledger.uuid_value(v->'preview_id')AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version=2 THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_tag_prepare_v1_base(actor,w,raw);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'THEN PERFORM banking.collection_error('P2800');END $$;
CREATE OR REPLACE FUNCTION banking.pending_collection_label_command(actor uuid,w uuid)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE c banking.collection_label_command;p banking.collection_label_preview;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND principal_id=actor AND acknowledged_at IS NULL;
 IF FOUND THEN
  IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN PERFORM banking.collection_error('P2801');END IF;
  SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_label_reauthorize(actor,w,p);IF p.wire_version=2 THEN PERFORM banking.collection_error('P2804');END IF;
 END IF;RETURN banking.collection_tag_pending_v1_base(actor,w);
END $$;

CREATE OR REPLACE FUNCTION banking.transition_collection_label_command(actor uuid,w uuid,id uuid,operation text,raw text)RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE c banking.collection_label_command;p banking.collection_label_preview;v jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY[]::text[]);
 SELECT * INTO c FROM banking.collection_label_command WHERE workspace_id=w AND command_id=id AND principal_id=actor;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 SELECT * INTO p FROM banking.collection_label_preview WHERE workspace_id=w AND preview_id=c.preview_id;PERFORM banking.collection_tag_reauthorize(actor,w,p);IF p.wire_version=2 THEN PERFORM banking.collection_error('P2804');END IF;RETURN banking.collection_tag_transition_v1_base(actor,w,id,operation,raw);
END $$;
DROP TRIGGER preserve_collection_rules ON banking.collection_label_state;
DROP FUNCTION banking.collection_rule_state_guard();
DROP TRIGGER collection_rule_initialize ON banking.collection_workspace;
DROP FUNCTION banking.collection_rule_initialize();
DROP TABLE banking.collection_rule_decision;
DROP TABLE banking.collection_rule_preview_result;
ALTER TABLE banking.collection_label_state DROP CONSTRAINT collection_rule_current_fk;
ALTER TABLE banking.collection_label_state DROP COLUMN ruleset_revision;
DROP TABLE banking.collection_rule_edition;
ALTER TABLE banking.collection_label_preview DROP CONSTRAINT collection_label_preview_wire_version_check;
ALTER TABLE banking.collection_label_preview ADD CONSTRAINT collection_label_preview_wire_version_check CHECK(wire_version IN(1,2));
GRANT EXECUTE ON FUNCTION banking.get_collection_taxonomy_v3(uuid,uuid),banking.prepare_collection_label_command_v2(uuid,uuid,text),banking.pending_collection_label_command_v2(uuid,uuid),banking.transition_collection_label_command_v2(uuid,uuid,uuid,text,text)TO bookkeeping_api;
COMMIT;
