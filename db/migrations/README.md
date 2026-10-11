# SQL migration workflow

No product migration is approved yet. SQL fixtures live in `tests/migrations/`
and are never deployed. First review [database contracts](../../docs/contracts/database.md).

```sh
make migration-create NAME=initial_ledger
# Complete the contract and both generated directions; generated SQL fails until edited.
make migration-seal
make migration-check
make migration-test
```

Docker is needed for creation/testing; offline file checks need Bash, Git and
`shasum`. Pulls happen only in explicit Docker commands. The test uses an internal
network, no published ports, tmpfs data and disposable trust authentication.
No real password, inherited DSN, Infisical or GCP access is used. Only resources
created by that test are removed. Do not attach other containers to its network.

Use one reviewed logical change per sequential pair. Rebase before numbering;
resolve concurrent numbering collisions before merge. Once merged, do not edit
or rename SQL: add a new forward migration. `migration-check` checks the manifest;
CI also compares SQL with the PR base to reject modification/deletion. Resealing
is for new reviewed migrations, not a way to rewrite applied history.

Use explicit `BEGIN`/`COMMIT` for ordinary DDL. A down migration must really reverse
the change where safely possible; if not, raise an explicit exception explaining
the forward-repair/restore requirement. A comment-only down falsely advances the
version without reversing the schema. Never use production down as app rollback.

The harness tests fresh install, replay, synthetic down/up, transactional SQL
failure and dirty-state rejection, then applies any approved product migrations
to another fresh database. It does not reverse product migrations automatically.
Dirty state requires inspection and an incident-specific repair decision, never
an automatic `force`. Production migration execution and applied-checksum receipts
remain JB-003/JB-005 work. No live-database wrapper is exposed by this seed.
