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

it("displays and previews multiline literal condition text without input sanitization", async () => {
  const value = "Fictional\nCafe";
  const preview = vi.fn();
  render(
    <RuleEditor
      catalog={{
        ...collectionCatalog,
        rules: [
          {
            ...rule,
            conditions: [{ field: "description", operator: "contains", value }],
          },
        ],
      }}
      sources={collectionSources}
      disabled={false}
      onPreview={preview}
    />,
  );
  await userEvent.click(screen.getByText("Manage rules", { exact: true }));
  expect(
    (
      screen.getByLabelText("Rule 1 condition 1 value", {
        exact: true,
      }) as HTMLTextAreaElement
    ).value,
  ).toBe(value);
  await userEvent.clear(screen.getByLabelText("Rule 1 name", { exact: true }));
  await userEvent.type(
    screen.getByLabelText("Rule 1 name", { exact: true }),
    "Fictional multiline rule",
  );
  await userEvent.click(
    screen.getByRole("button", { name: /^Preview rule changes$/ }),
  );
  expect(preview.mock.calls[0][0][0].conditions[0].value).toBe(value);
});

it("filters the inventory without losing other draft edits or mounting multiple editors", async () => {
  const preview = vi.fn();
  const second: RuleDefinition = {
    ...rule,
    rule_id: "86000000-0000-4000-8000-000000000002",
    name: "Fictional train rule",
    conditions: [
      { field: "description", operator: "contains", value: "Train" },
    ],
  };
  render(
    <RuleEditor
      catalog={{ ...collectionCatalog, rules: [rule, second] }}
      sources={collectionSources}
      disabled={false}
      onPreview={preview}
    />,
  );
  await userEvent.click(screen.getByText("Manage rules", { exact: true }));
  expect(
    screen.getByRole("table", { name: "Private categorization rules" }),
  ).toBeTruthy();
  await userEvent.clear(screen.getByLabelText("Rule 1 name", { exact: true }));
  await userEvent.type(
    screen.getByLabelText("Rule 1 name", { exact: true }),
    "Updated meal rule",
  );
  await userEvent.type(
    screen.getByRole("searchbox", { name: "Search rules" }),
    "train",
  );
  expect(
    screen.queryByRole("button", { name: "Updated meal rule" }),
  ).toBeNull();
  await userEvent.click(
    screen.getByRole("button", { name: "Fictional train rule" }),
  );
  expect(screen.queryByLabelText("Rule 1 name", { exact: true })).toBeNull();
  expect(screen.getByLabelText("Rule 2 name", { exact: true })).toBeTruthy();
  await userEvent.click(screen.getByRole("button", { name: "Close editor" }));
  expect(screen.queryByLabelText("Rule 2 name", { exact: true })).toBeNull();
  await userEvent.click(
    screen.getByRole("button", { name: "Preview rule changes" }),
  );
  expect(preview).toHaveBeenCalledExactlyOnceWith([
    { ...rule, name: "Updated meal rule" },
    second,
  ]);
});

it("locks table editing and bulk rule actions while saved work owns the workspace", async () => {
  const preview = vi.fn();
  render(
    <RuleEditor
      catalog={{ ...collectionCatalog, rules: [rule] }}
      sources={collectionSources}
      disabled
      onPreview={preview}
    />,
  );
  await userEvent.click(screen.getByText("Manage rules", { exact: true }));
  for (const name of [
    "Import YAML",
    "Export YAML",
    "Add rule",
    "Fictional meal rule",
    "Close editor",
    "Preview rule changes",
  ]) {
    expect(
      (screen.getByRole("button", { name }) as HTMLButtonElement).matches(
        ":disabled",
      ),
    ).toBe(true);
  }
  await userEvent.click(
    screen.getByRole("button", { name: "Fictional meal rule" }),
  );
  expect(preview).not.toHaveBeenCalled();
});
