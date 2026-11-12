import { afterEach, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { APIError, type API } from "../bookkeeping/api";
import { AutomaticCategorization } from "./AutomaticCategorization";
import {
  CategorizationSummary,
  winningRuleNames,
} from "./CategorizationSummary";
import {
  categorizationOf,
  createAutomationPort,
  validAutomationStatus,
  type AutomationStatus,
  type AutomationPort,
  type Categorization,
} from "./automation";
import { CollectionFailure, type ReviewRow } from "./types";
import {
  collectionCatalog,
  collectionRow,
} from "../../tests/collection-fixture";
afterEach(cleanup);
afterEach(() => vi.useRealTimers());
const workspace = "88000000-0000-4000-8000-000000000001";
const status = (patch: Partial<AutomationStatus> = {}): AutomationStatus => ({
  profile: "private-collection-automation-v1",
  workspace_id: workspace,
  revision: "0",
  enabled: false,
  processing_state: "disabled",
  pending_count: "3000",
  attention_count: "17",
  blocked_reason: null,
  last_run: null,
  ...patch,
});
const automatic: Categorization = {
  state: "automatic",
  origin: "rule",
  reason: null,
  rule_names: ["Fictional travel"],
  provider: null,
  decision_id: "88000000-0000-4000-8000-000000000002",
};
function settings(
  port: AutomationPort,
  patch: Partial<Parameters<typeof AutomaticCategorization>[0]> = {},
) {
  const p = {
    workspace,
    port,
    canClassify: true,
    locked: false,
    onLockChange: vi.fn(),
    onDenied: vi.fn(),
    onChanged: vi.fn(),
    onAttention: vi.fn(),
    ...patch,
  };
  return { ...render(<AutomaticCategorization {...p} />), props: p };
}
it("validates scoped status, bounded counts and historical run identity", () => {
  expect(validAutomationStatus(status(), workspace)).toBe(true);
  for (const patch of [
    { workspace_id: "88000000-0000-4000-8000-000000000099" },
    { pending_count: "-1" },
    { attention_count: "1.0" },
    { revision: "01" },
    { pending_count: "1".repeat(20) },
    { enabled: true },
  ])
    expect(validAutomationStatus(status(patch), workspace)).toBe(false);
  expect(
    validAutomationStatus(
      status({
        last_run: {
          profile: "private-collection-auto-run-v1",
          workspace_id: workspace,
          run_id: automatic.decision_id!,
          state: "committed",
          processed_count: "100",
          attention_count: "2",
        },
      }),
      workspace,
    ),
  ).toBe(true);
  expect(
    validAutomationStatus(
      status({
        last_run: {
          profile: "private-collection-auto-run-v1",
          workspace_id: "wrong",
          run_id: automatic.decision_id!,
          state: "committed",
          processed_count: "100",
          attention_count: "2",
        },
      }),
      workspace,
    ),
  ).toBe(false);
});
it("falls back safely for malformed annotations without turning suggestions into categories", () => {
  expect(
    categorizationOf({ ...automatic, reason: "new-unknown-reason" })?.state,
  ).toBe("unavailable");
  expect(categorizationOf({ ...automatic, decision_id: null })?.state).toBe(
    "unavailable",
  );
  expect(categorizationOf(undefined)).toBeUndefined();
  const row = collectionRow(0, {
    owner_topic: { state: "unclassified" },
  }) as ReviewRow;
  render(<CategorizationSummary row={row} />);
  expect(screen.getByText("Needs category")).toBeTruthy();
  expect(screen.queryByText(/Automatic/)).toBeNull();
});
it("shows saved categories and protected manual origin without an acknowledgement action", () => {
  const row = collectionRow(0, {
    owner_topic: { state: "topic", topic_id: "travel", name: "Travel" },
  }) as ReviewRow;
  const view = render(
    <CategorizationSummary row={{ ...row, categorization: automatic }} />,
  );
  expect(screen.getByText("Travel")).toBeTruthy();
  expect(screen.getByText("Automatic · Your rule")).toBeTruthy();
  expect(screen.queryByRole("button")).toBeNull();
  view.rerender(
    <CategorizationSummary
      row={{
        ...row,
        categorization: {
          ...automatic,
          state: "manual",
          origin: "manual",
          rule_names: [],
        },
      }}
    />,
  );
  expect(screen.getByText("Your choice")).toBeTruthy();
  expect(screen.queryByText(/Automatic/)).toBeNull();
});
it("never attributes an earlier rule to a renamed current edition", () => {
  const catalog = {
    ...collectionCatalog,
    rules: [
      {
        rule_id: "rule",
        name: "New name",
        status: "active" as const,
        source_account_id: null,
        priority: 0,
        conditions: [
          {
            field: "merchant" as const,
            operator: "exact" as const,
            value: "Fictional",
          },
        ],
        output: { kind: "uncategorized" as const, tag_ids: [] },
      },
    ],
  };
  expect(
    winningRuleNames([{ rule_id: "rule", rule_revision: "3" }], catalog),
  ).toEqual(["An earlier saved rule"]);
  expect(
    winningRuleNames(
      [{ rule_id: "rule", rule_revision: catalog.ruleset_revision }],
      catalog,
    ),
  ).toEqual(["New name"]);
});
it("retries exact unknown settings and locks competing work until confirmed receipt", async () => {
  const requests: unknown[] = [];
  let enabled = false;
  const port: AutomationPort = {
    read: vi.fn(async () =>
      status({ enabled, processing_state: enabled ? "pending" : "disabled" }),
    ),
    configure: vi.fn(async (_w, body) => {
      requests.push(structuredClone(body));
      enabled = true;
      if (requests.length === 1) throw new CollectionFailure("uncertain");
      return status({
        revision: "1",
        enabled: true,
        processing_state: "pending",
      });
    }),
  };
  const h = settings(port),
    user = userEvent.setup();
  await user.click(
    await screen.findByRole("button", { name: "Enable automatic categories" }),
  );
  await screen.findByText(/settings result is unknown/);
  expect(h.props.onLockChange).toHaveBeenLastCalledWith(true);
  expect(
    screen.queryByRole("button", { name: "Enable automatic categories" }),
  ).toBeNull();
  await user.click(
    screen.getByRole("button", { name: "Retry same settings change" }),
  );
  await screen.findByText(/^On$/);
  expect(requests).toHaveLength(2);
  expect(requests[1]).toEqual(requests[0]);
  expect(h.props.onChanged).toHaveBeenCalledOnce();
  await waitFor(() =>
    expect(h.props.onLockChange).toHaveBeenLastCalledWith(false),
  );
  expect(screen.queryByRole("button", { name: /Acknowledge/ })).toBeNull();
});
it("retains uncertain intent when a retry receives a definite rejection", async () => {
  const configure = vi
    .fn()
    .mockRejectedValueOnce(new CollectionFailure("uncertain"))
    .mockRejectedValue(new CollectionFailure("stale"));
  const h = settings({ read: async () => status(), configure }),
    user = userEvent.setup();
  await user.click(
    await screen.findByRole("button", { name: "Enable automatic categories" }),
  );
  await user.click(
    await screen.findByRole("button", { name: "Retry same settings change" }),
  );
  expect(configure.mock.calls[1]).toEqual(configure.mock.calls[0]);
  expect(h.props.onLockChange).toHaveBeenLastCalledWith(true);
  expect(
    screen.queryByRole("button", { name: "Refresh category status" }),
  ).toBeNull();
});
it("read-only status offers no implicit enable and retains whole-result counts", async () => {
  const configure = vi.fn();
  const h = settings(
    { read: async () => status(), configure },
    { canClassify: false },
  );
  await screen.findByText("3000 awaiting categorization · 17 need attention");
  expect(
    screen.queryByRole("button", { name: "Enable automatic categories" }),
  ).toBeNull();
  await userEvent.click(
    screen.getByRole("button", { name: "Needs attention" }),
  );
  expect(h.props.onAttention).toHaveBeenCalledOnce();
  expect(configure).not.toHaveBeenCalled();
});
it("denied status cannot become a false enabled state", async () => {
  const h = settings({
    read: async () => {
      throw new CollectionFailure("denied");
    },
    configure: vi.fn(),
  });
  await waitFor(() => expect(h.props.onDenied).toHaveBeenCalledOnce());
  expect(screen.queryByText(/^On$/)).toBeNull();
});
it("fences late old status when workspace changes", async () => {
  let done!: (v: AutomationStatus) => void;
  const read = vi.fn(
    () =>
      new Promise<AutomationStatus>((resolve) => {
        done = resolve;
      }),
  );
  const h = settings({ read, configure: vi.fn() });
  h.rerender(
    <AutomaticCategorization
      {...h.props}
      workspace="88000000-0000-4000-8000-000000000099"
      port={{
        read: async () => {
          throw new CollectionFailure("unavailable");
        },
        configure: vi.fn(),
      }}
    />,
  );
  await act(async () =>
    done(status({ enabled: true, processing_state: "idle" })),
  );
  expect(screen.queryByText(/^On$/)).toBeNull();
});
it("uses authenticated automation route and the identical raw body on uncertain retry", async () => {
  const bodies: (string | undefined)[] = [];
  const api = vi.fn(async (path: string, body?: string) => {
    expect(path).toBe(`/v1/collections/${workspace}/automation`);
    bodies.push(body);
    if (bodies.length === 1) throw new APIError(0, true);
    return status({
      enabled: true,
      processing_state: "pending",
      revision: "1",
    });
  }) as unknown as API;
  const port = createAutomationPort(api),
    request = { key: "fixed-key", expected_revision: "0", enabled: true };
  await expect(port.configure(workspace, request)).rejects.toMatchObject({
    kind: "uncertain",
  });
  await port.configure(workspace, request);
  expect(bodies).toEqual([
    '{"key":"fixed-key","expected_revision":"0","enabled":true}',
    '{"key":"fixed-key","expected_revision":"0","enabled":true}',
  ]);
});

it("only displays supported high-confidence retained provider attribution as automatic", () => {
  const provider = {
    name: "Plaid",
    version: "v2",
    primary: "FOOD_AND_DRINK",
    detailed: "FOOD_AND_DRINK_COFFEE",
    confidence: "HIGH",
    map_edition: "plaid-pfc-primary-2026-11-11",
  };
  expect(
    categorizationOf({
      ...automatic,
      origin: "provider",
      rule_names: [],
      provider,
    })?.state,
  ).toBe("automatic");
  for (const patch of [
    { confidence: "LOW" },
    { version: "v9" },
    { primary: "x".repeat(129) },
    { map_edition: "invented" },
  ])
    expect(
      categorizationOf({
        ...automatic,
        origin: "provider",
        rule_names: [],
        provider: { ...provider, ...patch },
      })?.state,
    ).toBe("unavailable");
});
it("fresh mount reads current policy without replaying an unknown old settings request", async () => {
  const configure = vi
    .fn()
    .mockRejectedValue(new CollectionFailure("uncertain"));
  const first = settings({ read: async () => status(), configure });
  await userEvent.click(
    await screen.findByRole("button", { name: "Enable automatic categories" }),
  );
  await screen.findByText(/settings result is unknown/);
  first.unmount();
  const nextConfigure = vi.fn();
  settings({
    read: async () =>
      status({ revision: "1", enabled: true, processing_state: "idle" }),
    configure: nextConfigure,
  });
  await screen.findByText(/^On$/);
  expect(configure).toHaveBeenCalledOnce();
  expect(nextConfigure).not.toHaveBeenCalled();
  expect(screen.queryByText(/not accepted/)).toBeNull();
});
it("notifies current denial after a confirmed configuration receipt when fresh disclosure is denied", async () => {
  let reads = 0;
  const h = settings({
    read: async () => {
      if (++reads === 1) return status();
      throw new CollectionFailure("denied");
    },
    configure: async () =>
      status({ revision: "1", enabled: true, processing_state: "idle" }),
  });
  await userEvent.click(
    await screen.findByRole("button", { name: "Enable automatic categories" }),
  );
  await waitFor(() => expect(h.props.onDenied).toHaveBeenCalledOnce());
  expect(screen.queryByText(/^On$/)).toBeNull();
});
it("treats wrong-workspace mutation responses as unknown rather than successful", async () => {
  const api = vi.fn(async () =>
    status({
      workspace_id: "88000000-0000-4000-8000-000000000099",
      enabled: true,
      processing_state: "idle",
    }),
  ) as unknown as API;
  await expect(
    createAutomationPort(api).configure(workspace, {
      key: "original",
      expected_revision: "0",
      enabled: true,
    }),
  ).rejects.toMatchObject({ kind: "uncertain" });
});

const pendingStatus = () =>
  status({ enabled: true, processing_state: "pending", pending_count: "100" });
const completedStatus = () =>
  status({
    enabled: true,
    processing_state: "idle",
    pending_count: "0",
    last_run: {
      profile: "private-collection-auto-run-v1",
      workspace_id: workspace,
      run_id: automatic.decision_id!,
      state: "committed",
      processed_count: "100",
      attention_count: "17",
    },
  });

it("observes pending work with reads and refreshes its committed run once without any category mutation", async () => {
  vi.useFakeTimers();
  const read = vi
    .fn()
    .mockResolvedValueOnce(pendingStatus())
    .mockResolvedValue(completedStatus());
  const configure = vi.fn(),
    onRun = vi.fn();
  const h = settings({ read, configure }, { onRun });
  await act(async () => {});
  expect(screen.getByText("Categorizing imported transactions…")).toBeTruthy();
  await act(async () => vi.advanceTimersByTime(5000));
  expect(read).toHaveBeenCalledTimes(2);
  expect(onRun).toHaveBeenCalledExactlyOnceWith(automatic.decision_id);
  expect(screen.queryByText("Categorizing imported transactions…")).toBeNull();
  h.rerender(<AutomaticCategorization {...h.props} active={false} />);
  h.rerender(<AutomaticCategorization {...h.props} active />);
  await act(async () => {});
  expect(read).toHaveBeenCalledTimes(3);
  expect(onRun).toHaveBeenCalledOnce();
  expect(configure).not.toHaveBeenCalled();
});

it("bounds pending status reads and cancels them when the view closes", async () => {
  vi.useFakeTimers();
  const read = vi.fn().mockResolvedValue(pendingStatus());
  const h = settings({ read, configure: vi.fn() });
  await act(async () => {});
  for (let n = 0; n < 6; n++)
    await act(async () => vi.advanceTimersByTime(5000));
  expect(read).toHaveBeenCalledTimes(7);
  await act(async () => vi.advanceTimersByTime(60000));
  expect(read).toHaveBeenCalledTimes(7);
  h.unmount();
  await act(async () => vi.advanceTimersByTime(60000));
  expect(read).toHaveBeenCalledTimes(7);
});

it("does not refresh a catalog from a late status reply after competing work locks the view", async () => {
  vi.useFakeTimers();
  let finish!: (v: AutomationStatus) => void;
  const read = vi
    .fn()
    .mockResolvedValueOnce(pendingStatus())
    .mockImplementation(
      () =>
        new Promise<AutomationStatus>((resolve) => {
          finish = resolve;
        }),
    );
  const onRun = vi.fn(),
    h = settings({ read, configure: vi.fn() }, { onRun });
  await act(async () => {});
  await act(async () => vi.advanceTimersByTime(5000));
  expect(read).toHaveBeenCalledTimes(2);
  h.rerender(<AutomaticCategorization {...h.props} locked />);
  await act(async () => finish(completedStatus()));
  await act(async () => vi.advanceTimersByTime(60000));
  expect(onRun).not.toHaveBeenCalled();
  expect(read).toHaveBeenCalledTimes(2);
});

it("pauses status polling without discarding an in-flight settings receipt when the view becomes inactive", async () => {
  let finish!: (v: AutomationStatus) => void,
    enabled = false;
  const port: AutomationPort = {
    read: async () =>
      status({ enabled, processing_state: enabled ? "idle" : "disabled" }),
    configure: () =>
      new Promise<AutomationStatus>((resolve) => {
        finish = resolve;
      }),
  };
  const h = settings(port);
  await userEvent.click(
    await screen.findByRole("button", { name: "Enable automatic categories" }),
  );
  h.rerender(<AutomaticCategorization {...h.props} active={false} />);
  enabled = true;
  await act(async () =>
    finish(status({ revision: "1", enabled: true, processing_state: "idle" })),
  );
  expect(h.props.onChanged).toHaveBeenCalledOnce();
  expect(h.props.onLockChange).toHaveBeenLastCalledWith(false);
  expect(
    screen.queryByRole("button", { name: "Retry same settings change" }),
  ).toBeNull();
});
