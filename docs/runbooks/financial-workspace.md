<!-- stage-runbook: true -->
# Financial workspace review

## Status and boundary

PARTIAL: issue #52 synthetic design prototype. No application route, auth bypass,
posting action, cloud resource, financial export, live data or browser persistence.
Owner review and real browser acceptance remain pending.

## Stage ledger

- 2026-10-13: reviewed current UI boundary, corpus, source discipline and FW01–02.
- Built the standalone [preview](../prototypes/financial-workspace.html) with
  exact corpus copies and bounded navigation/state demonstrations.
- Browser skill setup reported `Browser is not available: iab`; read its
  troubleshooting guidance. No visual inspection or real keyboard claim.
- FW-DOM automated synthetic checks passed using the existing local jsdom
  dependency; no dependency or application package changed.

## Current evidence

FW-DOM checks compare embedded books to the checked-in corpus, assert exact
fixed money strings, report → account → journal → return navigation, preserved
period/currency selections, focused heading, entity-change stale detail clearing,
separate all-LLC cards, February empty-data warning, loading/error/denied/stale
hiding, explicit retry, pending capabilities, and absent network/storage APIs.
This is DOM behavior evidence; it cannot verify layout, contrast, native keyboard
interaction, screen-reader output or runtime authorization.

Repository `make check test build`: PASS on 2026-10-13 (Go vet, race tests,
contract/link checks, migration artifact checks and command builds).
No SQL/API changes; database integration tests are outside this design-only scope.
Coordinator reran FW-DOM successfully after integration and aligned the prototype
labels with #51: selected-period income and cumulative unclosed earnings. The
prototype still defines no report API/schema; its fixed figures are illustrative.

## Remaining work

Owner: open the HTML locally, review report meaning/navigation, then record
approval or corrections in #52. Browser reviewers: inspect desktop and narrow
viewport layouts, Tab/Enter flows, skip link, focus contrast and table narration.
Implementation agents: use #51 reviewed server report contracts; retain current
identity boundary; prove runtime scope/revision handling. No implementation
acceptance checkbox is satisfied by a static response-state simulation.

## PR handoff

Issue #52 stays open. Files are confined to design, prototype, source notes and
this runbook. Root coordinator links the source record and merges independent
agent work. Lease is ready for REVIEW after passing required checks and commit.

### Reproduce FW-DOM

After installing the locked web dependencies, run from the repository root:

```sh
node <<'NODE'
const fs = require('node:fs');
const assert = require('node:assert/strict');
const { JSDOM } = require('./web/node_modules/jsdom');
const html = fs.readFileSync('docs/prototypes/financial-workspace.html', 'utf8');
const dom = new JSDOM(html, { runScripts: 'dangerously' });
const d = dom.window.document;
// Use only the trusted checked-in synthetic HTML; never execute supplied documents.
function click(selector) { assert(d.querySelector(selector)); d.querySelector(selector).click(); }
// Model selector changes without claiming native keyboard or browser coverage.
function select(id, value) { d.getElementById(id).value = value; d.getElementById(id).dispatchEvent(new dom.window.Event('change')); }
// Inspect the current financial surface, excluding embedded fixture source.
function surface() { return d.getElementById('surface').textContent; }
assert.deepEqual(JSON.parse(d.getElementById('fixture').textContent), JSON.parse(fs.readFileSync('tests/corpus/golden-month.json')).books);
assert(surface().includes('1,145.00'));
click('[data-view="Reports"]'); assert(surface().includes('130.00'));
select('currency', 'original'); click('[data-account="Assets:Checking"]');
click('[data-journal="AC-02"]'); assert(surface().includes('200.00'));
assert.equal(d.activeElement.id, 'heading');
click('[data-back="journals"]'); click('[data-back="report"]');
assert.equal(d.getElementById('currency').value, 'original');
assert.equal(d.getElementById('period').value, '2026-01');
click('[data-account="Assets:Checking"]'); click('[data-journal="AC-02"]');
select('entity', 'b-usd'); assert(!surface().includes('AC-02')); assert(!surface().includes('1,145.00'));
select('entity', 'all'); assert(surface().includes('not a consolidated')); assert.equal(d.querySelectorAll('[data-book]').length, 2);
click('[data-book="a-usd"]'); select('period', '2026-02'); assert(surface().includes('not a verified zero'));
select('period', '2026-01');
for (const state of ['loading', 'error', 'denied', 'stale']) { select('state', state); assert(!surface().includes('1,145.00')); assert(!surface().includes('AC-02')); }
click('[data-retry]'); assert(surface().includes('1,145.00'));
click('[data-view="Review inbox"]'); assert(surface().includes('not pending ledger commands'));
click('[data-view="Reconciliation"]'); assert(surface().includes('planned'));
assert(!/fetch\(|localStorage|sessionStorage|XMLHttpRequest/.test(html));
console.log('PASS FW-DOM');
NODE
```

Observed run used the same locked jsdom from an existing local worktree cache;
replace only the module path if reusing that cache. Browser validation remains
separate, as documented above.
