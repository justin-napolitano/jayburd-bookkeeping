"use client";
import { useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import type { Book, Currency, Entity } from "./types";
import type { components } from "../lib/api.generated";
type Attempt = components["schemas"]["ConnectionAttempt"];
type Listing = components["schemas"]["ConnectionList"];
type Begin = components["schemas"]["ConnectionBeginResult"];
type Binding = components["schemas"]["ConnectionBindingRequest"];
// Public status and action shapes come only from the coordinator's frozen OpenAPI.
type AttemptPage = components["schemas"]["ConnectionAttemptList"];
type StreamStatus = components["schemas"]["ConnectionStreamStatus"];
type StreamAction =
  components["schemas"]["ConnectionStreamOperation"]["action"];
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
// Only the reviewed opaque Hosted Link path may become browser navigation.
function validHostedURL(value: string): boolean {
  return (
    value.length <= 2048 &&
    /^https:\/\/secure\.plaid\.com\/hl\/[A-Za-z0-9_-]+$/.test(value)
  );
}
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
  "abandoned",
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
            ["active", "paused", "disconnected"].includes(a.state) &&
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
  const [hosted, setHosted] = useState<Begin | null>(null);
  const [enrollmentStarted, setEnrollmentStarted] = useState(false);
  const [list, setList] = useState<Listing | null>(null);
  const [attempt, setAttempt] = useState<Attempt | null>(null);
  const [attempts, setAttempts] = useState<AttemptPage | null>(null);
  const [expiredAttempt, setExpiredAttempt] = useState(false);
  const [bindingExpired, setBindingExpired] = useState(false);
  const [status, setStatus] = useState<StreamStatus | null>(null);
  const [statusAccount, setStatusAccount] = useState("");
  const [operation, setOperation] = useState<StreamAction | null>(null);
  const [inspectID, setInspectID] = useState("");
  const [message, setMessage] = useState("");
  const [selection, setSelection] = useState<
    Record<string, { ledger: string; currency: string }>
  >({});
  const [date, setDate] = useState("");
  const [consent, setConsent] = useState(false);
  const [abandonConsent, setAbandonConsent] = useState(false);
  const [disconnect, setDisconnect] = useState<
    Listing["items"][number]["accounts"][number] | null
  >(null);
  const [pending, setPending] = useState<{
    path: string;
    body: string;
    key: string;
    kind: "bind" | "disconnect" | "stream";
    id: string;
    scope?: {
      stream_id: string;
      source_account_id: string;
      sync_item_id: string;
    };
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
      setHosted(null);
      setList(null);
      setAttempts(null);
      setStatus(null);
      setOperation(null);
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
  // Authenticate both snapshots before offering enrollment; unfinished_count covers every page.
  const checkAttempts = (page: AttemptPage) => {
    if (
      !page ||
      !Array.isArray(page.items) ||
      page.items.length > 100 ||
      typeof page.unfinished_count !== "string" ||
      !/^(0|[1-9][0-9]*)$/.test(page.unfinished_count) ||
      (page.next_cursor !== null && !uuid.test(page.next_cursor)) ||
      new Set(page.items.map((a) => a.attempt_id)).size !== page.items.length
    )
      throw Error("response");
    page.items.forEach((a) => {
      checkedAttempt(a);
      if (
        typeof a.expired_unclaimed !== "boolean" ||
        typeof a.binding_expired !== "boolean"
      )
        throw Error("response");
    });
    return page;
  };
  const readList = async () => {
    const current = life.current;
    setList(null);
    setAttempts(null);
    setStatus(null);
    setOperation(null);
    const [listing, discovery] = await Promise.allSettled([
      api<Listing>(`${base}/connections`),
      api<AttemptPage>(`${base}/connection-attempts?limit=100`),
    ]);
    if (!current.live) return;
    // A failed Item listing must not hide successful durable-attempt discovery.
    if (discovery.status === "fulfilled") {
      const page = checkAttempts(discovery.value);
      setAttempts(page);
      if (page.unfinished_count === "0") setEnrollmentStarted(false);
    }
    if (listing.status === "fulfilled") {
      if (!validList(listing.value)) throw Error("response");
      setList(listing.value);
    }
    if (discovery.status === "rejected") throw discovery.reason;
    if (listing.status === "rejected") throw listing.reason;
  };
  // A status is useful only for the exact currently selected Item/source/stream scope.
  const checkedStatus = (
    value: StreamStatus,
    expected: {
      stream_id: string;
      source_account_id: string;
      sync_item_id: string;
    },
  ) => {
    if (
      !value ||
      value.stream_id !== expected.stream_id ||
      value.source_account_id !== expected.source_account_id ||
      value.sync_item_id !== expected.sync_item_id ||
      !["active", "paused", "disconnected"].includes(value.state) ||
      !itemStates.includes(value.item_state) ||
      typeof value.stream_revision !== "string" ||
      !revision.test(value.stream_revision) ||
      typeof value.item_generation !== "string" ||
      !revision.test(value.item_generation) ||
      typeof value.binding_revision !== "string" ||
      !revision.test(value.binding_revision) ||
      typeof value.requested_generation !== "string" ||
      !revision.test(value.requested_generation) ||
      typeof value.completed_generation !== "string" ||
      !/^(0|[1-9][0-9]*)$/.test(value.completed_generation) ||
      BigInt(value.completed_generation) > BigInt(value.requested_generation) ||
      typeof value.queued !== "boolean" ||
      typeof value.lease_active !== "boolean" ||
      value.provider_updated_at !== null ||
      (value.last_completed_at !== null &&
        !Number.isFinite(Date.parse(value.last_completed_at))) ||
      ![
        "TRANSACTIONS_UPDATE_STATUS_UNKNOWN",
        "NOT_READY",
        "INITIAL_UPDATE_COMPLETE",
        "HISTORICAL_UPDATE_COMPLETE",
      ].includes(value.readiness) ||
      (value.last_error !== null &&
        ![
          "pagination_mutation",
          "provider_failure",
          "decoder_rejected",
          "cancelled",
          "item_changed",
          "lease_expired",
          "account_disconnected",
          "account_paused",
        ].includes(value.last_error))
    )
      throw Error("response");
    return value;
  };
  const readStatus = (
    account: Listing["items"][number]["accounts"][number],
    itemID: string,
  ) =>
    run(async () => {
      const current = life.current;
      setStatus(null);
      setOperation(null);
      setStatusAccount(account.account_id);
      const value = await api<StreamStatus>(
        `${base}/connections/streams/${account.stream_id}/status`,
      );
      if (!current.live) return;
      setStatus(checkedStatus(value, { ...account, sync_item_id: itemID }));
    });
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
  const inspect = (id: string, expired = false, bindingExpiry = false) =>
    run(async () => {
      const current = life.current;
      setAttempt(null);
      const discovered = attempts?.items.find((a) => a.attempt_id === id);
      setExpiredAttempt(discovered?.expired_unclaimed ?? expired);
      setBindingExpired(discovered?.binding_expired ?? bindingExpiry);
      setSelection({});
      setConsent(false);
      setAbandonConsent(false);
      let value: Attempt;
      try {
        value = await api<Attempt>(`${base}/connection-attempts/${id}`);
      } catch (error) {
        if (!(error instanceof APIError) || ![403, 404].includes(error.status))
          throw error;
        // A historical attempt may be inaccessible while the book remains authorized.
        // Recheck current scope before deciding whether to clear its financial view.
        await readList();
        if (!current.live) return;
        setMessage(
          "This attempt is unavailable. Your current Connections access was rechecked; no bank setup was completed.",
        );
        return;
      }
      if (!current.live) return;
      setAttempt(checkedAttempt(value, id));
      setInspectID(id);
      // Recheck discovery before releasing an earlier local enrollment latch.
      if (discovered?.binding_expired ?? bindingExpiry) await readList();
    });
  // Only this fresh, in-memory Begin reply may authorize a single Link handoff.
  const start = () =>
    run(async () => {
      const current = life.current;
      if (
        !link ||
        !attempts ||
        attempts.unfinished_count !== "0" ||
        pending ||
        (enrollmentStarted &&
          !["active", "disconnected"].includes(attempt?.state ?? ""))
      )
        return;
      setEnrollmentStarted(true);
      setExpiredAttempt(false);
      setBindingExpired(false);
      setAttempt(null);
      setSelection({});
      setConsent(false);
      setAbandonConsent(false);
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
      if (begun.hosted_link_url) {
        if (
          !validHostedURL(begun.hosted_link_url) ||
          !begun.expires_at ||
          !Number.isFinite(Date.parse(begun.expires_at)) ||
          Date.parse(begun.expires_at) <= Date.now()
        )
          throw Error("handoff");
        setHosted(begun);
        setMessage(
          "Open Plaid Sandbox in a new tab, finish setup, then return here to check completion.",
        );
        return;
      }
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
  // Only an explicit check resolves Hosted Link. Unknown replies discard the handoff.
  const completeHosted = () =>
    run(async () => {
      const current = life.current;
      const handoff = hosted;
      if (!handoff) return;
      setHosted(null);
      if (
        !handoff.expires_at ||
        !Number.isFinite(Date.parse(handoff.expires_at)) ||
        Date.parse(handoff.expires_at) <= Date.now()
      ) {
        setMessage(
          "Bank setup expired. Refresh attempts before starting again.",
        );
        return;
      }
      let value: Attempt;
      try {
        value = await api<Attempt>(
          `${base}/connection-attempts/${handoff.attempt.attempt_id}/execute`,
          JSON.stringify({
            nonce: handoff.nonce,
            link_token: handoff.link_token,
          }),
        );
      } catch (error) {
        // A 409 can follow a committed exchange. Inspect that exact attempt;
        // never restore the handoff or repeat the provider mutation.
        if (error instanceof APIError && error.status === 409 && current.live) {
          const recovered = checkedAttempt(
            await api<Attempt>(`${base}/connection-attempts/${handoff.attempt.attempt_id}`),
            handoff.attempt.attempt_id,
          );
          if (!current.live) return;
          setAttempt(recovered);
          setInspectID(recovered.attempt_id);
          if (recovered.state === "credential_recovery_required") {
            setMessage(
              "Bank setup needs credential recovery. Review this attempt below; do not start another bank connection.",
            );
            return;
          }
        }
        throw error;
      }
      if (!current.live) return;
      const checked = checkedAttempt(value, handoff.attempt.attempt_id);
      setAttempt(checked);
      if (checked.state === "link_ready") {
        setHosted(handoff);
        setMessage(
          "Plaid has not reported completion yet. Finish setup, then check again.",
        );
      } else {
        await readList();
      }
    });
  // These two SQL-only operations have exact same-key replay; provider exchange does not.
  const submit = async (request: NonNullable<typeof pending>) => {
    const current = life.current;
    setPending(request);
    let value: Attempt & StreamStatus;
    try {
      value = await api(request.path, request.body, request.key);
    } catch (error) {
      if (current.live && error instanceof APIError && !error.uncertain) {
        setPending(null);
        if (request.kind === "stream") {
          setStatus(null);
          setOperation(null);
        }
      }
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
      setAbandonConsent(false);
    } else if (request.kind === "stream") {
      if (!request.scope) throw Error("response");
      checkedStatus(value, request.scope);
    } else if (value.stream_id !== request.id || value.state !== "disconnected")
      throw Error("response");
    setPending(null);
    setDisconnect(null);
    const receipt = request.kind === "stream" ? (value as StreamStatus) : null;
    await readList();
    if (!current.live) return;
    if (receipt) {
      const fresh = await api<StreamStatus>(
        `${base}/connections/streams/${request.id}/status`,
      );
      if (!current.live) return;
      setStatus(checkedStatus(fresh, receipt));
      setOperation(null);
      setMessage(
        "Operation recorded. A sync request is pending work, not confirmation of a provider pull or publication.",
      );
    }
  };
  // Build only an explicit new-source mapping; SQL checks all current grants and pins.
  const bind = () =>
    run(async () => {
      if (
        !attempt ||
        bindingExpired ||
        attempt.state !== "binding_review" ||
        !consent ||
        pending
      )
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
      {hosted?.hosted_link_url && (
        <div role="group" aria-label="Complete Sandbox bank setup">
          <p>
            Keep this tab open while you complete fictional bank setup in Plaid.
          </p>
          <a
            href={hosted.hosted_link_url}
            target="_blank"
            rel="noopener noreferrer"
            referrerPolicy="no-referrer"
          >
            Open Plaid Sandbox
          </a>
          <button
            disabled={locked || busy}
            onClick={() => void completeHosted()}
          >
            Check bank setup completion
          </button>
          <button
            disabled={locked || busy}
            onClick={() => {
              setHosted(null);
              setMessage(
                "Bank setup handoff cleared. Inspect the attempt; a new setup is available after its expiry.",
              );
            }}
          >
            Cancel bank setup
          </button>
        </div>
      )}
      <p>
        Connections for {entity.name} / {book.name}. Enrollment is not a balance
        or confirmation of complete bank history.
      </p>
      <p>
        Connect account prepares bank access for a book you created, then opens
        Plaid. You choose which accounts to share and review where they belong
        before importing transactions.
      </p>
      {!link && <p>New bank setup is unavailable in this deployment.</p>}
      <button
        type="button"
        disabled={
          mutable ||
          !link ||
          !attempts ||
          attempts.unfinished_count !== "0" ||
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
                    disabled={mutable}
                    onClick={() => void readStatus(account, item.sync_item_id)}
                  >
                    View sync status for {account.account_id}
                  </button>{" "}
                  <button
                    disabled={mutable || account.state === "disconnected"}
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
      <section aria-label="Connection recovery">
        <h3>Connection recovery</h3>
        {!attempts ? (
          <p>Attempt discovery unavailable. New enrollment is disabled.</p>
        ) : (
          <>
            <p>Unfinished attempts: {attempts.unfinished_count}</p>
            {attempts.items.length === 0 && <p>No attempts on this page.</p>}
            <ul>
              {attempts.items.map((a) => (
                <li key={a.attempt_id}>
                  {a.mode} —{" "}
                  {a.binding_expired
                    ? "binding review expired"
                    : a.expired_unclaimed
                      ? "expired before claim"
                      : a.state.replaceAll("_", " ")}{" "}
                  <button
                    disabled={mutable}
                    onClick={() =>
                      void inspect(
                        a.attempt_id,
                        a.expired_unclaimed,
                        a.binding_expired,
                      )
                    }
                  >
                    Review attempt {a.attempt_id}
                  </button>
                </li>
              ))}
            </ul>
            {attempts.next_cursor && (
              <button
                disabled={mutable}
                onClick={() =>
                  void run(async () => {
                    const current = life.current;
                    const cursor = attempts.next_cursor;
                    const page = await api<AttemptPage>(
                      `${base}/connection-attempts?limit=100&before=${cursor}`,
                    );
                    if (!current.live) return;
                    const valid = checkAttempts(page);
                    if (valid.next_cursor === cursor) throw Error("response");
                    setAttempts(valid);
                  })
                }
              >
                Older connection attempts
              </button>
            )}
          </>
        )}
      </section>
      {status && (
        <section aria-label="Account sync status">
          <h3>Sync status for {statusAccount}</h3>
          <p>
            Account acquisition: {status.state}. Provider Item:{" "}
            {status.item_state.replaceAll("_", " ")}.
          </p>
          <p>Readiness: {status.readiness.replaceAll("_", " ")}</p>
          <p>
            Requested generation {status.requested_generation}; completed
            generation {status.completed_generation}.
          </p>
          <p>
            {status.queued
              ? "Sync work is queued."
              : "No newer sync request is queued."}{" "}
            {status.lease_active
              ? "A worker lease is active; publication is not yet confirmed."
              : "No active worker lease."}
          </p>
          <p>
            Last complete publication:{" "}
            {status.last_completed_at ?? "None recorded"}
          </p>
          <p>Last provider update: Unknown — this timestamp is not supplied.</p>
          {status.last_error && (
            <p>
              Last acquisition outcome: {status.last_error.replaceAll("_", " ")}
            </p>
          )}
          <p>
            Acquisition does not post journals or prove that all bank history
            has arrived.
          </p>
          <button
            disabled={
              mutable ||
              status.state !== "active" ||
              status.item_state !== "active"
            }
            onClick={() => setOperation("pause")}
          >
            Pause account sync
          </button>{" "}
          <button
            disabled={
              mutable ||
              status.state !== "paused" ||
              status.item_state !== "active"
            }
            onClick={() => setOperation("resume")}
          >
            Resume account sync
          </button>{" "}
          <button
            disabled={
              mutable ||
              status.state !== "active" ||
              status.item_state !== "active"
            }
            onClick={() => setOperation("request")}
          >
            Request sync now
          </button>
          {operation && (
            <fieldset disabled={mutable}>
              <legend>Confirm account sync operation</legend>
              <p>
                {operation === "request"
                  ? "Request another acquisition"
                  : operation === "pause"
                    ? "Pause acquisition"
                    : "Resume acquisition"}{" "}
                for {statusAccount} only. Sibling accounts and posted journals
                are unchanged.
              </p>
              <p>
                Current Item generation {status.item_generation}; binding
                revision {status.binding_revision}; stream revision{" "}
                {status.stream_revision}.
              </p>
              <button
                onClick={() =>
                  void run(() =>
                    submit({
                      kind: "stream",
                      id: status.stream_id,
                      scope: {
                        stream_id: status.stream_id,
                        source_account_id: status.source_account_id,
                        sync_item_id: status.sync_item_id,
                      },
                      path: `${base}/connections/streams/${status.stream_id}/operations`,
                      body: JSON.stringify({
                        action: operation,
                        expected_item_generation: status.item_generation,
                        expected_binding_revision: status.binding_revision,
                        expected_stream_revision: status.stream_revision,
                      }),
                      key: crypto.randomUUID(),
                    }),
                  )
                }
              >
                Confirm {operation === "request" ? "sync request" : operation}
              </button>
              <button onClick={() => setOperation(null)}>
                Cancel sync operation
              </button>
            </fieldset>
          )}
        </section>
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
            The{" "}
            {pending.kind === "bind"
              ? "binding"
              : pending.kind === "stream"
                ? "stream operation"
                : "disconnect"}{" "}
            result is unresolved. Retrying uses the same request and key.
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
        Unfinished enrollment is discovered after reload. Inspect and reconcile
        it before another new connection. Expired unclaimed attempts cannot be
        replayed. Reconnect remains unavailable.
      </p>
      {attempt && (
        <article>
          <h3>Connection attempt</h3>
          <p>Attempt ID: {attempt.attempt_id}</p>
          <p>Historical attempt status: {attempt.state.replaceAll("_", " ")}</p>
          <p>Handoff expires: {attempt.expires_at}</p>
          {bindingExpired && (
            <p>
              Binding review expired; start a fresh connection. The known
              credential history is retained and its provider exchange must not
              be replayed.
            </p>
          )}
          {expiredAttempt && (
            <p>
              Expired before claim. This attempt cannot be replayed; a new
              connection requires a fresh Link session.
            </p>
          )}
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
          {attempt.mode === "new" &&
            attempt.state === "credential_recovery_required" &&
            attempt.sync_item_id === null && (
              <details>
                <summary>Credential unavailable: start over</summary>
                <p>
                  Use this only after confirming the credential cannot be
                  recovered. This preserves the failed attempt and does not
                  remove the Item at Plaid. Orphaned Sandbox Item cleanup may
                  still be needed.
                </p>
                <label>
                  <input
                    type="checkbox"
                    checked={abandonConsent}
                    disabled={mutable}
                    onChange={(e) => setAbandonConsent(e.target.checked)}
                  />
                  I confirm this credential is unavailable and want to abandon
                  this setup.
                </label>
                <button
                  disabled={mutable || !abandonConsent}
                  onClick={() =>
                    void run(async () => {
                      const current = life.current;
                      const value = await api<Attempt>(
                        `${base}/connection-attempts/${attempt.attempt_id}/abandon`,
                        JSON.stringify({ confirm_unrecoverable: true }),
                      );
                      if (!current.live) return;
                      setAttempt(checkedAttempt(value, attempt.attempt_id));
                      setAbandonConsent(false);
                      await readList();
                      setMessage(
                        "Setup abandoned. Its history is retained. A new connection requires a fresh Plaid session.",
                      );
                    })
                  }
                >
                  Abandon unavailable setup
                </button>
              </details>
            )}
          {!expiredAttempt &&
            !bindingExpired &&
            attempt.state === "binding_review" &&
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
                        setAbandonConsent(false);
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
                                setAbandonConsent(false);
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
                                setAbandonConsent(false);
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
