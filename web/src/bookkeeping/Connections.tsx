"use client";
import { useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import type { Book, Currency, Entity } from "./types";
import type { components } from "../lib/api.generated";
type Attempt = components["schemas"]["ConnectionAttempt"];
type Listing = components["schemas"]["ConnectionList"];
type Begin = components["schemas"]["ConnectionBeginResult"];
type Binding = components["schemas"]["ConnectionBindingRequest"];
// Only explicitly injected adapters handle transient Link tokens; no SDK loads here.
export type ConnectionLink = {
  open(input: {
    linkToken: string;
    signal: AbortSignal;
  }): Promise<{ publicToken: string } | null>;
};
export type ConnectionsProps = {
  api: API;
  entity: Entity;
  book: Book;
  currencies: Currency[];
  locked: boolean;
  denied: () => void;
  onReview: (sourceID: string) => void;
  link?: ConnectionLink;
};
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
const revision = /^[1-9][0-9]*$/;
const states = [
  "link_pending",
  "link_ready",
  "exchange_claimed",
  "credential_pending",
  "binding_review",
  "active",
  "credential_recovery_required",
  "disconnect_pending",
  "disconnected",
];
const itemStates = [
  "active",
  "paused",
  "disconnected",
  "credential_recovery_required",
  "disconnect_pending",
  "reauthorization_required",
];
// Runtime responses still require shape and identity checks despite generated TypeScript.
function validList(value: Listing): boolean {
  return (
    Array.isArray(value?.items) &&
    value.items.length <= 100 &&
    new Set(value.items.map((i) => i.sync_item_id)).size ===
      value.items.length &&
    value.items.every(
      (i) =>
        uuid.test(i.sync_item_id) &&
        typeof i.item_id === "string" &&
        i.item_id.length <= 256 &&
        itemStates.includes(i.state) &&
        revision.test(i.item_generation) &&
        Array.isArray(i.accounts) &&
        i.accounts.length <= 25 &&
        new Set(i.accounts.map((a) => a.stream_id)).size ===
          i.accounts.length &&
        i.accounts.every(
          (a) =>
            uuid.test(a.stream_id) &&
            uuid.test(a.source_account_id) &&
            typeof a.account_id === "string" &&
            a.account_id.length <= 256 &&
            ["active", "disconnected"].includes(a.state) &&
            revision.test(a.binding_revision),
        ),
    )
  );
}
// Remounting destroys tokens, pending confirmations and snapshots on any scope/lock change.
export function Connections(props: ConnectionsProps) {
  return (
    <ConnectionWorkspace
      key={`${props.entity.entity_id}/${props.book.book_id}/${props.book.book_revision}/${props.locked}`}
      {...props}
    />
  );
}
// A shared flight latch covers all mutations, while lifetime checks prevent late effects.
function ConnectionWorkspace({
  api,
  entity,
  book,
  currencies,
  locked,
  denied,
  onReview,
  link,
}: ConnectionsProps) {
  const base = `/v1/entities/${entity.entity_id}/books/${book.book_id}`;
  const life = useRef({
    live: true,
    flight: false,
    abort: new AbortController(),
  });
  const [busy, setBusy] = useState(false);
  const [enrollmentStarted, setEnrollmentStarted] = useState(false);
  const [list, setList] = useState<Listing | null>(null);
  const [attempt, setAttempt] = useState<Attempt | null>(null);
  const [inspectID, setInspectID] = useState("");
  const [message, setMessage] = useState("");
  const [selection, setSelection] = useState<
    Record<string, { ledger: string; currency: string }>
  >({});
  const [date, setDate] = useState("");
  const [consent, setConsent] = useState(false);
  const [disconnect, setDisconnect] = useState<
    Listing["items"][number]["accounts"][number] | null
  >(null);
  const [pending, setPending] = useState<{
    path: string;
    body: string;
    key: string;
    kind: "bind" | "disconnect";
    id: string;
  } | null>(null);
  // Public attempt metadata is checked before it can select a mutation path.
  const checkedAttempt = (value: Attempt, id?: string): Attempt => {
    if (
      !value ||
      value.entity_id !== entity.entity_id ||
      value.book_id !== book.book_id ||
      !uuid.test(value.attempt_id) ||
      (id && value.attempt_id !== id) ||
      !states.includes(value.state) ||
      !["new", "reconnect", "rotate", "remove"].includes(value.mode) ||
      !Number.isFinite(Date.parse(value.expires_at)) ||
      (value.sync_item_id !== null && !uuid.test(value.sync_item_id))
    )
      throw Error("response");
    if (
      value.manifest !== null &&
      (!Array.isArray(value.manifest.accounts) ||
        value.manifest.accounts.length > 25 ||
        new Set(value.manifest.accounts.map((a) => a.account_id)).size !==
          value.manifest.accounts.length ||
        value.manifest.accounts.some(
          (a) =>
            typeof a.account_id !== "string" ||
            !a.account_id.length ||
            a.account_id.length > 256 ||
            !["depository", "credit", "loan", "investment", "other"].includes(
              a.type,
            ) ||
            (a.iso_currency_code !== null &&
              !/^[A-Z]{3}$/.test(a.iso_currency_code)),
        ))
    )
      throw Error("response");
    return value;
  };
  // Never print error payloads, provider diagnostics or transient credentials.
  const fail = (error: unknown) => {
    if (error instanceof APIError && [401, 403, 404].includes(error.status)) {
      setList(null);
      setAttempt(null);
      setPending(null);
      setSelection({});
      setDisconnect(null);
      setMessage(
        "Connections are unavailable in this deployment or access was denied.",
      );
      if (error.status !== 404) denied();
    } else
      setMessage(
        "This operation could not be confirmed. Refresh status and review before proceeding.",
      );
  };
  // Keep one mutation in flight; the latch closes synchronously before React rerenders.
  const run = async (work: () => Promise<void>) => {
    const current = life.current;
    if (!current.live || current.flight || locked) return;
    current.flight = true;
    setBusy(true);
    setMessage("");
    try {
      await work();
    } catch (error) {
      if (current.live) fail(error);
    } finally {
      current.flight = false;
      if (current.live) setBusy(false);
    }
  };
  // List status has no financial balance or acquisition completeness implication.
  const readList = async () => {
    const current = life.current;
    const value = await api<Listing>(`${base}/connections`);
    if (!current.live) return;
    if (!validList(value)) throw Error("response");
    setList(value);
  };
  useEffect(() => {
    const current = { live: true, flight: false, abort: new AbortController() };
    life.current = current;
    if (!locked) void run(readList);
    return () => {
      current.live = false;
      current.abort.abort();
    };
    // Scope and lock are handled by the keyed outer component; API must be stable.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api]);
  // Inspect does not recreate Link or replay exchange after uncertainty.
  const inspect = (id: string) =>
    run(async () => {
      const current = life.current;
      setAttempt(null);
      setSelection({});
      setConsent(false);
      const value = await api<Attempt>(`${base}/connection-attempts/${id}`);
      if (!current.live) return;
      setAttempt(checkedAttempt(value, id));
      setInspectID(id);
    });
  // Only this fresh, in-memory Begin reply may authorize a single Link handoff.
  const start = () =>
    run(async () => {
      const current = life.current;
      if (
        !link ||
        pending ||
        (enrollmentStarted &&
          !["active", "disconnected"].includes(attempt?.state ?? ""))
      )
        return;
      setEnrollmentStarted(true);
      setAttempt(null);
      setSelection({});
      setConsent(false);
      const begun = await api<Begin>(
        `${base}/connection-attempts`,
        JSON.stringify({
          mode: "new",
          sync_item_id: null,
          expected_generation: null,
          approved_stream_ids: [],
        }),
        crypto.randomUUID(),
      );
      if (!current.live) return;
      const a = checkedAttempt(begun.attempt);
      setAttempt(a);
      setInspectID(a.attempt_id);
      if (
        !uuid.test(begun.nonce) ||
        !begun.link_token ||
        a.mode !== "new" ||
        a.state !== "link_ready"
      )
        throw Error("handoff");
      const result = await link.open({
        linkToken: begun.link_token,
        signal: life.current.abort.signal,
      });
      if (!current.live) return;
      if (!result) {
        setMessage(
          "Bank setup canceled. The known attempt remains available for inspection.",
        );
        return;
      }
      // No catch retries this irreversible handoff: unknown replies require inspection.
      const value = await api<Attempt>(
        `${base}/connection-attempts/${a.attempt_id}/execute`,
        JSON.stringify({
          nonce: begun.nonce,
          link_token: begun.link_token,
          public_token: result.publicToken,
        }),
      );
      if (!current.live) return;
      setAttempt(checkedAttempt(value, a.attempt_id));
      await readList();
    });
  // These two SQL-only operations have exact same-key replay; provider exchange does not.
  const submit = async (request: NonNullable<typeof pending>) => {
    const current = life.current;
    setPending(request);
    let value: Attempt & { stream_id?: string; state: string };
    try {
      value = await api(request.path, request.body, request.key);
    } catch (error) {
      if (current.live && error instanceof APIError && !error.uncertain)
        setPending(null);
      throw error;
    }
    if (!current.live) return;
    if (request.kind === "bind") {
      const a = checkedAttempt(value, request.id);
      if (
        a.state !== "active" ||
        !Array.isArray(a.streams) ||
        !a.streams.length ||
        a.streams.some(
          (s) =>
            !uuid.test(s.stream_id) ||
            !uuid.test(s.source_account_id) ||
            typeof s.overlap_review_required !== "boolean",
        )
      )
        throw Error("response");
      setAttempt(a);
      setSelection({});
      setConsent(false);
    } else if (value.stream_id !== request.id || value.state !== "disconnected")
      throw Error("response");
    setPending(null);
    setDisconnect(null);
    await readList();
  };
  // Build only an explicit new-source mapping; SQL checks all current grants and pins.
  const bind = () =>
    run(async () => {
      if (!attempt || attempt.state !== "binding_review" || !consent || pending)
        return;
      const bindings: Binding["bindings"] = Object.entries(selection)
        .filter(([, v]) => v.ledger && v.currency)
        .map(([account_id, v]) => ({
          account_id,
          ledger_account_id: v.ledger,
          currency_version_id: v.currency,
          source_account_id: null,
          expected_binding_revision: null,
          binding_date: date,
          binding_evidence_ref: crypto.randomUUID(),
        }));
      if (
        !bindings.length ||
        new Set(bindings.map((b) => b.ledger_account_id)).size !==
          bindings.length
      ) {
        setMessage(
          "Choose distinct ledger accounts and matching currencies for every selected account.",
        );
        return;
      }
      await submit({
        kind: "bind",
        id: attempt.attempt_id,
        path: `${base}/connection-attempts/${attempt.attempt_id}/bindings`,
        body: JSON.stringify({ consent_grants: true, bindings }),
        key: crypto.randomUUID(),
      });
    });
  const unavailable = locked || busy;
  const mutable = unavailable || !!pending;
  return (
    <section aria-label="Connections" aria-busy={busy}>
      <p>
        Connections for {entity.name} / {book.name}. Enrollment is not a balance
        or confirmation of complete bank history.
      </p>
      {!link && <p>New bank setup is unavailable in this deployment.</p>}
      <button
        type="button"
        disabled={
          mutable ||
          !link ||
          (enrollmentStarted &&
            !["active", "disconnected"].includes(attempt?.state ?? ""))
        }
        onClick={() => void start()}
      >
        Connect account
      </button>{" "}
      <button
        type="button"
        disabled={unavailable}
        onClick={() =>
          void run(async () => {
            setList(null);
            await readList();
          })
        }
      >
        Refresh connections
      </button>
      {message && <p role="alert">{message}</p>}
      {locked && (
        <p>
          Finish the current bookkeeping recovery before managing connections.
        </p>
      )}
      {list === null ? (
        <p>Connection status unavailable.</p>
      ) : list.items.length === 0 ? (
        <p>No connected Items in this book.</p>
      ) : (
        list.items.map((item) => (
          <article key={item.sync_item_id}>
            <h3>Provider Item {item.item_id}</h3>
            <p>Current connection status: {item.state.replaceAll("_", " ")}</p>
            <ul>
              {item.accounts.map((account) => (
                <li key={account.stream_id}>
                  Account {account.account_id} — {account.state}{" "}
                  <button
                    disabled={unavailable}
                    onClick={() => onReview(account.source_account_id)}
                  >
                    Review transactions for {account.account_id}
                  </button>{" "}
                  <button
                    disabled={mutable || account.state !== "active"}
                    onClick={() => setDisconnect(account)}
                  >
                    Disconnect {account.account_id}
                  </button>
                </li>
              ))}
            </ul>
          </article>
        ))
      )}
      {disconnect && (
        <fieldset disabled={mutable}>
          <legend>Confirm account disconnect</legend>
          <p>
            Stop acquisition for {disconnect.account_id} only. Other accounts
            and posted history are retained. This does not remove the provider
            Item.
          </p>
          <button
            onClick={() =>
              void run(() =>
                submit({
                  kind: "disconnect",
                  id: disconnect.stream_id,
                  path: `${base}/connections/streams/${disconnect.stream_id}/disconnect`,
                  body: JSON.stringify({
                    expected_binding_revision: disconnect.binding_revision,
                  }),
                  key: crypto.randomUUID(),
                }),
              )
            }
          >
            Confirm disconnect
          </button>
          <button onClick={() => setDisconnect(null)}>Cancel disconnect</button>
        </fieldset>
      )}
      {pending && (
        <div role="status">
          <p>
            The {pending.kind === "bind" ? "binding" : "disconnect"} result is
            unresolved. Retrying uses the same request and key.
          </p>
          <button
            disabled={unavailable}
            onClick={() => void run(() => submit(pending))}
          >
            Resolve saved request
          </button>
        </div>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (uuid.test(inspectID)) void inspect(inspectID);
        }}
      >
        <label>
          Known connection attempt ID
          <input
            value={inspectID}
            onChange={(e) => setInspectID(e.target.value)}
            disabled={mutable}
            autoComplete="off"
          />
        </label>
        <button disabled={mutable || !uuid.test(inspectID)}>
          Inspect attempt
        </button>
      </form>
      <p>
        An unfinished enrollment must be inspected before starting another.
        Attempts cannot yet be rediscovered after a lost browser session. Keep
        the non-secret attempt ID for inspection. Pause, manual sync and
        reconnect are not available here.
      </p>
      {attempt && (
        <article>
          <h3>Connection attempt</h3>
          <p>Attempt ID: {attempt.attempt_id}</p>
          <p>Historical attempt status: {attempt.state.replaceAll("_", " ")}</p>
          <p>Handoff expires: {attempt.expires_at}</p>
          {[
            "exchange_claimed",
            "credential_pending",
            "credential_recovery_required",
            "disconnect_pending",
          ].includes(attempt.state) && (
            <p>
              This attempt is unresolved. Do not repeat the provider exchange.
              Reconciliation can only check an already recorded credential; it
              cannot repair a lost exchange response.
            </p>
          )}
          <button
            disabled={
              mutable ||
              !["credential_pending", "credential_recovery_required"].includes(
                attempt.state,
              )
            }
            onClick={() =>
              void run(async () => {
                const current = life.current;
                const value = await api<Attempt>(
                  `${base}/connection-attempts/${attempt.attempt_id}/reconcile`,
                  "{}",
                );
                if (!current.live) return;
                setAttempt(checkedAttempt(value, attempt.attempt_id));
                await readList();
              })
            }
          >
            Reconcile recorded credential
          </button>
          {attempt.state === "binding_review" &&
            attempt.mode === "new" &&
            attempt.manifest && (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  void bind();
                }}
              >
                <fieldset disabled={mutable}>
                  <legend>Review verified account bindings</legend>
                  <label>
                    Binding date
                    <input
                      type="date"
                      required
                      value={date}
                      onChange={(e) => {
                        setDate(e.target.value);
                        setSelection({});
                        setConsent(false);
                      }}
                    />
                  </label>
                  {attempt.manifest.accounts.map((account) => (
                    <div key={account.account_id}>
                      <p>
                        {account.account_id} — {account.type},{" "}
                        {account.iso_currency_code ?? "currency unavailable"}
                      </p>
                      {account.type === "depository" &&
                      account.iso_currency_code ? (
                        <>
                          <label>
                            Ledger account for {account.account_id}
                            <select
                              value={
                                selection[account.account_id]?.ledger ?? ""
                              }
                              onChange={(e) => {
                                setSelection((s) => ({
                                  ...s,
                                  [account.account_id]: {
                                    ledger: e.target.value,
                                    currency:
                                      s[account.account_id]?.currency ?? "",
                                  },
                                }));
                                setConsent(false);
                              }}
                            >
                              <option value="">Do not bind</option>
                              {book.accounts
                                .filter(
                                  (a) =>
                                    date &&
                                    a.category === "asset" &&
                                    a.normal_side === "debit" &&
                                    a.monetary &&
                                    a.active_from <= date &&
                                    (!a.active_to || date < a.active_to),
                                )
                                .map((a) => (
                                  <option
                                    key={a.account_id}
                                    value={a.account_id}
                                  >
                                    {a.code} {a.name}
                                  </option>
                                ))}
                            </select>
                          </label>
                          <label>
                            Currency version for {account.account_id}
                            <select
                              value={
                                selection[account.account_id]?.currency ?? ""
                              }
                              onChange={(e) => {
                                setSelection((s) => ({
                                  ...s,
                                  [account.account_id]: {
                                    ledger: s[account.account_id]?.ledger ?? "",
                                    currency: e.target.value,
                                  },
                                }));
                                setConsent(false);
                              }}
                            >
                              <option value="">Choose currency</option>
                              {currencies
                                .filter(
                                  (c) =>
                                    date &&
                                    c.minor_unit !== null &&
                                    c.alphabetic_code ===
                                      account.iso_currency_code &&
                                    c.valid_from <= date &&
                                    (!c.valid_to || date < c.valid_to),
                                )
                                .map((c) => (
                                  <option
                                    key={c.currency_version_id}
                                    value={c.currency_version_id}
                                  >
                                    {c.alphabetic_code} from {c.valid_from}
                                  </option>
                                ))}
                            </select>
                          </label>
                        </>
                      ) : (
                        <p>This account cannot be bound in this profile.</p>
                      )}
                    </div>
                  ))}
                  <label>
                    <input
                      type="checkbox"
                      checked={consent}
                      onChange={(e) => setConsent(e.target.checked)}
                    />
                    I confirm these new source mappings and grant this account's
                    scoped acquisition access.
                  </label>
                  <button
                    disabled={
                      !consent ||
                      !date ||
                      !Object.values(selection).some(
                        (v) => v.ledger && v.currency,
                      ) ||
                      Object.values(selection).some(
                        (v) => !!v.ledger !== !!v.currency,
                      )
                    }
                  >
                    Confirm selected bindings
                  </button>
                </fieldset>
              </form>
            )}
        </article>
      )}
    </section>
  );
}
