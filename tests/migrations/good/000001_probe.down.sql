-- Destructive reversal is limited to the disposable fixture database.
BEGIN;
DROP TABLE migration_probe;
COMMIT;
