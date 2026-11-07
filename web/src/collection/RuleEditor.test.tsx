import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  collectionCatalog,
  collectionSources,
} from "../../tests/collection-fixture";
import { RuleEditor } from "./RuleEditor";
import type { RuleDefinition } from "./rules";
afterEach(cleanup);
const rule: RuleDefinition = {
  rule_id: "86000000-0000-4000-8000-000000000001",
  name: "Fictional meal rule",
  status: "active",
  source_account_id: null,
  priority: 0,
  conditions: [
    { field: "merchant", operator: "exact", value: "Fictional Cafe" },
  ],
  output: { kind: "uncategorized", tag_ids: [] },
};
describe("explicit rule edition editor", () => {
  it("previews complete stable IDs and literal conditions without running a browser matcher", async () => {
    const preview = vi.fn();
    render(
      <RuleEditor
        catalog={{ ...collectionCatalog, rules: [rule] }}
        sources={collectionSources}
        disabled={false}
        onPreview={preview}
      />,
    );
    await userEvent.click(screen.getByText("Manage rules", { exact: true }));
    await userEvent.clear(
      screen.getByLabelText("Rule 1 name", { exact: true }),
    );
    await userEvent.type(
      screen.getByLabelText("Rule 1 name", { exact: true }),
      "Changed meal rule",
    );
    await userEvent.selectOptions(
      screen.getByLabelText("Rule 1 condition 1 operator", { exact: true }),
      "regex",
    );
    await userEvent.clear(
      screen.getByLabelText("Rule 1 condition 1 value", { exact: true }),
    );
    await userEvent.type(
      screen.getByLabelText("Rule 1 condition 1 value", { exact: true }),
      "(",
    );
    expect(preview).not.toHaveBeenCalled();
    await userEvent.click(
      screen.getByRole("button", { name: /^Preview rule changes$/ }),
    );
    expect(preview).toHaveBeenCalledExactlyOnceWith([
      {
        ...rule,
        name: "Changed meal rule",
        conditions: [{ field: "merchant", operator: "regex", value: "(" }],
      },
    ]);
  });
  it("retains retired history and rejects excessive UTF8 condition bytes before preview", async () => {
    const preview = vi.fn();
    render(
      <RuleEditor
        catalog={{
          ...collectionCatalog,
          rules: [
            rule,
            {
              ...rule,
              rule_id: "86000000-0000-4000-8000-000000000002",
              name: "Retired meal rule",
              status: "retired",
            },
          ],
        }}
        sources={collectionSources}
        disabled={false}
        onPreview={preview}
      />,
    );
    await userEvent.click(screen.getByText("Manage rules", { exact: true }));
    await userEvent.clear(
      screen.getByLabelText("Rule 1 condition 1 value", { exact: true }),
    );
    await userEvent.click(
      screen.getByLabelText("Rule 1 condition 1 value", { exact: true }),
    );
    await userEvent.paste("😀".repeat(33));
    await userEvent.click(
      screen.getByRole("button", { name: /^Preview rule changes$/ }),
    );
    expect(screen.getByRole("alert").textContent).toContain("128 UTF-8 bytes");
    expect(preview).not.toHaveBeenCalled();
    await userEvent.click(
      screen.getByText("Retired rule history (1)", { exact: true }),
    );
    expect(
      screen.getByText("Retired meal rule · retired", { exact: true }),
    ).toBeTruthy();
    expect(
      screen.queryByLabelText("Rule 2 status", { exact: true }),
    ).toBeNull();
  });
});

it("mounts one bounded rule tag editor instead of100 hidden500-tag checklists", async () => {
  const id = (n: number) =>
    `89000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
  const rules = Array.from({ length: 100 }, (_, i) => ({
    ...rule,
    rule_id: id(i + 1),
    name: "Fictional rule " + (i + 1),
  }));
  const tags = Array.from({ length: 500 }, (_, i) => ({
    tag_id: id(i + 101),
    name: "Fictional tag " + (i + 1),
    status: "active" as const,
  }));
  const view = render(
    <RuleEditor
      catalog={{ ...collectionCatalog, rules, tags }}
      sources={collectionSources}
      disabled={false}
      onPreview={() => {}}
    />,
  );
  expect(view.container.querySelectorAll("input[type=checkbox]")).toHaveLength(
    0,
  );
  await userEvent.click(screen.getByText("Manage rules", { exact: true }));
  expect(view.container.querySelectorAll("input[type=checkbox]")).toHaveLength(
    500,
  );
});
