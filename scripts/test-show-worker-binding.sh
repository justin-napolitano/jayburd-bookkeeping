#!/usr/bin/env bash
# Check the lookup wrapper's bounded output without opening a database connection.
set -euo pipefail
cd "$(dirname "$0")/.."
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT
cat > "$fixture/psql" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "${JAYBURD_TEST_BINDING_RESPONSE:-}"
exit "${JAYBURD_TEST_BINDING_EXIT:-0}"
SH
chmod +x "$fixture/psql"
entity='9781d120-b826-4f77-949e-51ceda2c5e0f'
book='8b09e985-e348-4497-9bca-2b10eaf61417'
binding='11111111-1111-4111-8111-111111111111'
[[ "$(PATH="$fixture:$PATH" JAYBURD_TEST_BINDING_RESPONSE="$binding" scripts/show-worker-binding.sh "$entity" "$book")" == "$binding" ]]
if PATH="$fixture:$PATH" JAYBURD_TEST_BINDING_RESPONSE='' scripts/show-worker-binding.sh "$entity" "$book" >"$fixture/out" 2>"$fixture/err"; then exit 1; fi
[[ ! -s "$fixture/out" && "$(cat "$fixture/err")" == 'No active worker binding was found for that entity and book.' ]]
if PATH="$fixture:$PATH" JAYBURD_TEST_BINDING_EXIT=1 scripts/show-worker-binding.sh "$entity" "$book" >"$fixture/out" 2>"$fixture/err"; then exit 1; fi
[[ ! -s "$fixture/out" && "$(cat "$fixture/err")" == 'Lookup failed. Verify the dev Cloud SQL proxy, owner IAM login, and bookkeeping database.' ]]
if PATH="$fixture:$PATH" scripts/show-worker-binding.sh wrong "$book" >"$fixture/out" 2>"$fixture/err"; then exit 1; fi
[[ ! -s "$fixture/out" && "$(cat "$fixture/err")" == 'Entity and book IDs must both be UUIDs.' ]]
echo 'PASS: scoped worker binding wrapper'
