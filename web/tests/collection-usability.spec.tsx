import { expect, test } from "@playwright/experimental-ct-react";
import { CollectionReviewHarness } from "./CollectionReviewHarness";

// Hands-on synthetic review: inspect the rendered page before interacting; no provider/private DOM is used.
for (const width of [390, 1440]) {
  test(`synthetic collection usability review at${width}px`, async ({
    mount,
    page,
  }, info) => {
    await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 });
    await mount(<CollectionReviewHarness />);
    await expect(
      page.getByText("1–25 of 3000 matching transactions"),
    ).toBeVisible();
    await page.screenshot({ path: info.outputPath("fictional-initial.png") });
    const first = page.getByRole("checkbox").first();
    expect((await first.boundingBox())!.y).toBeLessThan(844);
    await first.focus();
    await page.keyboard.press("Space");
    await expect(first).toBeChecked();
    await page.screenshot({
      path: info.outputPath("fictional-first-selection.png"),
    });
    const action = page.getByRole("button", {
      name: "Preview labels for 1 selected",
      exact: true,
    });
    const manage = page.getByText("Manage topics", { exact: true });
    console.info(
      JSON.stringify({
        width,
        firstSelectionY: (await first.boundingBox())!.y,
        bulkActionY: (await action.boundingBox())!.y,
        topicManagerY: (await manage.boundingBox())!.y,
      }),
    );
    expect((await action.boundingBox())!.y).toBeLessThan(750);
    expect((await manage.boundingBox())!.y).toBeLessThan(600);
    await manage.click();
    await expect(
      page.getByLabel("Topic 1 name", { exact: true }),
    ).toBeVisible();
    await page.screenshot({
      path: info.outputPath("fictional-topic-editor.png"),
    });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    expect(
      await page.evaluate(() => [localStorage.length, sessionStorage.length]),
    ).toEqual([0, 0]);
  });
}
