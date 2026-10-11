import { test, expect } from "@playwright/experimental-ct-react";
import { Harness } from "./Harness";
const book = {
  book_id: "b1",
  name: "Operating",
  book_revision: "0",
  policies: [
    {
      policy_id: "p1",
      functional_currency: "USD",
      functional_currency_version_id: "c1",
      effective_from: "2026-01-01",
      effective_to: null,
    },
  ],
  periods: [],
  accounts: [
    {
      account_id: "a1",
      code: "1000",
      name: "Cash",
      category: "asset",
      normal_side: "debit",
      monetary: true,
      active_from: "2026-01-01",
      active_to: null,
    },
    {
      account_id: "a2",
      code: "3000",
      name: "Capital",
      category: "equity",
      normal_side: "credit",
      monetary: false,
      active_from: "2026-01-01",
      active_to: null,
    },
  ],
};
const workspace = {
  entities: [
    {
      entity_id: "e1",
      name: "Synthetic LLC",
      kind: "legal",
      permissions: ["read", "post"],
      books: [book],
    },
    {
      entity_id: "e2",
      name: "Other synthetic LLC",
      kind: "legal",
      permissions: ["read"],
      books: [{ ...book, book_id: "b2" }],
    },
  ],
  currencies: [
    {
      currency_version_id: "c1",
      alphabetic_code: "USD",
      minor_unit: 2,
      valid_from: "2026-01-01",
      valid_to: null,
    },
  ],
  development_policy: "dev-accrual-usd-v1",
};
// Browser tests intercept every financial route with fixed synthetic responses.
test("split review posts only after confirmation and retries the exact uncertain request", async ({
  mount,
  page,
}) => {
  let posts: { key: string | undefined; body: string | null }[] = [];
  await page.route("**/v1/**", async (route) => {
    const req = route.request();
    if (req.url().endsWith("/workspace"))
      return route.fulfill({ json: workspace });
    posts.push({ key: req.headers()["idempotency-key"], body: req.postData() });
    if (posts.length === 2)
      return route.fulfill({ status: 404, json: { code: "not_found" } });
    if (posts.length === 1)
      return route.fulfill({
        status: 503,
        json: { code: "temporarily_unavailable" },
      });
    return route.fulfill({
      status: 201,
      json: {
        journal_id: "j1",
        book_revision: "1",
        accounting_date: "2026-01-01",
        description: "Synthetic split",
      },
    });
  });
  await mount(<Harness />);
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await page.getByRole("combobox", {name:"Policy", exact:true}).selectOption("p1");
  await page.getByLabel("Description", { exact: true }).fill("Synthetic split");
  await page.getByRole("button", { name: "Add split line" }).click();
  for (let i = 1; i <= 3; i++) {
    await page
      .getByLabel(`Line ${i} account`, { exact: true })
      .selectOption(i === 2 ? "a2" : "a1");
    await page
      .getByLabel(`Line ${i} original amount`, { exact: true })
      .fill(i === 2 ? "100.00" : "50.00");
    await page
      .getByLabel(`Line ${i} functional amount`, { exact: true })
      .fill(i === 2 ? "100.00" : "50.00");
  }
  await page
    .getByRole("button", { name: "Review journal", exact: true })
    .click();
  expect(posts).toHaveLength(0);
  await page.getByRole("button", { name: "Post journal", exact: true }).click();
  await expect(
    page.getByText("Unconfirmed outcome — keep this page open"),
  ).toBeVisible();
  await expect(page.getByLabel("Description", { exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Retry unchanged request" }).click();
  await expect(page.getByRole("alert")).toContainText(
    "previous request may have committed",
  );
  await expect(page.getByLabel("Entity and book")).toBeDisabled();
  await expect(page.getByLabel("Description", { exact: true })).toBeDisabled();
  await page.getByRole("button", { name: "Retry unchanged request" }).click();
  await expect(page.getByText("Posting confirmed")).toBeVisible();
  expect(posts).toHaveLength(3);
  expect(posts[1]).toEqual(posts[2]);
  expect(posts[0]).toEqual(posts[1]);
  expect(JSON.parse(posts[0].body!).lines).toHaveLength(3);
});
test("setup requires review and produces confirmed creation only after server response", async ({
  mount,
  page,
}) => {
  let setups = 0;
  await page.route("**/v1/**", (route) =>
    route.request().url().endsWith("/workspace")
      ? route.fulfill({ json: { ...workspace, entities: [] } })
      : (setups++,
        route.fulfill({
          status: 201,
          json: { entity_id: "new", book_id: "new" },
        })),
  );
  await mount(<Harness />);
  await page.getByText("Create an entity and book", { exact: true }).click();
  await page.getByLabel("Entity name", { exact: true }).fill("Fictional LLC");
  await page.getByLabel("Book name", { exact: true }).fill("Operating");
  await page.getByLabel("Period starts on").fill("2026-01-01");
  await page.getByLabel("Period ends before").fill("2027-01-01");
  await page.getByLabel("Functional currency metadata").selectOption("c1");
  for (let i = 1; i <= 2; i++) {
    await page
      .getByLabel(`Account ${i} code`, { exact: true })
      .fill(i === 1 ? "1000" : "3000");
    await page
      .getByLabel(`Account ${i} name`, { exact: true })
      .fill(i === 1 ? "Cash" : "Capital");
  }
  await page.getByRole("button", { name: "Review book setup" }).click();
  expect(setups).toBe(0);
  await page.getByRole("button", { name: "Create book", exact: true }).click();
  await expect(page.getByText(/Book creation confirmed/)).toBeVisible();
  expect(setups).toBe(1);
});
test("denied entity switch clears the previous report and does not show stale financial values", async ({
  mount,
  page,
}) => {
  await page.route("**/v1/**", (route) => {
    const url = route.request().url();
    if (url.endsWith("/workspace")) return route.fulfill({ json: workspace });
    if (url.includes("/e2/"))
      return route.fulfill({ status: 404, json: { code: "not_found" } });
    return route.fulfill({
      json: {
        through_date: "2026-01-01",
        snapshot_revision: "0",
        functional_currency: "USD",
        total_debits: "123.45",
        total_credits: "123.45",
        accounts: [],
      },
    });
  });
  await mount(<Harness />);
  await page.getByLabel("Entity and book").selectOption("e1/b1");
  await page.getByRole("button", { name: "Load trial balance" }).click();
  await expect(page.getByRole("table")).toBeVisible();
  await page.getByLabel("Entity and book").selectOption("e2/b2");
  await expect(page.getByRole("table")).toHaveCount(0);
  await page.getByRole("button", { name: "Load trial balance" }).click();
  await expect(page.getByLabel("Entity and book")).toHaveValue("");
  await expect(page.getByText("123.45", { exact: true })).toHaveCount(0);
});
