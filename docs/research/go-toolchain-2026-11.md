# Go patch toolchain sources — 484

Reviewed2026-11-11 for the synthetic acceptance blocker on482/PR483.
This changes build toolchain selection, not a financial message, currency mapping,
accounting policy or database contract; existing standards applicability remains.

| Source | Review depth | Evidence / limit |
| --- | --- | --- |
| [Go advisory JSON](https://vuln.go.dev/ID/GO-2026-6859.json) | Entire public JSON via credential-free IPv4 curl; browser retriever unavailable | REVIEWED GO-2026-6859/CVE-2026-97054; net/http MaxBytesReader header race can crash an HTTP/1 server under the described concurrent body/header handling. Fixed1.26.11 or1.27.4. Scanner reachability is not an application exploit demonstration. |
| [Official release history](https://go.dev/doc/devel/release#go1.26.11) |1.26.11 paragraph and release policy |1.26.11 released2026-11-10, including net/http and other security fixes.1.26 is still supported. |
| [Go downloads](https://go.dev/dl/?mode=json) | Selected1.26.11 stable record and platform entries | Official patch is available; no system-wide installer or tracked artifact downloaded. |
| [Official Docker image tag metadata](https://hub.docker.com/v2/repositories/library/golang/tags/1.26.11-alpine) | Entire public JSON via credential-free curl; browser retriever unavailable | OCI index sha256:ae52b5bc6ed045a9ed26ab03dc19327a802a9cc9aa8b27ed57f5c6b60579b0bb; linux amd64 child f496f8c537ed506626ee1a6913afb32c45068a48697163c088f604cc775a0c38 and arm64 child f00ba21309295da612e60611d28ad4c7b9424a74cb42b9fc7ecebaeb83c387a2. Public metadata establishes selected pin; canonical build/container checks must execute it. |

Choose the smallest supported patch1.26.10→1.26.11 consistently in module,
CI and Go builder/probe references. Deferring the patch or suppressing scanner
findings would leave the required gate failing. Moving to1.27.4 adds an unrelated
major language/runtime upgrade and is unnecessary for this fix. Keep image
digests, dependency requirements, checks and timeouts otherwise intact.

Acceptance: exact-head Seed uses1.26.11 for check/test/build and unchanged
govulncheck, executes source-backed restore/load and existing isolated container
proofs. A source/build patch does not remediate already deployed binaries; no
deployed runtime, real-data readiness or exploit-remediation certification is
claimed in this synthetic-only phase.
