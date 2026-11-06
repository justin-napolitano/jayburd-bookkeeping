import { expect, test } from "@playwright/experimental-ct-react";
import { CollectionReviewHarness } from "./CollectionReviewHarness";

test("server pagination clears selection and keeps whole-result count", async ({
  mount,
  page,
}) => {
  await mount(<CollectionReviewHarness />);
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await expect(page.getByRole("checkbox")).toHaveCount(25);
  await page.getByRole("checkbox").first().check();
  await page.getByRole("button", { name: "Next page" }).click();
  await expect(
    page.getByText("26–50 of 3000 matching transactions"),
  ).toBeVisible();
  await expect(page.getByText("0 selected on this page")).toBeVisible();
  await expect(
    page.getByText("Fictional purchase 1", { exact: true }),
  ).toHaveCount(0);
});

test("390px keyboard review has no horizontal clipping and keeps held money unavailable", async ({
  mount,
  page,
}, info) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mount(<CollectionReviewHarness held />);
  await expect(
    page.getByText("Money unavailable", { exact: true }),
  ).toBeVisible();
  const firstRow = page.getByText("Fictional purchase 1", { exact: true });
  await page.screenshot({
    path: info.outputPath("fictional-collection-initial.png"),
    fullPage: true,
  });
  expect((await firstRow.boundingBox())!.y).toBeLessThan(844);
  await info.attach("default-font-first-row-y", {
    body: String((await firstRow.boundingBox())!.y),
    contentType: "text/plain",
  });
  const held = page
    .getByRole("listitem")
    .filter({ hasText: "Fictional unsupported_currency hold" });
  await expect(held).not.toContainText("0.00 USD");
  const first = page.getByRole("checkbox").first();
  await first.focus();
  await page.keyboard.press("Space");
  await expect(first).toBeChecked();
  const choice = page.getByLabel("Label selected transactions");
  await expect(choice).toBeEnabled();
  await choice.focus();
  await expect(choice).toBeFocused();
  await page.keyboard.press("e");
  await page.keyboard.press("Tab");
  await expect(choice).toHaveValue("uncategorized");
  const preview = page.getByRole("button", {
    name: "Preview labels for 1 selected",
  });
  await preview.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Review 1 selected transactions" }),
  ).toBeFocused();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  expect(
    await page.evaluate(() => [localStorage.length, sessionStorage.length]),
  ).toEqual([0, 0]);
  await page.screenshot({
    path: info.outputPath("fictional-collection-mobile.png"),
    fullPage: true,
  });
});

test("390px larger font metrics retain the first row and usable controls", async ({
  mount,
  page,
}, info) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mount(<CollectionReviewHarness held />);
  // Stress wrapping and line boxes beyond the default local system font.
  // The original default-font fold/keyboard test remains independent above.
  await page.addStyleTag({
    content:
      "html { font-size: 17px; } body { font-family: Arial, sans-serif; line-height: 1.7; }",
  });
  const firstRow = page.getByText("Fictional purchase 1", { exact: true });
  await expect(firstRow).toBeVisible();
  const firstRowY = (await firstRow.boundingBox())!.y;
  expect(firstRowY).toBeLessThan(844);
  for (const control of [
    page.getByRole("button", { name: "Refresh transactions" }),
    page.getByRole("button", { name: "Apply filters" }),
    page.getByRole("combobox", { name: "Account", exact: true }),
    page.getByRole("searchbox", { name: "Search transactions" }),
    page.getByText("Book, topic and date filters", { exact: true }),
  ]) {
    const bounds = (await control.boundingBox())!;
    expect(bounds.height).toBeGreaterThanOrEqual(44);
    expect(bounds.width).toBeGreaterThanOrEqual(44);
  }
  expect(
    await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
  ).toBe(true);
  await info.attach("larger-font-first-row-y", {
    body: String(firstRowY),
    contentType: "text/plain",
  });
  await page.screenshot({
    path: info.outputPath("fictional-collection-larger-font.png"),
    fullPage: true,
  });
});

test("lost prepare reply recovers the saved intent without executing or acknowledging", async ({
  mount,
  page,
}) => {
  await mount(<CollectionReviewHarness scenario="lost_prepare" />);
  await page.getByRole("checkbox").first().check();
  await page
    .getByLabel("Label selected transactions")
    .selectOption("topic:dining");
  await page
    .getByRole("button", { name: "Preview labels for 1 selected" })
    .click();
  await page.getByRole("button", { name: "Save label review" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "saved review outcome is unknown",
  );
  await page.getByRole("button", { name: "Recover saved review" }).click();
  await expect(
    page.getByText("This saved review has not been applied."),
  ).toBeVisible();
  await expect(page.getByLabel("Fixture executions")).toHaveText("0");
  await expect(page.getByLabel("Fixture acknowledgements")).toHaveText("0");
  await page
    .getByRole("button", { name: "Confirm labels for 1 selected" })
    .click();
  await expect(
    page.getByText("Labels applied. No journal was created."),
  ).toBeVisible();
  await expect(page.getByLabel("Fixture executions")).toHaveText("1");
  await expect(page.getByLabel("Fixture acknowledgements")).toHaveText("0");
});

test("saved command discovery and cancellation both require explicit acknowledgement", async ({
  mount,
  page,
}) => {
  await mount(<CollectionReviewHarness scenario="pending" />);
  await expect(
    page.getByText("This saved review has not been applied."),
  ).toBeVisible();
  await expect(page.getByLabel("Fixture executions")).toHaveText("0");
  await page.getByRole("button", { name: "Cancel saved review" }).click();
  await expect(page.getByText("Saved label review cancelled.")).toBeVisible();
  await expect(page.getByLabel("Fixture acknowledgements")).toHaveText("0");
  await page.getByRole("button", { name: "Acknowledge result" }).click();
  await expect(
    page.getByText("1–25 of 3000 matching transactions"),
  ).toBeVisible();
  await expect(page.getByLabel("Fixture acknowledgements")).toHaveText("1");
});
