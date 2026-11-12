import { expect, test } from "@playwright/experimental-ct-react";
import { AutomaticCategorizationHarness } from "./AutomaticCategorizationHarness";
for (const width of [390, 1440])
  test(`persisted automatic categories and exact unknown settings recovery at ${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 });
    await mount(<AutomaticCategorizationHarness />);
    const rows = page.getByRole("list", { name: "Transaction rows" });
    await expect(
      rows.getByText("Automatic · Bank category").first(),
    ).toBeVisible();
    await expect(rows.getByText("Your choice")).toBeVisible();
    const first = rows.getByRole("checkbox").first();
    await expect(first).toBeVisible();
    await expect(first).toBeEnabled();
    await page.screenshot({
      path: info.outputPath(`fictional-automatic-initial-${width}.png`),
    });
    expect((await first.boundingBox())!.y).toBeLessThan(844);
    await first.focus();
    await page.keyboard.press("Space");
    await expect(first).toBeChecked();
    await page.keyboard.press("Space");
    await expect(first).not.toBeChecked();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await page
      .getByRole("button", { name: "Needs attention", exact: true })
      .click();
    await expect(
      page.getByText("1–1 of 1 matching transactions"),
    ).toBeVisible();
    await expect(
      rows.getByText("Needs category", { exact: true }),
    ).toBeVisible();
    await expect(
      rows.getByText("The bank category needs your review"),
    ).toBeVisible();
    await page.getByText("Settings", { exact: true }).click();
    await page
      .getByRole("button", {
        name: "Turn off automatic categories",
        exact: true,
      })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "settings result is unknown",
    );
    await expect(
      page.getByRole("button", { name: "Refresh transactions", exact: true }),
    ).toBeDisabled();
    await expect(rows.getByRole("checkbox").first()).toBeDisabled();
    const accounts = page
      .getByRole("navigation", { name: "Finance tools", includeHidden: true })
      .getByRole("button", {
        name: "Accounts",
        exact: true,
        includeHidden: true,
      });
    await expect(accounts).toBeDisabled();
    await page
      .getByRole("button", { name: "Retry same settings change", exact: true })
      .focus();
    await page.keyboard.press("Enter");
    await expect(page.getByText(/^Off$/)).toBeVisible();
    const requests = JSON.parse(
      (await page.getByLabel("Fictional automation requests").textContent())!,
    );
    expect(requests).toHaveLength(2);
    expect(requests[1]).toEqual(requests[0]);
    await expect(page.getByLabel("Fictional configuration effects")).toHaveText(
      "1",
    );
    expect(
      await page.evaluate(() => [localStorage.length, sessionStorage.length]),
    ).toEqual([0, 0]);
    await page.screenshot({
      path: info.outputPath(`fictional-automatic-${width}.png`),
      fullPage: true,
    });
  });
