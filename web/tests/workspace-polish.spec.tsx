import { test, expect } from "@playwright/experimental-ct-react";
import type { Page } from "playwright-core";
import { ReportsHarness } from "./ReportsHarness";
import { Harness } from "./Harness";
import { positionWorkspace, positionReport } from "./position-fixture";

// Only fictional pending/read outcomes drive presentation; no command is automatically executed.
async function servePolish(
  page: Page,
  pending: "idle" | "prepared" | "denied" = "idle",
) {
  const writes: string[] = [];
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  await page.route("**/v1/**", async (route) => {
    const url = new URL(route.request().url());
    if (route.request().method() !== "GET") writes.push(url.pathname);
    if (url.pathname === "/v1/workspace")
      return route.fulfill({ json: positionWorkspace });
    if (url.pathname === "/v1/commands/pending") {
      if (pending === "denied")
        return route.fulfill({ status: 404, json: { code: "not_found" } });
      return route.fulfill({
        json: {
          command:
            pending === "idle"
              ? null
              : {
                  command_id: "fictional-prepared-import",
                  operation: "import",
                  entity_id: "studio",
                  book_id: "studio-operating",
                  state: "prepared",
                  acknowledged: false,
                  result: null,
                  created_at: "2026-10-21T14:00:00Z",
                  body: {
                    batch_id: "fictional-batch",
                    preview_id: "fictional-preview",
                    preview_revision: "1",
                    binding_revision: "1",
                    source_revision: "0",
                    profile: "jayburd-synthetic-statement-csv/1",
                  },
                },
        },
      });
    }
    if (url.pathname.endsWith("/reports/balance-sheet"))
      return route.fulfill({
        json: positionReport(
          url.pathname.split("/")[5],
          url.searchParams.get("through_date")!,
          url.searchParams.get("compare_through_date")!,
        ),
      });
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  return writes;
}

test("report comparison is an inline keyboard target and initial state explains explicit reading", async ({
  mount,
  page,
}, info) => {
  await mount(<ReportsHarness />);
  await expect(
    page.getByText(/Choose a report and dates, then select Run report/),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Run report", exact: true }),
  ).toBeVisible();
  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 844 });
    const label = page.locator("label.check-option");
    const checkbox = page.getByRole("checkbox", {
      name: "Compare another period",
      exact: true,
    });
    const labelBox = await label.boundingBox();
    const box = await checkbox.boundingBox();
    expect(box!.width).toBeLessThan(28);
    expect(box!.height).toBeLessThan(28);
    expect(labelBox!.height).toBeGreaterThanOrEqual(44);
    expect(
      Math.abs(box!.y + box!.height / 2 - (labelBox!.y + labelBox!.height / 2)),
    ).toBeLessThan(3);
    await label.click();
    await expect(checkbox).toBeChecked();
    await checkbox.focus();
    await page.keyboard.press("Space");
    await expect(checkbox).not.toBeChecked();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath(`report-controls-${width}.png`),
      fullPage: true,
    });
  }
});

test("only idle recovery moves to Settings while its read controls remain available", async ({
  mount,
  page,
}, info) => {
  const writes = await servePolish(page);
  await mount(<Harness />);
  await expect(
    page.getByRole("button", { name: "Transactions", exact: true }),
  ).toBeEnabled();
  await expect(
    page.getByRole("button", { name: "Check saved command" }),
  ).toBeHidden();
  await page.getByText("Settings", { exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Check saved command" }),
  ).toBeVisible();
  await expect(page.getByText("Book setup", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Check saved command" }).click();
  await expect(
    page.getByRole("button", { name: "Transactions", exact: true }),
  ).toBeEnabled();
  await page.getByText("Settings", { exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Check saved command" }),
  ).toBeHidden();
  await page.screenshot({
    path: info.outputPath("idle-settings-collapsed.png"),
    fullPage: true,
  });
  expect(writes).toEqual([]);
});

for (const pending of ["prepared", "denied"] as const) {
  test(`${pending} saved work remains visible outside Settings and never unlocks navigation`, async ({
    mount,
    page,
  }) => {
    const writes = await servePolish(page, pending);
    await mount(<Harness />);
    const attention = page.getByRole("region", {
      name: "Saved command",
      exact: true,
    });
    await expect(attention).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Transactions", exact: true }),
    ).toBeDisabled();
    await expect(
      page.getByRole("button", { name: "Check saved command" }),
    ).toBeVisible();
    if (pending === "prepared")
      await expect(
        page.getByText("Saved for review.", { exact: false }),
      ).toBeVisible();
    else
      await expect(
        page.getByRole("button", { name: "Clear inaccessible saved command" }),
      ).toBeVisible();
    expect(writes).toEqual([]);
  });
}
