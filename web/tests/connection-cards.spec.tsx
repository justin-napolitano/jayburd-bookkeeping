import { test, expect } from "@playwright/experimental-ct-react";
import { AccountLabelsHarness } from "./AccountLabelsHarness";

// Existing projections only: management disclosure/cancellation must not send a request.
for (const width of [1440, 390]) {
  test(`registered accounts keep primary review and explicit secondary management at ${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 844 });
    await mount(<AccountLabelsHarness multiple />);
    const active = page.getByRole("listitem", {
      name: "Connected Bank account 1",
      exact: true,
    });
    const stopped = page.getByRole("listitem", {
      name: "Connected Bank account 2",
      exact: true,
    });
    await expect(
      active.getByRole("heading", { name: "Bank account 1", exact: true }),
    ).toBeVisible();
    await expect(active).toContainText("Connection: active");
    await expect(stopped).toContainText("Connection: disconnected");
    const review = active.getByRole("button", {
      name: "Review transactions for Bank account 1",
    });
    await expect(review).toHaveText("Transactions");
    await review.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByLabel("Reviewed synthetic source")).toHaveText(
      "00000000-0000-4000-8000-000000000007",
    );
    await expect(
      active.getByRole("button", { name: "Disconnect Bank account 1" }),
    ).toBeHidden();
    await expect(
      active.getByText("Provider account ID: private-card-a", { exact: true }),
    ).toBeHidden();
    const manage = active.getByText("Manage Bank account 1", { exact: true });
    await manage.focus();
    await page.keyboard.press("Enter");
    await expect(
      active.getByRole("button", { name: "Disconnect Bank account 1" }),
    ).toBeVisible();
    await expect(page.getByLabel("Synthetic mutation count")).toHaveText("0");
    const technical = active.getByText(
      "Technical account details for Bank account 1",
      { exact: true },
    );
    await technical.focus();
    await page.keyboard.press("Enter");
    await expect(
      active.getByText("Provider account ID: private-card-a", { exact: true }),
    ).toBeVisible();
    await page.keyboard.press("Enter");
    await active
      .getByRole("button", { name: "Disconnect Bank account 1" })
      .click();
    await expect(
      page.getByRole("group", { name: "Confirm account disconnect" }),
    ).toBeVisible();
    await expect(page.getByLabel("Synthetic mutation count")).toHaveText("0");
    await page
      .getByRole("button", { name: "Cancel disconnect", exact: true })
      .click();
    await expect(
      page.getByRole("group", { name: "Confirm account disconnect" }),
    ).toHaveCount(0);
    await manage.click();
    await stopped.getByText("Manage Bank account 2", { exact: true }).click();
    await expect(
      stopped.getByRole("button", { name: "Disconnect Bank account 2" }),
    ).toBeDisabled();
    await expect(
      stopped.getByRole("button", {
        name: "Review transactions for Bank account 2",
      }),
    ).toBeEnabled();
    await stopped.getByText("Manage Bank account 2", { exact: true }).click();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page.screenshot({
      path: info.outputPath(`connected-accounts-${width}.png`),
      fullPage: true,
    });
    await page.evaluate(() => {
      document.body.style.fontSize = "200%";
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await expect(page.getByLabel("Synthetic mutation count")).toHaveText("0");
  });
}
