import { expect, test } from "@playwright/experimental-ct-react";
import { AutomaticAccountingHarness } from "./AutomaticAccountingHarness";
for (const width of [1280, 390])
  test(`automatic accounting policy is explicit and compact at ${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: 900 });
    await mount(<AutomaticAccountingHarness />);
    await expect(
      page.getByText(
        "Configuration is saved. Automatic posting is not active in this deployment.",
      ),
    ).toBeVisible();
    await page.getByRole("button", { name: "Configure accounting" }).click();
    await page
      .getByRole("button", { name: "Review accounting configuration" })
      .click();
    await expect(
      page.getByRole("table", { name: "Mappings to save" }),
    ).toBeVisible();
    await expect(page.getByLabel("Configuration writes")).toHaveText("0");
    await expect(
      page.getByRole("button", { name: "Save accounting configuration" }),
    ).toBeVisible();
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth,
    );
    expect(overflow).toBe(false);
    if (width === 390) {
      const mappings = page.getByRole("region", {
        name: "Accounting configuration mappings",
        exact: true,
      });
      await expect
        .poll(() =>
          mappings.evaluate(
            (element) => element.scrollWidth > element.clientWidth,
          ),
        )
        .toBe(true);
      await mappings.focus();
      await mappings.press("ArrowRight");
      await expect
        .poll(() => mappings.evaluate((element) => element.scrollLeft))
        .toBeGreaterThan(0);
    }
    await page.screenshot({
      path: info.outputPath(`automatic-accounting-${width}.png`),
      fullPage: true,
    });
  });
