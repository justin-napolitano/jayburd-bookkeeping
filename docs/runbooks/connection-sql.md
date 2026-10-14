# Connection SQL delivery
<!-- stage-runbook: true -->

## Status and boundary

#101/#102; lease CONNECTION-SQL-101. Implemented migration 13 synthetic durable
connection/credential-reference foundation. No tokens, provider calls, secret-store
writes or cloud mutation. Go connection/acquisition lanes own injected orchestration;
passing SQL does not establish a hosted connection or close live qualification gates.
Contract: [connection SQL](../contracts/connection-sql.md).

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Review | AGENTS/index/bibliography/provider lifecycle and SQL 12; official Items and update-mode selected sections refreshed. |
| Contract | Coordinator approved explicit connect permission, owner-configured worker identity, one-shot claims, explicit bounded provider-first grants and lifecycle fences before DDL. Exact functions shared with both Go lanes. |
| Implement | Migration 13 adds actor/scope/expiry/nonce attempts, one-shot claim and manifest-before-secret publication, immutable credential receipts, explicit binding, scoped management and current acquisition context/resume. |
| Review refinements | Same-Item/account verification, precise affected-stream pins, per-account vs Item fences, exact UTC expiry and malformed-container diagnostics; worker configuration disable explicitly distinguished from revoking enrolled acquisition. |
| SQL proof | Disposable PostgreSQL migrations 1–13 plus **12 top-level groups and 5 named subtests PASS**. Actual operator/API/sync logins; no role impersonation at runtime boundaries. |
| Default checks | `make check`, `make test`, `make build` PASS with Go 1.26.9 and isolated caches. |
| Integration | Coordinator owns runner/Make/CI/schema-version and prior privilege-fixture extensions; shared combined regression runs after this commit is integrated. |

## Current evidence

Executed on 2026-10-13:

```sh
PATH=/private/tmp/jayburd-toolchain-1.26.9/go/bin:$PATH \
GOCACHE=/private/tmp/jayburd-go-cache GOMODCACHE=/private/tmp/jayburd-go-modcache \
bash scripts/test-connection-state.sh
```

The host Docker credential helper required normal sandbox escalation. Tests used
only private disposable synthetic PostgreSQL databases and fixed fictional logins;
no inherited DSNs or credential files. Proof log:
`/private/tmp/jayburd-connection-sql-proof.log`.

- `TestConnectionOneShotAndRecovery`: first-only Link/exchange execution grants,
  uncertain exchange not retried, manifest-before-secret confirmation, resolved
  reference acknowledgement and explicit binding without journal creation.
- `TestConnectionProviderFirstAndOverlap`: two new provider sources, bounded explicit
  source grants, safe public status/listing and overlap hold for an existing source.
- `TestConnectionRotationAndDisconnect`: one account disconnect leaves sibling
  acquisition valid; rotation fences shared Item before remote work, publishes a new
  reference/version, preserves disconnected state and rejects old worker claims.
- `TestConnectionRemovalUncertainty`: unknown provider reply remains pending; exact
  confirmation/replay disconnects every approved sibling without blind removal retry.
- `TestConnectionReconnectIdentity`: different Item or missing bound account rejected;
  same Item update preserves source identities/reference/version.
- `TestConnectionClaimCommitRecovery`: deliberately discard an application-visible
  committed claim reply, then contend two real replay sessions. Both return no
  execution permission; exactly one reservation event. This models reply loss, not
  an injected network partition inside PostgreSQL COMMIT.
- `TestConnectionRevocationAndScope`: current read/import/connect/source-bind,
  wrong entity and disabled service gates apply to successful attempt reads/replay.
- `TestConnectionManifestAndBindingGates`: unknown token field and duplicate provider
  metadata rejected; five unverified/account-class/currency/grant/duplicate selection
  subtests leave no partial binding.
- `TestConnectionWorkerContextAndPrivileges`: current fenced opaque-reference lookup,
  explicit absent resume, role/table/helper denials and immutable claim identity.
- `TestConnectionExpiryAndAtomicBinding`: strict UTC/expired handoff; one invalid
  account rolls back all selected source/grant publication.
- `TestConnectionMalformedContainers`: null/object/scalar account and binding
  containers return fixed diagnostics, including malformed nested entries.

Default logs: `/private/tmp/jayburd-connection-check.log`,
`/private/tmp/jayburd-connection-test.log`, `/private/tmp/jayburd-connection-build.log`.
Connection-service and acquisition integration evidence belongs to their lanes and
the coordinator's combined runbook; this lane does not claim their results.

## Remaining work

Live Link/session-association qualification, provider/Infisical permissions and
verified secret-store adapters, approved retention/cleanup, hosted execution and
Sandbox/production acceptance remain explicit external gates. Existing runtime
Infisical read-only policy is not expanded by SQL reference publication. New accounts
are not silently added to existing Items; a separately reviewed binding expansion
must approve their scope. Overlap release and posted correction resolution remain
separate; source and journal evidence is retained across disconnect/removal.

An exchange_claimed attempt after a crash is unresolved recovery, never retry
permission. No lease-based claim takeover or automatic remote retries exist. A lost
secret acknowledgement resolves only its preallocated reference/verified manifest.
A configuration change affects connection management enrollment authority; stopping
existing acquisition requires its current grant revocation or Item/account lifecycle.

## PR handoff

Owned files: migration 13 pair/checksum, `internal/connectionstate` SQL tests,
contract and this runbook. Migrations 1–12 untouched. Coordinator's copied
`scripts/test-connection-state.sh` stays untracked and is not committed here.
Stage guard and diff check required before every implementation commit. No push,
provider access or deployment from this lane.

## Follow-up: credential context and exclusive remote operations

Independent service review found that current actor authorization alone did not
revalidate Item generation before credential reads. Private context now rejects
stale nonterminal Item/sibling pins and expired unclaimed attempts while public
status stays readable. Begin and first claim reject Items with unresolved remote
operations, preventing overlapping rotation/removal/reconnect calls.

`TestConnectionPrivateContextFenceAndExclusiveClaim` PASS in the disposable
PostgreSQL proof: two attempts prepared before a claim, rejection of the second
claim/context, rejection of all new lifecycle modes after the claim, generation
change rejection before credential access, and retained public status reads.
The focused proof now passes 12 top-level groups and 5 named subtests. Unknown
operations without a persisted manifest still require separately reviewed recovery;
there is no automatic retry or fresh-attempt escape.
