import { expect, test } from "@playwright/experimental-ct-react";
import { CollectionFiltersDetailHarness } from "./CollectionFiltersDetailHarness";
import { filterBook, filterCurrency } from "./collection-filter-detail-fixture";

for (const width of [390, 1440])
  test(`rich filters and authorized detail preserve exact intent and keyboard return at ${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 });
    let component = await mount(<CollectionFiltersDetailHarness />);
    await expect(
      page.getByText("1–25 of 30 matching transactions"),
    ).toBeVisible();
    await page.getByRole("checkbox").first().check();
    await page
      .getByText("Book, topic and date filters", { exact: true })
      .click();
    await page
      .getByLabel("Merchant match", { exact: true })
      .selectOption("contains");
    await page
      .getByLabel("Merchant text", { exact: true })
      .fill("Fictional Cafe");
    await page
      .getByLabel("Original currency for amount filter", { exact: true })
      .selectOption(filterCurrency.currency_version_id);
    await page
      .getByLabel("Minimum original amount", { exact: true })
      .fill("9007199254740993.0100");
    await page
      .getByLabel("Maximum original amount", { exact: true })
      .fill("9007199254740993.02");
    await page
      .getByLabel("Specific book", { exact: true })
      .selectOption(`book:${filterBook.entity_id}:${filterBook.book_id}`);
    await page
      .getByLabel("Source status", { exact: true })
      .selectOption("booked");
    await page
      .getByLabel("Journal status", { exact: true })
      .selectOption("unavailable");
    await page
      .getByLabel("Institution", { exact: true })
      .selectOption("unavailable");
    await page
      .getByLabel("Legal owner", { exact: true })
      .selectOption("unknown");
    await page
      .getByLabel("Sort transactions by", { exact: true })
      .selectOption("amount");
    await page
      .getByLabel("Sort direction", { exact: true })
      .selectOption("asc");
    await page
      .getByRole("button", { name: "Apply filters", exact: true })
      .click();
    await expect(
      page.getByText("0 selected on this page", { exact: true }),
    ).toBeVisible();
    const query = JSON.parse(
      (await page.getByLabel("Last fictional query").textContent()) ?? "{}",
    );
    expect(query.profile).toBe("private-collection-review-v3");
    expect(query.merchant).toEqual({
      kind: "contains",
      value: "Fictional Cafe",
    });
    expect(query.amount).toEqual({
      currency_version_id: filterCurrency.currency_version_id,
      minimum: "9007199254740993.0100",
      maximum: "9007199254740993.02",
      direction: "all",
    });
    expect(query.book).toEqual({
      kind: "book",
      entity_id: filterBook.entity_id,
      book_id: filterBook.book_id,
    });
    expect(query.status).toEqual({ source: "booked", journal: "unavailable" });
    expect(query.sort).toEqual({ field: "amount", order: "asc" });
    expect(query.cursor).toBeNull();
    expect(query.snapshot).toBeNull();
    const row = page.getByRole("button", {
      name: "Fictional purchase 1",
      exact: true,
    });
    await row.focus();
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("heading", { name: "Transaction detail", exact: true }),
    ).toBeFocused();
    await expect(page.getByLabel("Fictional journal reads")).toHaveText("0");
    await page.getByRole("button", { name: /^View posted journal / }).click();
    await expect(
      page.getByRole("heading", { name: "Posted journal", exact: true }),
    ).toBeFocused();
    await expect(
      page.getByRole("list", { name: "Immutable posted journal lines" }),
    ).toContainText("9007199254740993.01 EUR");
    await expect(
      page.getByRole("list", { name: "Immutable posted journal lines" }),
    ).toContainText("9907919180215092.311 USD");
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath("fictional-source-journal-detail.png"),
    });
    await page
      .getByRole("button", { name: "Back to transactions", exact: true })
      .click();
    await expect(row).toBeFocused();
    await expect(
      page.getByText("1–25 of 30 matching transactions"),
    ).toBeVisible();
    await expect(page.getByLabel("Fictional detail reads")).toHaveText("1");
    expect(
      await page.evaluate(() => [localStorage.length, sessionStorage.length]),
    ).toEqual([0, 0]);
    await component.unmount();
    component = await mount(<CollectionFiltersDetailHarness />);
    await expect(page.getByLabel("Fictional detail reads")).toHaveText("0");
    await expect(
      page.getByRole("region", { name: "Source transaction detail" }),
    ).toHaveCount(0);
    await expect(
      page.getByText("1–25 of 30 matching transactions"),
    ).toBeVisible();
  });
