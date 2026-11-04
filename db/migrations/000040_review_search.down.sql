-- Reverse only the new scoped read; immutable observations and journals are untouched.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION banking.search_review_items(uuid,uuid,uuid,text);
COMMIT;
