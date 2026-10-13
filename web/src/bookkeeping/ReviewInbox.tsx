"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { APIError, type API } from "./api";
import { formatAmount } from "./amount";
import type { Book, Entity, Currency, Journal } from "./types";
import type { components } from "../lib/api.generated";

// Reviewed HTTP projections come only from the coordinator-owned schema generator.
export type ReviewSource = components["schemas"]["ReviewSource"];
export type ReviewRow = components["schemas"]["ReviewRow"];
export type ReviewPage = components["schemas"]["ReviewPage"];
export type ReviewDraft = components["schemas"]["ReviewDraft"];
export type ReviewDetail = components["schemas"]["ReviewItem"];
export type ReviewCategories = components["schemas"]["ReviewCategories"];
export type SourcePostCommand = Extract<
  components["schemas"]["RegisterCommand"],
  { operation: "source_post" }
>;
// Recovery owns registration/execution; this component only hands over explicitly reviewed intent.
export type ReviewInboxProps = {
  api: API;
  entity: Entity;
  book: Book;
  currencies: Currency[];
  locked: boolean;
  active?: boolean;
  prepare: (command: SourcePostCommand) => void;
  denied: () => void;
};
// Scope and lock changes destroy sensitive snapshots and pending presentation state.
export function ReviewInbox(props: ReviewInboxProps) {
  return (
    <Inbox
      key={`${props.entity.entity_id}/${props.book.book_id}/${props.book.book_revision}/${props.locked}/${props.active ?? true}`}
      {...props}
    />
  );
}
// A single request generation suppresses late source/page/detail responses.
function Inbox({
  api,
  entity,
  book,
  currencies,
  locked,
  active = true,
  prepare,
  denied,
}: ReviewInboxProps) {
  const base = `/v1/entities/${encodeURIComponent(entity.entity_id)}/books/${encodeURIComponent(book.book_id)}`;
  const [sources, setSources] = useState<ReviewSource[] | null>(null);
  const [source, setSource] = useState("");
  const [sourceAttempt, setSourceAttempt] = useState(0);
  const [filter, setFilter] = useState("needs_review");
  const [page, setPage] = useState<ReviewPage | null>(null);
  const [detail, setDetail] = useState<ReviewDetail | null>(null);
  const [categories, setCategories] = useState<ReviewCategories | null>(null);
  const [journal, setJournal] = useState<Journal | null>(null);
  const [draft, setDraft] = useState<ReviewDraft | null>(null);
  const [category, setCategory] = useState("");
  const [date, setDate] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [retry, setRetry] = useState<{ body: string; key: string } | null>(
    null,
  );
  const epoch = useRef(0);
  const alive = useRef(true);
  const flight = useRef(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const available = active && !locked;
  // Errors never surface raw API/provider diagnostics or substitute zero balances.
  const failure = (e: unknown) => {
    setError(
      e instanceof APIError
        ? e.message
        : "Review data is unavailable. Retry the same request or refresh.",
    );
    if (e instanceof APIError && [401, 403, 404].includes(e.status)) {
      setSources(null);
      setPage(null);
      setDetail(null);
      setJournal(null);
      setDraft(null);
      setCategories(null);
      denied();
    }
  };
  useEffect(() => {
    alive.current = true;
    setSources(null);
    setPage(null);
    setDetail(null);
    setJournal(null);
    setDraft(null);
    setCategories(null);
    setError("");
    if (!available)
      return () => {
        alive.current = false;
        epoch.current++;
      };
    const version = ++epoch.current;
    void api<{ items: ReviewSource[] }>(`${base}/sources`)
      .then((value) => {
        if (alive.current && version === epoch.current) setSources(value.items);
      })
      .catch((e) => {
        if (alive.current && version === epoch.current) failure(e);
      });
    return () => {
      alive.current = false;
      epoch.current++;
    };
    // Scope is remounted by the wrapper; denial callbacks are not request dependencies.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, base, available, sourceAttempt]);
  const resetDetail = () => {
    setDetail(null);
    setJournal(null);
    setCategories(null);
    setDraft(null);
    setRetry(null);
    setCategory("");
  };
  // Opaque cursor continuation keeps SQL's source snapshot; refresh starts a new snapshot.
  const load = async (selected = source, state = filter, cursor?: string) => {
    const version = ++epoch.current;
    setPage(null);
    resetDetail();
    setError("");
    if (!selected || !available) {
      setBusy(false);
      return;
    }
    setBusy(true);
    const query = new URLSearchParams({
      source_account_id: selected,
      limit: "50",
    });
    if (state) query.set("state", state);
    if (cursor) query.set("cursor", cursor);
    try {
      const value = await api<ReviewPage>(`${base}/review-items?${query}`);
      if (alive.current && version === epoch.current) {
        if (value.source_account_id !== selected)
          throw new Error("Mismatched source");
        setPage(value);
      }
    } catch (e) {
      if (alive.current && version === epoch.current) failure(e);
    } finally {
      if (alive.current && version === epoch.current) setBusy(false);
    }
  };
  // Fetch current item and category authority together; no categories are inferred from account names.
  const select = async (id: string) => {
    const version = ++epoch.current;
    resetDetail();
    setBusy(true);
    setError("");
    try {
      const value = await api<ReviewDetail>(
        `${base}/review-items/${encodeURIComponent(id)}`,
      );
      if (!alive.current || version !== epoch.current) return;
      if (value.source_account_id !== source || value.review_id !== id)
        throw new Error("Mismatched review");
      setDetail(value);
      setDraft(value.draft);
      setDate(value.booked_date);
      setDescription(value.description);
      if (
        (value.state === "needs_review" && value.blockers.length === 0) ||
        (value.state === "draft_ready" &&
          value.blockers.length > 0 &&
          value.blockers.every((blocker) => blocker === "draft_stale"))
      ) {
        const choices = await api<ReviewCategories>(
          `${base}/review-items/${encodeURIComponent(id)}/categories`,
        );
        if (!alive.current || version !== epoch.current) return;
        if (choices.source_account_id !== source || choices.review_id !== id)
          throw new Error("Mismatched categories");
        setCategories(choices);
        setDate(choices.accounting_date);
      }
      heading.current?.focus();
    } catch (e) {
      if (alive.current && version === epoch.current) {
        resetDetail();
        failure(e);
      }
    } finally {
      if (alive.current && version === epoch.current) setBusy(false);
    }
  };
  // Read the actual immutable journal under current scope authority; never synthesize it from a draft.
  const readJournal = async () => {
    if (!detail?.journal_id || !available || busy) return;
    const id = detail.journal_id;
    const version = ++epoch.current;
    setJournal(null);
    setBusy(true);
    setError("");
    try {
      const value = await api<Journal>(
        `${base}/journals/${encodeURIComponent(id)}`,
      );
      if (!alive.current || version !== epoch.current) return;
      if (
        value.entity_id !== entity.entity_id ||
        value.book_id !== book.book_id ||
        value.journal_id !== id
      )
        throw new Error("Mismatched journal");
      setJournal(value);
    } catch (e) {
      if (alive.current && version === epoch.current) failure(e);
    } finally {
      if (alive.current && version === epoch.current) setBusy(false);
    }
  };
  // Retry preserves the exact request/key after uncertainty; creating a draft never posts.
  const create = async (event: FormEvent) => {
    event.preventDefault();
    if (
      !detail ||
      !categories ||
      categories.fx_required ||
      !category ||
      !available ||
      flight.current
    )
      return;
    const request = retry ?? {
      key: crypto.randomUUID(),
      body: JSON.stringify({
        expected_review_revision: detail.review_revision,
        source_revision: detail.source_revision,
        binding_revision: detail.binding_revision,
        expected_book_revision: categories.book_revision,
        category_account_id: category,
        policy_id: categories.policy_id,
        accounting_date: date,
        description,
      }),
    };
    flight.current = true;
    setRetry(request);
    setBusy(true);
    setError("");
    const version = ++epoch.current;
    try {
      const value = await api<ReviewDraft>(
        `${base}/review-items/${encodeURIComponent(detail.review_id)}/drafts`,
        request.body,
        request.key,
      );
      if (!alive.current || version !== epoch.current) return;
      if (value.review_id !== detail.review_id)
        throw new Error("Mismatched draft");
      setDraft(value);
      setDetail({
        ...detail,
        state: "draft_ready",
        blockers: [],
        draft: value,
        draft_id: value.draft_id,
        review_revision: value.review_revision,
        source_revision: value.source_revision,
        binding_revision: value.binding_revision,
        book_revision: value.book_revision,
      });
      setRetry(null);
      setPage(null);
      setCategories(null);
    } catch (e) {
      if (alive.current && version === epoch.current) {
        failure(e);
        if (e instanceof APIError && !e.uncertain) setRetry(null);
      }
    } finally {
      if (alive.current && version === epoch.current) {
        flight.current = false;
        setBusy(false);
      }
    }
  };
  const amount = (row: ReviewRow) =>
    `${formatAmount(row.amount, currencies.find((c) => c.currency_version_id === row.currency_version_id)?.minor_unit)} ${row.currency}`;
  if (!active) return null;
  return (
    <section aria-labelledby="review-inbox-title">
      <h2 id="review-inbox-title">Review transactions</h2>
      <p>
        Imported transactions are unposted until you review a draft and
        explicitly confirm its command. Posted balances exclude unposted items.
      </p>
      {locked && (
        <p role="status">
          Resolve the pending command before reviewing more transactions.
        </p>
      )}
      <div className="workspace-actions">
        <label>
          Source account
          <select
            value={source}
            disabled={!available || busy || !!retry}
            onChange={(e) => {
              setSource(e.target.value);
              void load(e.target.value);
            }}
          >
            <option value="">Select a source account</option>
            {sources?.map((s) => (
              <option key={s.source_account_id} value={s.source_account_id}>
                {s.display_name} · {s.currency} · {s.state}
              </option>
            ))}
          </select>
        </label>
        <label>
          Review status
          <select
            value={filter}
            disabled={!available || busy || !!retry}
            onChange={(e) => {
              setFilter(e.target.value);
              void load(source, e.target.value);
            }}
          >
            <option value="needs_review">Needs review</option>
            <option value="draft_ready">Draft ready</option>
            <option value="posted">Posted</option>
            <option value="overlap_hold">Overlap hold</option>
            <option value="">All statuses</option>
          </select>
        </label>
        <button
          type="button"
          disabled={!source || !available || busy || !!retry}
          onClick={() => void load()}
        >
          Refresh transactions
        </button>
      </div>
      {!sources && error && available && (
        <button
          type="button"
          onClick={() => setSourceAttempt((attempt) => attempt + 1)}
        >
          Retry source accounts
        </button>
      )}
      {sources?.length === 0 && (
        <p>No source accounts are available in this book.</p>
      )}
      {error && <p role="alert">{error}</p>}
      {busy && <p role="status">Loading review data…</p>}
      {page && (
        <>
          <p className="report-stamp">
            Source revision {page.source_revision}. This is a review list, not a
            posted-balance report.
          </p>
          <div className="table-scroll">
            <table>
              <caption>
                Source transactions · {filter || "all statuses"}
              </caption>
              <thead>
                <tr>
                  <th scope="col">Date</th>
                  <th scope="col">Description</th>
                  <th scope="col">Direction</th>
                  <th scope="col">Amount</th>
                  <th scope="col">Status</th>
                  <th scope="col">Action</th>
                </tr>
              </thead>
              <tbody>
                {page.items.map((row) => (
                  <tr key={row.review_id}>
                    <td>{row.booked_date}</td>
                    <td>{row.description}</td>
                    <td>{row.direction}</td>
                    <td>{amount(row)}</td>
                    <td>{row.state.replaceAll("_", " ")}</td>
                    <td>
                      <button
                        type="button"
                        disabled={busy || !!retry || !available}
                        onClick={() => void select(row.review_id)}
                      >
                        Review {row.description}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {page.items.length === 0 && <p>No transactions match this status.</p>}
          <button
            type="button"
            disabled={!page.next_cursor || busy || !available || !!retry}
            onClick={() =>
              void load(source, filter, page.next_cursor ?? undefined)
            }
          >
            Next transactions
          </button>
        </>
      )}
      {detail && (
        <section aria-labelledby="review-detail-title">
          <h3 id="review-detail-title" ref={heading} tabIndex={-1}>
            {detail.description}
          </h3>
          <p>
            {detail.booked_date} · {detail.direction} · {amount(detail)} ·{" "}
            {detail.state.replaceAll("_", " ")}
          </p>
          {detail.blockers.length > 0 && (
            <p role="alert">Review blocked: {detail.blockers.join(", ")}</p>
          )}
          {detail.journal_id && (
            <>
              <p>Posted journal: {detail.journal_id}</p>
              <button
                type="button"
                disabled={!available || busy}
                onClick={() => void readJournal()}
              >
                View posted journal
              </button>
            </>
          )}
          {journal && (
            <section aria-label="Posted journal">
              <h4>Posted journal</h4>
              <p>
                {journal.accounting_date} · {journal.description} · Book
                revision {journal.book_revision}
              </p>
              <div className="table-scroll">
                <table>
                  <caption>Immutable posted journal lines</caption>
                  <thead>
                    <tr>
                      <th scope="col">Account</th>
                      <th scope="col">Side</th>
                      <th scope="col">Original amount</th>
                      <th scope="col">Functional amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {journal.lines.map((line, index) => (
                      <tr key={index}>
                        <td>
                          {book.accounts.find(
                            (a) => a.account_id === line.account_id,
                          )?.name ?? line.account_id}
                        </td>
                        <td>{line.side}</td>
                        <td>
                          {line.transaction_amount} {line.transaction_currency}
                        </td>
                        <td>
                          {line.functional_amount} {journal.functional_currency}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}
          {categories?.fx_required && (
            <p role="alert">
              An explicit FX valuation is required. This workflow cannot draft
              this transaction; its original currency and amount are preserved.
            </p>
          )}
          {categories && !categories.fx_required && (
            <form onSubmit={(e) => void create(e)}>
              <fieldset disabled={busy || !available || !!retry}>
                <legend>Review the category and accounting details</legend>
                <label>
                  Category account
                  <select
                    required
                    value={category}
                    onChange={(e) => setCategory(e.target.value)}
                  >
                    <option value="">Select a category account</option>
                    {categories.accounts.map((a) => (
                      <option key={a.account_id} value={a.account_id}>
                        {a.code} · {a.name} · {a.category}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Accounting date
                  <input required type="date" readOnly value={date} />
                </label>
                <label>
                  Journal description
                  <input
                    required
                    maxLength={500}
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </label>
                <p>
                  Policy {categories.policy_id} · Functional currency{" "}
                  {categories.functional_currency} · Book revision{" "}
                  {categories.book_revision}
                </p>
              </fieldset>
              <button type="submit" disabled={busy || !available || !category}>
                {retry ? "Retry same draft request" : "Create reviewed draft"}
              </button>
              <p>Creating this draft does not post a journal.</p>
            </form>
          )}
          {draft && (
            <>
              <h4>
                {detail.state === "posted"
                  ? "Posted journal draft"
                  : detail.blockers.length
                    ? "Previous draft requires review"
                    : "Draft awaiting explicit posting"}
              </h4>
              <p>
                {draft.posting_body.accounting_date} ·{" "}
                {draft.posting_body.description}
              </p>
              <div className="table-scroll">
                <table>
                  <caption>Exact proposed journal lines</caption>
                  <thead>
                    <tr>
                      <th scope="col">Account</th>
                      <th scope="col">Side</th>
                      <th scope="col">Original amount</th>
                      <th scope="col">Functional amount</th>
                    </tr>
                  </thead>
                  <tbody>
                    {draft.posting_body.lines.map((line, index) => (
                      <tr key={index}>
                        <td>
                          {book.accounts.find(
                            (a) => a.account_id === line.account_id,
                          )?.name ?? line.account_id}
                        </td>
                        <td>{line.side}</td>
                        <td>
                          {line.transaction_amount} {line.transaction_currency}
                        </td>
                        <td>{line.functional_amount}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {detail.state === "draft_ready" &&
                detail.blockers.length === 0 && (
                  <>
                    <button
                      type="button"
                      disabled={!available || busy}
                      onClick={() =>
                        prepare({
                          operation: "source_post",
                          entity_id: entity.entity_id,
                          book_id: book.book_id,
                          body: {
                            review_id: draft.review_id,
                            draft_id: draft.draft_id,
                            review_revision: draft.review_revision,
                            source_revision: draft.source_revision,
                            binding_revision: draft.binding_revision,
                            expected_book_revision: draft.book_revision,
                            posting_body: draft.posting_body,
                          },
                        })
                      }
                    >
                      Review posting command
                    </button>
                    <p>
                      Command recovery will ask you to confirm execution. This
                      button does not execute it.
                    </p>
                  </>
                )}
            </>
          )}
        </section>
      )}
    </section>
  );
}
