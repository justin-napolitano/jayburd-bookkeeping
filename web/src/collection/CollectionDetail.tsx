import { useCallback, useEffect, useRef, useState } from "react";
import { formatAmount } from "../bookkeeping/amount";
import {
  CollectionFailure,
  validSourceDetail,
  type CollectionPort,
  type DetailRequest,
  type SourceDetail,
  type BookChoice,
  type Catalog,
} from "./types";
import { CollectionJournal } from "./CollectionJournal";
import type { JournalIdentity, JournalReader } from "./journal";
import { RuleSuggestionView, RuleApplicationView } from "./RuleViews";
import styles from "./collection.module.css";

// A focused read keeps the list's bounded page in its parent and never expands a selected mutation.
export function CollectionDetail({
  request,
  books = [],
  catalog,
  port,
  readJournal,
  onBack,
  onDenied,
}: {
  request: DetailRequest;
  books?: BookChoice[];
  catalog: Catalog;
  port: CollectionPort;
  readJournal?: JournalReader;
  onBack: () => void;
  onDenied: () => void;
}) {
  const [detail, setDetail] = useState<SourceDetail | null>(null),
    [journal, setJournal] = useState<JournalIdentity | null>(null),
    [busy, setBusy] = useState(true),
    [error, setError] = useState("");
  const epoch = useRef(0),
    heading = useRef<HTMLHeadingElement>(null),
    denied = useRef(onDenied);
  denied.current = onDenied;
  const load = useCallback(async () => {
    const n = ++epoch.current;
    setDetail(null);
    setJournal(null);
    setBusy(true);
    setError("");
    try {
      const value = await port.detail(request);
      if (n !== epoch.current) return;
      if (!validSourceDetail(value, request))
        throw new CollectionFailure("stale");
      setDetail(value);
    } catch (reason) {
      if (n !== epoch.current) return;
      if (reason instanceof CollectionFailure && reason.kind === "denied")
        denied.current();
      else
        setError(
          reason instanceof CollectionFailure && reason.kind === "stale"
            ? "This source detail changed. Return to the list and refresh before continuing."
            : "Source detail is unavailable. Retry or return to the list.",
        );
    } finally {
      if (n === epoch.current) setBusy(false);
    }
  }, [port, request]);
  useEffect(() => {
    void load();
    return () => {
      epoch.current++;
    };
  }, [load]);
  useEffect(() => {
    if (detail && !busy) heading.current?.focus();
  }, [detail, busy]);
  return (
    <section
      className={`${styles.workspace} ${styles.detail}`}
      aria-label="Source transaction detail"
      aria-busy={busy}
    >
      <button className="secondary" onClick={onBack}>
        Back to transactions
      </button>
      <h2 tabIndex={-1} ref={heading}>
        Transaction detail
      </h2>
      {busy && <p role="status">Loading current authorized source detail…</p>}
      {error && (
        <>
          <p role="alert">{error}</p>
          <button onClick={() => void load()}>Retry transaction detail</button>
        </>
      )}
      {detail && (
        <>
          <h3>{detail.row.description}</h3>
          <p>
            {detail.row.booked_date} · {detail.row.account_name}
          </p>
          <p>
            {detail.row.money.state === "held"
              ? `Money unavailable: ${detail.row.money.reason.replaceAll("_", " ")}`
              : `${detail.row.money.direction} ${formatAmount(detail.row.money.amount, detail.row.money.minor_unit)} ${detail.row.money.currency}`}
          </p>
          {detail.row.money.state === "available" && (
            <p>
              Currency reference {detail.row.money.currency_version_id}, valid
              from {detail.row.money.valid_from}
              {detail.row.money.valid_to
                ? ` until ${detail.row.money.valid_to} (exclusive)`
                : ""}
              ; {detail.row.money.minor_unit} decimal places.
            </p>
          )}
          <p>
            Merchant: {detail.row.merchant ?? "Not supplied"}. Source:{" "}
            {detail.row.source_state}. Journal:{" "}
            {detail.row.journal_state.replaceAll("_", " ")}.
          </p>
          <p>Institution unavailable. Legal owner unknown.</p>
          <p>
            Primary topic:{" "}
            {detail.row.owner_topic.state === "topic"
              ? detail.row.owner_topic.name
              : detail.row.owner_topic.state === "uncategorized"
                ? "Explicit Uncategorized"
                : detail.row.owner_topic.state === "stale"
                  ? "Needs confirmation"
                  : "Not classified"}
          </p>
          <p>
            Strategic tags:{" "}
            {detail.row.owner_tags.state === "unclassified"
              ? "Not classified"
              : detail.row.owner_tags.tags
                  .map(
                    (tag) =>
                      tag.name + (tag.status === "retired" ? " (retired)" : ""),
                  )
                  .join(", ") || "None"}
            {detail.row.owner_tags.state === "stale"
              ? " — needs confirmation"
              : ""}
          </p>
          <p>
            Provider suggestion:{" "}
            {detail.row.provider_suggestion.state === "attributed"
              ? `${detail.row.provider_suggestion.label} (${detail.row.provider_suggestion.taxonomy_version}; ${detail.row.provider_suggestion.confidence})`
              : "Unavailable"}
            . This is separate from an owner decision or accounting treatment.
          </p>
          <h3>Saved rule provenance</h3>
          <RuleApplicationView application={detail.row.rule_application} />
          <h3>Current rule suggestion</h3>
          <RuleSuggestionView
            suggestion={detail.row.rule_suggestion}
            catalog={catalog}
          />
          <h3>Relevant book entries</h3>
          {detail.links.state === "unavailable" ? (
            <p>Book entries are unavailable with your current access.</p>
          ) : detail.links.state === "unassigned" ? (
            <p>This source is unassigned to a book.</p>
          ) : !detail.links.journals.length ? (
            <p>No posted journal is linked in the currently readable book.</p>
          ) : (
            <ul className={styles.reviewRows}>
              {detail.links.journals.map((anchor) => (
                <li
                  key={
                    anchor.entity_id +
                    "/" +
                    anchor.book_id +
                    "/" +
                    anchor.journal_id
                  }
                >
                  <details>
                    <summary>Journal reference</summary>
                    <p>
                      Journal {anchor.journal_id}; linked source revision{" "}
                      {anchor.source_revision}.
                    </p>
                  </details>
                  {readJournal && (
                    <button
                      onClick={() =>
                        setJournal({
                          entity_id: anchor.entity_id,
                          book_id: anchor.book_id,
                          journal_id: anchor.journal_id,
                        })
                      }
                    >
                      View posted journal
                      {books.find(
                        (book) =>
                          book.entity_id === anchor.entity_id &&
                          book.book_id === anchor.book_id,
                      )
                        ? " in " +
                          books.find(
                            (book) =>
                              book.entity_id === anchor.entity_id &&
                              book.book_id === anchor.book_id,
                          )!.name
                        : ""}
                    </button>
                  )}
                </li>
              ))}
            </ul>
          )}
          {journal && readJournal && (
            <CollectionJournal
              key={
                journal.entity_id +
                "/" +
                journal.book_id +
                "/" +
                journal.journal_id
              }
              identity={journal}
              read={readJournal}
              onDenied={() => {
                setDetail(null);
                setJournal(null);
                denied.current();
              }}
            />
          )}
        </>
      )}
    </section>
  );
}
