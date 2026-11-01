# Finite Production acquisition

Source375; human review before actual grants/apply/exposure. Foundation already
owns identities, SQL users, buckets, keys and static secret destinations. This root
owns only publisher/consumer services, queue and narrow dispatch/caller grants,
always-paused scheduler and dedicated signed receiver. Null runtime is inert.
Use existing Production worker-runtime state bucket with a distinct production-
acquisition prefix. Never load/copy private state/plans/credential files into source.

Pin exact matching Production worker digest, configured single book binding,
numeric static versions and owner-reviewed UTC expiry <=7days. Do not generate a
new expiry at apply time. Receiver is private unless public_webhook_approved=true;
this opens only the signed receiver, not workers. Existing SQL/current scope and
all applicable signature/credential/Sandbox/recovery gates remain mandatory.

Both worker/receiver authority and the separate app pilot object must share the
reviewed deadline/static/key/profile metadata. App pilot=null retains expired
bank access and authorized SQL bookkeeping/recovery. Do not infer activation
from the independent credential-check readback deadline.

First apply is not scheduler resume or bank authorization. Independently verify
actual images/identities/SQL login/ingress, private worker denial and signed
receiver behavior. Manual Scheduler dispatch is separately reviewed; no silent
resume fallback. Stop: pause queue/scheduler and disable current book provider
scope/consent; API pilot=null preserves local recovery. Existing runtime expiry
also fences requests. Do not destroy retained evidence or use a Sandbox rollback.

See docs/contracts/finite-production-acquisition.md and its stage runbook. Capacity
min0/max1/CPU1/512Mi and queue1 controls do not guarantee USD25 total spend.
