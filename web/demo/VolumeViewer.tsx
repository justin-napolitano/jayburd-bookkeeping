import { useEffect, useRef, useState } from "react";
import { createAPI } from "../src/bookkeeping/api";
import type {
  DurableCommand,
  Journal,
  PendingCommand,
  RegisterCommand,
  TrialBalance,
  Workspace,
} from "../src/bookkeeping/types";
import type { components } from "../src/lib/api.generated";
import { decimal, depositCents, expectedCents, parseCents } from "./volume";
type Setup = components["schemas"]["SetupResult"];
type Work = {
  key: string;
  request: RegisterCommand;
  command?: DurableCommand;
  checked: boolean;
};
type Run = {
  count: number;
  posted: number;
  started: number;
  setup?: Setup;
  work?: Work;
  ids: Set<string>;
  recent: string[];
  revision: string;
  name: string;
  verified: boolean;
  actualBalance?: string;
};
const api = createAPI(
  async () => "synthetic.browser.proof",
  () => {},
);
// Synthetic-only runner sends real sequential commands; it never bypasses ledger validation.
export function VolumeViewer({
  onChanged,
  onActiveChange,
}: {
  onChanged: () => void;
  onActiveChange?: (active: boolean) => void;
}) {
  const [count, setCount] = useState(1000),
    [view, setView] = useState<Run | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [phase, setPhase] = useState("Ready");
  const run = useRef<Run | null>(null),
    alive = useRef(true),
    pause = useRef(false),
    working = useRef(false);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
      pause.current = true;
    };
  }, []);
  // Clone only the bounded visible receipt list; UUID uniqueness state stays off the DOM.
  function publish(r: Run) {
    if (alive.current) setView({ ...r, recent: [...r.recent] });
  }
  function guard() {
    if (!alive.current)
      throw new Error("Closed; recover pending work after reopening.");
  }
  // Retained key and command identity survive uncertain replies and are reused on retry.
  async function finishWork(r: Run) {
    const w = r.work!;
    guard();
    if (!w.command) {
      setPhase("Saving immutable request");
      w.command = await api<DurableCommand>(
        "/v1/commands",
        JSON.stringify(w.request),
        w.key,
      );
    }
    guard();
    if (w.command.state === "prepared") {
      setPhase("Executing SQL transaction");
      w.command = await api<DurableCommand>(
        `/v1/commands/${w.command.command_id}/execute`,
        "{}",
      );
    }
    guard();
    if (w.command.state !== "committed" || !w.command.result)
      throw new Error("Command did not commit; review pending work.");
    if (!w.checked) {
      if (w.request.operation === "setup") {
        const s = w.command.result as Setup;
        if (
          !s.accounts.find((a) => a.code === "1000") ||
          !s.accounts.find((a) => a.code === "2000")
        )
          throw new Error("Setup receipt mismatch");
        r.setup = s;
        r.revision = s.book_revision;
      } else {
        const j = w.command.result as Journal,
          s = r.setup!,
          amount = depositCents(r.posted);
        const bank = s.accounts.find((a) => a.code === "1000")!.account_id,
          capital = s.accounts.find((a) => a.code === "2000")!.account_id;
        if (
          r.ids.has(j.journal_id) ||
          j.entity_id !== s.entity_id ||
          j.book_id !== s.book_id ||
          BigInt(j.book_revision) !== BigInt(r.revision) + 1n ||
          j.lines.length !== 2 ||
          !j.lines.some((l) => l.account_id === bank && l.side === "debit") ||
          !j.lines.some(
            (l) => l.account_id === capital && l.side === "credit",
          ) ||
          j.lines.some(
            (l) =>
              l.transaction_currency !== "USD" ||
              parseCents(l.functional_amount) !== amount ||
              parseCents(l.transaction_amount) !== amount,
          )
        )
          throw new Error(
            "Committed receipt differs from the exact-cent oracle. Stop and review.",
          );
        r.ids.add(j.journal_id);
        r.revision = j.book_revision;
        r.posted++;
        r.recent.unshift(
          `#${r.posted} · USD ${decimal(amount)} · journal ${j.journal_id}`,
        );
        r.recent = r.recent.slice(0, 20);
      }
      w.checked = true;
      publish(r);
    }
    guard();
    setPhase("Acknowledging verified receipt");
    w.command = await api<DurableCommand>(
      `/v1/commands/${w.command.command_id}/acknowledge`,
      "{}",
    );
    if (!w.command.acknowledged || w.command.state !== "committed")
      throw new Error("Acknowledgement unconfirmed; retry the same operation.");
    r.work = undefined;
  }
  // Start explicitly authorizes the bounded synthetic sequence and receipt acknowledgements.
  async function drive(fresh: boolean) {
    if (working.current) return;
    working.current = true;
    pause.current = false;
    setBusy(true);
    setError("");
    onActiveChange?.(true);
    try {
      if (fresh) {
        const pending = await api<PendingCommand>("/v1/commands/pending");
        guard();
        if (pending.command)
          throw new Error(
            "Recover the existing command in the bookkeeping workspace before starting.",
          );
        const workspace = await api<Workspace>("/v1/workspace");
        guard();
        const currency = workspace.currencies.find(
          (c) =>
            c.alphabetic_code === "USD" &&
            c.minor_unit === 2 &&
            c.valid_from <= "2026-01-01" &&
            (!c.valid_to || c.valid_to > "2026-12-31"),
        );
        if (!currency)
          throw new Error(
            "USD metadata unavailable for this synthetic period.",
          );
        const r: Run = {
          count,
          posted: 0,
          started: performance.now(),
          ids: new Set(),
          recent: [],
          revision: "0",
          name: `Volume Lab ${crypto.randomUUID().slice(0, 8)}`,
          verified: false,
        };
        r.work = {
          key: crypto.randomUUID(),
          checked: false,
          request: {
            operation: "setup",
            entity_id: null,
            book_id: null,
            body: {
              entity_name: r.name,
              entity_kind: "legal",
              book_name: "Volume proof",
              starts_on: "2026-01-01",
              ends_before: "2027-01-01",
              functional_currency_version_id: currency.currency_version_id,
              accounts: [
                {
                  code: "1000",
                  name: "Bank",
                  category: "asset",
                  normal_side: "debit",
                  monetary: true,
                },
                {
                  code: "2000",
                  name: "Capital",
                  category: "equity",
                  normal_side: "credit",
                  monetary: false,
                },
              ],
            },
          },
        };
        run.current = r;
        publish(r);
      }
      const r = run.current!;
      while (alive.current) {
        if (r.work) await finishWork(r);
        guard();
        if (r.posted === r.count) break;
        if (pause.current) {
          setPhase("Paused after confirmed acknowledgement");
          return;
        }
        const s = r.setup!,
          amount = decimal(depositCents(r.posted));
        r.work = {
          key: crypto.randomUUID(),
          checked: false,
          request: {
            operation: "post",
            entity_id: s.entity_id,
            book_id: s.book_id,
            body: {
              expected_revision: r.revision,
              policy_id: s.policy_id,
              accounting_date: "2026-01-01",
              description: `Synthetic volume deposit ${r.posted + 1}`,
              lines: [
                {
                  account_id: s.accounts.find((a) => a.code === "1000")!
                    .account_id,
                  side: "debit",
                  transaction_currency: "USD",
                  transaction_amount: amount,
                  functional_amount: amount,
                  valuation_basis: "identity",
                },
                {
                  account_id: s.accounts.find((a) => a.code === "2000")!
                    .account_id,
                  side: "credit",
                  transaction_currency: "USD",
                  transaction_amount: amount,
                  functional_amount: amount,
                  valuation_basis: "identity",
                },
              ],
            },
          },
        };
      }
      guard();
      setPhase("Checking independent trial balance");
      const s = r.setup!,
        report = await api<TrialBalance>(
          `/v1/entities/${s.entity_id}/books/${s.book_id}/trial-balance?through_date=2026-12-31`,
        );
      guard();
      const expected = expectedCents(r.count),
        bank = report.accounts.find((a) => a.account_code === "1000"),
        capital = report.accounts.find((a) => a.account_code === "2000");
      if (
        report.entity_id !== s.entity_id ||
        report.book_id !== s.book_id ||
        report.functional_currency !== "USD" ||
        report.snapshot_revision !== r.revision ||
        r.ids.size !== r.count ||
        parseCents(report.total_debits) !== expected ||
        parseCents(report.total_credits) !== expected ||
        !bank ||
        !capital ||
        parseCents(bank.debit_balance) !== expected ||
        parseCents(bank.credit_balance) !== 0n ||
        parseCents(capital.credit_balance) !== expected ||
        parseCents(capital.debit_balance) !== 0n
      )
        throw new Error(
          "Final trial balance differs from independent exact-cent expectations.",
        );
      r.actualBalance = decimal(parseCents(bank.debit_balance));
      r.verified = true;
      publish(r);
      setPhase("Volume run verified");
    } catch (e) {
      if (alive.current) {
        setError(
          e instanceof Error ? e.message : "Run stopped; recover pending work.",
        );
        setPhase("Stopped for review");
      }
    } finally {
      working.current = false;
      if (alive.current) {
        setBusy(false);
        onActiveChange?.(false);
        onChanged();
      }
    }
  }
  const seconds = view ? (performance.now() - view.started) / 1000 : 0;
  return (
    <section aria-label="Volume transaction proof">
      <h2>Watch thousands of real postings</h2>
      <p>
        This synthetic run creates a separate USD book. Each deposit saves a
        command, commits through Go and PostgreSQL, verifies returned cents,
        then acknowledges it. Start authorizes acknowledgements for this run.
      </p>
      <label>
        Transaction count
        <select
          value={count}
          disabled={busy || (!!view && !view.verified)}
          onChange={(e) => setCount(Number(e.target.value))}
        >
          {[1000, 5000, 10000].map((n) => (
            <option key={n} value={n}>
              {n.toLocaleString("en-US")}
            </option>
          ))}
        </select>
      </label>
      <button
        disabled={busy || (!!view && !view.verified)}
        onClick={() => void drive(true)}
      >
        Start volume run
      </button>
      {busy && (
        <button
          onClick={() => {
            pause.current = true;
            setPhase("Pausing after current transaction");
          }}
        >
          Pause volume run
        </button>
      )}
      {!busy && view && !view.verified && (
        <button onClick={() => void drive(false)}>
          {error ? "Retry same volume operation" : "Resume volume run"}
        </button>
      )}
      <p role="status">{phase}</p>
      {error && <p role="alert">{error}</p>}
      {view && (
        <>
          <p>{view.name}</p>
          <progress
            max={view.count}
            value={view.posted}
            aria-label="Committed transactions"
          />
          <p>
            {view.posted.toLocaleString("en-US")} /{" "}
            {view.count.toLocaleString("en-US")} transactions committed ·{" "}
            {seconds.toFixed(1)}s elapsed ·{" "}
            {seconds ? (view.posted / seconds).toFixed(1) : "0"} postings/second
          </p>
          <p>
            Expected final bank debit and capital credit: USD{" "}
            {decimal(expectedCents(view.count))}.{" "}
            {view.verified &&
              "Actual trial balance matches; every returned journal ID was unique."}
          </p>
          {view.verified && (
            <p>
              Actual bank debit and capital credit from PostgreSQL: USD{" "}
              {view.actualBalance}.
            </p>
          )}
          <ol aria-label="Recent committed transactions">
            {view.recent.map((row) => (
              <li key={row}>{row}</li>
            ))}
          </ol>
        </>
      )}
      <p>
        Pause finishes the current transaction. Closing stops scheduling; a
        request already sent can still commit. Recover pending work in the
        bookkeeping workspace after reopening. Throughput measures this local
        harness, not production capacity.
      </p>
    </section>
  );
}
