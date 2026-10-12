# GCP foundation ownership and operations

## Decision

The development foundation was applied with owner authorization on 2026-10-12.
Terraform state was migrated to its private GCS backend. Build federation is
enabled for the merged owner-triggered, main-only image workflow; GitHub still has
no Terraform apply permission. Application/database deployment and live OIDC
exchange remain unverified. See the [evidence ledger](runbooks/gcp-delivery.md).

[Bootstrap instructions](../infra/gcp/foundation/README.md) own the exact commands.
The [delivery contract](gcp-delivery.md) owns the selected architecture.

## Applied bootstrap identity and state boundary

| Identity | Granted in this slice | Later owner |
| --- | --- | --- |
| Owner bootstrap | Existing separately supplied administrative authority; not minted here | Owner applies foundation and privately reviews state/plans |
| Builder | Writer on only the development image repository; narrowly scoped WIF impersonation, enabled for development | #27 builds/scans/promotes digests; no state, runtime impersonation or secrets |
| Deployer | Object admin on only application-state bucket; no WIF or cloud deployment rights | #27 adds minimum resource roles and a reviewed promotion gate |
| Migrator | Account only | #26 owns DDL role and bounded migration invocation |
| API, frontend | Separate accounts only | #26/#27 own database/connectivity and ingress privileges |
| Publisher, consumer | Separate accounts only | #7 owns task enqueue/handling and SQL receipt rights |
| Scheduler, task-invoker | Separate accounts only | #7 scopes invoke/token permissions to the specific handlers |

The table records the initial applied bootstrap. New, unapplied #27 source moves
deployer state access to a third runtime bucket and adds separately disabled
deployment federation; the [rollout contract](contracts/cloud-run-rollout.md) owns
its permissions and promotion gate. Foundation state has no application-deployer grant. Both buckets enforce uniform
access, public-access prevention, versioning and Terraform deletion guards. These grants
do not erase inherited owner/org permissions; verify effective IAM before apply.
Infisical impersonation/secret containers and values stay in #11; this foundation
creates no unused secret or machine credential.

## Promotion and recovery

Private main protection returned HTTP 403 on 2026-10-11; environments list was empty.
Neither a workflow label nor a manual convention is an enforced approval. Before
#27 enables publishing, owner reviews passing checks and initiates the exact main
workflow. Provider checks actor, immutable repository/owner IDs, event, ref and
workflow path. This limits build authority; repository administrators remain trusted.
No automatic apply or deployment follows a push, merge or issue label.

Only Terraform may own configured runtime revisions. #27 must serialize each dev
migration/deployment, accept an already scanned digest, stop on migration/readiness/
auth failure and roll back only to a schema-compatible digest. Runtime DDL is denied;
failed migrations never trigger force/automatic down. Retain last known good tagged
images; cleanup deletes only untagged images older than seven days.

On federation incident, owner disables pool/provider before reviewing trust;
already-issued access tokens can remain valid until expiry. Revoke scoped grants
where immediate containment is required. For state loss, owner restores a version
privately and verifies it against resources; never delete buckets or force-unlock
without identifying the lock owner. Default tests cannot exercise these cloud paths.

## Capacity, costs and acceptance inputs

Initial proposed DB pool budget for #26: API 5 per instance, publisher 2, consumer 2;
maxima 2/1/1 give 14 connections. Allow a second revision of each (28), two migrator
and 5 administrative connections: 35 total (migration lock plus execution). Verify the actual server connection limit
and load behavior; frontend has no SQL pool. This is a proposal for #26, not an
implemented pool setting or permission to exhaust the server.

Development recovery objectives proposed for owner acceptance: restore within four
hours and lose at most 24 hours of synthetic work through the backup path; PITR is
additionally tested. One-minute publisher activation is a schedule, not a delivery
SLA; measure enqueue/completion latency and investigate oldest pending work above
five minutes. No live-bookkeeping recovery objective is inferred from these defaults.

Budget alerts at 50%, 80% and 100% of USD 50 exclude credits and apply to the selected
project. They notify default billing recipients; owner must verify those recipients
and delivery in the cloud proof. Alerts never enforce a spend cap.

Reviewed Iowa SQL rates give an illustrative 730-hour baseline: db-g1-small at
$0.035/hour = $25.55; 10 GiB SSD at $0.000232877/GiB-hour = about $1.70;
10 GiB of used backup storage at $0.000109589/GiB-hour = about $0.80.
That subtotal is $28.05 before PITR growth, restore resources and other services.
Shared-core instances have no Cloud SQL SLA. See G10 in the
[source register](research/gcp-sources.md).

Planning envelope (not a quote): reserve $30 for database compute/initial storage,
$5 for backups/PITR and restore testing, $5 for application/publisher/consumer compute,
$1 for scheduler/tasks, $3 for state/artifacts/logging and $3 for authentication/secrets/
networking, leaving $3 contingency. These are allocation limits, not verified SKU
estimates. Require a regional calculator estimate before apply; do not assume free
allowances are unused across this billing account. Bank subscriptions are excluded.
Observed seven-day spend, synthetic restore timing, backlog latency and the 1,000-post
proof belong to #12. Project and billing metadata are now established and #5 is closed in GitHub.
Application auth, actual trust-denial tests, cost measurements and recovery
acceptance remain separate cloud evidence gates.
