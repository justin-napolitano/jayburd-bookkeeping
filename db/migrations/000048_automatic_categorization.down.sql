-- Empty-only downgrade: retained automatic decisions and source attribution are evidence.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE banking.collection_workspace,banking.collection_label_state IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE integration.sync_category_page,banking.provider_category_revision,banking.collection_automatic_run,banking.collection_automation_setting_receipt IN ACCESS EXCLUSIVE MODE;
DO $$BEGIN
 IF EXISTS(SELECT FROM integration.sync_category_page)OR EXISTS(SELECT FROM banking.provider_category_revision)
 OR EXISTS(SELECT FROM banking.collection_automatic_run)OR EXISTS(SELECT FROM banking.collection_automation_setting_receipt)
 OR EXISTS(SELECT FROM banking.collection_automation_state WHERE initialized OR revision<>0)
 THEN RAISE EXCEPTION 'automatic_categorization_retained_records';END IF;
END $$;
DROP FUNCTION banking.collection_query_rows(uuid,uuid);
ALTER FUNCTION banking.collection_automatic_previous_rows(uuid,uuid)RENAME TO collection_query_rows;
DROP FUNCTION banking.query_collection_review_v3(uuid,uuid,text);
ALTER FUNCTION banking.collection_automatic_previous_query(uuid,uuid,text)RENAME TO query_collection_review_v3;
GRANT EXECUTE ON FUNCTION banking.query_collection_review_v3(uuid,uuid,text)TO bookkeeping_api;
DROP TRIGGER collection_automatic_initialize ON banking.collection_workspace;
DROP TRIGGER preserve_automatic_run ON banking.collection_automatic_run;
-- Closed namespace list; no CASCADE can erase another slice's retained dependency.
DO $$DECLARE f regprocedure;BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE (n.nspname='banking'AND(p.proname LIKE 'collection_automatic_%'OR p.proname LIKE 'collection_automation_%'
 OR p.proname IN('automatic_category_metadata','get_collection_automation','set_collection_automation','prepare_collection_automatic_run','execute_collection_automatic_run')))
 OR(n.nspname='integration'AND p.proname IN('stage_sync_category_page','sync_category_context','complete_sync_categories'))
 LOOP EXECUTE format('DROP FUNCTION %s',f);END LOOP;
END $$;
ALTER TABLE banking.collection_automation_state DROP CONSTRAINT collection_automation_state_workspace_id_last_run_id_fkey;
DROP TABLE banking.collection_automatic_decision,banking.collection_automatic_run,banking.collection_automation_setting_receipt,banking.collection_automation_state,banking.collection_automation_edition;
DROP TABLE integration.sync_category_completion,banking.provider_category_revision,integration.sync_category_page;
DROP TABLE reference.analytic_category_pair,reference.analytic_category;
COMMIT;
