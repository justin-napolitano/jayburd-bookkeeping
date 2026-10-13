"use client";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { createAPI } from "./api";
import type { Book, Entity, Workspace, RegisterCommand } from "./types";
import { useCommands } from "./commands";
import { CommandPanel } from "./CommandPanel";
import { BookWorkspace } from "./BookWorkspace";
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
  const recovery = useCommands(api);
  const locked = recovery.locked;
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
  useEffect(() => {
    if (recovery.command?.state === "committed") void load();
  }, [recovery.command?.command_id, recovery.command?.state, load]);
  useEffect(() => {
    if (recovery.accessDenied) {
      setWorkspace(null);
      setScope("");
    }
  }, [recovery.accessDenied]);
  const selected = (recovery.accessDenied ? null : workspace)?.entities
    .flatMap((entity) => entity.books.map((book) => ({ entity, book })))
    .find((x) => `${x.entity.entity_id}/${x.book.book_id}` === scope);
  return (
    <>
      <section className="workspace-picker" aria-labelledby="workspace-title">
        <div className="heading">
          <h2 id="workspace-title">Your books</h2>
          <button disabled={loading || locked} onClick={() => void load()}>
            Refresh workspace
          </button>
        </div>
        {loading && <p role="status">Loading authorized books…</p>}
        {error && <p role="alert">{error}</p>}
        {workspace && !recovery.accessDenied && (
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
      <CommandPanel recovery={recovery} workspace={workspace} />
      {selected && (
        <BookWorkspace
          key={`${scope}/${recovery.command?.command_id ?? "none"}/${recovery.command?.state ?? "none"}`}
          {...selected}
          currencies={workspace?.currencies ?? []}
          api={api}
          prepare={recovery.prepare}
          locked={locked}
          denied={() => {
            setScope("");
            void load();
          }}
        >
          <ManualJournal
            {...selected}
            locked={locked}
            prepare={recovery.prepare}
          />
        </BookWorkspace>
      )}
      {workspace && !recovery.accessDenied && (
        <Setup
          workspace={workspace}
          locked={locked}
          prepare={recovery.prepare}
        />
      )}
    </>
  );
}
// Setup creates a new scope only; balances and authorization are never user-supplied.
function Setup({
  workspace,
  locked,
  prepare,
}: {
  workspace: Workspace;
  locked: boolean;
  prepare: (value: RegisterCommand) => void;
}) {
  const [accounts, setAccounts] = useState<AccountDraft[]>([
    blankAccount(),
    {
      ...blankAccount(),
      category: "equity",
      normal_side: "credit",
      monetary: false,
    },
  ]);
  const review = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    prepare({
      operation: "setup",
      entity_id: null,
      book_id: null,
      body: {
        entity_name: f.get("entity_name"),
        entity_kind: f.get("entity_kind"),
        book_name: f.get("book_name"),
        starts_on: f.get("starts_on"),
        ends_before: f.get("ends_before"),
        functional_currency_version_id: f.get("currency"),
        accounts,
      } as Extract<RegisterCommand, { operation: "setup" }>["body"],
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
          <fieldset disabled={locked}>
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
      </details>
    </section>
  );
}
// The selected scope owns manual draft strings; durable recovery owns review and execution.
function ManualJournal({
  entity,
  book,
  locked,
  prepare,
}: {
  entity: Entity;
  book: Book;
  locked: boolean;
  prepare: (value: RegisterCommand) => void;
}) {
  const [lines, setLines] = useState<LineDraft[]>([
    blankLine(),
    { ...blankLine(), side: "credit" },
  ]);
  const review = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    prepare({
      operation: "post",
      entity_id: entity.entity_id,
      book_id: book.book_id,
      body: {
        expected_revision: book.book_revision,
        policy_id: f.get("policy"),
        accounting_date: f.get("date"),
        description: f.get("description"),
        lines: lines.map(({ rate_observation_id, ...line }) =>
          line.valuation_basis === "spot"
            ? { ...line, rate_observation_id }
            : line,
        ),
      } as Extract<RegisterCommand, { operation: "post" }>["body"],
    });
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
              <fieldset disabled={locked}>
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
          </>
        )}
      </section>
    </>
  );
}
