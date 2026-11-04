import { test, expect } from "@playwright/experimental-ct-react";
import { Harness } from "./Harness";
import { positionWorkspace } from "./position-fixture";

// Real client classification must not turn missing authority into an idle command gate.
for (const failure of ["network", "503", "invalid JSON"] as const) {
  test(`failed pending discovery remains locked until explicit successful read: ${failure}`, async ({
    mount,
    page,
  }) => {
    let recovered = false;
    let reads = 0;
    const writes: string[] = [];
    await page.route("**/v1/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (route.request().method() !== "GET") writes.push(path);
      if (path === "/v1/workspace")
        return route.fulfill({ json: positionWorkspace });
      if (path === "/v1/commands/pending") {
        reads++;
        if (recovered) return route.fulfill({ json: { command: null } });
        if (failure === "network") return route.abort("failed");
        if (failure === "503")
          return route.fulfill({ status: 503, body: "private failure" });
        return route.fulfill({ status: 200, body: "invalid private JSON" });
      }
      return route.fulfill({ status: 404 });
    });
    await mount(<Harness />);
    await expect(page.getByRole("alert")).toHaveText(
      "Data is unavailable right now. Retry the read.",
    );
    await expect(
      page.getByRole("button", { name: "Transactions", exact: true }),
    ).toBeDisabled();
    await expect(
      page.getByRole("button", { name: "Clear inaccessible saved command" }),
    ).toHaveCount(0);
    expect(reads).toBe(1);
    expect(writes).toEqual([]);
    recovered = true;
    await page.getByRole("button", { name: "Check saved command" }).click();
    await expect(
      page.getByRole("button", { name: "Transactions", exact: true }),
    ).toBeEnabled();
    expect(reads).toBe(2);
    expect(writes).toEqual([]);
  });
}
