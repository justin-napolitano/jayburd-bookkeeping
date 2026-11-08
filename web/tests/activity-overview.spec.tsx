import { financeNavItem } from "./finance-navigation";
import { test, expect } from "@playwright/experimental-ct-react";
import { ActivityOverviewHarness } from "./ActivityOverviewHarness";
import {
  activitySources,
  activityPage,
  activityEntity,
  activityCurrencies,
} from "./activity-fixture";
import { Harness } from "./Harness";

for (const width of [390, 1440])
  test(`exact Overview drills and usable layout at ${width}px`, async ({
    page,
    mount,
  }, testInfo) => {
    await page.setViewportSize({ width, height: 900 });
    const calls: { method: string; url: string; body: string | null }[] = [];
    await page.route("**/v1/**", async (route) => {
      const r = route.request();
      calls.push({ method: r.method(), url: r.url(), body: r.postData() });
      if (new URL(r.url()).pathname.endsWith("/sources"))
        return route.fulfill({ json: activitySources });
      if (new URL(r.url()).pathname.endsWith("/source-activity/query"))
        return route.fulfill({ json: activityPage(r.postDataJSON()) });
      return route.fulfill({ status: 404, json: { code: "not_found" } });
    });
    await mount(<ActivityOverviewHarness />);
    await expect(
      page.getByText("9,007,199,254,740,993.01", { exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: /View USD .* merchants/ }).click();
    await expect(
      page.getByText("Fictional merchant & cafe", { exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "View transactions", exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: "Fictional merchant & cafe" }),
    ).toBeFocused();
    await expect(page.locator(".activity-exact-amount")).toHaveText(
      "24.01 USD",
    );
    await page.getByRole("button", { name: "Next page" }).click();
    await expect(
      page.getByRole("cell", { name: "Fictional payment" }),
    ).toBeVisible();
    const final = JSON.parse(calls.at(-1)!.body!);
    expect(final.merchant).toEqual({
      status: "attributed",
      label: "Fictional merchant & cafe",
    });
    expect(final.snapshot.filter_digest).toBe("fictional-pin");
    expect(final.cursor).toBe("fictional-continuation");
    await page.screenshot({
      path: testInfo.outputPath(`overview-${width}.png`),
      fullPage: true,
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    expect(
      calls.every(
        (c) => c.method === "GET" || c.url.endsWith("/source-activity/query"),
      ),
    ).toBe(true);
    expect(
      await page.evaluate(
        () =>
          Object.keys(localStorage).length + Object.keys(sessionStorage).length,
      ),
    ).toBe(0);
    expect(new URL(page.url()).search).toBe("");
    await page.getByRole("button", { name: "Back to merchants" }).click();
    await expect(
      page.getByRole("heading", { name: "USD merchant activity" }),
    ).toBeFocused();
    await page.getByRole("button", { name: "Close details" }).click();
    await expect(
      page.getByRole("heading", { name: "Money in and out" }),
    ).toBeFocused();
    await page.addStyleTag({ content: ":root {font-size:200%;}" });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });

test("Overview is the daily default and hands the same source to Transactions", async ({
  page,
  mount,
}) => {
  await page.route("**/v1/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/v1/workspace")
      return route.fulfill({
        json: {
          entities: [activityEntity],
          currencies: activityCurrencies,
          development_policy: "dev-accrual-usd-v1",
        },
      });
    if (path === "/v1/commands/pending")
      return route.fulfill({ json: { command: null } });
    if (path.endsWith("/sources"))
      return route.fulfill({ json: activitySources });
    if (path.endsWith("/source-activity/query"))
      return route.fulfill({
        json: activityPage(route.request().postDataJSON()),
      });
    if (path.endsWith("/review-items"))
      return route.fulfill({
        json: {
          source_account_id: "s",
          source_revision: "4",
          sort: "date",
          order: "desc",
          matching_count: "0",
          start_index: "0",
          items: [],
          next_cursor: null,
        },
      });
    return route.fulfill({ status: 404, json: { code: "not_found" } });
  });
  await mount(<Harness />);
  await expect(
    page.getByRole("heading", { name: "Overview", exact: true }),
  ).toBeVisible();
  await expect(financeNavItem(page, "Overview")).toHaveAttribute(
    "aria-current",
    "page",
  );
  await page.getByRole("button", { name: "Personal", exact: true }).click();
  await expect(
    page.getByText("9,007,199,254,740,993.01", { exact: true }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page
    .getByRole("button", { name: "Review account transactions", exact: true })
    .click();
  await expect(
    page.getByRole("combobox", { name: "Source account", exact: true }),
  ).toHaveValue("s");
});
