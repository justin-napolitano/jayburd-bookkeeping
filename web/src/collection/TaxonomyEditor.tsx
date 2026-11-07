import { useState } from "react";
import { validTopics, type Topic } from "./types";
import styles from "./collection.module.css";

// Local drafts retain every stable identity; preview is the only route to a durable edit.
export function TaxonomyEditor({
  topics,
  disabled,
  onPreview,
}: {
  topics: Topic[];
  disabled: boolean;
  onPreview: (topics: Topic[]) => void;
}) {
  const [draft, setDraft] = useState(topics);
  const [error, setError] = useState("");
  const change = (id: string, patch: Partial<Topic>) => {
    setError("");
    setDraft((old) =>
      old.map((topic) =>
        topic.topic_id === id ? { ...topic, ...patch } : topic,
      ),
    );
  };
  return (
    <details className={styles.taxonomy}>
      <summary>Manage topics</summary>
      <p>
        Rename, organize or retire topics. Changes take effect only after review
        and confirmation. Retired topics remain in history.
      </p>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (disabled) return;
          if (!validTopics(draft)) {
            setError(
              "Use unique names within each parent, active parents, at most eight levels and 500 active topics. Retiring a parent requires moving or retiring its children.",
            );
            return;
          }
          onPreview(draft);
        }}
      >
        <fieldset disabled={disabled}>
          <legend>Private topic edition</legend>
          {draft.map((topic, index) => {
            const retired =
              topics.find((old) => old.topic_id === topic.topic_id)?.status ===
              "retired";
            return (
              <div className={styles.topicEdit} key={topic.topic_id}>
                {retired ? (
                  <p>{topic.name} · Retired (history only)</p>
                ) : (
                  <>
                    <label>
                      Topic {index + 1} name
                      <input
                        value={topic.name}
                        maxLength={160}
                        onChange={(event) =>
                          change(topic.topic_id, { name: event.target.value })
                        }
                      />
                    </label>
                    <label>
                      Topic {index + 1} parent
                      <select
                        value={topic.parent_id ?? ""}
                        onChange={(event) =>
                          change(topic.topic_id, {
                            parent_id: event.target.value || null,
                          })
                        }
                      >
                        <option value="">No parent</option>
                        {draft
                          .filter(
                            (parent) =>
                              parent.status === "active" &&
                              parent.topic_id !== topic.topic_id,
                          )
                          .map((parent) => (
                            <option
                              key={parent.topic_id}
                              value={parent.topic_id}
                            >
                              {parent.name || "Unnamed topic"}
                            </option>
                          ))}
                      </select>
                    </label>
                    <label>
                      Topic {index + 1} status
                      <select
                        value={topic.status}
                        onChange={(event) =>
                          change(topic.topic_id, {
                            status: event.target.value as Topic["status"],
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
              draft.filter((topic) => topic.status === "active").length >= 500
            }
            onClick={() =>
              setDraft((old) => [
                ...old,
                {
                  topic_id: crypto.randomUUID(),
                  name: "",
                  parent_id: null,
                  status: "active",
                },
              ])
            }
          >
            Add topic
          </button>
          <button
            type="submit"
            disabled={JSON.stringify(draft) === JSON.stringify(topics)}
          >
            Preview topic changes
          </button>
        </fieldset>
      </form>
      {error && <p role="alert">{error}</p>}
    </details>
  );
}

// Persisted taxonomy intent is displayed in full, including retirement and stable parent links.
export function TopicReview({ topics }: { topics: Topic[] }) {
  return (
    <ul className={styles.reviewRows} aria-label="Reviewed topic edition">
      {topics.map((topic) => (
        <li key={topic.topic_id}>
          <strong>{topic.name}</strong>
          <span>
            {topic.status === "retired" ? "Retired" : "Active"} · Parent:{" "}
            {topics.find((parent) => parent.topic_id === topic.parent_id)
              ?.name ?? "None"}
          </span>
        </li>
      ))}
      {!topics.length && <li>No topics in this edition.</li>}
    </ul>
  );
}
