# Image dependency patch sources — 2026-11-07

Scope: issue 454, rejected owner/main build 42341240949, blocking Sandbox 436.
No financial message, accounting semantics or ISO usage profile changes.

## IS01 — Go HTML parser advisory

Go Security Team, [GO-2026-6860](https://pkg.go.dev/vuln/GO-2026-6860),
published 2026-10-27, reviewed 2026-11-07. Read the complete description,
affected package/version/symbol table and references. It identifies quadratic
text coalescing in golang.org/x/net/html before v0.62.0, affecting Parse and
fragment/options variants. The project has no direct x/net/html imports; the
scanner finds module metadata in distributed Go binaries. Reachability is not
established here, and the release gate intentionally rejects installed findings.
The module-wide patch upgrades every binary through the shared go.mod pin.
Official module download resolved tag v0.62.0 to Go source commit
3de643db506517b3e652a13cc0b56034cea6b8e6 and verified module checksums.

## IS02 — Alpine runtime packages

Alpine Linux, [musl package](https://pkgs.alpinelinux.org/package/v3.24/main/x86_64/musl),
[musl-utils package](https://pkgs.alpinelinux.org/package/v3.24/main/x86_64/musl-utils),
[zlib package](https://pkgs.alpinelinux.org/package/v3.24/main/x86_64/zlib),
and [v3.24 security database](https://secdb.alpinelinux.org/v3.24/main.json),
access attempted 2026-11-07. Search discovery exposed musl 1.2.6-r3 and
v3.24 zlib 1.3.2-r1 on another architecture; the full relevant pages/database
were unavailable through the web tool, host DNS, and a credential-free Docker
fetch (certificate validation failure). No TLS bypass was used. This is an
access limitation, not full advisory review.

Exact repair bounds come from retained Trivy 0.75.0 rejected artifacts:
13562536146 reports CVE-2026-107891 in musl/musl-utils 1.2.6-r2, fixed r3,
and CVE-2026-85091 in zlib 1.3.2-r0, fixed r1. Extend the existing runtime
APK minimum-version mechanism on the unchanged digest-pinned Node 24.18.0 base.
Alternative: advance the whole base image, changing more runtime dependencies
and requiring a new digest review. Both require an actual final-image scan;
version constraints alone are not evidence that the image is clean.

## Acceptance and limits

Preserve scanner severity, ignore-unfixed=false, exact runtime proof, trusted
identity gates and no-rebuild publication. Source checks and runtime compatibility
must pass; a fresh trusted main build must report zero HIGH/CRITICAL findings
for all four candidates before rollout. No exploitability, SLSA or security
certification claim follows from these dependency changes.
