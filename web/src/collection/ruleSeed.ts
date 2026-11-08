import type { LabelCommand } from "./types";
import type { RuleOutput, RuleSeed } from "./rules";
// Only a single confirmed manual correction can seed an inert draft; no rule or historical batch is auto-created.
export function correctionRuleSeed(command: LabelCommand): RuleSeed | null {
  if (
    command.state !== "committed" ||
    command.preview.selection.length !== 1 ||
    command.preview.review_rows.length !== 1
  )
    return null;
  const action = command.preview.action,
    row = command.preview.review_rows[0];
  if (!["topic", "uncategorized", "tags"].includes(action.kind)) return null;
  let output: RuleOutput | null = null;
  const tags =
    action.kind === "tags" &&
    command.preview.profile !== "private-collection-preview-v1"
      ? (command.preview.tag_results[0]?.tags.map((tag) => tag.tag_id) ?? [])
      : "owner_tags" in row
        ? row.owner_tags.tags.map((tag) => tag.tag_id)
        : [];
  if (action.kind === "topic")
    output = { kind: "topic", topic_id: action.topic_id, tag_ids: tags };
  else if (action.kind === "uncategorized")
    output = { kind: "uncategorized", tag_ids: tags };
  else if (row.owner_topic.state === "topic")
    output = {
      kind: "topic",
      topic_id: row.owner_topic.topic_id,
      tag_ids: tags,
    };
  else if (row.owner_topic.state === "uncategorized")
    output = { kind: "uncategorized", tag_ids: tags };
  return {
    workspace_id: command.workspace_id,
    command_id: command.command_id,
    prepare_key: command.prepare_key,
    source_id: row.source_id,
    condition: row.merchant
      ? { field: "merchant", operator: "exact", value: row.merchant }
      : { field: "description", operator: "contains", value: row.description },
    output,
  };
}
