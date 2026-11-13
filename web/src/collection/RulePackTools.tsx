import { useEffect, useId, useRef, useState } from "react";
import type { Catalog, SourceChoice } from "./types";
import type { RuleDefinition } from "./rules";
import { exampleRulesYaml, exportRulesYaml, importRulesYaml } from "./ruleYaml";
import styles from "./rulePacks.module.css";

type Props = {
  rules: RuleDefinition[];
  catalog: Catalog;
  sources: SourceChoice[];
  disabled: boolean;
  onImport: (rules: RuleDefinition[]) => void;
};

// Compare typed values rather than property insertion order from the visual editor.
const content = (rule: RuleDefinition) =>
  JSON.stringify([
    rule.rule_id,
    rule.name,
    rule.status,
    rule.source_account_id,
    rule.priority,
    rule.conditions.map((condition) => [
      condition.field,
      condition.operator,
      condition.value,
    ]),
    rule.output.kind,
    rule.output.kind === "topic" ? rule.output.topic_id : null,
    [...rule.output.tag_ids].sort(),
  ]);

// Files become local drafts only; the existing saved server review owns persistence.
export function RulePackTools({
  rules,
  catalog,
  sources,
  disabled,
  onImport,
}: Props) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [prepared, setPrepared] = useState<{
    baseline: string;
    rules: RuleDefinition[];
  } | null>(null);
  const [notice, setNotice] = useState("");
  const inputID = useId(),
    textID = useId();
  const epoch = useRef(0);
  const baseline = JSON.stringify(rules);
  const current = useRef({ disabled, baseline });
  current.current = { disabled, baseline };
  useEffect(
    () => () => {
      epoch.current++;
    },
    [],
  );
  useEffect(() => {
    if (disabled) epoch.current++;
  }, [disabled]);
  const replaceText = (value: string) => {
    epoch.current++;
    setText(value);
    setPrepared(null);
    setError("");
    setNotice("");
  };
  const message = (reason: unknown) =>
    reason instanceof Error
      ? reason.message
      : "The rule pack could not be read.";
  const ready = !disabled && prepared?.baseline === baseline ? prepared : null;
  const added =
    ready?.rules.filter(
      (rule) => !rules.some((old) => old.rule_id === rule.rule_id),
    ).length ?? 0;
  const updated =
    ready?.rules.filter((rule) => {
      const old = rules.find((item) => item.rule_id === rule.rule_id);
      return old && content(old) !== content(rule);
    }).length ?? 0;
  return (
    <>
      <button
        type="button"
        className="secondary"
        disabled={disabled}
        onClick={() => {
          setOpen(!open);
          setError("");
        }}
      >
        Import YAML
      </button>
      <button
        type="button"
        className="secondary"
        disabled={disabled}
        onClick={() => {
          try {
            const yaml = exportRulesYaml(rules, catalog, sources);
            const url = URL.createObjectURL(
              new Blob([yaml], { type: "application/yaml;charset=utf-8" }),
            );
            const link = document.createElement("a");
            link.href = url;
            link.download = "jayburd-rules.yaml";
            link.click();
            setTimeout(() => URL.revokeObjectURL(url), 1000);
            setError("");
            setNotice(
              "YAML exported. Rule identities and history are included.",
            );
          } catch (reason) {
            setError(message(reason));
          }
        }}
      >
        Export YAML
      </button>
      {notice && (
        <p role="status" className={styles.notice}>
          {notice}
        </p>
      )}
      {open && (
        <section className={styles.importer} aria-label="Import YAML rules">
          <div className={styles.heading}>
            <h4>Import a rule pack</h4>
            <button
              type="button"
              className="secondary"
              disabled={disabled}
              onClick={() => {
                epoch.current++;
                setOpen(false);
              }}
            >
              Close import
            </button>
          </div>
          <p>
            Choose a YAML file or paste rules below. Existing rules stay unless
            updated by their exported ID. Nothing is saved until you review and
            confirm the changes.
          </p>
          <div className={styles.actions}>
            <label htmlFor={inputID}>
              YAML file
              <input
                id={inputID}
                type="file"
                accept=".yaml,.yml,application/yaml,text/yaml,text/plain"
                disabled={disabled}
                onChange={async (event) => {
                  const file = event.target.files?.[0];
                  event.target.value = "";
                  if (!file || current.current.disabled) return;
                  const attempt = ++epoch.current;
                  const startedAt = current.current.baseline;
                  setPrepared(null);
                  setError("");
                  setNotice("");
                  if (file.size > 262144) {
                    setError("Use a YAML file no larger than 256 KiB.");
                    return;
                  }
                  try {
                    const value = await file.text();
                    if (
                      attempt !== epoch.current ||
                      current.current.disabled ||
                      current.current.baseline !== startedAt
                    )
                      return;
                    setText(value);
                  } catch {
                    if (attempt === epoch.current)
                      setError(
                        "The file could not be read. Try another file or paste its YAML.",
                      );
                  }
                }}
              />
            </label>
            <button
              type="button"
              className="secondary"
              disabled={disabled}
              onClick={() => {
                try {
                  replaceText(exampleRulesYaml(catalog, sources));
                } catch (reason) {
                  setError(message(reason));
                }
              }}
            >
              Use fictional example
            </button>
          </div>
          <label htmlFor={textID}>Rule pack YAML</label>
          <textarea
            id={textID}
            value={text}
            spellCheck={false}
            disabled={disabled}
            onChange={(event) => replaceText(event.target.value)}
            rows={12}
            maxLength={262144}
          />
          <p className={styles.hint}>
            Version 1 packs use conditions and topic names or IDs. Export your
            rules to get a reusable template. Imported text is data, never
            executable code.
          </p>
          <button
            type="button"
            disabled={disabled || !text.trim()}
            onClick={() => {
              try {
                setPrepared({
                  baseline,
                  rules: importRulesYaml(text, rules, catalog, sources),
                });
                setError("");
              } catch (reason) {
                setPrepared(null);
                setError(message(reason));
              }
            }}
          >
            Validate YAML
          </button>
          {ready && (
            <div className={styles.summary}>
              <p role="status">
                {added} added · {updated} updated · {rules.length - updated}{" "}
                existing rules preserved
              </p>
              <button
                type="button"
                disabled={disabled || (!added && !updated)}
                onClick={() => {
                  if (
                    !ready ||
                    current.current.disabled ||
                    current.current.baseline !== ready.baseline
                  )
                    return;
                  onImport(ready.rules);
                  setPrepared(null);
                  setOpen(false);
                  setNotice(
                    "Rule pack added to your draft. Review rule changes to save it.",
                  );
                }}
              >
                Add to draft
              </button>
            </div>
          )}
        </section>
      )}
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
    </>
  );
}
