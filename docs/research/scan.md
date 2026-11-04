# Same-version scanner installer —437

Reviewed2026-11-04, public first-party source only. No financial records, private
environment, credential, provider data, Terraform state/plan or cloud mutation.

| Source / reviewed depth | Fact | Project choice / limit |
| --- | --- | --- |
| Aqua Trivy [installer at75c4dc0](https://github.com/aquasecurity/trivy/blob/75c4dc0f45c5d7ffd05ae26df1e0c666787bdf2a/contrib/install.sh), download construction/client/checksum/curl functions | v0.75.0 installer routes the binary through get.trivy.dev; curl exit6 stops installation. | Both hosted attempts and independent local request failed DNS before scan; not a vulnerability finding or passed scan. |
| Official [release asset API](https://api.github.com/repos/aquasecurity/trivy/releases/tags/v0.75.0), named Linux64bit asset metadata; [asset](https://github.com/aquasecurity/trivy/releases/download/v0.75.0/trivy_0.75.0_Linux-64bit.tar.gz) downloaded without execution | API digest SHA256c6e65abddb348e25f10549df887045629cf28cc72453cd1c63acb717316b3f3f matches independently downloaded bytes. | Use that exact existing version/hash/official URL. SHA matching does not establish tool safety or scan accuracy; no signature/SLSA claim. |
| [Trivy action ed142fd](https://github.com/aquasecurity/trivy-action/blob/ed142fd0673e97e23eac54620cfb913e5ce36c25/action.yaml), skip-setup input, setup step and scanner environment propagation | Supports separate installation while retaining the same scanner invocation/report/action. | Disable built-in **installation** only after pinned verified install; severity HIGH,CRITICAL, unfixed inclusion, exit1, actual image and report remain unchanged. |

Decision: repair Sandbox build-only installation with explicit hosted context,
bounded TLS download and checksum before extraction, binary execution or PATH
publication. No alternate version/user-selected URL/hash, fallback unverified
binary, scan suppression, new product dependency or Production workflow change.
Alternative retry was attempted once and reproduced the DNS failure. Downgrading
or skipping scanning would not preserve the release boundary. No upstream code
copied. Existing threat: compromised/mismatched download→exact fixed hash before
execution; high-impact CI context→install before cloud auth, existing owner/main
WIF unchanged. Human requested deployment and this bounded prerequisite repair;
actual hosted scan success remains required before any image promotion.

Named evidence: test-image-scanner-install.sh rejects default-local invocation and
corrupt bytes before extraction/execution/PATH; core/release/actionlint checks;
actual hosted same-version installer, candidate runtime and unchanged HIGH/CRITICAL
scan/publication. Synthetic negatives alone do not establish positive installation.
No financial message/ISO mapping change; existing G15 image/provenance limits apply.
