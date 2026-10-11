-- Synthetic tooling fixture; never part of the product schema.
BEGIN;
CREATE TABLE migration_probe (id integer PRIMARY KEY);
INSERT INTO migration_probe VALUES (1);
COMMIT;
