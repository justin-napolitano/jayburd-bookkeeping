-- Verify that a failed transactional migration leaves no partial DDL.
BEGIN;
CREATE TABLE must_rollback (id integer PRIMARY KEY);
SELECT 1 / 0;
COMMIT;
