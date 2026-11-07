import { expect, test } from "@playwright/experimental-ct-react";
import { TagSelection } from "../src/collection/TagEditor";
import { CollectionReviewHarness } from "./CollectionReviewHarness";

test("500-tag mobile selection stays bounded, searchable and keyboard operable", async ({
  mount,
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const tags = Array.from({ length: 500 }, (_, i) => ({
    tag_id: `42000000-0000-4000-8000-${String(i + 1).padStart(12, "0")}`,
    name: `Fictional tag ${i + 1}`,
    status: "active" as const,
  }));
  await mount(
    <TagSelection
      tags={tags}
      count={1}
      stale={false}
      disabled={false}
      onPreview={() => {}}
    />,
  );
  await page.getByText("Change strategic tags", { exact: true }).click();
  const list = page.getByRole("group", { name: "Available strategic tags" });
  expect((await list.boundingBox())!.height).toBeLessThan(280);
  const preview = page.getByRole("button", {
    name: "Preview tags for 1 selected",
  });
  expect((await preview.boundingBox())!.y).toBeLessThan(844);
  await page
    .getByLabel("Find strategic tags", { exact: true })
    .fill("Fictional tag 500");
  await expect(page.getByRole("checkbox")).toHaveCount(1);
  const tag = page.getByRole("checkbox", {
    name: "Fictional tag 500",
    exact: true,
  });
  await tag.focus();
  await page.keyboard.press("Space");
  await expect(tag).toBeChecked();
  await expect(preview).toBeEnabled();
  await page
    .getByLabel("Find strategic tags", { exact: true })
    .fill("unmatched");
  await expect(
    page.getByText("1 tags selected.", { exact: true }),
  ).toBeVisible();
  await expect(preview).toBeEnabled();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("tag intent is saved, confirmed and acknowledged independently from the primary topic", async ({
  mount,
  page,
}) => {
  await mount(<CollectionReviewHarness />);
  await page.getByRole("checkbox").first().check();
  await page.getByText("Change strategic tags", { exact: true }).click();
  await page.getByRole("checkbox", { name: "Travel", exact: true }).check();
  await page
    .getByRole("button", { name: "Preview tags for 1 selected" })
    .click();
  await expect(
    page.getByRole("heading", {
      name: "Review tags for 1 selected",
      exact: true,
    }),
  ).toBeFocused();
  await expect(
    page.getByRole("list", { name: "Resulting strategic tags" }),
  ).toContainText("Travel");
  await page
    .getByRole("button", { name: "Save tag review", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Saved tag review", exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Fixture executions")).toHaveText("0");
  await page
    .getByRole("button", { name: "Confirm tags for 1 selected", exact: true })
    .click();
  await expect(
    page.getByText("Tags applied. No journal was created.", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Fixture acknowledgements")).toHaveText("0");
  await page
    .getByRole("button", { name: "Acknowledge result", exact: true })
    .click();
  await expect(page.getByLabel("Fixture acknowledgements")).toHaveText("1");
});
