import type { useCommands } from "./commands";
import type { Workspace } from "./types";
import { currencyAt, formatAmount } from "./amount";
// Recovered work is review-only until the owner explicitly chooses its next transition.
export function CommandPanel({
  recovery: r,
  workspace,
}: {
  recovery: ReturnType<typeof useCommands>;
  workspace: Workspace | null;
}) {
  const c = r.command;
  const body = c && ("posting_body" in c.body ? c.body.posting_body : c.body);
  const entity = workspace?.entities.find((e) => e.entity_id === c?.entity_id);
  const book = entity?.books.find((b) => b.book_id === c?.book_id);
  const policyID = body && "policy_id" in body ? body.policy_id : null;
  const policy = book?.policies.find((p) => p.policy_id === policyID);
  const functionalMinorUnit = workspace?.currencies.find(
    (currency) =>
      currency.currency_version_id === policy?.functional_currency_version_id,
  )?.minor_unit;
  const accountingDate =
    body && "accounting_date" in body ? body.accounting_date : "";
  return (
    <section
      className={
        !c && !r.registration && r.ready && !r.error
          ? "command-idle"
          : "command-attention"
      }
      aria-labelledby="command-title"
    >
      <h2 id="command-title">Saved command</h2>
      {!r.ready && (
        <p role="status">Checking saved work before enabling new entries…</p>
      )}
      {r.error && <p role="alert">{r.error}</p>}
      {r.busy && (
        <p role="status">Waiting for the authoritative command result…</p>
      )}
      {!c && !r.registration && r.ready && (
        <p>No unacknowledged command. New work is available.</p>
      )}
      <button disabled={r.busy} onClick={() => void r.reload()}>
        Check saved command
      </button>
      {r.registration && !c && (
        <>
          <p>
            Registration is unconfirmed. Retry this same registration to
            discover its durable result. No new entry is enabled.
          </p>
          <button disabled={r.busy} onClick={r.retryRegistration}>
            Retry registration
          </button>
        </>
      )}
      {c && body && (
        <>
          <p>
            Command <code>{c.command_id}</code> · {c.state}
          </p>
          {c.operation !== "setup" && (
            <p>
              Target: {entity?.name ?? c.entity_id} / {book?.name ?? c.book_id}
            </p>
          )}
          {c.state === "prepared" && (
            <p>
              Saved for review. Last confirmed state: prepared. Another tab may
              have advanced it; the server rechecks before confirmation or
              cancellation.
            </p>
          )}
          {c.state === "committed" && (
            <div role="status">
              <h3>
                {c.operation === "post" || c.operation === "source_post"
                  ? "Posting confirmed"
                  : c.operation === "import"
                    ? "Import confirmed"
                    : "Book creation confirmed"}
              </h3>
              <p>
                The committed result remains saved until you acknowledge it.
                Acknowledging never reverses or changes the books.
              </p>
              {c.result && "journal_id" in c.result && (
                <p>
                  Journal {c.result.journal_id} · revision{" "}
                  {c.result.book_revision}
                </p>
              )}
            </div>
          )}
          {c.state === "cancelled" && (
            <p role="status">
              Cancellation confirmed. This command did not execute. Acknowledge
              before preparing new work.
            </p>
          )}
          {c.operation === "source_post" && "review_id" in c.body && (
            <p>
              Source-linked journal · Review item {c.body.review_id}. Posting
              and its source link commit together.
            </p>
          )}
          {"batch_id" in body ? (
            <>
              <p>Statement import · preview revision {body.preview_revision}</p>
              <p>
                This workspace does not yet display the statement preview.
                Confirm imports only in a client that shows the reviewed rows.
                You can cancel saved work or acknowledge its completed result
                here.
              </p>
              {c.result && "created_count" in c.result && (
                <p>
                  {c.result.created_count} new review items ·{" "}
                  {c.result.existing_count} existing observations. No journals
                  were posted by this import.
                </p>
              )}
            </>
          ) : "lines" in body ? (
            <>
              <p>
                <strong>{body.description}</strong> · Accounting date{" "}
                {body.accounting_date}
              </p>
              <p>
                Functional currency:{" "}
                {policy?.functional_currency ?? "See saved policy metadata"} ·
                Policy {body.policy_id}
              </p>
              <div className="table-scroll">
                <table>
                  <caption>
                    {c.state === "committed"
                      ? "Committed journal request"
                      : "Journal lines for review"}
                  </caption>
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
                    {body.lines.map((line, i) => (
                      <tr key={i}>
                        <th scope="row">
                          {book?.accounts.find(
                            (a) => a.account_id === line.account_id,
                          )?.name ?? line.account_id}
                        </th>
                        <td>{line.side}</td>
                        <td>
                          {formatAmount(
                            line.transaction_amount,
                            currencyAt(
                              workspace?.currencies ?? [],
                              line.transaction_currency,
                              accountingDate,
                            )?.minor_unit,
                          )}{" "}
                          {line.transaction_currency}
                        </td>
                        <td>
                          {formatAmount(
                            line.functional_amount,
                            functionalMinorUnit,
                          )}{" "}
                          {policy?.functional_currency}
                        </td>
                        <td>
                          {line.valuation_basis}
                          {line.rate_observation_id && (
                            <small>
                              {" "}
                              · Rate observation {line.rate_observation_id}
                            </small>
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
                <strong>{body.entity_name}</strong> / {body.book_name}
              </p>
              <p>
                Period: {body.starts_on} until before {body.ends_before}. USD
                functional currency.
              </p>
              <ul>
                {body.accounts.map((a, i) => (
                  <li key={i}>
                    {a.code} · {a.name} · {a.category} · normal {a.normal_side}{" "}
                    · {a.monetary ? "monetary" : "non-monetary"}
                  </li>
                ))}
              </ul>
            </>
          )}
          <details>
            <summary>Exact saved request and result</summary>
            <pre>
              {JSON.stringify({ body: c.body, result: c.result }, null, 2)}
            </pre>
          </details>
          {c.state === "prepared" ? (
            <>
              <button
                disabled={r.busy || !r.ready || c.operation === "import"}
                onClick={() => void r.transition("execute")}
              >
                {c.operation === "post" || c.operation === "source_post"
                  ? "Post journal"
                  : c.operation === "import"
                    ? "Review import in originating client"
                    : "Create book"}
              </button>
              <button
                disabled={r.busy || !r.ready}
                onClick={() => void r.transition("cancel")}
              >
                Cancel saved command
              </button>
            </>
          ) : (
            <button
              disabled={r.busy || !r.ready}
              onClick={() => void r.transition("acknowledge")}
            >
              Acknowledge result
            </button>
          )}
        </>
      )}
    </section>
  );
}
