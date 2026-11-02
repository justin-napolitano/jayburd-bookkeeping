import { useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import type { components } from "../lib/api.generated";

// The frozen SQL snapshot supplies display/eligibility; labels never establish account identity.
type Options = components["schemas"]["RegistrationOptions"];
type Reason = NonNullable<Options["accounts"][number]["reason"]>;
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
const decimal = /^(0|[1-9][0-9]*)$/;
// These explain server decisions; this module never reimplements product or currency eligibility.
const reasons: Record<Reason, string> = {
  already_registered: "Registered",
  source_disconnected: "Disconnected; use connection recovery",
  unsupported_account:
    "This account type is not supported by the current profile",
  currency_unavailable: "A matching dated currency reference is unavailable",
  currency_ambiguous: "Currency references need review",
  policy_unavailable: "This book needs a current accounting policy",
  connection_not_ready: "This connection needs attention before registration",
  attempt_expired:
    "This setup expired; inspect or cancel it before starting again",
  consent_expired: "Bank access expired; connection recovery is required",
  capacity_exhausted: "The approved account limit has been reached",
  code_space_exhausted: "The account chart needs review before adding accounts",
};

// Validate the owned snapshot before displaying choices; TypeScript does not validate network bytes.
export function checkedRegistrationOptions(
  value: Options,
  entity: string,
  book: string,
  attempt: string,
): Options {
  if (
    !value ||
    value.entity_id !== entity ||
    value.book_id !== book ||
    value.attempt_id !== attempt ||
    !uuid.test(value.sync_item_id) ||
    typeof value.book_name !== "string" ||
    !value.book_name ||
    value.book_name.length > 200 ||
    (value.binding_date !== null &&
      (!/^\d{4}-\d{2}-\d{2}$/.test(value.binding_date) ||
        !Number.isFinite(Date.parse(value.binding_date)))) ||
    (value.max_sources !== null &&
      (typeof value.max_sources !== "string" ||
        !/^([1-9]|1[0-9]|2[0-5])$/.test(value.max_sources))) ||
    typeof value.current_sources !== "string" ||
    !decimal.test(value.current_sources) ||
    typeof value.remaining_sources !== "string" ||
    !/^(0|[1-9]|1[0-9]|2[0-5])$/.test(value.remaining_sources) ||
    value.history_target_days !== "90" ||
    value.history_request_mode !== "provider_default" ||
    value.history_guaranteed !== false ||
    typeof value.can_register !== "boolean" ||
    (value.blocked_reason !== null &&
      ![
        "policy_unavailable",
        "connection_not_ready",
        "attempt_expired",
        "consent_expired",
        "capacity_exhausted",
      ].includes(value.blocked_reason)) ||
    !Array.isArray(value.accounts) ||
    value.accounts.length < 1 ||
    value.accounts.length > 25 ||
    new Set(value.accounts.map((row) => row.account_id)).size !==
      value.accounts.length
  )
    throw Error("Account choices are unavailable. Refresh this connection.");
  for (const row of value.accounts) {
    if (
      !row ||
      typeof row.account_id !== "string" ||
      !/^[!-~]{1,256}$/.test(row.account_id) ||
      typeof row.display_label !== "string" ||
      !row.display_label ||
      row.display_label.length > 80 ||
      ![
        "depository",
        "credit",
        "loan",
        "investment",
        "other",
        "payroll",
      ].includes(row.type) ||
      (row.subtype !== null &&
        (typeof row.subtype !== "string" ||
          !/^[a-z0-9][a-z0-9 -]{0,63}$/.test(row.subtype))) ||
      (row.iso_currency_code !== null &&
        (typeof row.iso_currency_code !== "string" ||
          !/^[A-Z]{3}$/.test(row.iso_currency_code))) ||
      typeof row.registered !== "boolean" ||
      row.registered !==
        ["already_registered", "source_disconnected"].includes(
          row.reason ?? "",
        ) ||
      typeof row.eligible !== "boolean" ||
      (row.reason !== null && !Object.hasOwn(reasons, row.reason)) ||
      row.eligible !== (row.reason === null) ||
      (row.registered &&
        (row.eligible ||
          !["already_registered", "source_disconnected"].includes(
            row.reason ?? "",
          ))) ||
      (row.registered
        ? !uuid.test(row.stream_id ?? "") ||
          !uuid.test(row.source_account_id ?? "") ||
          !["active", "paused", "disconnected"].includes(row.source_state ?? "")
        : row.stream_id !== null ||
          row.source_account_id !== null ||
          row.source_state !== null)
    )
      throw Error("Account choices are unavailable. Refresh this connection.");
  }
  if (
    value.can_register !== value.accounts.some((row) => row.eligible) ||
    (value.blocked_reason !== null && value.can_register)
  ) {
    throw Error("Account choices are unavailable. Refresh this connection.");
  }
  return value;
}

// Selection is memory-only. Review persists the exact plan through the existing principal recovery gate.
export function RegistrationPicker({
  api,
  entity,
  book,
  attempt,
  locked,
  prepare,
  denied,
  onReview,
}: {
  api: API;
  entity: string;
  book: string;
  attempt: string;
  locked: boolean;
  prepare: (ids: string[]) => Promise<unknown>;
  denied: () => void;
  onReview: (source: string) => void;
}) {
  const [options, setOptions] = useState<Options | null>(null);
  const [selected, setSelected] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);
  const [preparing, setPreparing] = useState(false);
  const mounted = useRef(true);
  const heading = useRef<HTMLHeadingElement>(null);
  const deniedRef = useRef(denied);
  deniedRef.current = denied;
  useEffect(() => {
    let live = true;
    mounted.current = true;
    setOptions(null);
    setSelected([]);
    setError("");
    void api<Options>(
      `/v1/entities/${entity}/books/${book}/connection-attempts/${attempt}/registration`,
      undefined,
      undefined,
    )
      .then((value) => {
        if (live)
          setOptions(checkedRegistrationOptions(value, entity, book, attempt));
      })
      .catch((failure: unknown) => {
        if (!live) return;
        setOptions(null);
        setError(
          "Account choices are unavailable. Refresh this connection or inspect setup recovery.",
        );
        if (failure instanceof APIError && [403, 404].includes(failure.status))
          deniedRef.current();
      });
    return () => {
      live = false;
      mounted.current = false;
    };
  }, [api, entity, book, attempt, revision]);
  // An explicit setup/resume/refresh reveals the choice list; background import reads never steal focus.
  useEffect(() => {
    if (!options) return;
    const frame = requestAnimationFrame(() => heading.current?.focus());
    return () => cancelAnimationFrame(frame);
  }, [options?.attempt_id]);
  const capacity = options ? BigInt(options.remaining_sources) : 0n;
  const eligibleSelection =
    !!options &&
    selected.length > 0 &&
    BigInt(selected.length) <= capacity &&
    selected.every((id) =>
      options.accounts.some((row) => row.account_id === id && row.eligible),
    );
  return (
    <section
      className="registration-picker"
      aria-label="Choose accounts to register"
    >
      <h3 ref={heading} tabIndex={-1}>
        Choose accounts to register
      </h3>
      {error && <p role="alert">{error}</p>}
      {!options && !error && <p role="status">Loading account choices…</p>}
      {options && (
        <>
          <p>
            Register in <strong>{options.book_name}</strong>. We create a
            separate ledger account for each selected bank account.
          </p>
          <p>
            Target: the last 90 days. The current bank request uses provider
            defaults; available history depends on the bank.
          </p>
          {options.blocked_reason && (
            <p role="status">{reasons[options.blocked_reason]}</p>
          )}
          <ul className="registration-choices">
            {options.accounts.map((row) => (
              <li key={row.account_id} className="registration-choice">
                {row.eligible ? (
                  <label className="registration-choice-label">
                    <input
                      type="checkbox"
                      checked={selected.includes(row.account_id)}
                      disabled={
                        locked ||
                        preparing ||
                        (!selected.includes(row.account_id) &&
                          BigInt(selected.length) >= capacity)
                      }
                      onChange={(event) =>
                        setSelected((current) =>
                          event.target.checked
                            ? [...current, row.account_id]
                            : current.filter((id) => id !== row.account_id),
                        )
                      }
                    />
                    <span>
                      <strong>{row.display_label}</strong>
                      <span className="registration-secondary">
                        {row.subtype ?? row.type} ·{" "}
                        {row.iso_currency_code ?? "Currency unavailable"}
                      </span>
                    </span>
                  </label>
                ) : (
                  <div>
                    <strong>{row.display_label}</strong>
                    <span className="registration-secondary">
                      {row.subtype ?? row.type} ·{" "}
                      {row.iso_currency_code ?? "Currency unavailable"}
                    </span>
                  </div>
                )}
                <span className="registration-choice-state">
                  {row.reason ? reasons[row.reason] : "Available"}
                  {row.registered && row.source_state !== "active"
                    ? ` (${row.source_state})`
                    : ""}
                </span>
                {row.registered && row.source_account_id && (
                  <button
                    type="button"
                    disabled={locked || preparing}
                    onClick={() => onReview(row.source_account_id!)}
                  >
                    View transactions for {row.display_label}
                  </button>
                )}
              </li>
            ))}
          </ul>
          <p>
            {selected.length} selected. {options.remaining_sources} permitted{" "}
            {options.max_sources === null
              ? "per registration"
              : "under the approved limit"}
            .
          </p>
          <button
            type="button"
            disabled={
              locked || preparing || !options.can_register || !eligibleSelection
            }
            onClick={() => {
              setPreparing(true);
              void prepare([...selected])
                .catch(() => {
                  if (mounted.current)
                    setError(
                      "Registration is unconfirmed. Check saved work before trying again.",
                    );
                })
                .finally(() => {
                  if (mounted.current) setPreparing(false);
                });
            }}
          >
            Review selected accounts
          </button>
          <p>
            Review the exact saved mappings before confirming registration.
            Registration imports source data and does not post journals.
          </p>
          <p>
            Only accounts already shared with this connection appear here.
            Sharing another account requires separate bank consent; existing
            sharing must be preserved.
          </p>
        </>
      )}
      <button
        type="button"
        disabled={locked || preparing}
        onClick={() => setRevision((current) => current + 1)}
      >
        Refresh account choices
      </button>
    </section>
  );
}
