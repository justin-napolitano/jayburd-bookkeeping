<!-- stage-runbook: true -->
# Offline Production candidate — #316 stage B

## Status and boundary

PROD316-CANDIDATE; build/316-production-candidate from4e4a172. Source only.
No runtime/IAM/SQL/cloud/provider/credential/private data or spending changes.

## Stage ledger

| Stage | Evidence |
| --- | --- |
| Resume | Live316/337 open;308/311–315 source merged;352 merged after all five checks |
| Design | Closed candidate-only public metadata schema, exact-byte hash, seven-day window, numeric pins, paused defaults; no deploy emitter |
| Implementation | Standalone validator and synthetic negative fixtures; make test wiring |
| Validation | 34 focused release tests PASS; make check/test/build PASS; staged gate required |

## Current evidence

Installed Go1.26.5 with external temporary modfile/cache used locally; tracked
Go1.26.10 unchanged. No product integration gates needed for offline tooling.
Candidate validation cannot establish evidence truth or grant activation. Four
image components include the worker image used by the webhook command. Subsequent
build verification must prove that command and source ancestry. No receipt hashes
are presented as real approvals. USD25/month GCP target excludes Plaid costs.

## Remaining work

Independent trusted receipt/build/ancestry checks, Production migration identity,
infrastructure/workflow composition and actual337 decision/hosted qualification.
Human security review before merging this sensitive release-validation code.

## PR handoff

Refs #316, partial stage B only; parent remains open. No migration or API changes.
Synthetic fixtures contain no real identities or credentials. Existing Sandbox
release validators remain unchanged. No deployment or provider activation.
