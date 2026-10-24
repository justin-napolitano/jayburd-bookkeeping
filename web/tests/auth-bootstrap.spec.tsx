import { test, expect } from "@playwright/experimental-ct-react";
import { AuthBootstrapHarness } from "./AuthBootstrapHarness";

const publicFixture = {
  apiKey: "synthetic-public",
  projectId: "synthetic-project",
  authDomain: "synthetic-project.firebaseapp.com",
};

/** Public configuration is fictional; browser never submits credentials or calls a real identity provider. */
test("delayed configuration stays loading and retries a transient failure on desktop and mobile", async ({
  mount,
  page,
}) => {
  let externalRequests = 0;
  await page.route("https://**/*", async (route) => {
    externalRequests++;
    await route.abort();
  });
  for (const viewport of [
    { width: 1280, height: 800 },
    { width: 390, height: 844 },
  ]) {
    await page.setViewportSize(viewport);
    let release!: () => void;
    const pending = new Promise<void>((resolve) => {
      release = resolve;
    });
    let requests = 0;
    await page.route("**/api/auth-config", async (route) => {
      requests++;
      if (requests === 1) {
        await pending;
        await route.fulfill({ status: 503, body: "" });
      } else await route.fulfill({ json: publicFixture });
    });
    const component = await mount(<AuthBootstrapHarness />);
    await expect(
      page.getByRole("heading", { name: "Preparing sign-in" }),
    ).toBeVisible();
    await expect(page.getByText("Sign-in is not configured")).toHaveCount(0);
    await expect(page.getByLabel("Password")).toHaveCount(0);
    release();
    await expect(
      page.getByRole("heading", { name: "Sign-in is temporarily unavailable" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Retry sign-in" }).focus();
    await page.keyboard.press("Enter");
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByText("Sign-in is not configured")).toHaveCount(0);
    expect(requests).toBe(2);
    await component.unmount();
    await page.unroute("**/api/auth-config");
  }
  expect(externalRequests).toBe(0);
});
