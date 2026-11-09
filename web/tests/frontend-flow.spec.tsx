import { expect, test } from "@playwright/experimental-ct-react";
import { FrontendReview } from "../review/FrontendReview";

for (const width of [390, 768, 1024, 1280, 1440]) {
  test(`${width}px integrated navigation and transaction controls stay reachable`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    await mount(<FrontendReview />);
    await expect(
      page.getByRole("button", { name: "Studio supplies", exact: true }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    if (width < 960)
      await page.getByRole("button", { name: "Menu", exact: true }).click();
    const nav = page.getByRole("navigation", { name: "Finance tools" });
    for (const label of [
      "Transactions",
      "Accounts",
      "Imported activity",
      "Posted entries",
    ]) {
      const control = nav.getByRole("button", { name: label, exact: true });
      await expect(control).toBeVisible();
      const box = (await control.boundingBox())!;
      expect(box.x).toBeGreaterThanOrEqual(0);
      expect(box.x + box.width).toBeLessThanOrEqual(width);
      expect(box.height).toBeGreaterThanOrEqual(44);
    }
    await nav
      .getByRole("button", { name: "Transactions", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Studio supplies", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Studio supplies", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Back to transactions", exact: true }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Back to transactions", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Studio supplies", exact: true }),
    ).toBeFocused();
    await page.screenshot({
      path: info.outputPath(`frontend-flow-${width}.png`),
      fullPage: true,
    });
  });
}

test("guided sample setup preserves explicit save, execute and acknowledgement", async ({
  mount,
  page,
}, info) => {
  await page.setViewportSize({ width: 1024, height: 900 });
  await mount(<FrontendReview />);
  await expect(
    page.getByRole("button", { name: "Studio supplies", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Try workspace setup", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Set up private transactions", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Save setup review", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Create private workspace", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Save setup review", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Create private workspace", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Acknowledge setup result", exact: true }),
  ).toBeVisible();
  await page.screenshot({
    path: info.outputPath("frontend-setup-completion.png"),
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "Acknowledge setup result", exact: true })
    .click();
  await page
    .getByRole("button", {
      name: "Review connection for Fictional records",
      exact: true,
    })
    .click();
  await expect(
    page.getByText(/Existing automatic bank syncing/).first(),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Save setup review", exact: true })
    .click();
  await page
    .getByRole("button", {
      name: "Adopt reviewed connection history",
      exact: true,
    })
    .click();
  await expect(
    page.getByText(/Existing connection history adopted/),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Acknowledge setup result", exact: true })
    .click();
  await page
    .getByText("Private workspace and existing connections", { exact: true })
    .click();
  await expect(
    page.getByText(/No existing connection is currently eligible for adoption/),
  ).toBeVisible();
  await expect(page.getByRole("alert")).toHaveCount(0);
  expect(
    await page.evaluate(() => [localStorage.length, sessionStorage.length]),
  ).toEqual([0, 0]);
});
