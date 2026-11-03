# Categorization/rules source review —412/413/414

## Scope, review and limitations

Owner requested reusable categorization and explainable rules before a custom
model. Reviewed2026-11-03 public sources only; no private records or provider
payloads. This register supports proposed design, not a claim of integrated
classification, independent accuracy or approved financial recognition.

| ID / source | Reviewed sections/version | Establishes and limits |
| --- | --- | --- |
| CR01 [Actual rules](https://actualbudget.org/docs/budgeting/rules/) | How Rules Work, Condition Types/Fields, Automatic Rules; mutable public documentation, selected sections read | Exact/contains/regex, ordered stages, specific overrides and editable correction-derived payee/category rules. Their ordering and broad actions are not automatically Jayburd policy. |
| CR02 [Actual license](https://github.com/actualbudget/actual/blob/master/LICENSE.txt) | Complete MIT license text, current master | Copyright/permission notices accompany substantial copied software. No source copied and no dependency added in407; selected files must be pinned/reviewed at reuse. |
| CR03 [hledger1.52](https://hledger.org/1.52/hledger.html#csv) | CSV conditional if blocks, matchers and if tables, selected sections read | Field/description matching and ordered conditional assignments. Useful explicit examples; CSV account assignment differs from private spending labels. No runtime/code copied or license-conformance claim. |
| CR04 [Plaid Transactions API](https://plaid.com/docs/api/products/transactions/) / [PFC migration](https://plaid.com/docs/transactions/pfc-migration/) | PFC fields/confidence/version and v1/v2 migration selections | Provider supplies primary/detailed/confidence/version; version defaults differ by enablement date. Proprietary enrichment, not OSS or an ISO category standard; current Item taxonomy/content not inspected.413 must pin supported wire and source revision mapping. |
| CR05 [Plotnikova et al.,2018](https://publications.kse.ua/publications/building-payment-classification-models-rules-1421) | Author-affiliated publication summary/abstract only; full chapter unread | Describes rule bootstrap followed by customer labels and ML; identifies maintenance/ambiguity limitations. Reported case-study outcomes are not Jayburd thresholds or independently reproduced evidence. |
| CR06 [Toran et al.,2023](https://arxiv.org/abs/2305.18430) | v2,10June2023; abstract only, full paper/code unreviewed | Heuristics/domain knowledge for weakly supervised classification. Preprint, not automatic peer-review or an available licensed production rule corpus. Deferred model research only. |

## Interpretation, alternatives and acceptance

Our proposed inference: attributed Plaid fallback plus private approved rules
offers useful explainability before a model. Compare provider-only (little
personalization), a universal hand-maintained merchant list (ambiguous merchants/
maintenance), and model-first (privacy/evaluation/operational burden). Do not
assume a merchant uniquely determines personal/business purpose or loan split.
Actual supports a correction/preview workflow, but its last-writer conflict
policy is not accepted here:414 must define deterministic specificity and
equal-priority abstention/owner resolution. Preserve original provider evidence.

414 owns bounded matching, precedence/scope/version/replay and preview-to-apply
fixed tests.413 owns provider unknown/version/correction/attribution fixtures.
417 owns accounting/ISO4217/financial semantic mappings before aggregate SQL.
Source updates require edition/access-date review; no citation alone authorizes
DDL, provider backfill, private training, automatic journals or cloud activation.

## 407 standards and security assessment

407 changes navigation to existing authorized reads, with no new financial wire,
amount/currency/date mapping or SQL/API. ISO20022 message selection and audit
export mapping are not applicable to this navigation-only slice; no exchange/
export is introduced. Existing ISO4217 metadata/decimal projections remain
unchanged.413/414/417 require fresh applicability records for their contracts.
Restricted-data trust boundaries remain the existing verified browser/API/book/
SQL authority and command recovery. No new integration, dependency, grant or
browser persistence. Scope/denied/lock/no-mutation tests are407's relevant threat
mitigations. Human PR review still precedes merge; no compliance claim.
