"use client";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import { accountLabels } from "./accountLabels";
import { RegistrationPicker } from "./RegistrationPicker";
import { checkedStatus, itemStates } from "./connectionStatus";
import { AccountImportStatus } from "./AccountImportStatus";
import type { Book, Currency, Entity, DurableCommand } from "./types";
import type { components } from "../lib/api.generated";
type Attempt = components["schemas"]["ConnectionAttempt"];
type Listing = components["schemas"]["ConnectionList"];
type Begin = components["schemas"]["ConnectionBeginResult"];
type Binding = components["schemas"]["ConnectionBindingRequest"];
type BindingSelection = Record<
  string,
  {
    ledger: string;
    currency: string;
    create: boolean;
    code: string;
    name: string;
  }
>;
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
  // Only the shared principal gate may prepare normal account registration.
  prepareRegistration?: (
    entity: string,
    book: string,
    attempt: string,
    ids: string[],
  ) => Promise<DurableCommand | undefined>;
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
const automaticDepositorySubtypes = new Set([
  "checking",
  "savings",
  "money market",
  "cash management",
]);
// A reviewed product/subtype pair determines only the proposed chart shape.
function automaticClassification(type: string, subtype: string | null) {
  if (
    type === "depository" &&
    subtype &&
    automaticDepositorySubtypes.has(subtype)
  )
    return {
      category: "asset" as const,
      side: "debit" as const,
      code: 1100,
      label: "Bank",
    };
  if (type === "credit" && subtype === "credit card")
    return {
      category: "liability" as const,
      side: "credit" as const,
      code: 2100,
      label: "Credit card",
    };
  if (
    type === "loan" &&
    subtype &&
    ["student", "mortgage", "auto", "consumer", "line of credit"].includes(
      subtype,
    )
  )
    return {
      category: "liability" as const,
      side: "credit" as const,
      code: 2200,
      label: "Loan",
    };
  return null;
}
// Readiness is visible beside the action; a disabled financial command never hides its cause.
function bindingBlockers(
  selection: BindingSelection,
  date: string,
  consent: boolean,
): string[] {
  const values = Object.values(selection);
  const selected = values.filter((value) => value.ledger && value.currency);
  const blockers: string[] = [];
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) blockers.push("choose a binding date");
  if (!selected.length)
    blockers.push("create or select a ledger account with a matching currency");
  if (values.some((value) => !!value.ledger !== !!value.currency))
    blockers.push("complete or clear every partial account selection");
  if (new Set(selected.map((value) => value.ledger)).size !== selected.length)
    blockers.push(
      "use a distinct ledger account for each selected bank account",
    );
  if (
    values.some(
      (value) => value.create && (!value.code.trim() || !value.name.trim()),
    )
  )
    blockers.push("provide a code and name for every new ledger account");
  if (!consent)
    blockers.push("acknowledge the source mappings and access grant");
  return blockers;
}
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

// Customer status names describe the next step without exposing transport states.
function setupLabel(state: string): string {
  return (
    (
      {
        link_pending: "Starting bank setup",
        link_ready: "Waiting for bank setup",
        exchange_claimed: "Confirming bank connection",
        credential_pending: "Confirming bank connection",
        binding_review: "Choose your accounts",
        active: "Connected",
        credential_recovery_required: "Bank setup needs attention",
        disconnect_pending: "Canceling bank setup",
        disconnected: "Disconnected",
        abandoned: "Canceled",
      } as Record<string, string>
    )[state] ?? "Bank setup needs attention"
  );
}
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
  prepareRegistration,
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
  const [hostedURLExpired, setHostedURLExpired] = useState(false);
  const [enrollmentStarted, setEnrollmentStarted] = useState(false);
  const [list, setList] = useState<Listing | null>(null);
  const [attempt, setAttempt] = useState<Attempt | null>(null);
  const [attempts, setAttempts] = useState<AttemptPage | null>(null);
  const [expiredAttempt, setExpiredAttempt] = useState(false);
  const [bindingExpired, setBindingExpired] = useState(false);
  const [status, setStatus] = useState<StreamStatus | null>(null);
  const [statusAccount, setStatusAccount] = useState("");
  const [operation, setOperation] = useState<StreamAction | null>(null);
  const [message, setMessage] = useState("");
  const [selection, setSelection] = useState<BindingSelection>({});
  const [date, setDate] = useState("");
  const [consent, setConsent] = useState(false);
  const [advanced, setAdvanced] = useState(false);
  const autoReturn = useRef<{ attempt: string; checks: number } | null>(null);
  const [cancelReview, setCancelReview] = useState("");
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
  // Labels are display-only; every request and review handoff retains the immutable ID.
  const accountLabel = accountLabels([
    ...(list?.items.flatMap((item) =>
      item.accounts.map((account) => account.account_id),
    ) ?? []),
    ...(attempt?.manifest?.accounts.map((account) => account.account_id) ?? []),
  ]);
  // Bound automatic status traffic across the book; additional rows retain explicit sync-status controls.
  const autoStatus = new Set(
    list?.items
      .flatMap((item) =>
        item.accounts
          .filter((account) => account.state === "active")
          .map((account) => account.stream_id),
      )
      .slice(0, 8) ?? [],
  );
  // URL expiry disables navigation only; server recovery has its own authoritative deadline.
  useEffect(() => {
    setHostedURLExpired(false);
    if (!hosted?.expires_at) return;
    const expires = Date.parse(hosted.expires_at);
    let timer: ReturnType<typeof setTimeout>;
    const update = () => {
      const remaining = expires - Date.now();
      if (remaining <= 0) setHostedURLExpired(true);
      else timer = setTimeout(update, Math.min(remaining, 2_147_483_647));
    };
    update();
    return () => clearTimeout(timer);
  }, [hosted?.attempt.attempt_id, hosted?.expires_at]);
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
            ![
              "depository",
              "credit",
              "loan",
              "investment",
              "other",
              "payroll",
            ].includes(a.type) ||
            (a.subtype !== null &&
              !/^[a-z0-9][a-z0-9 -]{0,63}$/.test(a.subtype)) ||
            (a.iso_currency_code !== null &&
              !/^[A-Z]{3}$/.test(a.iso_currency_code)),
        ))
    )
      throw Error("response");
    return value;
  };
  // Proposals come only from the reviewed book/currency snapshot and remain editable before submission.
  const proposeBindings = (bindingDate: string) => {
    const used = new Set(book.accounts.map((account) => account.code));
    const nextCode = { asset: 1100, credit: 2100, loan: 2200 };
    let ordinal = 0;
    const proposals: typeof selection = {};
    for (const account of attempt?.manifest?.accounts ?? []) {
      const classification = automaticClassification(
        account.type,
        account.subtype,
      );
      if (!classification || !account.iso_currency_code) continue;
      const currency = currencies.find(
        (candidate) =>
          candidate.alphabetic_code === account.iso_currency_code &&
          candidate.minor_unit !== null &&
          candidate.valid_from <= bindingDate &&
          (!candidate.valid_to || bindingDate < candidate.valid_to),
      );
      if (!currency) continue;
      const group =
        account.type === "depository"
          ? "asset"
          : account.type === "credit"
            ? "credit"
            : "loan";
      while (used.has(String(nextCode[group]))) nextCode[group] += 1;
      const code = String(nextCode[group]);
      used.add(code);
      nextCode[group] += 1;
      ordinal += 1;
      proposals[account.account_id] = {
        ledger: crypto.randomUUID(),
        currency: currency.currency_version_id,
        create: true,
        code,
        name: `${classification.label} ${account.subtype} ${ordinal}`,
      };
    }
    setSelection(proposals);
    setConsent(false);
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
    if (!current.live) return null;
    // A failed Item listing must not hide successful durable-attempt discovery.
    let page: AttemptPage | null = null;
    if (discovery.status === "fulfilled") {
      page = checkAttempts(discovery.value);
      setAttempts(page);
      if (page.unfinished_count === "0") setEnrollmentStarted(false);
    }
    if (listing.status === "fulfilled") {
      if (!validList(listing.value)) throw Error("response");
      setList(listing.value);
    }
    if (discovery.status === "rejected") throw discovery.reason;
    if (listing.status === "rejected") throw listing.reason;
    return page;
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
    if (!locked)
      void run(async () => {
        await readList();
      });
    return () => {
      current.live = false;
      current.abort.abort();
    };
    // Scope and lock are handled by the keyed outer component; API must be stable.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api]);
  // Finish defaults before the review form becomes interactive; never overwrite a reviewed user date later.
  useLayoutEffect(() => {
    if (
      (!prepareRegistration || advanced) &&
      attempt?.state === "binding_review" &&
      attempt.manifest &&
      !bindingExpired &&
      !date
    ) {
      const today = new Date().toISOString().slice(0, 10);
      setDate(today);
      proposeBindings(today);
    }
    // Proposal inputs are immutable within the keyed book workspace.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attempt, bindingExpired, date, advanced, prepareRegistration]);
  // Inspect does not recreate Link or replay exchange after uncertainty.
  const inspect = (id: string, expired = false, bindingExpiry = false) =>
    run(async () => {
      const current = life.current;
      setAttempt(null);
      setAdvanced(false);
      const discovered = attempts?.items.find((a) => a.attempt_id === id);
      setExpiredAttempt(discovered?.expired_unclaimed ?? expired);
      setBindingExpired(discovered?.binding_expired ?? bindingExpiry);
      setSelection({});
      setConsent(false);
      setCancelReview("");
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
          "This bank setup is unavailable. Your account access was rechecked; no bank setup was completed.",
        );
        return;
      }
      if (!current.live) return;
      setAttempt(checkedAttempt(value, id));

      // Recheck discovery before releasing an earlier local enrollment latch.
      if (discovered?.binding_expired ?? bindingExpiry) await readList();
    });
  // A fresh Begin reply alone authorizes a new, memory-only Link handoff.
  const beginNewConnection = async (current: typeof life.current) => {
    if (!link || !current.live) return;
    setEnrollmentStarted(true);
    setExpiredAttempt(false);
    setBindingExpired(false);
    setAttempt(null);
    setSelection({});
    setConsent(false);
    setCancelReview("");
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
        "Open Plaid in a new tab, finish setup, then return here to check completion.",
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
        "Bank setup canceled. You can review any unfinished setup below.",
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
  };
  const start = () =>
    run(async () => {
      if (
        !link ||
        !attempts ||
        attempts.unfinished_count !== "0" ||
        pending ||
        (enrollmentStarted &&
          !["active", "disconnected"].includes(attempt?.state ?? ""))
      )
        return;
      await beginNewConnection(life.current);
    });
  // Cancellation is durable and serialized against exchange by the server. A lost
  // reply is resolved by reading the same attempt; never infer success locally.
  const cancelSetup = (old: Attempt, retry: boolean) =>
    run(async () => {
      const current = life.current;
      const path = `${base}/connection-attempts/${old.attempt_id}`;
      let result: Attempt;
      try {
        result = checkedAttempt(
          await api<Attempt>(
            `${path}/abandon`,
            JSON.stringify({ confirm_unrecoverable: true }),
          ),
          old.attempt_id,
        );
      } catch {
        result = checkedAttempt(await api<Attempt>(path), old.attempt_id);
      }
      if (!current.live) return;
      setAttempt(result);
      const page = await readList();
      if (!current.live) return;
      if (result.state !== "abandoned") {
        setMessage(
          "Setup could not be canceled. Check its status before trying again.",
        );
        return;
      }
      setHosted(null);
      setEnrollmentStarted(false);
      setCancelReview("");
      setMessage(
        "Setup canceled. You can connect an account whenever you’re ready.",
      );
      if (retry && page?.unfinished_count === "0")
        await beginNewConnection(current);
    });
  // Retry first resolves the old claim, then uses the server's stored-credential
  // guard before starting a new Link session. An uncertain reply stops here.
  const retryConnection = (old: Attempt | null = attempt, retry = true) =>
    run(async () => {
      const current = life.current;
      if (
        (retry && !link) ||
        !old ||
        hosted ||
        pending ||
        old.mode !== "new" ||
        old.sync_item_id !== null ||
        old.state !== "credential_recovery_required"
      )
        return;
      const path = `${base}/connection-attempts/${old.attempt_id}`;
      let recovered: Attempt;
      let credentialAbsent = false;
      try {
        recovered = checkedAttempt(
          await api<Attempt>(`${path}/reconcile`, "{}"),
          old.attempt_id,
        );
      } catch (error) {
        if (
          !(error instanceof APIError) ||
          error.status !== 409 ||
          error.uncertain
        )
          throw error;
        credentialAbsent = true;
        recovered = checkedAttempt(await api<Attempt>(path), old.attempt_id);
      }
      if (!current.live) return;
      setAttempt(recovered);
      if (recovered.state !== "credential_recovery_required") {
        await readList();
        if (current.live)
          setMessage("Connection status updated. Review it below.");
        return;
      }
      if (!credentialAbsent) {
        setMessage(
          "Connection status is still pending. Try again in a moment.",
        );
        return;
      }
      let abandoned: Attempt;
      try {
        abandoned = checkedAttempt(
          await api<Attempt>(
            `${path}/abandon`,
            JSON.stringify({ confirm_unrecoverable: true }),
          ),
          old.attempt_id,
        );
      } catch (error) {
        if (
          error instanceof APIError &&
          error.status === 409 &&
          !error.uncertain
        ) {
          const fresh = checkedAttempt(
            await api<Attempt>(path),
            old.attempt_id,
          );
          if (!current.live) return;
          setAttempt(fresh);
          await readList();
          if (current.live)
            setMessage(
              "Connection status changed. Check it before trying again.",
            );
          return;
        }
        throw error;
      }
      if (!current.live) return;
      if (abandoned.state !== "abandoned") throw Error("response");
      setAttempt(abandoned);
      const page = await readList();
      if (!current.live) return;
      if (!page || page.unfinished_count !== "0") {
        setMessage(
          "Another setup needs attention. Check its status before trying again.",
        );
        return;
      }
      setHosted(null);
      setEnrollmentStarted(false);
      if (retry) await beginNewConnection(current);
      else
        setMessage(
          "Setup canceled. You can connect an account whenever you’re ready.",
        );
    });
  // A conflict requires inspection of this exact attempt, never another exchange or automatic restart.
  const inspectCompletionConflict = async (
    error: unknown,
    id: string,
    current: typeof life.current,
  ): Promise<boolean> => {
    if (!(error instanceof APIError) || error.status !== 409 || !current.live)
      return false;
    const recovered = checkedAttempt(
      await api<Attempt>(`${base}/connection-attempts/${id}`),
      id,
    );
    if (!current.live) return true;
    setAttempt(recovered);

    if (recovered.state === "link_ready") {
      const discovery = await readList();
      if (!current.live) return true;
      const discovered = discovery?.items.find(
        (item) => item.attempt_id === id,
      );
      const expired =
        discovered?.expired_unclaimed ??
        Date.parse(recovered.expires_at) <= Date.now();
      setExpiredAttempt(expired);
      if (expired) {
        setMessage(
          "This setup's recovery window has ended. Its exchange will not be retried. Review the current connection list before explicitly starting a fresh setup.",
        );
        return true;
      }
      setMessage(
        "This bank setup has not been completed in Jayburd. If you exited Plaid, use Cancel setup or Try again below. Otherwise finish Plaid and check this attempt again.",
      );
    } else if (recovered.state === "credential_recovery_required") {
      setMessage(
        "Bank setup needs attention. Review it below before connecting another bank.",
      );
    } else {
      await readList();
    }
    return true;
  };
  // An explicit check uses the server-retained encrypted Link token; reload does not lose it.
  const completeHosted = () =>
    run(async () => {
      const current = life.current;
      const handoff = hosted;
      if (!handoff) return;
      setHosted(null);
      let value: Attempt;
      try {
        value = await api<Attempt>(
          `${base}/connection-attempts/${handoff.attempt.attempt_id}/complete`,
          "{}",
        );
      } catch (error) {
        if (
          await inspectCompletionConflict(
            error,
            handoff.attempt.attempt_id,
            current,
          )
        )
          return;
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
  // Only explicitly opening this Hosted Link arms bounded same-attempt completion on return.
  useEffect(() => {
    if (!prepareRegistration || !hosted || locked) return;
    const onReturn = () => {
      const armed = autoReturn.current;
      if (
        !armed ||
        armed.attempt !== hosted.attempt.attempt_id ||
        armed.checks >= 3 ||
        busy ||
        pending ||
        life.current.flight
      )
        return;
      armed.checks++;
      void completeHosted();
    };
    window.addEventListener("focus", onReturn);
    return () => window.removeEventListener("focus", onReturn);
    // The scope/lock-keyed workspace owns the closure and finite in-memory counter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [prepareRegistration, hosted, locked, busy, pending]);
  // Recovery checks the exact durable attempt without needing a browser-held handoff.
  const checkHostedAttempt = (id: string) =>
    run(async () => {
      const current = life.current;
      let value: Attempt;
      try {
        value = checkedAttempt(
          await api<Attempt>(
            `${base}/connection-attempts/${id}/complete`,
            "{}",
          ),
          id,
        );
      } catch (error) {
        if (await inspectCompletionConflict(error, id, current)) return;
        throw error;
      }
      if (!current.live) return;
      setAttempt(value);

      if (value.state === "link_ready") {
        setMessage(
          "Plaid has not reported a completed bank setup yet. Check again after finishing there.",
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
      const blockers = bindingBlockers(selection, date, consent);
      if (
        !attempt ||
        bindingExpired ||
        attempt.state !== "binding_review" ||
        pending ||
        blockers.length
      )
        return;
      const bindings: Binding["bindings"] = Object.entries(selection)
        .filter(([, v]) => v.ledger && v.currency)
        .map(([account_id, v]) => {
          const account = attempt.manifest?.accounts.find(
            (item) => item.account_id === account_id,
          );
          const classification =
            account && automaticClassification(account.type, account.subtype);
          if (v.create && !classification) throw Error("binding");
          return {
            account_id,
            ledger_account_id: v.ledger,
            currency_version_id: v.currency,
            source_account_id: null,
            expected_binding_revision: null,
            binding_date: date,
            binding_evidence_ref: crypto.randomUUID(),
            ...(v.create
              ? {
                  new_ledger_account: {
                    account_id: v.ledger,
                    code: v.code,
                    name: v.name,
                    category: classification!.category,
                    normal_side: classification!.side,
                    monetary: true as const,
                    active_from: date,
                  },
                }
              : {}),
          };
        });
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
  // Show only actionable setup history; discovery still scans the server-owned pages.
  const recoveryAttempts =
    attempts?.items.filter(
      (a) =>
        !["active", "disconnected", "abandoned"].includes(a.state) ||
        a.binding_expired ||
        a.expired_unclaimed,
    ) ?? [];
  return (
    <section aria-label="Connections" aria-busy={busy}>
      {hosted?.hosted_link_url && (
        <div role="group" aria-label="Complete bank setup">
          {hostedURLExpired ? (
            <p>
              This Plaid link has expired. If you already finished there, check
              completion below while the setup recovery window remains open.
            </p>
          ) : (
            <>
              <p>
                Finish bank setup in Plaid, then return here. If this tab
                closes, sign in and review the unfinished setup to check
                completion.
              </p>
              <a
                href={hosted.hosted_link_url}
                target="_blank"
                rel="noopener noreferrer"
                referrerPolicy="no-referrer"
                onClick={(event) => {
                  // Background tabs may delay timers; recheck expiry at the navigation boundary.
                  const expiry = Date.parse(hosted.expires_at ?? "");
                  if (!Number.isFinite(expiry) || expiry <= Date.now()) {
                    event.preventDefault();
                    setHostedURLExpired(true);
                  } else if (
                    prepareRegistration &&
                    autoReturn.current?.attempt !== hosted.attempt.attempt_id
                  ) {
                    autoReturn.current = {
                      attempt: hosted.attempt.attempt_id,
                      checks: 0,
                    };
                  }
                }}
              >
                Open Plaid
              </a>
              <p>Plaid link expires: {hosted.expires_at}</p>
              {prepareRegistration && (
                <p>
                  Return here after bank setup and we’ll check completion. You
                  can also check manually.
                </p>
              )}
            </>
          )}
          <button
            disabled={locked || busy}
            onClick={() => void completeHosted()}
          >
            Check bank setup completion
          </button>
          <button
            disabled={locked || busy}
            onClick={() => void cancelSetup(hosted.attempt, false)}
          >
            Cancel bank setup
          </button>
        </div>
      )}
      <div className="connections-header">
        <div>
          <h2>Accounts</h2>
          <p>
            Connect a bank and review the accounts you share with {book.name}.
          </p>
        </div>
        <div className="connections-header-actions">
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
            Connect bank
          </button>
          <button
            type="button"
            className="secondary"
            disabled={unavailable}
            onClick={() =>
              void run(async () => {
                setList(null);
                await readList();
              })
            }
          >
            Refresh
          </button>
        </div>
      </div>
      {attempts?.items
        .filter(
          (a) =>
            a.mode === "new" &&
            [
              "link_pending",
              "link_ready",
              "credential_recovery_required",
            ].includes(a.state) &&
            !a.expired_unclaimed &&
            !a.binding_expired,
        )
        .map((a) => (
          <div
            key={a.attempt_id}
            role="group"
            aria-label="Unfinished bank setup"
          >
            {prepareRegistration && !hosted && a.state === "link_ready" && (
              <button
                type="button"
                disabled={mutable}
                onClick={() => void inspect(a.attempt_id)}
              >
                Check bank setup status
              </button>
            )}
            <p>
              Your previous bank setup hasn’t finished. Try again to start a
              fresh session, or cancel setup.
            </p>
            <button
              disabled={mutable || !link}
              onClick={() =>
                void (a.state === "credential_recovery_required"
                  ? retryConnection(a, true)
                  : cancelSetup(a, true))
              }
            >
              Try again
            </button>{" "}
            <button
              disabled={mutable}
              onClick={() =>
                void (a.state === "credential_recovery_required"
                  ? retryConnection(a, false)
                  : cancelSetup(a, false))
              }
            >
              Cancel setup
            </button>
          </div>
        ))}
      {!link && <p>New bank setup is unavailable in this deployment.</p>}
      {message && <p role="alert">{message}</p>}
      {locked && (
        <p>
          Finish the current bookkeeping recovery before managing connections.
        </p>
      )}
      {list === null ? (
        <p>Connection status unavailable.</p>
      ) : list.items.length === 0 ? (
        <p>No bank accounts connected yet.</p>
      ) : (
        list.items.map((item, itemIndex) => (
          <article key={item.sync_item_id} className="connection-card">
            <div className="connection-heading">
              <h3>Bank connection {itemIndex + 1}</h3>
              <span className="connection-status">
                {item.state === "active"
                  ? "Connected"
                  : item.state.replaceAll("_", " ")}
              </span>
            </div>
            {prepareRegistration &&
              attempts?.items.some(
                (candidate) =>
                  candidate.sync_item_id === item.sync_item_id &&
                  candidate.mode === "new" &&
                  candidate.state === "active",
              ) && (
                <button
                  type="button"
                  disabled={mutable}
                  onClick={() => {
                    const owned = attempts.items.find(
                      (candidate) =>
                        candidate.sync_item_id === item.sync_item_id &&
                        candidate.mode === "new" &&
                        candidate.state === "active",
                    );
                    if (owned) void inspect(owned.attempt_id);
                  }}
                >
                  Add shared accounts
                </button>
              )}
            {item.accounts.length === 0 && (
              <p>
                No registered accounts are available in this book for this
                connection.
              </p>
            )}
            <ul className="connected-account-list">
              {item.accounts.map((account) => (
                <li
                  key={account.stream_id}
                  className="connected-account"
                  aria-label={`Connected ${accountLabel(account.account_id)}`}
                >
                  <div className="connected-account-heading">
                    <h4>{accountLabel(account.account_id)}</h4>
                    <p>Connection: {account.state.replaceAll("_", " ")}</p>
                  </div>
                  {prepareRegistration && autoStatus.has(account.stream_id) && (
                    <AccountImportStatus
                      api={api}
                      entity={entity.entity_id}
                      book={book.book_id}
                      item={item.sync_item_id}
                      account={account}
                      locked={mutable}
                      denied={denied}
                    />
                  )}
                  <div className="connected-account-actions">
                    <button
                      className="secondary"
                      aria-label={`Review transactions for ${accountLabel(account.account_id)}`}
                      disabled={unavailable}
                      onClick={() => onReview(account.source_account_id)}
                    >
                      Transactions
                    </button>
                    <button
                      className="secondary"
                      aria-label={`View sync status for ${accountLabel(account.account_id)}`}
                      disabled={mutable}
                      onClick={() =>
                        void readStatus(account, item.sync_item_id)
                      }
                    >
                      Sync status
                    </button>
                  </div>
                  <details className="account-management">
                    <summary>Manage {accountLabel(account.account_id)}</summary>
                    <button
                      className="secondary"
                      disabled={mutable || account.state === "disconnected"}
                      onClick={() => setDisconnect(account)}
                    >
                      Disconnect {accountLabel(account.account_id)}
                    </button>
                  </details>
                </li>
              ))}
            </ul>
          </article>
        ))
      )}
      {prepareRegistration &&
        attempts?.items
          .filter(
            (candidate) =>
              candidate.mode === "new" &&
              [
                "binding_review",
                "credential_pending",
                "exchange_claimed",
              ].includes(candidate.state) &&
              !candidate.binding_expired &&
              !candidate.expired_unclaimed,
          )
          .map((candidate, index, rows) => (
            <button
              key={candidate.attempt_id}
              type="button"
              disabled={mutable}
              onClick={() => void inspect(candidate.attempt_id)}
            >
              {candidate.state === "binding_review"
                ? "Resume account selection"
                : "Review unfinished bank setup"}
              {rows.length > 1 ? ` ${index + 1}` : ""}
            </button>
          ))}
      {(!attempts ||
        recoveryAttempts.length > 0 ||
        attempts.unfinished_count !== "0") && (
        <section
          aria-label="Unfinished connections"
          className="connection-recovery-list"
        >
          <h3>Finish connecting your bank</h3>
          {!attempts ? (
            <p>
              We couldn’t check unfinished bank setup. Refresh before connecting
              another bank.
            </p>
          ) : (
            <>
              {attempts.unfinished_count !== "0" && (
                <p>Unfinished bank setups: {attempts.unfinished_count}</p>
              )}
              {recoveryAttempts.length === 0 && (
                <p>No unfinished setup on this page.</p>
              )}
              <ul>
                {recoveryAttempts.map((a, index) => (
                  <li key={a.attempt_id} className="connection-recovery-item">
                    <span>
                      {a.binding_expired
                        ? "Account selection expired"
                        : a.expired_unclaimed
                          ? "Bank setup expired"
                          : "Bank setup needs attention"}
                    </span>
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
                      Review bank setup
                      {recoveryAttempts.length > 1 ? ` ${index + 1}` : ""}
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
                  Check earlier bank setups
                </button>
              )}
            </>
          )}
        </section>
      )}
      {status && (
        <section aria-label="Account sync status">
          <h3>Sync status for {accountLabel(statusAccount)}</h3>
          <p>Account sync is {status.state.replaceAll("_", " ")}.</p>
          <p>
            {status.readiness === "HISTORICAL_UPDATE_COMPLETE"
              ? "Historical import completed"
              : status.readiness === "INITIAL_UPDATE_COMPLETE"
                ? "Initial history imported"
                : "Import not completed yet"}
          </p>
          <p>
            {status.delivery_state === "quarantined"
              ? "Sync needs attention. Try again after resolving the connection issue."
              : status.delivery_state === "retry_wait"
                ? "Sync is waiting to retry."
                : status.lease_active
                  ? "Sync is in progress. New transactions are not confirmed yet."
                  : status.queued
                    ? "Sync is queued. New transactions are not confirmed yet."
                    : "No sync is waiting."}
          </p>
          <p>
            Last import: {status.last_completed_at ?? "No completed import yet"}
          </p>
          {(status.decoder_error || status.last_error) && (
            <p role="status">
              Some transactions could not be imported. Your posted entries have
              not changed.
            </p>
          )}
          <p>
            Imported transactions still need review. An import does not confirm
            complete bank history.
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
                  ? "Request a sync"
                  : operation === "pause"
                    ? "Pause syncing"
                    : "Resume syncing"}{" "}
                for {accountLabel(statusAccount)} only. Sibling accounts and
                posted journals are unchanged.
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
            Stop syncing for {accountLabel(disconnect.account_id)} only. Other
            accounts and posted history are retained. Your bank connection is
            retained for the other accounts.
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
              ? "account setup"
              : pending.kind === "stream"
                ? "sync change"
                : "disconnect"}{" "}
            result is unconfirmed. Check again to safely finish the same
            request.
          </p>
          <button
            disabled={unavailable}
            onClick={() => void run(() => submit(pending))}
          >
            Resolve saved request
          </button>
        </div>
      )}
      {attempt && (
        <article>
          <h3>Bank setup</h3>
          <p>Setup status: {setupLabel(attempt.state)}</p>
          {!["active", "disconnected", "abandoned"].includes(attempt.state) && (
            <button
              disabled={mutable}
              onClick={() =>
                void inspect(attempt.attempt_id, expiredAttempt, bindingExpired)
              }
            >
              Check this bank setup
            </button>
          )}
          {bindingExpired && (
            <p>
              Account selection expired. Start a fresh connection; your earlier
              setup history is retained.
            </p>
          )}
          {expiredAttempt && (
            <p>
              This bank setup expired before it completed. Connect again to
              start a fresh session.
            </p>
          )}
          {attempt.mode === "new" &&
            attempt.state === "link_ready" &&
            !expiredAttempt &&
            !hosted && (
              <button
                disabled={mutable}
                onClick={() => void checkHostedAttempt(attempt.attempt_id)}
              >
                Check bank setup completion
              </button>
            )}
          {attempt.state === "credential_recovery_required" &&
          attempt.mode === "new" &&
          attempt.sync_item_id === null ? (
            <>
              <p>
                This bank setup did not finish. Trying again checks for a saved
                connection before opening a new Plaid session. The earlier
                attempt stays in your history.
              </p>
              <button
                disabled={mutable || !link}
                onClick={() => void retryConnection()}
              >
                Try connecting again
              </button>
            </>
          ) : ["credential_pending", "credential_recovery_required"].includes(
              attempt.state,
            ) ? (
            <button
              disabled={mutable}
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
              Check connection status
            </button>
          ) : null}
          {prepareRegistration &&
            !advanced &&
            !expiredAttempt &&
            !bindingExpired &&
            attempt.mode === "new" &&
            ["binding_review", "active"].includes(attempt.state) &&
            attempt.manifest && (
              <RegistrationPicker
                api={api}
                entity={entity.entity_id}
                book={book.book_id}
                attempt={attempt.attempt_id}
                locked={mutable}
                prepare={(ids) =>
                  prepareRegistration(
                    entity.entity_id,
                    book.book_id,
                    attempt.attempt_id,
                    ids,
                  )
                }
                denied={denied}
                onReview={onReview}
              />
            )}
          {prepareRegistration &&
            !expiredAttempt &&
            !bindingExpired &&
            attempt.state === "binding_review" &&
            attempt.manifest && (
              <button
                type="button"
                disabled={mutable}
                aria-expanded={advanced}
                onClick={() => setAdvanced((current) => !current)}
              >
                {advanced ? "Hide advanced mapping" : "Advanced mapping"}
              </button>
            )}
          {(!prepareRegistration || advanced) &&
            !expiredAttempt &&
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
                        proposeBindings(e.target.value);
                      }}
                    />
                  </label>
                  <button
                    type="button"
                    disabled={!date}
                    onClick={() => proposeBindings(date)}
                  >
                    Create distinct accounts for all supported bank accounts
                  </button>
                  <p>
                    Labels identify accounts in this view. Bank names and masked
                    numbers are not available here; labels may change when the
                    account set changes.
                  </p>
                  {attempt.manifest.accounts.map((account) => (
                    <div key={account.account_id}>
                      <p>
                        {accountLabel(account.account_id)} — {account.type}
                        {account.subtype ? ` / ${account.subtype}` : ""},{" "}
                        {account.iso_currency_code ?? "currency unavailable"}
                      </p>
                      {automaticClassification(account.type, account.subtype) &&
                      account.iso_currency_code ? (
                        <>
                          <label>
                            Ledger account for{" "}
                            {accountLabel(account.account_id)}
                            <select
                              value={
                                selection[account.account_id]?.create
                                  ? "new"
                                  : (selection[account.account_id]?.ledger ??
                                    "")
                              }
                              onChange={(e) => {
                                // Skipping a source removes its whole proposal, including hidden fields.
                                if (!e.target.value) {
                                  setSelection((current) => {
                                    const next = { ...current };
                                    delete next[account.account_id];
                                    return next;
                                  });
                                  setConsent(false);
                                  return;
                                }
                                const previous = selection[account.account_id];
                                const create = e.target.value === "new";
                                setSelection((s) => ({
                                  ...s,
                                  [account.account_id]: {
                                    ledger: create
                                      ? previous?.create && previous.ledger
                                        ? previous.ledger
                                        : crypto.randomUUID()
                                      : e.target.value,
                                    currency: previous?.currency ?? "",
                                    create,
                                    code: previous?.code ?? "",
                                    name: previous?.name ?? "",
                                  },
                                }));
                                setConsent(false);
                              }}
                            >
                              <option value="">Do not bind</option>
                              <option value="new">
                                Create a new ledger account
                              </option>
                              {book.accounts
                                .filter(
                                  (a) =>
                                    date &&
                                    a.category ===
                                      automaticClassification(
                                        account.type,
                                        account.subtype,
                                      )?.category &&
                                    a.normal_side ===
                                      automaticClassification(
                                        account.type,
                                        account.subtype,
                                      )?.side &&
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
                          {selection[account.account_id]?.create && (
                            <>
                              <label>
                                Account code for{" "}
                                {accountLabel(account.account_id)}
                                <input
                                  required
                                  maxLength={64}
                                  value={
                                    selection[account.account_id]?.code ?? ""
                                  }
                                  onChange={(e) => {
                                    setSelection((current) => ({
                                      ...current,
                                      [account.account_id]: {
                                        ...current[account.account_id],
                                        code: e.target.value,
                                      },
                                    }));
                                    setConsent(false);
                                  }}
                                />
                              </label>
                              <label>
                                Account name for{" "}
                                {accountLabel(account.account_id)}
                                <input
                                  required
                                  maxLength={200}
                                  value={
                                    selection[account.account_id]?.name ?? ""
                                  }
                                  onChange={(e) => {
                                    setSelection((current) => ({
                                      ...current,
                                      [account.account_id]: {
                                        ...current[account.account_id],
                                        name: e.target.value,
                                      },
                                    }));
                                    setConsent(false);
                                  }}
                                />
                              </label>
                            </>
                          )}
                          <label>
                            Currency version for{" "}
                            {accountLabel(account.account_id)}
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
                                    create:
                                      s[account.account_id]?.create ?? false,
                                    code: s[account.account_id]?.code ?? "",
                                    name: s[account.account_id]?.name ?? "",
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
                  {(() => {
                    const blockers = bindingBlockers(selection, date, consent);
                    return blockers.length ? (
                      <p id="binding-readiness" role="status">
                        To confirm: {blockers.join("; ")}.
                      </p>
                    ) : (
                      <p id="binding-readiness" role="status">
                        All selected bindings are ready to confirm.
                      </p>
                    );
                  })()}
                  <button
                    aria-describedby="binding-readiness"
                    disabled={
                      bindingBlockers(selection, date, consent).length > 0
                    }
                  >
                    Confirm selected bindings
                  </button>
                </fieldset>
              </form>
            )}
          {attempt.mode === "new" &&
            ["binding_review", "disconnect_pending"].includes(
              attempt.state,
            ) && (
              <section aria-label="Cancel bank setup">
                {attempt.state === "disconnect_pending" && (
                  <p>
                    Cancellation is waiting for the bank connection to be
                    removed. Resume the same cancellation; do not start another
                    bank connection.
                  </p>
                )}
                {cancelReview !== attempt.attempt_id ? (
                  <button
                    type="button"
                    disabled={mutable}
                    onClick={() => setCancelReview(attempt.attempt_id)}
                  >
                    {attempt.state === "disconnect_pending"
                      ? "Resume canceling setup"
                      : "Cancel setup and start over"}
                  </button>
                ) : (
                  <fieldset disabled={mutable}>
                    <legend>Confirm bank setup cancellation</legend>
                    <p>
                      Remove this unfinished bank connection and keep its
                      history. No ledger accounts or journals will be created. A
                      new setup becomes available only after removal is
                      confirmed.
                    </p>
                    <button
                      type="button"
                      onClick={() => void cancelSetup(attempt, !!link)}
                    >
                      Confirm cancellation
                    </button>{" "}
                    <button type="button" onClick={() => setCancelReview("")}>
                      Keep this setup
                    </button>
                  </fieldset>
                )}
              </section>
            )}
        </article>
      )}
    </section>
  );
}
