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
  // Do not execute a recovered account plan until its book and exact currency references are visible.
  const registrationReady =
    c?.operation !== "register_accounts" ||
    (!!book &&
      !!body &&
      "bindings" in body &&
      body.bindings.every((binding) =>
        workspace?.currencies.some(
          (currency) =>
            currency.currency_version_id === binding.currency_version_id,
        ),
      ));
  // Idle recovery still runs in the owning hook; it needs no permanent customer inventory.
  if (
    !c &&
    !r.registration &&
    r.ready &&
    !r.error &&
    !r.accessDenied &&
    !r.busy
  )
    return null;
  return (
    <section
      className="command-attention saved-work"
      aria-busy={r.busy}
      aria-labelledby="command-title"
    >
      <h2 id="command-title">Saved work</h2>
      {!r.ready && !r.error && !r.accessDenied && (
        <p role="status">Checking for unfinished work…</p>
      )}
      {r.error && <p role="alert">{r.error}</p>}
      {r.accessDenied && (
        <>
          <p>
            This saved work is no longer available with your current access.
            Clearing it does not apply or reverse any changes.
          </p>
          <button disabled={r.busy} onClick={() => void r.clearInaccessible()}>
            Clear unavailable saved work
          </button>
        </>
      )}
      {r.busy && (
        <p role="status">
          Waiting for confirmation. Please keep this page open…
        </p>
      )}
      <button
        className="secondary"
        disabled={r.busy}
        onClick={() => void r.reload()}
      >
        Check saved work
      </button>
      {r.registration && !c && (
        <>
          <p>
            Saving is unconfirmed. Retry this same request to check whether it
            was saved. Other changes remain paused.
          </p>
          <button disabled={r.busy} onClick={r.retryRegistration}>
            Retry saving review
          </button>
        </>
      )}
      {c && body && (
        <>
          {c.operation !== "setup" && (
            <p>
              Book: {entity?.name ?? "Entity name unavailable"} /{" "}
              {book?.name ?? "Book name unavailable"}
            </p>
          )}
          {c.state === "prepared" && (
            <p>
              Last checked: ready for your confirmation. Review the details
              below. Changes made in another tab will be checked again before
              confirming or cancelling.
            </p>
          )}
          {c.state === "prepared" && c.operation === "post" && (
            <p role="status">
              Accounting entries must start from an imported transaction or
              retained statement row. This saved manual request cannot be
              posted. Cancel it to return to source review.
            </p>
          )}
          {c.state === "committed" && (
            <div role="status">
              <h3>
                {c.operation === "post" || c.operation === "source_post"
                  ? "Posting confirmed"
                  : c.operation === "import"
                    ? "Import confirmed"
                    : c.operation === "register_accounts"
                      ? "Account registration confirmed"
                      : "Book creation confirmed"}
              </h3>
              <p>
                Your result is saved. Acknowledge it to finish this review and
                continue. Acknowledging does not change your books.
              </p>
              {c.result && "journal_id" in c.result && (
                <p>Posted entry reference: {c.result.journal_id}</p>
              )}
            </div>
          )}
          {c.state === "cancelled" && (
            <p role="status">
              Cancellation confirmed. No changes were applied by this review.
              Acknowledge the result before starting new work.
            </p>
          )}
          {c.operation === "source_post" && "review_id" in c.body && (
            <p>
              Based on the imported transaction you reviewed. When posted, the
              entry and its source reference are saved together.
            </p>
          )}
          {"batch_id" in body ? (
            <>
              <h3>Saved statement import</h3>
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
          ) : "bindings" in body ? (
            <>
              <h3>Accounts to register</h3>
              <p>
                These accounts use the saved mappings below. Registration
                imports source data; it does not post journals.
              </p>
              <p>
                Binding date: {body.binding_date}. Setup expires:{" "}
                {body.expires_at}.
              </p>
              <ul>
                {body.bindings.map((binding) => (
                  <li key={binding.ledger_account_id}>
                    {binding.new_ledger_account?.code}{" "}
                    {binding.new_ledger_account?.name ??
                      book?.accounts.find(
                        (account) =>
                          account.account_id === binding.ledger_account_id,
                      )?.name ??
                      "Existing ledger account"}
                    {" · "}
                    {workspace?.currencies.find(
                      (currency) =>
                        currency.currency_version_id ===
                        binding.currency_version_id,
                    )?.alphabetic_code ?? "Currency metadata unavailable"}
                    {binding.new_ledger_account && (
                      <>
                        {" "}
                        · Ledger available from{" "}
                        {binding.new_ledger_account.active_from}
                      </>
                    )}
                  </li>
                ))}
              </ul>
              {!registrationReady && (
                <p role="status">
                  Load the current book and currency metadata before confirming
                  this registration.
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
                {policy?.functional_currency ?? "Currency name unavailable"}
              </p>
              <div className="table-scroll">
                <table>
                  <caption>
                    {c.state === "committed"
                      ? "Posted entry details"
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
                              · Exchange rate reference{" "}
                              {line.rate_observation_id}
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
          {c.state === "prepared" ? (
            <>
              {c.operation !== "post" && (
                <button
                  disabled={
                    r.busy ||
                    !r.ready ||
                    c.operation === "import" ||
                    !registrationReady
                  }
                  onClick={() => void r.transition("execute")}
                >
                  {c.operation === "source_post"
                    ? "Post journal"
                    : c.operation === "import"
                      ? "Review import in originating client"
                      : c.operation === "register_accounts"
                        ? "Register selected accounts"
                        : "Create book"}
                </button>
              )}
              <button
                disabled={r.busy || !r.ready}
                onClick={() => void r.transition("cancel")}
              >
                Cancel saved review
              </button>
            </>
          ) : (
            <button
              disabled={r.busy || !r.ready}
              onClick={() => void r.transition("acknowledge")}
            >
              {c.operation === "register_accounts"
                ? "Done"
                : "Acknowledge result"}
            </button>
          )}
        </>
      )}
    </section>
  );
}
