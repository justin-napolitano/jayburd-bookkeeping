import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { collectionCatalog } from "../../tests/collection-fixture";
import { RulePackTools } from "./RulePackTools";
import type { RuleDefinition } from "./rules";
import { exportRulesYaml } from "./ruleYaml";

afterEach(cleanup);
const topic = "f1000000-0000-4000-8000-000000000001";
const catalog = {
  ...collectionCatalog,
  topics: [
    {
      topic_id: topic,
      name: "Food and drink",
      parent_id: null,
      status: "active" as const,
    },
  ],
};
const pack = `version: 1
rules:
  - name: Fictional cafe
    conditions:
      - field: merchant
        operator: exact
        value: Fictional Cafe
    topic: {name: Food and drink}
`;
const existing: RuleDefinition = {
  rule_id: "f2000000-0000-4000-8000-000000000001",
  name: "Unsaved existing rule",
  status: "active",
  source_account_id: null,
  priority: 0,
  conditions: [
    { field: "description", operator: "exact", value: "Fictional rail" },
  ],
  output: { kind: "uncategorized", tag_ids: [] },
};
describe("local YAML rule pack workflow", () => {
  it("treats an unchanged export as a no-op even with visual-editor property order", async () => {
    const { priority, ...rest } = existing;
    const reordered = { ...rest, priority };
    const onImport = vi.fn();
    render(
      <RulePackTools
        rules={[reordered]}
        catalog={catalog}
        sources={[]}
        disabled={false}
        onImport={onImport}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "Import YAML" }));
    await userEvent.click(screen.getByLabelText("Rule pack YAML"));
    await userEvent.paste(exportRulesYaml([existing], catalog, []));
    await userEvent.click(
      screen.getByRole("button", { name: "Validate YAML" }),
    );
    expect(
      screen.getByText("0 added · 0 updated · 1 existing rules preserved"),
    ).toBeTruthy();
    expect(
      (
        screen.getByRole("button", {
          name: "Add to draft",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(onImport).not.toHaveBeenCalled();
  });
  it("validates without saving and preserves current unsaved draft rules on explicit add", async () => {
    const onImport = vi.fn();
    render(
      <RulePackTools
        rules={[existing]}
        catalog={catalog}
        sources={[]}
        disabled={false}
        onImport={onImport}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "Import YAML" }));
    // Paste preserves literal YAML braces.
    await userEvent.click(screen.getByLabelText("Rule pack YAML"));
    await userEvent.paste(pack);
    await userEvent.click(
      screen.getByRole("button", { name: "Validate YAML" }),
    );
    expect(onImport).not.toHaveBeenCalled();
    expect(
      screen.getByText("1 added · 0 updated · 1 existing rules preserved"),
    ).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Add to draft" }));
    expect(onImport).toHaveBeenCalledTimes(1);
    const result = onImport.mock.calls[0][0] as RuleDefinition[];
    expect(result).toHaveLength(2);
    expect(result.find((rule) => rule.rule_id === existing.rule_id)).toEqual(
      existing,
    );
    expect(
      result.find((rule) => rule.name === "Fictional cafe")?.output,
    ).toEqual({ kind: "topic", topic_id: topic, tag_ids: [] });
    expect(screen.getByRole("status").textContent).toContain(
      "Review rule changes to save it",
    );
  });
  it("invalidates a validated pack when drafts change or access locks", async () => {
    const onImport = vi.fn();
    const view = render(
      <RulePackTools
        rules={[]}
        catalog={catalog}
        sources={[]}
        disabled={false}
        onImport={onImport}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "Import YAML" }));
    await userEvent.click(screen.getByLabelText("Rule pack YAML"));
    await userEvent.paste(pack);
    await userEvent.click(
      screen.getByRole("button", { name: "Validate YAML" }),
    );
    view.rerender(
      <RulePackTools
        rules={[existing]}
        catalog={catalog}
        sources={[]}
        disabled={false}
        onImport={onImport}
      />,
    );
    expect(screen.queryByRole("button", { name: "Add to draft" })).toBeNull();
    view.rerender(
      <RulePackTools
        rules={[existing]}
        catalog={catalog}
        sources={[]}
        disabled={true}
        onImport={onImport}
      />,
    );
    expect(
      (
        screen.getByRole("button", {
          name: "Validate YAML",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);
    expect(onImport).not.toHaveBeenCalled();
  });
  it("rejects oversized files before reading and ignores a read completed after a lock", async () => {
    const onImport = vi.fn();
    const view = render(
      <RulePackTools
        rules={[]}
        catalog={catalog}
        sources={[]}
        disabled={false}
        onImport={onImport}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "Import YAML" }));
    const large = new File([" ".repeat(262145)], "rules.yaml", {
      type: "application/yaml",
    });
    await userEvent.upload(screen.getByLabelText("YAML file"), large);
    expect(screen.getByRole("alert").textContent).toContain("256 KiB");
    let resolve!: (value: string) => void;
    const slow = new File([pack], "rules.yaml", { type: "application/yaml" });
    Object.defineProperty(slow, "text", {
      value: () =>
        new Promise<string>((done) => {
          resolve = done;
        }),
    });
    await userEvent.upload(screen.getByLabelText("YAML file"), slow);
    view.rerender(
      <RulePackTools
        rules={[]}
        catalog={catalog}
        sources={[]}
        disabled={true}
        onImport={onImport}
      />,
    );
    resolve(pack);
    await Promise.resolve();
    expect(
      (screen.getByLabelText("Rule pack YAML") as HTMLTextAreaElement).value,
    ).toBe("");
    expect(onImport).not.toHaveBeenCalled();
  });
});
