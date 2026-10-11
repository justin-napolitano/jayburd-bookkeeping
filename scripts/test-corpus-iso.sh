#!/usr/bin/env bash
# Exercise a pinned provisional schema with original synthetic messages, offline.
set -euo pipefail
cd "$(dirname "$0")/.."
schema=${1:-.local/external/camt.053.001.14.xsd}
expected=fdad51c5748d8b05056bb6dc140952e6c87ec749e63368df6834e4f36f9e6a9f
actual=$(shasum -a 256 "$schema")
[[ "${actual%% *}" == "$expected" ]] || { echo 'Unexpected schema digest' >&2; exit 1; }
mkdir -p .local/corpus
xmllint --nonet --noout --schema "$schema" tests/corpus/iso/statement.xml
sed 's/camt.053.001.14/camt.053.001.99/' tests/corpus/iso/statement.xml > .local/corpus/wrong-namespace.xml
sed 's/>200.00</>-200.00</' tests/corpus/iso/statement.xml > .local/corpus/negative-amount.xml
for invalid in wrong-namespace negative-amount; do
  if xmllint --nonet --noout --schema "$schema" ".local/corpus/$invalid.xml" > ".local/corpus/$invalid.log" 2>&1; then
    echo "FAIL: schema accepted $invalid" >&2; exit 1
  fi
done
# XSD cannot establish financial consistency: this mismatch still validates.
sed 's/>1200.00</>9999.00</' tests/corpus/iso/statement.xml > .local/corpus/wrong-closing-balance.xml
xmllint --nonet --noout --schema "$schema" .local/corpus/wrong-closing-balance.xml
echo 'PASS: schema smoke and negative cases; financial semantics and official provenance remain separate gates'
