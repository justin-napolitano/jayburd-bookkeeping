import { financeNavItem } from "./finance-navigation";
import { expect, test } from "@playwright/experimental-ct-react";
import { Harness } from "./Harness";
import type { LabelCommand } from "../src/collection/types";

const workspace = "11000000-0000-4000-8000-000000000001";
const a = "71000000-0000-4000-8000-000000000001",
  b = "71000000-0000-4000-8000-000000000002";
for (const restored of ["same", "other"] as const) {
  test(`owner recovery mounting preserves unknown A and handles ${restored} available claim`, async ({
    mount,
    page,
  }) => {
    let status: "none" | "inaccessible" | "available" = "inaccessible";
    let recoveryID = a,
      clears = 0,
      scopedReads = 0,
      executions = 0,
      acknowledgements = 0;
    const clearIDs: string[] = [];
    const snapshot = {
      authority_revision: "1",
      source_snapshot: "a".repeat(64),
      taxonomy_revision: "1",
      ruleset_revision: "0",
      map_revision: "0",
      decision_revision: "0",
    };
    let pending: LabelCommand | null = {
      workspace_id: workspace,
      command_id: a,
      prepare_key: "72000000-0000-4000-8000-000000000001",
      state: "prepared",
      acknowledged: false,
      preview: {
        profile: "private-collection-preview-v1",
        workspace_id: workspace,
        preview_id: "73000000-0000-4000-8000-000000000001",
        expires_at: "2020-01-01T00:00:00Z",
        snapshot,
        action: { kind: "taxonomy", topics: [] },
        selection: [],
        selected_count: "0",
        review_rows: [],
      },
    };
    await page.route("**/v*/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      const body = route.request().postData()
        ? route.request().postDataJSON()
        : undefined;
      const reply = (json: unknown) => route.fulfill({ json });
      if (path === "/v1/workspace")
        return reply({
          entities: [],
          currencies: [],
          development_policy: "dev-accrual-usd-v1",
        });
      if (path === "/v1/commands/pending") return reply({ command: null });
      if (path === "/v1/collection-recovery")
        return reply({
          profile: "private-collection-recovery-v1",
          state: status,
          recovery_id: status === "none" ? null : recoveryID,
        });
      if (path === "/v1/collection-recovery/clear") {
        clearIDs.push(body.recovery_id);
        if (++clears === 1)
          return route.fulfill({
            status: 503,
            json: { code: "temporarily_unavailable" },
          });
        return reply({
          profile: "private-collection-recovery-clear-v1",
          recovery_id: a,
          cleared: true,
        });
      }
      if (path === "/v1/collection") {
        scopedReads++;
        return reply({
          profile: "private-source-workspace-v1",
          workspace_id: workspace,
          workspace_revision: "1",
          capabilities: ["read", "classify"],
        });
      }
      if (path.endsWith("/taxonomy"))
        return reply({
          profile: "private-collection-catalog-v3",
          workspace_id: workspace,
          authority_revision: "1",
          taxonomy_revision: "1",
          ruleset_revision: "0",
          map_revision: "0",
          tag_revision: "1",
          books: [],
          currencies: [],
          institution: { state: "unavailable" },
          legal_owner: { state: "unknown" },
          topics: [],
          rules: [],
          tags: [],
          sources: [],
        });
      if (path.endsWith("/review/query"))
        return reply({
          profile: "private-collection-review-v3",
          workspace_id: workspace,
          snapshot: { ...snapshot, tag_revision: "1" },
          selector: body,
          matching_count: "0",
          start_index: "0",
          items: [],
          next_cursor: null,
          partial: false,
        });
      const wrap = () => ({
        profile: "private-collection-command-v3",
        workspace_id: workspace,
        command: pending,
      });
      if (path.endsWith("/label-commands/pending")) return reply(wrap());
      if (pending && path.endsWith("/execute")) {
        executions++;
        pending.state = "committed";
        return reply(wrap());
      }
      if (pending && path.endsWith("/acknowledge")) {
        acknowledgements++;
        pending.acknowledged = true;
        const result = structuredClone(wrap());
        pending = null;
        status = "none";
        return reply(result);
      }
      return route.fulfill({ status: 404, json: { code: "not_found" } });
    });
    await mount(<Harness />);
    await page
      .getByRole("button", {
        name: "Clear inaccessible private review",
        exact: true,
      })
      .click();
    await expect(
      page.getByText(/Clearing the saved private review is unconfirmed/),
    ).toBeVisible();
    status = "available";
    recoveryID = restored === "same" ? a : b;
    if (restored === "other" && pending)
      pending = {
        ...pending,
        command_id: b,
        prepare_key: "72000000-0000-4000-8000-000000000002",
      };
    await page
      .getByRole("button", { name: "Check private recovery", exact: true })
      .click();
    const saved = page.getByRole("heading", {
      name: "Saved topic review",
      exact: true,
    });
    if (restored === "same") {
      await expect(saved).toBeVisible();
      await page
        .getByRole("button", { name: "Confirm topic changes", exact: true })
        .click();
      await expect(
        page.getByText("Topics updated. No journal was created.", {
          exact: true,
        }),
      ).toBeVisible();
      await page
        .getByRole("button", { name: "Acknowledge result", exact: true })
        .click();
      await expect(financeNavItem(page, "Transactions")).toBeDisabled();
      await expect(
        page.getByRole("button", {
          name: "Retry clearing the same private review",
          exact: true,
        }),
      ).toBeVisible();
      expect(clears).toBe(1);
    } else {
      await expect(saved).toHaveCount(0);
      expect(scopedReads).toBe(0);
      expect(executions).toBe(0);
      expect(acknowledgements).toBe(0);
    }
    await page
      .getByRole("button", {
        name: "Retry clearing the same private review",
        exact: true,
      })
      .click();
    if (restored === "same")
      await expect(financeNavItem(page, "Transactions")).toBeEnabled();
    else {
      await expect(saved).toBeVisible();
      expect(executions).toBe(0);
      expect(acknowledgements).toBe(0);
    }
    expect(clearIDs).toEqual([a, a]);
    expect(
      await page.evaluate(() => [localStorage.length, sessionStorage.length]),
    ).toEqual([0, 0]);
  });
}
