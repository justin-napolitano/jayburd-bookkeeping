-- Read-only query/detail functions retain no business data; existing v1/v2 paths remain.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION banking.get_collection_detail(uuid,uuid,text);
DROP FUNCTION banking.query_collection_review_v3(uuid,uuid,text);
DROP FUNCTION banking.get_collection_taxonomy_v3(uuid,uuid);
DROP FUNCTION banking.collection_query_amount(jsonb,integer);
DROP FUNCTION banking.collection_query_rows(uuid,uuid);
COMMIT;
