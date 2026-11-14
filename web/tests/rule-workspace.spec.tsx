import { expect, test } from "@playwright/experimental-ct-react";
import { RuleWorkspaceHarness } from "./RuleWorkspaceHarness";

const yaml = `version: 1
rules:
  - id: 86000000-0000-4000-8000-000000000001
    name: Fictional cafe meals
    conditions:
      - field: merchant
        operator: exact
        value: Fictional Cafe
    topic:
      name: Dining
  - id: 86000000-0000-4000-8000-000000000002
    name: Fictional lunch counter
    conditions:
      - field: description
        operator: contains
        value: Fictional Lunch
    topic:
      id: d1000000-0000-4000-8000-000000000001
`;

for (const width of [1280, 390]) {
  test(`YAML bulk draft stays inert and rule inventory stays bounded at ${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    await mount(<RuleWorkspaceHarness />);
    await page.getByText("Manage rules", { exact: true }).click();
    await page
      .getByRole("button", { name: "Import YAML", exact: true })
      .click();
    await page.getByLabel("Rule pack YAML", { exact: true }).fill(yaml);
    await page
      .getByRole("button", { name: "Validate YAML", exact: true })
      .click();
    await expect(
      page.getByText("2 added · 0 updated · 0 existing rules preserved", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(page.getByLabel("Synthetic rule preview")).toHaveText(
      "No preview",
    );
    await page
      .getByRole("button", { name: "Add to draft", exact: true })
      .click();
    const table = page.getByRole("table", {
      name: "Private categorization rules",
    });
    await expect(table.getByRole("row")).toHaveCount(3);
    await expect(page.getByLabel("Synthetic rule preview")).toHaveText(
      "No preview",
    );
    await page
      .getByRole("button", { name: "Fictional cafe meals", exact: true })
      .click();
    await expect(page.getByLabel("Rule 1 name", { exact: true })).toBeVisible();
    await page
      .getByLabel("Rule 1 name", { exact: true })
      .fill("Fictional cafe lunch");
    await page
      .getByRole("searchbox", { name: "Search rules", exact: true })
      .fill("counter");
    await expect(table.getByRole("row")).toHaveCount(2);
    await page
      .getByRole("button", { name: "Fictional lunch counter", exact: true })
      .click();
    await expect(page.getByLabel("Rule 1 name", { exact: true })).toHaveCount(
      0,
    );
    await expect(page.getByLabel("Rule 2 name", { exact: true })).toBeVisible();
    await page.screenshot({
      path: info.outputPath(`rule-workspace-editor-${width}.png`),
      fullPage: true,
    });
    await page
      .getByRole("button", { name: "Close editor", exact: true })
      .click();
    await page
      .getByRole("searchbox", { name: "Search rules", exact: true })
      .clear();
    await expect(table.getByRole("row")).toHaveCount(3);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath(`rule-workspace-${width}.png`),
      fullPage: true,
    });
    await page
      .getByRole("button", { name: "Preview rule changes", exact: true })
      .click();
    const preview = JSON.parse(
      (await page.getByLabel("Synthetic rule preview").textContent()) ?? "null",
    );
    expect(preview).toHaveLength(2);
    expect(preview.map((rule: { name: string }) => rule.name)).toEqual([
      "Fictional cafe lunch",
      "Fictional lunch counter",
    ]);
    expect(
      preview.every((rule: { rule_id: string }) =>
        /^[a-f0-9-]{36}$/.test(rule.rule_id),
      ),
    ).toBe(true);
  });
}
