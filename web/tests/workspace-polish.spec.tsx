import { financeNavItem } from "./finance-navigation";
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
  const reads: string[] = [];
  await page.clock.install({ time: new Date("2026-10-21T15:00:00Z") });
  await page.route("**/v1/**", async (route) => {
    const url = new URL(route.request().url());
    if (route.request().method() !== "GET") writes.push(url.pathname);
    else reads.push(url.pathname);
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
  return { writes, reads };
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

test("idle discovery leaves no debug inventory and sends no mutations", async ({
  mount,
  page,
}, info) => {
  const { writes, reads } = await servePolish(page);
  await mount(<Harness />);
  await expect(financeNavItem(page, "Review accounting")).toBeEnabled();
  await expect(
    page.getByRole("region", { name: "Saved work", exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Check saved work", exact: true }),
  ).toHaveCount(0);
  expect(reads).toContain("/v1/commands/pending");
  expect(writes).toEqual([]);
  await page.screenshot({
    path: info.outputPath("idle-customer-workspace.png"),
    fullPage: true,
  });
});

for (const pending of ["prepared", "denied"] as const) {
  test(`${pending} saved work remains contextual and never unlocks navigation`, async ({
    mount,
    page,
  }, info) => {
    const { writes } = await servePolish(page, pending);
    await mount(<Harness />);
    const attention = page.getByRole("region", {
      name: "Saved work",
      exact: true,
    });
    await expect(attention).toBeVisible();
    await expect(financeNavItem(page, "Review accounting")).toBeDisabled();
    await expect(
      page.getByRole("button", { name: "Check saved work" }),
    ).toBeVisible();
    if (pending === "prepared") {
      await expect(
        attention.getByText(/Last checked: ready for your confirmation/),
      ).toBeVisible();
      await expect(
        attention.getByRole("button", {
          name: "Cancel saved review",
          exact: true,
        }),
      ).toBeEnabled();
      await expect(
        page.getByText("fictional-prepared-import", { exact: true }),
      ).toHaveCount(0);
      await expect(attention.locator("pre")).toHaveCount(0);
    } else {
      await expect(
        page.getByRole("button", { name: "Clear unavailable saved work" }),
      ).toBeVisible();
      await expect(
        attention.getByText("Checking for unfinished work…", { exact: true }),
      ).toHaveCount(0);
    }
    for (const width of [390, 768, 1024, 1280, 1440]) {
      await page.setViewportSize({ width, height: 900 });
      const exit = attention.getByRole("button", {
        name:
          pending === "prepared"
            ? "Cancel saved review"
            : "Clear unavailable saved work",
        exact: true,
      });
      await expect(exit).toBeVisible();
      const box = (await exit.boundingBox())!;
      expect(box.x).toBeGreaterThanOrEqual(0);
      expect(box.x + box.width).toBeLessThanOrEqual(width);
      expect(box.height).toBeGreaterThanOrEqual(44);
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      await page.screenshot({
        path: info.outputPath(`saved-work-${pending}-${width}.png`),
        fullPage: true,
      });
    }
    expect(writes).toEqual([]);
  });
}
