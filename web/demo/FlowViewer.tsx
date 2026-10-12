import React, { useCallback, useEffect, useRef, useState } from "react";
import { APIError, createAPI } from "../src/bookkeeping/api";
import { currencyAt, formatAmount } from "../src/bookkeeping/amount";
import type {
  DurableCommand,
  PendingCommand,
  RegisterCommand,
  TrialBalance,
  Workspace,
} from "../src/bookkeeping/types";
import type { components } from "../src/lib/api.generated";

type Setup = components["schemas"]["SetupResult"];
const steps = [
  "Check pending work and USD metadata",
  "Prepare Flow Lab book",
  "Execute setup",
  "Acknowledge setup",
  "Prepare deposit 0.10",
  "Execute deposit 0.10",
  "Acknowledge deposit",
  "Prepare deposit 0.20",
  "Execute deposit 0.20",
  "Acknowledge deposit",
  "Prepare fee 0.01",
  "Execute fee",
  "Replay same execution",
  "Acknowledge fee",
  "Prepare unbalanced debit 0.02 / credit 0.01",
  "Attempt unbalanced posting",
  "Cancel rejected entry",
  "Acknowledge cancellation",
  "Verify trial balance",
];

/** Demo-only observer of real API outcomes; no invented internal database tracing. */
export function FlowViewer({
  onChanged,
  onActiveChange,
}: {
  onChanged: () => void;
  onActiveChange?: (active: boolean) => void;
}) {
  const [index, setIndex] = useState(0);
  const [running, setRunning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [timeline, setTimeline] = useState<string[]>([]);
  const [report, setReport] = useState<TrialBalance | null>(null);
  const alive = useRef(true);
  const lock = useRef(false);
  const command = useRef<DurableCommand | null>(null);
  const book = useRef<Setup | null>(null);
  const currency = useRef("");
  const revision = useRef("");
  const journalID = useRef("");
  const api = useRef(
    createAPI(
      async () => "synthetic.browser.proof",
      () => {},
    ),
  ).current;
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  // Guard asynchronous observations after a viewer unmounts.
  const record = useCallback((message: string) => {
    if (alive.current) setTimeline((old) => [...old, message]);
  }, []);
  // One explicit step advances a durable boundary; uncertain results stop for recovery.
  const next = useCallback(async () => {
    if (lock.current || failure || index >= steps.length) return;
    lock.current = true;
    setBusy(true);
    onActiveChange?.(true);
    record(`Request sent: ${steps[index]}`);
    try {
      // Registration keys are created only once; failures stop instead of making a new request.
      const prepare = async (request: RegisterCommand) => {
        command.current = await api<DurableCommand>(
          "/v1/commands",
          JSON.stringify(request),
          crypto.randomUUID(),
        );
        if (command.current.state !== "prepared")
          throw new Error("Preparation was not confirmed.");
        record(
          `Prepared command ${command.current.command_id}. Execution has not been requested by this step.`,
        );
      };
      // Replay uses the existing command identifier and unchanged saved request.
      const transition = async (action: string) => {
        if (!command.current) throw new Error("Missing command.");
        command.current = await api<DurableCommand>(
          `/v1/commands/${command.current.command_id}/${action}`,
          "{}",
        );
        return command.current;
      };
      if (index === 0) {
        const pending = await api<PendingCommand>("/v1/commands/pending");
        if (pending.command)
          throw new Error(
            "Resolve the existing saved request in the bookkeeping panel first.",
          );
        const workspace = await api<Workspace>("/v1/workspace");
        const usd = currencyAt(workspace.currencies, "USD", "2026-01-15");
        if (!usd || usd.minor_unit !== 2)
          throw new Error("Reviewed USD metadata unavailable.");
        currency.current = usd.currency_version_id;
        record(
          "Ready: no pending command; USD metadata confirms two decimal places.",
        );
      } else if (index === 1) {
        await prepare({
          operation: "setup",
          entity_id: null,
          book_id: null,
          body: {
            entity_name: `Flow Lab ${crypto.randomUUID().slice(0, 8)}`,
            entity_kind: "legal",
            book_name: "Watched transactions",
            starts_on: "2026-01-01",
            ends_before: "2027-01-01",
            functional_currency_version_id: currency.current,
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
                monetary: true,
              },
              {
                code: "4400",
                name: "Bank fees",
                category: "expense",
                normal_side: "debit",
                monetary: false,
              },
            ],
          },
        });
      } else if (index === 2) {
        const result = await transition("execute");
        if (
          result.state !== "committed" ||
          !result.result ||
          !("accounts" in result.result)
        )
          throw new Error("Setup not confirmed.");
        book.current = result.result;
        revision.current = result.result.book_revision;
        record(
          `Commit confirmed: book ${result.result.book_id}; revision ${revision.current}.`,
        );
      } else if ([3, 6, 9, 13, 17].includes(index)) {
        const result = await transition("acknowledge");
        if (!result.acknowledged)
          throw new Error("Acknowledgement not confirmed.");
        record(
          `Acknowledged command ${result.command_id}; next request is now allowed.`,
        );
      } else if ([4, 7, 10, 14].includes(index)) {
        const setup = book.current;
        if (!setup) throw new Error("Missing book.");
        const amount =
          index === 4
            ? "0.10"
            : index === 7
              ? "0.20"
              : index === 10
                ? "0.01"
                : "0.02";
        // Exact fixed decimal strings travel unchanged; account references come from real SQL setup.
        const line = (
          code: string,
          side: "debit" | "credit",
          value: string,
        ) => {
          const account = setup.accounts.find((a) => a.code === code);
          if (!account) throw new Error("Missing synthetic account.");
          return {
            account_id: account.account_id,
            side,
            transaction_currency: "USD",
            transaction_amount: value,
            functional_amount: value,
            valuation_basis: "identity" as const,
          };
        };
        await prepare({
          operation: "post",
          entity_id: setup.entity_id,
          book_id: setup.book_id,
          body: {
            expected_revision: revision.current,
            policy_id: setup.policy_id,
            accounting_date: "2026-01-15",
            description: steps[index],
            lines: [
              line(index === 10 ? "4400" : "1000", "debit", amount),
              line(
                index === 10 ? "1000" : "2000",
                "credit",
                index === 14 ? "0.01" : amount,
              ),
            ],
          },
        });
      } else if ([5, 8, 11, 12].includes(index)) {
        const result = await transition("execute");
        if (
          result.state !== "committed" ||
          !result.result ||
          !("journal_id" in result.result)
        )
          throw new Error("Posting not confirmed.");
        const journal = result.result;
        if (
          index === 12 &&
          (journalID.current !== journal.journal_id ||
            revision.current !== journal.book_revision)
        )
          throw new Error("Replay changed its result.");
        journalID.current = journal.journal_id;
        revision.current = journal.book_revision;
        record(
          `${index === 12 ? "Replay verified: same journal and revision" : "Commit confirmed"}: journal ${journal.journal_id}; revision ${journal.book_revision}.`,
        );
      } else if (index === 15) {
        let rejected = false;
        try {
          await transition("execute");
        } catch (error) {
          if (!(error instanceof APIError) || error.status !== 422) throw error;
          rejected = true;
        }
        if (!rejected)
          throw new Error("Unbalanced entry unexpectedly accepted.");
        const pending = await api<PendingCommand>("/v1/commands/pending");
        if (
          !pending.command ||
          pending.command.command_id !== command.current?.command_id ||
          pending.command.state !== "prepared"
        )
          throw new Error("Rejected state not confirmed.");
        record(
          "Expected rejection confirmed: HTTP 422; same command remains prepared for explicit cancellation.",
        );
      } else if (index === 16) {
        const result = await transition("cancel");
        if (result.state !== "cancelled")
          throw new Error("Cancellation not confirmed.");
        record(`Cancellation confirmed: command ${result.command_id}.`);
      } else if (index === 18) {
        const setup = book.current;
        if (!setup) throw new Error("Missing book.");
        const balance = await api<TrialBalance>(
          `/v1/entities/${setup.entity_id}/books/${setup.book_id}/trial-balance?through_date=2026-01-31`,
        );
        const expected = [
          ["1000", "0.29", "0.00"],
          ["2000", "0.00", "0.30"],
          ["4400", "0.01", "0.00"],
        ];
        if (
          balance.functional_currency !== "USD" ||
          formatAmount(balance.total_debits, 2) !== "0.30" ||
          formatAmount(balance.total_credits, 2) !== "0.30" ||
          balance.snapshot_revision !== revision.current ||
          balance.accounts.length !== 3 ||
          expected.some(([code, debit, credit]) => {
            const row = balance.accounts.find((a) => a.account_code === code);
            return (
              !row ||
              formatAmount(row.debit_balance, 2) !== debit ||
              formatAmount(row.credit_balance, 2) !== credit
            );
          })
        )
          throw new Error(
            "Trial balance differs from the independently fixed expected cents.",
          );
        if (alive.current) setReport(balance);
        record(
          "Cents verified: bank USD 0.29; capital credit USD 0.30; fees debit USD 0.01; total debits and credits USD 0.30.",
        );
      }
      if (alive.current) {
        setIndex((old) => old + 1);
        onChanged();
        if (index === steps.length - 1) onActiveChange?.(false);
      }
    } catch (error) {
      if (alive.current) {
        setRunning(false);
        setFailure(
          error instanceof Error ? error.message : "The suite stopped.",
        );
        onChanged();
        onActiveChange?.(false);
      }
    } finally {
      lock.current = false;
      if (alive.current) setBusy(false);
    }
  }, [api, failure, index, onActiveChange, onChanged, record]);
  // Pause cancels future steps, never pretends to cancel an already-sent transaction.
  useEffect(() => {
    if (!running || busy || failure || index >= steps.length) return;
    const timer = window.setTimeout(() => {
      void next();
    }, 1100);
    return () => window.clearTimeout(timer);
  }, [running, busy, failure, index, next]);
  return (
    <section aria-label="Watch transaction flows">
      <h2>Watch transaction flows</h2>
      <p>
        Fresh Flow Lab book → deposit USD 0.10 → deposit USD 0.20 → fee USD 0.01
        → replay the same posting → reject an unbalanced entry → verify bank USD
        0.29.
      </p>
      <p>
        Prepare saves the request. Execute posts atomically. Acknowledge
        releases the next request. This timeline shows observed HTTP boundaries,
        not internal SQL tracing.
      </p>
      <p>
        Run continuously authorizes this fixed synthetic suite, including
        acknowledgements. Pause stops future steps; an in-flight request can
        still commit.
      </p>
      <p role="status">
        {index === steps.length
          ? "Suite complete — exact cents verified."
          : `Step ${index + 1} of ${steps.length}: ${steps[index]}`}
      </p>
      <button
        onClick={() => void next()}
        disabled={busy || running || !!failure || index === steps.length}
      >
        Next step
      </button>{" "}
      <button
        onClick={() => setRunning(true)}
        disabled={busy || running || !!failure || index === steps.length}
      >
        Run continuously
      </button>{" "}
      <button onClick={() => setRunning(false)} disabled={!running}>
        Pause
      </button>
      {failure && (
        <p role="alert">
          {failure} Stop here and recover the saved request in the bookkeeping
          panel. Do not start another suite until it is resolved. Refresh after
          recovery to start a fresh suite.
        </p>
      )}
      <ol aria-label="Transaction flow timeline">
        {timeline.map((entry, i) => (
          <li key={i}>{entry}</li>
        ))}
      </ol>
      {report && (
        <p data-testid="flow-result">
          Verified bank balance: <strong>USD 0.29</strong>. Total debits: USD{" "}
          {formatAmount(report.total_debits, 2)}. Total credits: USD{" "}
          {formatAmount(report.total_credits, 2)}. Book revision:{" "}
          {report.snapshot_revision}.
        </p>
      )}
    </section>
  );
}
