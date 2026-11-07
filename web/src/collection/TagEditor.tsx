import { useId, useState } from "react";
import { validTags, type LabelAction, type Tag } from "./types";
import styles from "./collection.module.css";

// A separate complete catalog keeps many-to-many tags independent of the single primary topic.
export function TagEditor({
  tags,
  disabled,
  onPreview,
}: {
  tags: Tag[];
  disabled: boolean;
  onPreview: (tags: Tag[]) => void;
}) {
  const [draft, setDraft] = useState(tags);
  const [error, setError] = useState("");
  const statusLabel = useId();
  return (
    <details className={styles.taxonomy}>
      <summary>Manage tags</summary>
      <p>
        Tags add context without changing the primary topic. Rename or retire
        tags; retired identities remain in history.
      </p>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (disabled) return;
          if (!validTags(draft)) {
            setError(
              "Use unique tag names with 1–80 characters and at most 500 active tags.",
            );
            return;
          }
          onPreview(draft);
        }}
      >
        <fieldset disabled={disabled}>
          <legend>Private tag catalog</legend>
          {draft.map((tag, index) => {
            const retired = tags.some(
              (old) => old.tag_id === tag.tag_id && old.status === "retired",
            );
            const change = (patch: Partial<Tag>) => {
              setError("");
              setDraft((old) =>
                old.map((item) =>
                  item.tag_id === tag.tag_id ? { ...item, ...patch } : item,
                ),
              );
            };
            return (
              <div key={tag.tag_id} className={styles.topicEdit}>
                {retired ? (
                  <p>{tag.name} · Retired (history only)</p>
                ) : (
                  <>
                    <label>
                      Tag {index + 1} name
                      <input
                        value={tag.name}
                        maxLength={160}
                        onChange={(event) =>
                          change({ name: event.target.value })
                        }
                      />
                    </label>
                    <label>
                      <span id={`${statusLabel}-${tag.tag_id}`}>
                        Tag {index + 1} status
                      </span>
                      <select
                        aria-labelledby={`${statusLabel}-${tag.tag_id}`}
                        value={tag.status}
                        onChange={(event) =>
                          change({
                            status: event.target.value as Tag["status"],
                          })
                        }
                      >
                        <option value="active">Active</option>
                        <option value="retired">Retired</option>
                      </select>
                    </label>
                  </>
                )}
              </div>
            );
          })}
          <button
            type="button"
            className="secondary"
            disabled={
              draft.length >= 1000 ||
              draft.filter((tag) => tag.status === "active").length >= 500
            }
            onClick={() =>
              setDraft((old) => [
                ...old,
                { tag_id: crypto.randomUUID(), name: "", status: "active" },
              ])
            }
          >
            Add tag
          </button>
          <button
            disabled={JSON.stringify(draft) === JSON.stringify(tags)}
            type="submit"
          >
            Preview tag catalog
          </button>
        </fieldset>
      </form>
      {error && <p role="alert">{error}</p>}
    </details>
  );
}

// Tag intent is explicit; stale prior sets require the owner to choose replacement rather than silently merging.
export function TagSelection({
  tags,
  count,
  stale,
  disabled,
  onPreview,
}: {
  tags: Tag[];
  count: number;
  stale: boolean;
  disabled: boolean;
  onPreview: (action: Extract<LabelAction, { kind: "tags" }>) => void;
}) {
  const [operation, setOperation] = useState<"add" | "remove" | "replace">(
    "add",
  );
  const [ids, setIDs] = useState<string[]>([]);
  const operationLabel = useId();
  const [search, setSearch] = useState("");
  return (
    <details className={styles.tagChoice}>
      <summary>Change strategic tags</summary>
      <fieldset disabled={disabled}>
        <legend>Strategic tags</legend>
        <p>Tags never replace the primary topic. Choose up to 25 tags.</p>
        <label>
          <span id={operationLabel}>Tag operation</span>
          <select
            aria-labelledby={operationLabel}
            value={operation}
            onChange={(event) => {
              setOperation(event.target.value as typeof operation);
              setIDs([]);
            }}
          >
            <option value="add">Add tags</option>
            <option value="remove">Remove tags</option>
            <option value="replace">Replace tags</option>
          </select>
        </label>
        {stale && (
          <p role="status">
            Some selected tags need reconfirmation. Choose Replace tags to
            review a new complete set.
          </p>
        )}
        <label>
          Find strategic tags
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>
        <div
          className={styles.tagChecks}
          role="group"
          aria-label="Available strategic tags"
          tabIndex={0}
        >
          {tags
            .filter(
              (tag) =>
                (operation === "remove" || tag.status === "active") &&
                tag.name
                  .toLocaleLowerCase()
                  .includes(search.toLocaleLowerCase()),
            )
            .map((tag) => (
              <label key={tag.tag_id} className={styles.tagCheck}>
                <input
                  type="checkbox"
                  checked={ids.includes(tag.tag_id)}
                  disabled={!ids.includes(tag.tag_id) && ids.length >= 25}
                  onChange={(event) =>
                    setIDs((old) =>
                      event.target.checked
                        ? [...old, tag.tag_id]
                        : old.filter((id) => id !== tag.tag_id),
                    )
                  }
                />
                <span>
                  {tag.name}
                  {tag.status === "retired" ? " (retired)" : ""}
                </span>
              </label>
            ))}
        </div>
        {!tags.length && <p>No tags yet. Add a tag in Manage tags.</p>}
        <p>
          {ids.length} tags selected.
          {operation === "replace" && ids.length === 0
            ? " This explicitly clears the tag set after confirmation."
            : ""}
        </p>
        <button
          type="button"
          disabled={
            disabled ||
            !count ||
            (operation !== "replace" && ids.length === 0) ||
            (stale && operation !== "replace")
          }
          onClick={() =>
            onPreview({ kind: "tags", operation, tag_ids: [...ids].sort() })
          }
        >
          Preview tags for {count} selected
        </button>
      </fieldset>
    </details>
  );
}

export function TagCatalogReview({ tags }: { tags: Tag[] }) {
  return (
    <ul className={styles.reviewRows} aria-label="Reviewed tag catalog">
      {tags.map((tag) => (
        <li key={tag.tag_id}>
          <strong>{tag.name}</strong>
          <span>{tag.status === "retired" ? "Retired" : "Active"}</span>
        </li>
      ))}
      {!tags.length && <li>No tags in this catalog.</li>}
    </ul>
  );
}
