import { describe, expect, it } from "vitest";
import {
  collectionCatalog,
  collectionPage,
  collectionPreview,
  collectionQuery,
  collectionRow,
  collectionSnapshot,
  collectionWorkspace,
  heldCollectionRow,
  nullNumericCollectionRow,
  removedHeldCollectionRow,
} from "../../tests/collection-fixture";
import {
  samePreview,
  selectionKey,
  selectionOf,
  validCommand,
  validPage,
  validPreview,
  type LabelCommand,
} from "./types";

describe("collection safe projection", () => {
  it("accepts a4096-character opaque cursor and rejects4097 without loosening identity bounds", () => {
    const q = collectionQuery();
    const page = {
      ...collectionPage(q),
      matching_count: "2",
      next_cursor: "a".repeat(4096),
    };
    expect(validPage(page, q, collectionWorkspace, collectionCatalog)).toBe(
      true,
    );
    expect(
      validPage(
        { ...page, next_cursor: "a".repeat(4097) },
        q,
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(false);
    expect(
      validPage(
        {
          ...page,
          items: [{ ...page.items[0], observation_id: "b".repeat(2049) }],
        },
        q,
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(false);
  });
  it("rejects a wrong-account row even when the response echoes the requested account selector", () => {
    const q = collectionQuery({ source_account_id: "fictional-checking" });
    const p = collectionPage(q, [collectionRow(1)]);
    expect(validPage(p, q, collectionWorkspace, collectionCatalog)).toBe(false);
  });
  it("preserves a removed observation's prior held/null monetary projection", () => {
    expect(
      validPage(
        collectionPage(collectionQuery(), [removedHeldCollectionRow()]),
        collectionQuery(),
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(true);
  });
  it("keeps available dated currency money valid when the stored numeric code is null", () => {
    const row = nullNumericCollectionRow();
    expect(
      validPage(
        collectionPage(collectionQuery(), [row]),
        collectionQuery(),
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(true);
    expect(row.money.state === "available" && row.money.numeric_code).toBe(
      null,
    );
  });
  it("preserves exact large original quantities and dated numeric currency identity", () => {
    const row = collectionRow();
    if (row.money.state !== "available") throw Error("fixture");
    row.money.amount = "9007199254740993.01";
    expect(
      validPage(
        collectionPage(collectionQuery(), [row]),
        collectionQuery(),
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(true);
    expect(row.money.numeric_code).toBe("840");
    expect(row.money.valid_from).toBe("2000-01-01");
  });
  it.each(["unsupported_currency", "invalid_precision"] as const)(
    "keeps %s holds unavailable rather than fabricated numeric facts",
    (reason) => {
      const row = heldCollectionRow(reason),
        page = collectionPage(collectionQuery(), [row]);
      expect(
        validPage(
          page,
          collectionQuery(),
          collectionWorkspace,
          collectionCatalog,
        ),
      ).toBe(true);
      const invalid = structuredClone(page);
      Object.assign(invalid.items[0].money, { amount: "0", currency: "USD" });
      expect(
        validPage(
          invalid,
          collectionQuery(),
          collectionWorkspace,
          collectionCatalog,
        ),
      ).toBe(false);
      const invented = collectionPage();
      invented.items[0].source_state = "held";
      expect(
        validPage(
          invented,
          collectionQuery(),
          collectionWorkspace,
          collectionCatalog,
        ),
      ).toBe(false);
    },
  );
  it("rejects mismatched workspace, filters, snapshots, duplicate composite identities and excessive rows", () => {
    const edits = [
      (p: ReturnType<typeof collectionPage>) => {
        p.workspace_id = "foreign";
      },
      (p: ReturnType<typeof collectionPage>) => {
        p.selector.search = "another";
      },
      (p: ReturnType<typeof collectionPage>) => {
        p.snapshot.source_snapshot = "unbound";
      },
      (p: ReturnType<typeof collectionPage>) => {
        p.snapshot.authority_revision = "2";
      },
      (p: ReturnType<typeof collectionPage>) => {
        p.items.push(p.items[0]);
      },
      (p: ReturnType<typeof collectionPage>) => {
        p.start_index = "3000";
      },
    ];
    for (const edit of edits) {
      const p = structuredClone(collectionPage());
      edit(p);
      expect(
        validPage(p, collectionQuery(), collectionWorkspace, collectionCatalog),
      ).toBe(false);
    }
    const q = collectionQuery({
      snapshot: { ...collectionSnapshot, source_snapshot: "b".repeat(64) },
    });
    expect(
      validPage(collectionPage(), q, collectionWorkspace, collectionCatalog),
    ).toBe(false);
  });
  it("uses workspace/source/observation composite identity and source revision equality", () => {
    const a = collectionRow(),
      b = { ...a, source_id: "other-source" };
    expect(selectionKey(a)).not.toBe(selectionKey(b));
    const p = collectionPage(collectionQuery(), [a, b]);
    expect(
      validPage(p, collectionQuery(), collectionWorkspace, collectionCatalog),
    ).toBe(true);
  });
  it("validates whole immutable recovered intent, including money, source pins and explicit Uncategorized", () => {
    const row = collectionRow(),
      request = {
        workspace_id: collectionWorkspace.workspace_id,
        snapshot: collectionSnapshot,
        selection: [selectionOf(row)],
        action: { kind: "uncategorized" as const },
      },
      p = collectionPreview(request, [row]);
    expect(validPreview(p, request)).toBe(true);
    const command: LabelCommand = {
      workspace_id: collectionWorkspace.workspace_id,
      command_id: "c",
      prepare_key: "k",
      state: "prepared",
      acknowledged: false,
      preview: p,
    };
    expect(validCommand(command, collectionWorkspace, collectionCatalog)).toBe(
      true,
    );
    expect(
      validCommand(
        { ...command, acknowledged: true },
        collectionWorkspace,
        collectionCatalog,
      ),
    ).toBe(false);
    const changed = structuredClone(p);
    if (changed.review_rows[0].money.state === "available")
      changed.review_rows[0].money.amount = "1.00";
    expect(samePreview(p, changed)).toBe(false);
    expect(validPreview({ ...p, action: null }, request)).toBe(false);
    expect(
      validPreview(
        { ...p, action: { kind: "uncategorized", topic_id: "dining" } },
        request,
      ),
    ).toBe(false);
    expect(
      validPreview(
        { ...p, selection: [], review_rows: [], selected_count: "0" },
        { ...request, selection: [] },
      ),
    ).toBe(false);
    expect(
      validPreview(
        {
          ...p,
          selection: [{ ...request.selection[0], observation_revision: "2" }],
        },
        request,
      ),
    ).toBe(false);
  });
});
