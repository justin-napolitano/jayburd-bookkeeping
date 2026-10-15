BEGIN;
SET LOCAL ROLE bookkeeping_owner;
DROP FUNCTION ledger.clear_inaccessible_command(uuid,uuid,text);
COMMIT;
