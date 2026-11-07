import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import type { Catalog, SourceChoice } from "./types";
import {
  canonicalRules,
  ruleCatalogError,
  type RuleCondition,
  type RuleDefinition,
  type RuleOutput,
  type RuleSeed,
} from "./rules";
import { RuleCatalogReview } from "./RuleViews";
import styles from "./collection.module.css";
type Draft = Omit<RuleDefinition, "priority"> & {
  priorityText: string;
  conditionKeys: string[];
};
const draftOf = (rule: RuleDefinition): Draft => {
  const { priority, ...rest } = rule;
  return {
    ...rest,
    priorityText: String(priority),
    conditionKeys: rule.conditions.map(() => crypto.randomUUID()),
  };
};
const wireOf = (rule: Draft): RuleDefinition => {
  const { priorityText, conditionKeys: _keys, ...rest } = rule;
  return {
    ...rest,
    priority: /^-?\d+$/.test(priorityText) ? Number(priorityText) : NaN,
  };
};
function Choice({
  label,
  value,
  onChange,
  children,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <label>
      <span id={id}>{label}</span>
      <select
        aria-labelledby={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {children}
      </select>
    </label>
  );
}

function OutputTags({
  catalog,
  ids,
  disabled,
  onChange,
}: {
  catalog: Catalog;
  ids: string[];
  disabled: boolean;
  onChange: (ids: string[]) => void;
}) {
  const [search, setSearch] = useState("");
  return (
    <fieldset disabled={disabled}>
      <legend>Rule output tags</legend>
      <label>
        Find rule output tags
        <input
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
      </label>
      <div
        className={styles.tagChecks}
        role="group"
        aria-label="Available rule output tags"
        tabIndex={0}
      >
        {catalog.tags
          .filter((tag) =>
            tag.name.toLocaleLowerCase().includes(search.toLocaleLowerCase()),
          )
          .map((tag) => (
            <label key={tag.tag_id} className={styles.tagCheck}>
              <input
                type="checkbox"
                checked={ids.includes(tag.tag_id)}
                disabled={
                  !ids.includes(tag.tag_id) &&
                  (tag.status !== "active" || ids.length >= 20)
                }
                onChange={(event) =>
                  onChange(
                    event.target.checked
                      ? [...ids, tag.tag_id].sort()
                      : ids.filter((id) => id !== tag.tag_id),
                  )
                }
              />
              {tag.name}
              {tag.status === "retired" ? " (retired)" : ""}
            </label>
          ))}
      </div>
      <p>
        {ids.length} of 20 output tags selected. Manual tag sets remain
        independent.
      </p>
    </fieldset>
  );
}

// Complete immutable editions use the same saved-review engine; edits never evaluate or persist rules locally.
export function RuleEditor({
  catalog,
  sources,
  disabled,
  onPreview,
  seed,
  onSeedConsumed,
}: {
  catalog: Catalog;
  sources: SourceChoice[];
  disabled: boolean;
  onPreview: (rules: RuleDefinition[]) => void;
  seed?: RuleSeed | null;
  onSeedConsumed?: () => void;
}) {
  const [draft, setDraft] = useState<Draft[]>(() => catalog.rules.map(draftOf)),
    [open, setOpen] = useState(false),
    [error, setError] = useState(""),
    [selectedRule, setSelectedRule] = useState<string | null>(
      () =>
        catalog.rules.find((rule) => rule.status === "active")?.rule_id ?? null,
    );
  const heading = useRef<HTMLHeadingElement>(null),
    focusRequested = useRef(false),
    consumed = useRef<string | null>(null);
  const add = (seedValue?: RuleSeed) => {
    const rule: RuleDefinition = {
      rule_id: crypto.randomUUID(),
      name: "",
      status: "active",
      source_account_id: seedValue?.source_id ?? null,
      priority: 0,
      conditions: [
        seedValue?.condition ?? {
          field: "merchant",
          operator: "exact",
          value: "",
        },
      ],
      output: seedValue?.output ?? { kind: "topic", topic_id: "", tag_ids: [] },
    };
    focusRequested.current = true;
    setDraft((old) => [draftOf(rule), ...old]);
    setSelectedRule(rule.rule_id);
    setOpen(true);
    setError("");
  };
  useEffect(() => {
    if (seed && consumed.current !== seed.command_id + "/" + seed.prepare_key) {
      consumed.current = seed.command_id + "/" + seed.prepare_key;
      add(seed);
      onSeedConsumed?.();
    }
  }, [seed, onSeedConsumed]);
  useEffect(() => {
    if (open && focusRequested.current) {
      focusRequested.current = false;
      heading.current?.focus();
    }
  }, [open, draft]);
  const change = (id: string, patch: Partial<Draft>) => {
    setError("");
    setDraft((old) =>
      old.map((rule) => (rule.rule_id === id ? { ...rule, ...patch } : rule)),
    );
  };
  const active = draft.filter(
    (rule) =>
      !catalog.rules.some(
        (old) => old.rule_id === rule.rule_id && old.status === "retired",
      ),
  );
  const retired = catalog.rules.filter((rule) => rule.status === "retired");
  return (
    <details
      className={styles.taxonomy}
      open={open}
      onToggle={(event) => setOpen(event.currentTarget.open)}
    >
      <summary>Manage rules</summary>
      {open && (
        <>
          <h3 ref={heading} tabIndex={-1}>
            Rule conditions and outputs
          </h3>
          <p>
            Manual decisions win. Account rules precede workspace rules;
            equal-rank disagreement abstains. Rules only suggest analytic labels
            and never create journals.
          </p>
          <p>
            All conditions must match. Text matching is case-sensitive; regex
            syntax and work are checked by the server. A merchant match alone
            does not establish business purpose or accounting treatment.
          </p>
          <form
            onSubmit={(event) => {
              event.preventDefault();
              if (disabled) return;
              const rules = draft.map(wireOf),
                message = ruleCatalogError(
                  rules,
                  catalog.rules,
                  catalog,
                  sources,
                );
              if (message) {
                setError(message);
                return;
              }
              onPreview(canonicalRules(rules));
            }}
          >
            <fieldset disabled={disabled}>
              <legend>Private rule catalog</legend>
              <button
                type="button"
                className="secondary"
                disabled={
                  draft.length >= 1000 ||
                  draft.filter((rule) => rule.status === "active").length >= 100
                }
                onClick={() => add()}
              >
                Add rule
              </button>
              {active.map((rule, index) => (
                <details
                  key={rule.rule_id}
                  open={selectedRule === rule.rule_id}
                  onToggle={(event) => {
                    if (event.currentTarget.open) setSelectedRule(rule.rule_id);
                    else if (selectedRule === rule.rule_id)
                      setSelectedRule(null);
                  }}
                >
                  <summary>
                    Rule {index + 1}: {rule.name || "New rule"}
                  </summary>
                  {selectedRule === rule.rule_id && (
                    <fieldset className={styles.ruleEdit}>
                      <legend>Rule {index + 1}</legend>
                      <div className={styles.topicEdit}>
                        <label>
                          Rule {index + 1} name
                          <input
                            maxLength={160}
                            value={rule.name}
                            onChange={(event) =>
                              change(rule.rule_id, { name: event.target.value })
                            }
                          />
                        </label>
                        <Choice
                          label={`Rule ${index + 1} status`}
                          value={rule.status}
                          onChange={(value) =>
                            change(rule.rule_id, {
                              status: value as RuleDefinition["status"],
                            })
                          }
                        >
                          <option value="active">Active</option>
                          <option value="retired">Retired</option>
                        </Choice>
                        <Choice
                          label={`Rule ${index + 1} scope`}
                          value={rule.source_account_id ?? ""}
                          onChange={(value) =>
                            change(rule.rule_id, {
                              source_account_id: value || null,
                            })
                          }
                        >
                          <option value="">Whole private workspace</option>
                          {sources.map((source) => (
                            <option
                              key={source.source_id}
                              value={source.source_id}
                            >
                              {source.name}
                            </option>
                          ))}
                        </Choice>
                        <label>
                          Rule {index + 1} priority
                          <input
                            inputMode="numeric"
                            value={rule.priorityText}
                            maxLength={9}
                            onChange={(event) =>
                              change(rule.rule_id, {
                                priorityText: event.target.value,
                              })
                            }
                          />
                        </label>
                      </div>
                      {rule.conditions.map((condition, conditionIndex) => {
                        const update = (patch: Partial<RuleCondition>) =>
                          change(rule.rule_id, {
                            conditions: rule.conditions.map((old, i) =>
                              i === conditionIndex ? { ...old, ...patch } : old,
                            ),
                          });
                        const label = `Rule ${index + 1} condition ${conditionIndex + 1}`;
                        return (
                          <div
                            key={rule.conditionKeys[conditionIndex]}
                            className={styles.ruleCondition}
                          >
                            <Choice
                              label={label + " field"}
                              value={condition.field}
                              onChange={(value) =>
                                update({
                                  field: value as RuleCondition["field"],
                                  operator:
                                    value === "date" ? "on_or_after" : "exact",
                                  value: "",
                                })
                              }
                            >
                              {[
                                "merchant",
                                "description",
                                "account",
                                "currency_version",
                                "direction",
                                "date",
                              ].map((field) => (
                                <option key={field} value={field}>
                                  {field.replaceAll("_", " ")}
                                </option>
                              ))}
                            </Choice>
                            <Choice
                              label={label + " operator"}
                              value={condition.operator}
                              onChange={(value) =>
                                update({
                                  operator: value as RuleCondition["operator"],
                                })
                              }
                            >
                              {(condition.field === "merchant" ||
                              condition.field === "description"
                                ? ["exact", "contains", "regex"]
                                : condition.field === "date"
                                  ? ["on_or_after", "on_or_before"]
                                  : ["exact"]
                              ).map((operator) => (
                                <option key={operator} value={operator}>
                                  {operator.replaceAll("_", " ")}
                                </option>
                              ))}
                            </Choice>
                            {condition.field === "account" ? (
                              <Choice
                                label={label + " value"}
                                value={condition.value}
                                onChange={(value) => update({ value })}
                              >
                                <option value="">Choose an account</option>
                                {sources.map((source) => (
                                  <option
                                    key={source.source_id}
                                    value={source.source_id}
                                  >
                                    {source.name}
                                  </option>
                                ))}
                              </Choice>
                            ) : condition.field === "currency_version" ? (
                              <Choice
                                label={label + " value"}
                                value={condition.value}
                                onChange={(value) => update({ value })}
                              >
                                <option value="">
                                  Choose a dated currency
                                </option>
                                {catalog.currencies.map((currency) => (
                                  <option
                                    key={currency.currency_version_id}
                                    value={currency.currency_version_id}
                                  >
                                    {currency.code} from {currency.valid_from}
                                  </option>
                                ))}
                              </Choice>
                            ) : condition.field === "direction" ? (
                              <Choice
                                label={label + " value"}
                                value={condition.value}
                                onChange={(value) => update({ value })}
                              >
                                <option value="">Choose a direction</option>
                                <option value="inflow">Inflow</option>
                                <option value="outflow">Outflow</option>
                              </Choice>
                            ) : (
                              <label>
                                {label} value
                                <input
                                  type={
                                    condition.field === "date" ? "date" : "text"
                                  }
                                  maxLength={128}
                                  value={condition.value}
                                  onChange={(event) =>
                                    update({ value: event.target.value })
                                  }
                                />
                              </label>
                            )}
                            <button
                              type="button"
                              className="secondary"
                              disabled={rule.conditions.length <= 1}
                              onClick={() =>
                                change(rule.rule_id, {
                                  conditions: rule.conditions.filter(
                                    (_, i) => i !== conditionIndex,
                                  ),
                                  conditionKeys: rule.conditionKeys.filter(
                                    (_, i) => i !== conditionIndex,
                                  ),
                                })
                              }
                            >
                              Remove condition {conditionIndex + 1} from rule{" "}
                              {index + 1}
                            </button>
                          </div>
                        );
                      })}
                      <button
                        type="button"
                        className="secondary"
                        disabled={rule.conditions.length >= 8}
                        onClick={() =>
                          change(rule.rule_id, {
                            conditions: [
                              ...rule.conditions,
                              {
                                field: "description",
                                operator: "contains",
                                value: "",
                              },
                            ],
                            conditionKeys: [
                              ...rule.conditionKeys,
                              crypto.randomUUID(),
                            ],
                          })
                        }
                      >
                        Add condition to rule {index + 1}
                      </button>
                      <Choice
                        label={`Rule ${index + 1} primary topic`}
                        value={
                          rule.output.kind === "uncategorized"
                            ? "uncategorized"
                            : rule.output.topic_id
                        }
                        onChange={(value) =>
                          change(rule.rule_id, {
                            output:
                              value === "uncategorized"
                                ? {
                                    kind: "uncategorized",
                                    tag_ids: rule.output.tag_ids,
                                  }
                                : {
                                    kind: "topic",
                                    topic_id: value,
                                    tag_ids: rule.output.tag_ids,
                                  },
                          })
                        }
                      >
                        <option value="">Choose a primary topic</option>
                        <option value="uncategorized">
                          Explicit Uncategorized
                        </option>
                        {catalog.topics
                          .filter((topic) => topic.status === "active")
                          .map((topic) => (
                            <option key={topic.topic_id} value={topic.topic_id}>
                              {topic.name}
                            </option>
                          ))}
                      </Choice>
                      <OutputTags
                        catalog={catalog}
                        ids={rule.output.tag_ids}
                        disabled={disabled}
                        onChange={(tag_ids) =>
                          change(rule.rule_id, {
                            output: { ...rule.output, tag_ids },
                          })
                        }
                      />
                    </fieldset>
                  )}
                </details>
              ))}
              <button
                type="submit"
                disabled={
                  disabled ||
                  JSON.stringify(draft.map(wireOf)) ===
                    JSON.stringify(catalog.rules)
                }
              >
                Preview rule changes
              </button>
            </fieldset>
          </form>
          {error && <p role="alert">{error}</p>}
          {!!retired.length && (
            <details>
              <summary>Retired rule history ({retired.length})</summary>
              <RuleCatalogReview rules={retired} catalog={catalog} />
            </details>
          )}
        </>
      )}
    </details>
  );
}
