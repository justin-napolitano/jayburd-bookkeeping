import { describe, expect, it } from "vitest";
import {
  collectionCatalog,
  collectionPreview,
  collectionRow,
  collectionSnapshot,
  collectionWorkspace,
} from "../../tests/collection-fixture";
import {
  samePreview,
  selectionOf,
  validCommand,
  validPreview,
  validTags,
  type LegacyPreview,
  type PreviewRequest,
} from "./types";

const request = (
  operation: "add" | "remove" | "replace" = "add",
): PreviewRequest => ({
  workspace_id: collectionWorkspace.workspace_id,
  snapshot: collectionSnapshot,
  selection: [selectionOf(collectionRow())],
  action: {
    kind: "tags",
    operation,
    tag_ids: [collectionCatalog.tags[0].tag_id],
  },
});
describe("independent strategic tag contract", () => {
  it("validates complete stable catalogs with codepoint names and exact active bounds", () => {
    expect(
      validTags([{ ...collectionCatalog.tags[0], name: "😀".repeat(80) }]),
    ).toBe(true);
    expect(
      validTags([{ ...collectionCatalog.tags[0], name: "😀".repeat(81) }]),
    ).toBe(false);
    expect(
      validTags([
        collectionCatalog.tags[0],
        { ...collectionCatalog.tags[1], name: collectionCatalog.tags[0].name },
      ]),
    ).toBe(false);
    expect(validTags([{ ...collectionCatalog.tags[0], status: "other" }])).toBe(
      false,
    );
  });
  it("requires exact resulting sets and preserves the primary topic", () => {
    const r = request();
    const p = collectionPreview(r, [collectionRow()]);
    expect(validPreview(p, r)).toBe(true);
    expect(p.review_rows[0].owner_topic).toEqual({ state: "unclassified" });
    expect(validPreview({ ...p, tag_results: [] }, r)).toBe(false);
    expect(
      validPreview(
        { ...p, tag_results: [{ ...p.tag_results[0], tags: [] }] },
        r,
      ),
    ).toBe(false);
    expect(
      validPreview(
        {
          ...p,
          tag_results: [{ ...p.tag_results[0], observation_revision: "2" }],
        },
        r,
      ),
    ).toBe(false);
  });
  it("rejects stale additive intent but accepts explicit empty replacement", () => {
    const row = collectionRow(0, {
      owner_tags: { state: "stale", tags: [collectionCatalog.tags[0]] },
    });
    const r = request();
    expect(validPreview(collectionPreview(r, [row]), r)).toBe(false);
    const replacement = {
      ...r,
      action: {
        kind: "tags" as const,
        operation: "replace" as const,
        tag_ids: [],
      },
    };
    expect(
      validPreview(collectionPreview(replacement, [row]), replacement),
    ).toBe(true);
    const emptyAdd = {
      ...r,
      action: { kind: "tags" as const, operation: "add" as const, tag_ids: [] },
    };
    expect(
      validPreview(collectionPreview(emptyAdd, [collectionRow()]), emptyAdd),
    ).toBe(false);
  });
  it("preserves historical v1 saved previews without fabricating v2 pins or tags", () => {
    const modern = collectionPreview(
      { ...request(), action: { kind: "uncategorized" } },
      [collectionRow()],
    );
    const { tag_results: _results, ...base } = modern;
    const { tag_revision: _tag, ...snapshot } = base.snapshot;
    const legacy: LegacyPreview = {
      ...base,
      profile: "private-collection-preview-v1",
      snapshot,
      action: { kind: "uncategorized" },
      review_rows: base.review_rows.map(({ owner_tags: _tags, ...row }) => row),
    };
    const command = {
      workspace_id: collectionWorkspace.workspace_id,
      command_id: "saved-old",
      prepare_key: "same-old-key",
      state: "prepared",
      acknowledged: false,
      preview: legacy,
    };
    expect(validCommand(command, collectionWorkspace, collectionCatalog)).toBe(
      true,
    );
    expect(samePreview(legacy, structuredClone(legacy))).toBe(true);
    expect(samePreview(legacy, modern)).toBe(false);
    expect(
      validCommand(
        { ...command, preview: { ...legacy, snapshot: modern.snapshot } },
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(false);
    expect(
      validCommand(
        { ...command, preview: { ...legacy, review_rows: modern.review_rows } },
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(false);
  });
});
