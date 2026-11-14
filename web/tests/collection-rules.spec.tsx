import { expect, test } from "@playwright/experimental-ct-react";
import { CollectionRulesHarness } from "./CollectionRulesHarness";

for (const width of [390, 1440])
  test(`rule conflicts and exact saved application recover without implicit writes at ${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 });
    await mount(<CollectionRulesHarness />);
    const first = page
      .getByRole("list", { name: "Transaction rows" })
      .getByRole("listitem")
      .first();
    await first.getByText("Labels and provenance", { exact: true }).click();
    await expect(
      first.getByText(/Saved rule application needs confirmation; ruleset 3/),
    ).toBeVisible();
    await expect(first.getByText(/Equal-rank rules disagree/)).toBeVisible();
    await expect(page.getByLabel("Fictional rule executions")).toHaveText("0");
    const checks = page.getByRole("checkbox");
    await checks.nth(0).check();
    await checks.nth(1).check();
    await page
      .getByRole("button", {
        name: "Preview rules for 2 selected",
        exact: true,
      })
      .click();
    await expect(
      page.getByRole("heading", {
        name: "Review rules for 2 selected",
        exact: true,
      }),
    ).toBeFocused();
    await expect(
      page.getByText(
        "Eligible reviewed results: 1. Unchanged selected results: 1.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(
      page.getByRole("list", { name: "Reviewed rule results" }),
    ).toContainText("conflict");
    await expect(
      page.getByRole("list", { name: "Reviewed rule results" }),
    ).toContainText("Preserved manual fields: primary topic.");
    await page
      .getByRole("button", { name: "Save rule application", exact: true })
      .click();
    await expect(
      page.getByText(/saved review outcome is unknown/),
    ).toBeVisible();
    await expect(page.getByLabel("Fictional rule executions")).toHaveText("0");
    await page
      .getByRole("button", { name: "Retry saving same review", exact: true })
      .click();
    await expect(
      page.getByRole("heading", {
        name: "Saved rule application",
        exact: true,
      }),
    ).toBeFocused();
    const prepares = JSON.parse(
      (await page.getByLabel("Fictional rule prepares").textContent()) ?? "[]",
    );
    expect(prepares).toHaveLength(2);
    expect(prepares[0]).toEqual(prepares[1]);
    await page
      .getByRole("button", {
        name: "Confirm rules for 2 selected",
        exact: true,
      })
      .click();
    await expect(
      page.getByText("Rule results applied. No journal was created.", {
        exact: true,
      }),
    ).toBeVisible();
    await expect(page.getByLabel("Fictional rule acknowledgements")).toHaveText(
      "0",
    );
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath("fictional-rule-conflict-result.png"),
    });
    await page
      .getByRole("button", { name: "Acknowledge result", exact: true })
      .click();
    await expect(page.getByLabel("Fictional rule acknowledgements")).toHaveText(
      "1",
    );
    expect(
      await page.evaluate(() => [localStorage.length, sessionStorage.length]),
    ).toEqual([0, 0]);
  });

test("rule catalog editing previews complete intent before save and explicit confirmation", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mount(<CollectionRulesHarness scenario="catalog" />);
  await page.getByText("Manage rules", { exact: true }).click();
  await page.getByRole("button", { name: "Add rule", exact: true }).click();
  await page
    .getByLabel("Rule 1 name", { exact: true })
    .fill("Fictional meal strategy");
  await page
    .getByLabel("Rule 1 condition 1 value", { exact: true })
    .fill("Fictional Cafe");
  await page
    .getByLabel("Rule 1 primary topic", { exact: true })
    .selectOption("uncategorized");
  await page
    .getByRole("button", { name: "Preview rule changes", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Review rule changes", exact: true }),
  ).toBeFocused();
  await expect(page.getByLabel("Fictional rule executions")).toHaveText("0");
  await page
    .getByRole("button", { name: "Save rule review", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Saved rule review", exact: true }),
  ).toBeFocused();
  await page
    .getByRole("button", { name: "Confirm rule changes", exact: true })
    .click();
  await expect(
    page.getByText("Rules updated. No journal was created.", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Fictional rule acknowledgements")).toHaveText(
    "0",
  );
});

test("lost acknowledgement opens an inert correction seed only after its exact confirmed retry", async ({
  mount,
  page,
}) => {
  await mount(<CollectionRulesHarness scenario="seed" />);
  await page.getByRole("checkbox").first().check();
  await page
    .getByLabel("Label selected transactions", { exact: true })
    .selectOption("uncategorized");
  await page
    .getByRole("button", { name: "Preview labels for 1 selected", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Save label review", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Confirm labels for 1 selected", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Acknowledge and create rule", exact: true })
    .click();
  await expect(page.getByText(/saved review outcome is unknown/)).toBeVisible();
  await expect(
    page.getByRole("heading", {
      name: "Rules",
      exact: true,
    }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Retry same acknowledgement", exact: true })
    .click();
  await expect(
    page.getByLabel("Rule 1 condition 1 value", { exact: true }),
  ).toHaveValue("Fictional Cafe");
  await expect(page.getByLabel("Rule 1 name", { exact: true })).toHaveValue("");
  await expect(page.getByLabel("Fictional rule executions")).toHaveText("1");
  await expect(page.getByLabel("Fictional rule acknowledgements")).toHaveText(
    "1",
  );
  await expect(
    page.getByRole("heading", { name: "Review rule changes", exact: true }),
  ).toHaveCount(0);
});
