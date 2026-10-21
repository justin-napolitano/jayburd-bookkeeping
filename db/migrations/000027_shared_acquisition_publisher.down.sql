BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION integration.sync_delivery_bindings(integer);
DROP TABLE integration.sync_publisher_cursor;
DROP INDEX integration.sync_delivery_publisher_page;
COMMIT;
