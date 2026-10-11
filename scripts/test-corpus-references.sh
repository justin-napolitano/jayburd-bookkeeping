#!/usr/bin/env bash
# Compare original synthetic fixtures with pinned external engines, offline.
set -euo pipefail
cd "$(dirname "$0")/.."
hledger_bin=${1:-.local/external/hledger/hledger}
[[ -x "$hledger_bin" ]] || { echo 'Install the pinned hledger tool described in tests/corpus/README.md.' >&2; exit 1; }
"$hledger_bin" --version | grep -F 'hledger 1.52.4-' >/dev/null
uv tool run --offline --python 3.12 --from beancount==3.2.3 bean-check --version | grep -Fx 'Beancount 3.2.3' >/dev/null
mkdir -p .local/corpus
for pair in golden-month:a-usd golden-month:b-usd fx-lifecycle:fx-full fx-lifecycle:fx-partial fx-lifecycle:fx-conversion; do
  corpus=${pair%%:*}
  book=${pair#*:}
  base=".local/corpus/$book"
  go run ./cmd/corpus export "tests/corpus/$corpus.json" "$book" hledger > "$base.journal"
  go run ./cmd/corpus export "tests/corpus/$corpus.json" "$book" beancount > "$base.beancount"
  "$hledger_bin" -f "$base.journal" check
  uv tool run --offline --python 3.12 --from beancount==3.2.3 bean-check "$base.beancount"
  echo "PASS: $book fixed functional-balance assertions accepted by both engines"
done
# An intentionally wrong independent expectation must be rejected by both engines.
sed 's/1045 USD/1046 USD/' .local/corpus/a-usd.journal > .local/corpus/wrong-balance.journal
sed 's/1045 USD/1046 USD/' .local/corpus/a-usd.beancount > .local/corpus/wrong-balance.beancount
if "$hledger_bin" -f .local/corpus/wrong-balance.journal check > .local/corpus/hledger-negative.log 2>&1; then
  echo 'FAIL: hledger accepted wrong expected balance' >&2; exit 1
fi
if uv tool run --offline --python 3.12 --from beancount==3.2.3 bean-check .local/corpus/wrong-balance.beancount > .local/corpus/beancount-negative.log 2>&1; then
  echo 'FAIL: Beancount accepted wrong expected balance' >&2; exit 1
fi
echo 'PASS: external engines rejected wrong fixed expectations; FX-policy equivalence is not claimed'
