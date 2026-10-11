"use client";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { APIError, createAPI, type API } from "./api";
import type { Book, Entity, Journal, TrialBalance, Workspace } from "./types";
// UI account drafts carry strings unchanged to the authoritative setup transaction.
type AccountDraft = {
  code: string;
  name: string;
  category: string;
  normal_side: string;
  monetary: boolean;
};
type LineDraft = {
  account_id: string;
  side: string;
  transaction_currency: string;
  transaction_amount: string;
  functional_amount: string;
  valuation_basis: string;
  rate_observation_id: string;
};
type Pending = { key: string; body: string };
const today = () => new Date().toISOString().slice(0, 10);
const blankAccount = (): AccountDraft => ({
  code: "",
  name: "",
  category: "asset",
  normal_side: "debit",
  monetary: true,
});
const blankLine = (): LineDraft => ({
  account_id: "",
  side: "debit",
  transaction_currency: "USD",
  transaction_amount: "",
  functional_amount: "",
  valuation_basis: "identity",
  rate_observation_id: "",
});
// Workspace lifecycle discards stale responses and unmounts scope-owned financial state.
export function Bookkeeping({
  getToken,
  rejectSession,
}: {
  getToken: () => Promise<string>;
  rejectSession: () => void;
}) {
  const api = useMemo(
    () => createAPI(getToken, rejectSession),
    [getToken, rejectSession],
  );
  const [workspace, setWorkspace] = useState<Workspace | null>(null),
    [scope, setScope] = useState(""),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(false);
  const [locked, setLocked] = useState(false);
  const epoch = useRef(0);
  const load = useCallback(async () => {
    const n = ++epoch.current;
    setLoading(true);
    setError("");
    try {
      const w = await api<Workspace>("/v1/workspace");
      if (n === epoch.current) {
        setWorkspace(w);
        setScope((s) =>
          w.entities.some((e) =>
            e.books.some((b) => `${e.entity_id}/${b.book_id}` === s),
          )
            ? s
            : "",
        );
      }
    } catch (e) {
      if (n === epoch.current) {
        setWorkspace(null);
        setScope("");
        setError(e instanceof Error ? e.message : "Workspace unavailable.");
      }
    } finally {
      if (n === epoch.current) setLoading(false);
    }
  }, [api]);
  useEffect(() => {
    void load();
    return () => {
      epoch.current++;
    };
  }, [load]);
  const selected = workspace?.entities
    .flatMap((entity) => entity.books.map((book) => ({ entity, book })))
    .find((x) => `${x.entity.entity_id}/${x.book.book_id}` === scope);
  return (
    <>
      <section aria-labelledby="workspace-title">
        <div className="heading">
          <h2 id="workspace-title">Workspace</h2>
          <button disabled={loading || locked} onClick={() => void load()}>
            Refresh workspace
          </button>
        </div>
        {loading && <p role="status">Loading authorized books…</p>}
        {error && <p role="alert">{error}</p>}
        {workspace && (
          <>
            <label>
              Entity and book
              <select
                disabled={locked}
                value={scope}
                onChange={(e) => setScope(e.target.value)}
              >
                <option value="">Choose a book</option>
                {workspace.entities.map((e) => (
                  <optgroup label={e.name} key={e.entity_id}>
                    {e.books.map((b) => (
                      <option
                        key={b.book_id}
                        value={`${e.entity_id}/${b.book_id}`}
                      >
                        {e.name} / {b.name}
                      </option>
                    ))}
                  </optgroup>
                ))}
              </select>
            </label>
            {workspace.entities.length === 0 && (
              <p>
                No books are available. Create your first development book
                below.
              </p>
            )}
          </>
        )}
      </section>
      {selected && (
        <BookView
          key={scope}
          {...selected}
          api={api}
          refresh={load}
          locked={locked}
          lock={setLocked}
          denied={() => {
            setScope("");
            void load();
          }}
        />
      )}
      {workspace && (
        <Setup
          api={api}
          workspace={workspace}
          refresh={load}
          locked={locked}
          lock={setLocked}
        />
      )}
    </>
  );
}
// Snapshot/body/key are frozen at review; uncertain outcomes retain the identical request.
function useMutation(
  api: API,
  path: string,
  committed: (result: Journal | Record<string, unknown>) => void,
  lock: (value: boolean) => void,
) {
  const [pending, setPending] = useState<Pending | null>(null),
    [busy, setBusy] = useState(false),
    [uncertain, setUncertain] = useState(false),
    [error, setError] = useState("");
  const active = useRef(true);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  useEffect(() => {
    lock(!!pending);
    const prevent = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    if (pending) window.addEventListener("beforeunload", prevent);
    return () => {
      window.removeEventListener("beforeunload", prevent);
    };
  }, [pending, lock]);
  const review = (body: unknown) => {
    setPending({ key: crypto.randomUUID(), body: JSON.stringify(body) });
    setError("");
    setUncertain(false);
  };
  const send = async () => {
    if (!pending || busy) return;
    setBusy(true);
    setError("");
    try {
      const result = await api<Journal>(path, pending.body, pending.key);
      if (active.current) {
        setPending(null);
        setUncertain(false);
        committed(result);
      }
    } catch (e) {
      if (active.current) {
        const unknown = uncertain || !(e instanceof APIError) || e.uncertain;
        setUncertain(unknown);
        setError(
          uncertain
            ? "The previous request may have committed. Keep its request key and verify the result with an authorized operator before entering it again."
            : e instanceof Error
              ? e.message
              : "Request failed.",
        );
        if (!unknown) setPending(null);
      }
    } finally {
      if (active.current) setBusy(false);
    }
  };
  return {
    pending,
    busy,
    uncertain,
    error,
    review,
    send,
    cancel: () => {
      if (!uncertain && !busy) setPending(null);
    },
  };
}
// Setup creates a new scope only; balances and authorization are never user-supplied.
function Setup({
  api,
  workspace,
  refresh,
  locked,
  lock,
}: {
  api: API;
  workspace: Workspace;
  refresh: () => Promise<void>;
  locked: boolean;
  lock: (value: boolean) => void;
}) {
  const [accounts, setAccounts] = useState<AccountDraft[]>([
      blankAccount(),
      {
        ...blankAccount(),
        category: "equity",
        normal_side: "credit",
        monetary: false,
      },
    ]),
    [success, setSuccess] = useState("");
  const m = useMutation(
    api,
    "/v1/setup",
    () => {
      setSuccess(
        "Book creation confirmed. Refreshing the workspace; choose the new book when it appears above.",
      );
      void refresh();
    },
    lock,
  );
  const review = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setSuccess("");
    const f = new FormData(e.currentTarget);
    m.review({
      entity_name: f.get("entity_name"),
      entity_kind: f.get("entity_kind"),
      book_name: f.get("book_name"),
      starts_on: f.get("starts_on"),
      ends_before: f.get("ends_before"),
      functional_currency_version_id: f.get("currency"),
      accounts,
    });
  };
  return (
    <section>
      <details>
        <summary>Create an entity and book</summary>
        <p>
          Development policy: accrual, UTC, USD functional currency, half-even
          rounding. Foreign transaction currencies remain supported. Choose
          dates and your chart of accounts deliberately.
        </p>
        <form onSubmit={review}>
          <fieldset disabled={locked || !!m.pending || m.busy}>
            <div className="grid">
              <label>
                Entity name
                <input name="entity_name" required maxLength={200} />
              </label>
              <label>
                Entity kind
                <select name="entity_kind">
                  <option value="legal">Legal entity / LLC</option>
                  <option value="personal">Personal</option>
                </select>
              </label>
              <label>
                Book name
                <input name="book_name" required maxLength={200} />
              </label>
              <label>
                Period starts on
                <input name="starts_on" type="date" required />
              </label>
              <label>
                Period ends before
                <input name="ends_before" type="date" required />
              </label>
              <label>
                Functional currency metadata
                <select name="currency" required>
                  <option value="">Choose reviewed USD metadata</option>
                  {workspace.currencies
                    .filter(
                      (c) => c.alphabetic_code === "USD" && c.minor_unit === 2,
                    )
                    .map((c) => (
                      <option
                        key={c.currency_version_id}
                        value={c.currency_version_id}
                      >
                        USD · {c.valid_from} to {c.valid_to ?? "open"}
                      </option>
                    ))}
                </select>
              </label>
            </div>
            <h3>Accounts</h3>
            {accounts.map((a, i) => (
              <div className="account grid" key={i}>
                {(["code", "name"] as const).map((k) => (
                  <label key={k}>
                    {`Account ${i + 1} ${k}`}
                    <input
                      required
                      maxLength={k === "code" ? 64 : 200}
                      value={a[k]}
                      onChange={(e) =>
                        setAccounts((v) =>
                          v.map((x, j) =>
                            j === i ? { ...x, [k]: e.target.value } : x,
                          ),
                        )
                      }
                    />
                  </label>
                ))}
                <label>
                  Category
                  <select
                    aria-label={`Account ${i + 1} category`}
                    value={a.category}
                    onChange={(e) =>
                      setAccounts((v) =>
                        v.map((x, j) =>
                          j === i ? { ...x, category: e.target.value } : x,
                        ),
                      )
                    }
                  >
                    {["asset", "liability", "equity", "revenue", "expense"].map(
                      (v) => (
                        <option key={v}>{v}</option>
                      ),
                    )}
                  </select>
                </label>
                <label>
                  Normal side
                  <select
                    aria-label={`Account ${i + 1} normal side`}
                    value={a.normal_side}
                    onChange={(e) =>
                      setAccounts((v) =>
                        v.map((x, j) =>
                          j === i ? { ...x, normal_side: e.target.value } : x,
                        ),
                      )
                    }
                  >
                    <option>debit</option>
                    <option>credit</option>
                  </select>
                </label>
                <label className="check">
                  <input
                    type="checkbox"
                    checked={a.monetary}
                    onChange={(e) =>
                      setAccounts((v) =>
                        v.map((x, j) =>
                          j === i ? { ...x, monetary: e.target.checked } : x,
                        ),
                      )
                    }
                  />
                  Monetary account {i + 1}
                </label>
                {accounts.length > 2 && (
                  <button
                    type="button"
                    onClick={() =>
                      setAccounts((v) => v.filter((_, j) => j !== i))
                    }
                  >
                    Remove account {i + 1}
                  </button>
                )}
              </div>
            ))}
            <button
              type="button"
              disabled={accounts.length >= 100}
              onClick={() => setAccounts((v) => [...v, blankAccount()])}
            >
              Add account
            </button>
            <button>Review book setup</button>
          </fieldset>
        </form>
        <MutationReview mutation={m} label="Create book" />
        {success && <p role="status">{success}</p>}
      </details>
    </section>
  );
}
// The selected scope owns report requests, draft state and immutable committed result.
function BookView({
  entity,
  book,
  api,
  refresh,
  denied,
  locked,
  lock,
}: {
  entity: Entity;
  book: Book;
  api: API;
  refresh: () => Promise<void>;
  denied: () => void;
  locked: boolean;
  lock: (value: boolean) => void;
}) {
  const [lines, setLines] = useState<LineDraft[]>([
      blankLine(),
      { ...blankLine(), side: "credit" },
    ]),
    [journal, setJournal] = useState<Journal | null>(null),
    [report, setReport] = useState<TrialBalance | null>(null),
    [reportError, setReportError] = useState(""),
    [reportBusy, setReportBusy] = useState(false);
  const reportEpoch = useRef(0);
  useEffect(
    () => () => {
      reportEpoch.current++;
    },
    [],
  );
  const base = `/v1/entities/${entity.entity_id}/books/${book.book_id}`;
  const m = useMutation(
    api,
    `${base}/journals`,
    (r) => {
      reportEpoch.current++;
      setReportBusy(false);
      setJournal(r as Journal);
      setReport(null);
      void refresh();
    },
    lock,
  );
  const review = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    reportEpoch.current++;
    setReportBusy(false);
    setJournal(null);
    const f = new FormData(e.currentTarget);
    m.review({
      expected_revision: book.book_revision,
      policy_id: f.get("policy"),
      accounting_date: f.get("date"),
      description: f.get("description"),
      lines: lines.map(({ rate_observation_id, ...line }) =>
        line.valuation_basis === "spot"
          ? { ...line, rate_observation_id }
          : line,
      ),
    });
  };
  const trial = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const date = new FormData(e.currentTarget).get("through");
    const n = ++reportEpoch.current;
    setReport(null);
    setReportError("");
    setReportBusy(true);
    try {
      const r = await api<TrialBalance>(
        `${base}/trial-balance?through_date=${date}`,
      );
      if (n === reportEpoch.current) setReport(r);
    } catch (e) {
      if (n === reportEpoch.current) {
        setReportError(e instanceof Error ? e.message : "Report unavailable.");
        if (e instanceof APIError && [403, 404].includes(e.status)) denied();
      }
    } finally {
      if (n === reportEpoch.current) setReportBusy(false);
    }
  };
  const update = (i: number, k: keyof LineDraft, value: string) =>
    setLines((v) => v.map((x, j) => (j === i ? { ...x, [k]: value } : x)));
  return (
    <>
      <section>
        <p className="eyebrow">
          {entity.name} / {book.name} · revision {book.book_revision}
        </p>
        <h2>Manual journal</h2>
        <p>
          Source: manual entry. Nothing is posted until you review and confirm.
          Enter decimal amounts exactly; the ledger validates valuation and
          balance.
        </p>
        {!entity.permissions.includes("post") ? (
          <p>
            You have read access. Posting permission is required for new
            entries.
          </p>
        ) : (
          <>
            <form onSubmit={review}>
              <fieldset disabled={locked || !!m.pending || m.busy}>
                <div className="grid">
                  <label>
                    Accounting date
                    <input
                      name="date"
                      type="date"
                      required
                      defaultValue={today()}
                    />
                  </label>
                  <label>
                    Policy
                    <select name="policy" required>
                      <option value="">Choose policy</option>
                      {book.policies.map((p) => (
                        <option key={p.policy_id} value={p.policy_id}>
                          {p.functional_currency} · {p.effective_from} to{" "}
                          {p.effective_to ?? "open"}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label>
                    Description
                    <input name="description" required maxLength={500} />
                  </label>
                </div>
                {lines.map((line, i) => (
                  <div className="journal-line" key={i}>
                    <h3>Line {i + 1}</h3>
                    <div className="grid">
                      <label>
                        Account
                        <select
                          aria-label={`Line ${i + 1} account`}
                          required
                          value={line.account_id}
                          onChange={(e) =>
                            update(i, "account_id", e.target.value)
                          }
                        >
                          <option value="">Choose account</option>
                          {book.accounts.map((a) => (
                            <option key={a.account_id} value={a.account_id}>
                              {a.code} · {a.name}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label>
                        Side
                        <select
                          aria-label={`Line ${i + 1} side`}
                          value={line.side}
                          onChange={(e) => update(i, "side", e.target.value)}
                        >
                          <option>debit</option>
                          <option>credit</option>
                        </select>
                      </label>
                      <label>
                        Original currency
                        <input
                          aria-label={`Line ${i + 1} original currency`}
                          required
                          pattern="[A-Z]{3}"
                          value={line.transaction_currency}
                          onChange={(e) =>
                            update(i, "transaction_currency", e.target.value)
                          }
                        />
                      </label>
                      <label>
                        Original amount
                        <input
                          aria-label={`Line ${i + 1} original amount`}
                          required
                          inputMode="decimal"
                          pattern="[0-9]+([.][0-9]+)?"
                          value={line.transaction_amount}
                          onChange={(e) =>
                            update(i, "transaction_amount", e.target.value)
                          }
                        />
                      </label>
                      <label>
                        Functional amount
                        <input
                          aria-label={`Line ${i + 1} functional amount`}
                          required
                          inputMode="decimal"
                          pattern="[0-9]+([.][0-9]+)?"
                          value={line.functional_amount}
                          onChange={(e) =>
                            update(i, "functional_amount", e.target.value)
                          }
                        />
                      </label>
                      <label>
                        Valuation
                        <select
                          aria-label={`Line ${i + 1} valuation`}
                          value={line.valuation_basis}
                          onChange={(e) =>
                            update(i, "valuation_basis", e.target.value)
                          }
                        >
                          <option value="identity">Same currency</option>
                          <option value="spot">Reviewed spot rate</option>
                        </select>
                      </label>
                      {line.valuation_basis === "spot" && (
                        <label>
                          Rate observation ID
                          <input
                            aria-label={`Line ${i + 1} rate observation ID`}
                            required
                            value={line.rate_observation_id}
                            onChange={(e) =>
                              update(i, "rate_observation_id", e.target.value)
                            }
                          />
                        </label>
                      )}
                    </div>
                    {lines.length > 2 && (
                      <button
                        type="button"
                        onClick={() =>
                          setLines((v) => v.filter((_, j) => j !== i))
                        }
                      >
                        Remove line {i + 1}
                      </button>
                    )}
                  </div>
                ))}
                <button
                  type="button"
                  disabled={lines.length >= 1000}
                  onClick={() => setLines((v) => [...v, blankLine()])}
                >
                  Add split line
                </button>
                <button>Review journal</button>
              </fieldset>
            </form>
            <MutationReview mutation={m} label="Post journal" book={book} />
          </>
        )}
        {journal && (
          <div role="status">
            <h3>Posting confirmed</h3>
            <p>
              Journal {journal.journal_id} · revision {journal.book_revision}
            </p>
            <p>
              {journal.accounting_date} · {journal.description}
            </p>
            <p>
              This committed journal is immutable. Corrections require a linked
              reversal through the API; reversal UI is a later slice.
            </p>
          </div>
        )}
      </section>
      <section>
        <h2>Trial balance</h2>
        <form onSubmit={trial}>
          <label>
            Through date
            <input name="through" type="date" required defaultValue={today()} />
          </label>
          <button disabled={reportBusy || locked}>Load trial balance</button>
        </form>
        {reportBusy && <p role="status">Reading ledger snapshot…</p>}
        {reportError && <p role="alert">{reportError}</p>}
        {report && (
          <>
            <p>
              Through {report.through_date} · {report.functional_currency} ·
              snapshot {report.snapshot_revision}
            </p>
            <div className="table-scroll">
              <table>
                <caption>Account balances from posted journals</caption>
                <thead>
                  <tr>
                    <th>Account</th>
                    <th>Debit</th>
                    <th>Credit</th>
                  </tr>
                </thead>
                <tbody>
                  {report.accounts.map((a) => (
                    <tr key={a.account_id}>
                      <th scope="row">{a.account_code}</th>
                      <td>{a.debit_balance}</td>
                      <td>{a.credit_balance}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <th>Total</th>
                    <td>{report.total_debits}</td>
                    <td>{report.total_credits}</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </>
        )}
      </section>
    </>
  );
}
// Review presents exact accounting facts, with the serialized request available for recovery.
function MutationReview({
  mutation: m,
  label,
  book,
}: {
  mutation: ReturnType<typeof useMutation>;
  label: string;
  book?: Book;
}) {
  const draft = m.pending ? JSON.parse(m.pending.body) : null;
  return (
    <>
      {m.error && <p role="alert">{m.error}</p>}
      {m.pending && (
        <div className="review">
          <h3>
            {m.uncertain
              ? "Unconfirmed outcome — keep this page open"
              : "Review before committing"}
          </h3>
          <p>
            {m.uncertain
              ? "Retry uses the original key and exact body. Editing is locked until the outcome is resolved."
              : "Review the exact values below. The database applies accounting rules when you confirm."}
          </p>
          <p>
            Request key: <code>{m.pending.key}</code>
          </p>
          {draft.lines ? (
            <>
              <p>
                <strong>{draft.description}</strong> · Accounting date{" "}
                {draft.accounting_date}
              </p>
              <div className="table-scroll">
                <table>
                  <caption>Journal lines to post</caption>
                  <thead>
                    <tr>
                      <th>Account</th>
                      <th>Side</th>
                      <th>Original amount</th>
                      <th>Functional amount</th>
                      <th>Valuation</th>
                    </tr>
                  </thead>
                  <tbody>
                    {draft.lines.map((line: LineDraft, i: number) => (
                      <tr key={i}>
                        <th scope="row">
                          {book?.accounts.find(
                            (a) => a.account_id === line.account_id,
                          )?.name ?? line.account_id}
                        </th>
                        <td>{line.side}</td>
                        <td>
                          {line.transaction_amount} {line.transaction_currency}
                        </td>
                        <td>
                          {line.functional_amount}{" "}
                          {
                            book?.policies.find(
                              (p) => p.policy_id === draft.policy_id,
                            )?.functional_currency
                          }
                        </td>
                        <td>
                          {line.valuation_basis}
                          {line.rate_observation_id && (
                            <small> · {line.rate_observation_id}</small>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          ) : (
            <>
              <p>
                <strong>{draft.entity_name}</strong> ({draft.entity_kind}) /{" "}
                {draft.book_name}
              </p>
              <p>
                Open period: {draft.starts_on} through the day before{" "}
                {draft.ends_before}. Functional currency: USD.
              </p>
              <ul>
                {draft.accounts.map((a: AccountDraft, i: number) => (
                  <li key={i}>
                    {a.code} · {a.name} · {a.category} · normal {a.normal_side}{" "}
                    · {a.monetary ? "monetary" : "non-monetary"}
                  </li>
                ))}
              </ul>
            </>
          )}
          <details>
            <summary>Exact request for review or recovery</summary>
            <pre>{JSON.stringify(draft, null, 2)}</pre>
          </details>
          <button disabled={m.busy} onClick={() => void m.send()}>
            {m.busy
              ? "Waiting for commit…"
              : m.uncertain
                ? "Retry unchanged request"
                : label}
          </button>
          {!m.uncertain && (
            <button disabled={m.busy} onClick={m.cancel}>
              Edit before posting
            </button>
          )}
        </div>
      )}
    </>
  );
}
