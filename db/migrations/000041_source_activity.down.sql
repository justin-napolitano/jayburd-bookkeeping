-- Remove only the read boundary; immutable imported and financial history stays intact.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION banking.query_source_activity(uuid,uuid,uuid,text);
DROP FUNCTION banking.activity_facts(uuid,uuid,uuid);
DROP INDEX banking.review_assignment_activity_review;
COMMIT;
