"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { APIError, type API } from "./api";
import type { Book, Entity } from "./types";
import type { ReviewSource } from "./ReviewInbox";
import {
  accountingReasons,
  accountingID,
  readAccounting,
  saveAccounting,
  validAccountingConfiguration,
  type AccountingConfiguration,
  type AccountingPolicy,
  type AccountingStatus,
} from "./automaticAccountingClient";
import {
  exampleAccountingRules,
  exportAccountingRules,
  parseAccountingRules,
} from "./accountingRulesYaml";
import styles from "./automaticAccounting.module.css";

type Snapshot = {
  policy: AccountingPolicy;
  status: AccountingStatus;
  sources: ReviewSource[];
};
type Props = {
  api: API;
  entity: Entity;
  book: Book;
  locked: boolean;
  denied: () => void;
  onInspectReview?: (reviewID: string, sourceID?: string) => void;
};

// Book/access remounts discard private drafts; no policy body or token enters browser persistence.
export function AutomaticAccounting(props: Props) {
  const session = useRef({ api: props.api, generation: 0 });
  if (session.current.api !== props.api)
    session.current = {
      api: props.api,
      generation: session.current.generation + 1,
    };
  return (
    <AccountingPanel
      key={`${props.entity.entity_id}/${props.book.book_id}/${props.book.book_revision}/${props.locked}/${session.current.generation}`}
      {...props}
    />
  );
}
function AccountingPanel({
  api,
  entity,
  book,
  locked,
  denied,
  onInspectReview,
}: Props) {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null),
    [error, setError] = useState(""),
    [notice, setNotice] = useState("");
  const [loading, setLoading] = useState(false),
    [saving, setSaving] = useState(false),
    [unavailable, setUnavailable] = useState(false);
  const [editing, setEditing] = useState(false),
    [enabled, setEnabled] = useState(false),
    [bookPolicy, setBookPolicy] = useState(""),
    [from, setFrom] = useState(""),
    [through, setThrough] = useState("");
  const [yaml, setYaml] = useState(""),
    [review, setReview] = useState<AccountingConfiguration | null>(null),
    [uncertain, setUncertain] = useState(false);
  const pending = useRef<string | null>(null),
    flight = useRef(false),
    fileEpoch = useRef(0),
    alive = useRef(true),
    owner = useRef(api);
  owner.current = api;
  useEffect(
    () => () => {
      alive.current = false;
      fileEpoch.current++;
    },
    [],
  );
  const current = () => alive.current && owner.current === api;
  const load = useCallback(async () => {
    if (locked || flight.current) return;
    flight.current = true;
    setLoading(true);
    setError("");
    setSnapshot(null);
    setUnavailable(false);
    try {
      const [state, sources] = await Promise.all([
        readAccounting(api, entity.entity_id, book.book_id),
        api<{ profile: string; items: ReviewSource[] }>(
          `/v1/entities/${entity.entity_id}/books/${book.book_id}/sources`,
        ),
      ]);
      if (
        !sources ||
        sources.profile !== "source-list-v1" ||
        !Array.isArray(sources.items) ||
        sources.items.length > 100 ||
        !sources.items.every(
          (source) =>
            accountingID(source.source_account_id) &&
            typeof source.display_name === "string" &&
            source.display_name.length <= 500,
        ) ||
        new Set(sources.items.map((source) => source.source_account_id))
          .size !== sources.items.length
      )
        throw new APIError(0, true, "read");
      if (current()) setSnapshot({ ...state, sources: sources.items });
    } catch (reason) {
      if (!current()) return;
      setUnavailable(
        reason instanceof APIError && [403, 404].includes(reason.status),
      );
      setError(
        reason instanceof APIError && [403, 404].includes(reason.status)
          ? "Automatic accounting is unavailable for this book or your current access."
          : "Accounting status is unavailable. Whether automatic posting is active is unknown; refresh to check.",
      );
    } finally {
      if (current()) {
        flight.current = false;
        setLoading(false);
      }
    }
    // Scope-owned state is recreated when book/access changes; API identity still guards late replies.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [api, entity.entity_id, book.book_id, locked]);
  useEffect(() => {
    alive.current = true;
    void load();
    return () => {
      alive.current = false;
      fileEpoch.current++;
    };
  }, [load]);
  const blocked = locked || loading || saving || uncertain;
  const sourceName = (id: string) =>
    snapshot?.sources.find((source) => source.source_account_id === id)
      ?.display_name ?? "Source account";
  const accountName = (id: string) =>
    book.accounts.find((account) => account.account_id === id)?.name ??
    "Unavailable account";
  const invalidate = () => {
    fileEpoch.current++;
    setReview(null);
    setError("");
    setNotice("");
  };
  const openEditor = () => {
    if (!snapshot || blocked || editing) return;
    fileEpoch.current++;
    const config = snapshot.policy.policy;
    setEnabled(snapshot.policy.enabled);
    setBookPolicy(config?.book_policy_id ?? book.policies[0]?.policy_id ?? "");
    setFrom(config?.effective_from ?? book.policies[0]?.effective_from ?? "");
    setThrough(config?.effective_to ?? "");
    setYaml(
      config
        ? exportAccountingRules(config.rules)
        : exampleAccountingRules(book, snapshot.sources),
    );
    setReview(null);
    setError("");
    setEditing(true);
  };
  const download = () => {
    if (!snapshot || blocked) return;
    const contents = snapshot.policy.policy
      ? exportAccountingRules(snapshot.policy.policy.rules)
      : exampleAccountingRules(book, snapshot.sources);
    const url = URL.createObjectURL(
      new Blob([contents], { type: "application/yaml;charset=utf-8" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = "jayburd-accounting-rules.yaml";
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };
  const save = async () => {
    if (
      locked ||
      flight.current ||
      (!pending.current && (!review || !snapshot))
    )
      return;
    const body =
      pending.current ??
      JSON.stringify({
        key: crypto.randomUUID(),
        expected_revision: snapshot!.policy.revision,
        enabled,
        ...review,
      });
    if (new TextEncoder().encode(body).length > 131072) {
      setError(
        "The complete configuration exceeds 128 KiB. Use fewer or shorter mappings.",
      );
      return;
    }
    pending.current = body;
    flight.current = true;
    setSaving(true);
    setError("");
    try {
      await saveAccounting(api, entity.entity_id, book.book_id, body);
      if (!current()) return;
      fileEpoch.current++;
      pending.current = null;
      setUncertain(false);
      setReview(null);
      setEditing(false);
      setNotice(
        "Accounting configuration saved. Eligible entries use this policy without per-entry confirmation.",
      );
      flight.current = false;
      await load();
    } catch (reason) {
      if (!current()) return;
      if (
        reason instanceof APIError &&
        [401, 403, 404].includes(reason.status)
      ) {
        fileEpoch.current++;
        pending.current = null;
        setSnapshot(null);
        setYaml("");
        setReview(null);
        setEditing(false);
        setUncertain(false);
        setError(
          "Accounting access is unavailable. Check current access before making another change.",
        );
        denied();
      } else if (!(reason instanceof APIError) || reason.uncertain) {
        setUncertain(true);
        setError(
          "Saving this configuration is unconfirmed. Retry the same configuration to recover its result. If you leave this book, check its saved policy before editing again.",
        );
      } else {
        pending.current = null;
        setUncertain(false);
        setReview(null);
        setError(
          reason.status === 409
            ? "The policy changed. Close configuration, refresh status and review the current policy before saving again."
            : "The server rejected this configuration. Check the mappings, account policy and effective dates.",
        );
      }
    } finally {
      if (current()) {
        flight.current = false;
        setSaving(false);
      }
    }
  };
  const runtime = snapshot?.status.runtime_enabled,
    policyOn = snapshot?.policy.enabled;
  const removed =
    review && snapshot?.policy.policy
      ? snapshot.policy.policy.rules.filter(
          (old) => !review.rules.some((rule) => rule.rule_id === old.rule_id),
        ).length
      : 0;
  return (
    <section
      className={styles.panel}
      aria-label="Automatic accounting"
      aria-busy={loading || saving}
    >
      <div className={styles.heading}>
        <h3>Automatic accounting</h3>
        <div className={styles.actions}>
          <button
            type="button"
            className="secondary"
            disabled={blocked || editing}
            onClick={() => void load()}
          >
            Refresh accounting status
          </button>
          {snapshot && (
            <button
              type="button"
              className="secondary"
              disabled={blocked || editing}
              onClick={openEditor}
            >
              Configure accounting
            </button>
          )}
        </div>
      </div>
      {loading && <p role="status">Checking accounting policy and runtime…</p>}
      {locked && !snapshot && (
        <p>Finish saved work before configuring accounting.</p>
      )}
      {snapshot && (
        <>
          <p className={styles.states}>
            <span>
              Policy: <strong>{policyOn ? "On" : "Off"}</strong>
            </span>
            <span>
              Background posting:{" "}
              <strong>{runtime ? "Active" : "Not active"}</strong>
            </span>
          </p>
          <p role="status">
            {!runtime
              ? policyOn
                ? "Configuration is saved. Automatic posting is not active in this deployment."
                : "The policy is off. Background posting is not active in this deployment."
              : !policyOn
                ? "Automation is available. This book’s policy is off."
                : "Eligible source transactions post automatically under this policy. Unresolved items stay in review."}
          </p>
          {snapshot.status.blocked_reason && (
            <p role="status">
              {accountingReasons[snapshot.status.blocked_reason]}
            </p>
          )}
          <p>
            {snapshot.status.pending_count} awaiting accounting ·{" "}
            {snapshot.status.attention_count} need attention
          </p>
          {!!snapshot.policy.policy?.rules.length && !editing && (
            <div className={styles.tableScroll}>
              <table>
                <caption>Configured accounting mappings</caption>
                <thead>
                  <tr>
                    <th>Mapping</th>
                    <th>Exact description</th>
                    <th>Source</th>
                    <th>Account</th>
                  </tr>
                </thead>
                <tbody>
                  {snapshot.policy.policy.rules.map((rule) => (
                    <tr key={rule.rule_id}>
                      <th scope="row">{rule.name}</th>
                      <td>{rule.description}</td>
                      <td>{sourceName(rule.source_account_id)}</td>
                      <td>{accountName(rule.offset_account_id)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {snapshot.status.last_operation?.state === "committed" &&
            onInspectReview && (
              <button
                type="button"
                className="secondary"
                onClick={() =>
                  onInspectReview(snapshot.status.last_operation!.review_id)
                }
              >
                Inspect latest posted source
              </button>
            )}
          {!!snapshot.status.holds.length && (
            <details>
              <summary>
                Accounting exceptions ({snapshot.status.attention_count})
              </summary>
              <ul className={styles.holds}>
                {snapshot.status.holds.map((hold) => (
                  <li key={hold.review_id}>
                    <strong>{sourceName(hold.source_account_id)}</strong>
                    <span>{accountingReasons[hold.reason]}</span>
                    {onInspectReview && (
                      <button
                        type="button"
                        className="secondary"
                        onClick={() =>
                          onInspectReview(
                            hold.review_id,
                            hold.source_account_id,
                          )
                        }
                      >
                        Review source transaction
                      </button>
                    )}
                  </li>
                ))}
              </ul>
              {snapshot.status.holds_truncated && (
                <p>
                  Showing the first {snapshot.status.holds.length} exceptions;
                  the count above includes all exceptions.
                </p>
              )}
            </details>
          )}
        </>
      )}
      {notice && <p role="status">{notice}</p>}
      {error && <p role="alert">{error}</p>}
      {unavailable && (
        <p>
          Saved policy and runtime availability are separate. No enabled state
          is inferred from an unavailable read.
        </p>
      )}
      {editing && snapshot && (
        <div className={styles.editor}>
          <div className={styles.heading}>
            <h4>Accounting policy</h4>
            <button
              type="button"
              className="secondary"
              disabled={blocked}
              onClick={() => {
                fileEpoch.current++;
                setEditing(false);
                setReview(null);
                setError("");
              }}
            >
              Close configuration
            </button>
          </div>
          <fieldset disabled={blocked}>
            <legend>One-time configuration</legend>
            <label className={styles.check}>
              <input
                type="checkbox"
                checked={enabled}
                onChange={(event) => {
                  invalidate();
                  setEnabled(event.target.checked);
                }}
              />
              Enable automatic posting for these mappings
            </label>
            <div className={styles.fields}>
              <label>
                Book policy
                <select
                  value={bookPolicy}
                  onChange={(event) => {
                    invalidate();
                    setBookPolicy(event.target.value);
                  }}
                >
                  <option value="">Choose an existing policy</option>
                  {book.policies.map((policy) => (
                    <option key={policy.policy_id} value={policy.policy_id}>
                      {policy.functional_currency} from {policy.effective_from}
                      {policy.effective_to
                        ? ` until ${policy.effective_to}`
                        : ""}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Booked dates from
                <input
                  type="date"
                  value={from}
                  onChange={(event) => {
                    invalidate();
                    setFrom(event.target.value);
                  }}
                />
              </label>
              <label>
                Booked dates before (optional)
                <input
                  type="date"
                  value={through}
                  onChange={(event) => {
                    invalidate();
                    setThrough(event.target.value);
                  }}
                />
              </label>
            </div>
            <p>
              Import the complete future mapping set. Rule IDs preserve
              identity; historical entries and mappings remain unchanged. Only
              exact descriptions match. Transfer, card-payment, loan-split and
              unsupported currency/valuation cases stay held.
            </p>
            <div className={styles.actions}>
              <label>
                Accounting YAML file
                <input
                  type="file"
                  accept=".yaml,.yml,text/yaml,application/yaml"
                  onChange={async (event) => {
                    const file = event.target.files?.[0];
                    event.target.value = "";
                    if (!file || blocked) return;
                    invalidate();
                    const attempt = ++fileEpoch.current;
                    if (file.size > 131072) {
                      setError("Use a YAML file no larger than 128 KiB.");
                      return;
                    }
                    try {
                      const value = await file.text();
                      if (
                        current() &&
                        attempt === fileEpoch.current &&
                        !flight.current
                      ) {
                        setReview(null);
                        setYaml(value);
                      }
                    } catch {
                      if (current() && attempt === fileEpoch.current)
                        setError(
                          "The YAML file could not be read. Paste its text or choose another file.",
                        );
                    }
                  }}
                />
              </label>
              <button type="button" className="secondary" onClick={download}>
                Export accounting YAML
              </button>
            </div>
            <label>
              Accounting mapping YAML
              <textarea
                rows={10}
                value={yaml}
                spellCheck={false}
                maxLength={131072}
                onChange={(event) => {
                  invalidate();
                  setYaml(event.target.value);
                }}
              />
            </label>
            <button
              type="button"
              onClick={() => {
                fileEpoch.current++;
                try {
                  const rules = parseAccountingRules(
                    yaml,
                    book,
                    snapshot.sources,
                    snapshot.policy.policy?.rules ?? [],
                  );
                  const config = {
                    book_policy_id: bookPolicy,
                    effective_from: from,
                    effective_to: through || null,
                    rules,
                  };
                  if (
                    !validAccountingConfiguration(config) ||
                    !book.policies.some(
                      (policy) => policy.policy_id === bookPolicy,
                    )
                  )
                    throw new Error(
                      "Choose an existing book policy and valid effective dates.",
                    );
                  setReview(config);
                  setError("");
                } catch (reason) {
                  setReview(null);
                  setError(
                    reason instanceof Error
                      ? reason.message
                      : "The mapping configuration could not be read.",
                  );
                }
              }}
            >
              Review accounting configuration
            </button>
          </fieldset>
          {review && (
            <div
              className={styles.review}
              aria-label="Accounting configuration review"
            >
              <h4>Review before saving</h4>
              <p>
                Policy will be {enabled ? "on" : "off"}. Applies to booked dates
                from {review.effective_from}
                {review.effective_to
                  ? ` before ${review.effective_to}`
                  : " onward"}
                . {review.rules.length} mappings; {removed} existing mappings
                removed from future use.
              </p>
              <div className={styles.tableScroll}>
                <table>
                  <caption>Mappings to save</caption>
                  <thead>
                    <tr>
                      <th>Exact source description</th>
                      <th>Source account</th>
                      <th>Direction</th>
                      <th>Offset account</th>
                      <th>Treatment</th>
                    </tr>
                  </thead>
                  <tbody>
                    {review.rules.map((rule) => (
                      <tr key={rule.rule_id}>
                        <th scope="row">{rule.description}</th>
                        <td>{sourceName(rule.source_account_id)}</td>
                        <td>{rule.direction}</td>
                        <td>{accountName(rule.offset_account_id)}</td>
                        <td>{rule.treatment.replaceAll("_", " ")}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p>
                The server checks current permissions, account policy and source
                evidence. Saving an enabled policy allows eligible routine
                entries without another confirmation. Background posting is
                currently {runtime ? "active" : "not active"}.
              </p>
              <button
                type="button"
                disabled={blocked}
                onClick={() => void save()}
              >
                Save accounting configuration
              </button>
            </div>
          )}
        </div>
      )}
      {uncertain && (
        <button
          type="button"
          disabled={locked || saving}
          onClick={() => void save()}
        >
          Retry same accounting configuration
        </button>
      )}
    </section>
  );
}
