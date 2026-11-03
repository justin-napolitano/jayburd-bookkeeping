# Sandbox money-workspace audit —412

Authenticated hosted Sandbox in owner-selected Edge,2026-11-03UTC.
Desktop1720×954 and temporary390×844 viewport; default restored afterward.
13current-run screenshots inspected and saved in operator-local audit folder.
Only fictional Sandbox activity was inspected. No Production DOM, source
provider call, account mutation, saved label/draft, download or posting.

| Captures | Finding | Issue |
| --- | --- | --- |
|01–03 | Zero posted Position dominates; Transactions/Connections hidden in Tools; explicit book detour |407/412/418 |
|04–05 | Duplicate heading, large gaps, prominent accounting dates and repeated source selection |415/226 |
|05/12 | Sample18rows; no search/category/date controls or sortable headers in this observed Sandbox UI |408/415;406 already shipped elsewhere |
|06 | Sample detail appended below list; categorization unavailable with retry and no actionable reason |416/415/291 |
|07/09 | Connection card/actions use long provider/account IDs; Disconnect has equal visual weight |290/104 |
|09 | Last publication/provider unknown available, but buried beneath recovery; generation/lease terminology |408/417/418/420 |
|08/13 | Oversized comparison checkbox separated from label; reports require refresh without useful initial state |420 |
|10–12 | Mobile first view contains context/instructions instead of rows; wrapping dates/names, offscreen actions |415/420 |
|all | Idle recovery/setup and technical provenance distract; selected-book duplicate/h2 route titles |420 |
|inspected navigation | No category/merchant spending or opportunities workflow visible |413/414/417–419,242/243 |

Strengths: restrained design, exact currency labels, explicit scope, semantic
tables/native controls, focused detail and honest source-versus-posted boundary.
Recommendations preserve those while making daily activity immediately useful.
An early Connections DOM briefly reported unavailable during loading, then
displayed an active card: this is not evidence of persistent connection failure.

Limits: underlying categorization error undiagnosed, large-page performance not
tested in this hosted audit, no image-digest parity check and no Production
regression inferred from absent Sandbox sort controls. No full keyboard,
screen-reader, measured contrast or zoom audit; screenshots do not prove
accessibility or accounting correctness. Source-reviewed synthetic tests and
fresh visual captures belong to each delivery issue.

Operator-local report: /private/tmp/jayburd-sandbox-ui-audit/report.html. It is
not an external/public hosted artifact or a future-session dependency. The issue
map and this finding table retain the design evidence without publishing account
identifiers;407's new synthetic fixture screenshots document its actual change.
