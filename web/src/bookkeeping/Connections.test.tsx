import React, { StrictMode } from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { Connections, type ConnectionsProps } from "./Connections";
import { APIError, type API } from "./api";
import type { components } from "../lib/api.generated";
afterEach(cleanup);
const id = (n: number) =>
  `00000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const entity = {
  entity_id: id(1),
  name: "Fictional LLC",
  kind: "legal",
  permissions: ["read", "import"],
  books: [],
} as ConnectionsProps["entity"];
const book = {
  book_id: id(2),
  name: "Books",
  book_revision: "1",
  policies: [],
  periods: [],
  accounts: [
    {
      account_id: id(3),
      code: "1000",
      name: "Cash",
      category: "asset",
      normal_side: "debit",
      monetary: true,
      active_from: "2020-01-01",
      active_to: null,
    },
  ],
} as ConnectionsProps["book"];
const currencies = [
  {
    currency_version_id: id(4),
    alphabetic_code: "USD",
    minor_unit: 2,
    valid_from: "2020-01-01",
    valid_to: null,
  },
];
const attempt = {
  attempt_id: id(5),
  entity_id: id(1),
  book_id: id(2),
  mode: "new",
  state: "link_ready",
  expires_at: "2030-01-01T00:00:00Z",
  sync_item_id: null,
  manifest: null,
};
const manifest = {
  item_id: "synthetic-item",
  consent_expires_at: "2030-01-01T00:00:00Z",
  accounts: [
    {
      account_id: "synthetic-account",
      type: "depository",
      subtype: "checking",
      iso_currency_code: "USD",
    },
  ],
};
const account = {
  stream_id: id(7),
  source_account_id: id(8),
  account_id: "synthetic-account",
  state: "active",
  binding_revision: "2",
};
const listing = {
  items: [
    {
      sync_item_id: id(6),
      item_id: "synthetic-item",
      state: "active",
      item_generation: "1",
      accounts: [account],
    },
  ],
};
// Original synthetic responses exercise UI policy without claiming SQL or provider proof.
function props(api: API, defaultDiscovery = true): ConnectionsProps {
  return {
    api: defaultDiscovery
      ? ((async (path, body, key) =>
          path.includes("/connection-attempts?")
            ? { items: [], next_cursor: null, unfinished_count: "0" }
            : api(path, body, key)) as API)
      : api,
    entity,
    book,
    currencies,
    locked: false,
    denied: vi.fn(),
    onReview: vi.fn(),
  };
}
// Deferred replies expose stale renders and post-unmount effects deterministically.
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}

it("TestConnectionsScopeAndRecovery distinguishes missing deployment from empty", async () => {
  const api = vi
    .fn()
    .mockRejectedValue(new APIError(404, false)) as unknown as API;
  render(<Connections {...props(api)} />);
  await screen.findByRole("alert");
  expect(screen.queryByText("No connected Items in this book.")).toBeNull();
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  expect(screen.getByText("Connection status unavailable.")).toBeTruthy();
});
it("TestConnectionsScopeAndRecovery suppresses old scope and revoked data", async () => {
  const old = deferred<unknown>();
  const api = vi
    .fn()
    .mockReturnValueOnce(old.promise)
    .mockRejectedValue(new APIError(403, false)) as unknown as API;
  const p = props(api);
  const view = render(<Connections {...p} />);
  view.rerender(<Connections {...p} book={{ ...book, book_id: id(20) }} />);
  await screen.findByRole("alert");
  old.resolve(listing);
  await waitFor(() => expect(p.denied).toHaveBeenCalledOnce());
  expect(screen.queryByText("Provider Item synthetic-item")).toBeNull();
});
it("TestConnectionsScopeAndRecovery does not execute Link after scope locks", async () => {
  const handoff = deferred<{ publicToken: string } | null>();
  let signal: AbortSignal | undefined;
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections")
      ? { items: [] }
      : { attempt, nonce: id(9), link_token: "synthetic-link" },
  ) as unknown as API;
  const p = {
    ...props(api),
    link: {
      open: vi.fn(
        ({ signal: s }: { linkToken: string; signal: AbortSignal }) => {
          signal = s;
          return handoff.promise;
        },
      ),
    },
  };
  const view = render(<Connections {...p} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.click(screen.getByRole("button", { name: "Connect account" }));
  await waitFor(() => expect(p.link.open).toHaveBeenCalledOnce());
  view.rerender(<Connections {...p} locked />);
  expect(signal?.aborted).toBe(true);
  handoff.resolve({ publicToken: "synthetic-public" });
  await Promise.resolve();
  expect(
    vi.mocked(api).mock.calls.some(([path]) => path.endsWith("/execute")),
  ).toBe(false);
});
it("TestConnectionsScopeAndRecovery keeps unknown disconnect request exact", async () => {
  const mutations: [string, string | undefined, string | undefined][] = [];
  const api = vi.fn(async (path: string, body?: string, key?: string) => {
    if (!body) return listing;
    mutations.push([path, body, key]);
    if (mutations.length === 1) throw new APIError(0, true);
    return { stream_id: id(7), state: "disconnected" };
  }) as unknown as API;
  render(<Connections {...props(api)} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Disconnect synthetic-account" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Confirm disconnect" }));
  fireEvent.click(
    await screen.findByRole("button", { name: "Resolve saved request" }),
  );
  await waitFor(() => expect(mutations).toHaveLength(2));
  expect(mutations[1]).toEqual(mutations[0]);
  expect(mutations[0][1]).toBe('{"expected_binding_revision":"2"}');
});
it("TestConnectionsScopeAndRecovery inspects unknown exchange without retry", async () => {
  let executes = 0;
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.endsWith("/connection-attempts"))
      return { attempt, nonce: id(9), link_token: "synthetic-link" };
    if (path.endsWith("/execute")) {
      executes++;
      throw new APIError(503, true);
    }
    return { ...attempt, state: "exchange_claimed" };
  }) as unknown as API;
  render(
    <Connections
      {...props(api)}
      link={{ open: async () => ({ publicToken: "synthetic-public" }) }}
    />,
  );
  await screen.findByText("No connected Items in this book.");
  fireEvent.click(screen.getByRole("button", { name: "Connect account" }));
  await screen.findByRole("alert");
  fireEvent.click(screen.getByRole("button", { name: "Inspect attempt" }));
  await screen.findByText("Historical attempt status: exchange claimed");
  expect(executes).toBe(1);
  expect(
    screen.queryByRole("button", { name: "Try connecting again" }),
  ).toBeNull();
});
it("TestConnectionsScopeAndRecovery rejects mismatched attempt identity", async () => {
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections")
      ? { items: [] }
      : { ...attempt, entity_id: id(99) },
  ) as unknown as API;
  render(<Connections {...props(api)} link={{ open: vi.fn() }} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.change(screen.getByLabelText("Known connection attempt ID"), {
    target: { value: id(5) },
  });
  fireEvent.click(screen.getByRole("button", { name: "Inspect attempt" }));
  await screen.findByRole("alert");
  expect(
    screen.queryByText("Historical attempt status: link ready"),
  ).toBeNull();
});
it("TestConnectionsScopeAndRecovery ignores StrictMode's obsolete initial read", async () => {
  const old = deferred<unknown>();
  const api = vi
    .fn()
    .mockReturnValueOnce(old.promise)
    .mockResolvedValue({ items: [] }) as unknown as API;
  render(
    <StrictMode>
      <Connections {...props(api)} />
    </StrictMode>,
  );
  await screen.findByText("No connected Items in this book.");
  old.resolve(listing);
  await Promise.resolve();
  expect(screen.queryByText("Provider Item synthetic-item")).toBeNull();
});
it("TestConnectionsEnrollmentBinding explicitly maps verified account and opens review", async () => {
  let bindings: unknown;
  let enrolled = false;
  const api = vi.fn(async (path: string, body?: string) => {
    if (path.endsWith("/connections"))
      return enrolled ? listing : { items: [] };
    if (path.endsWith("/connection-attempts"))
      return { attempt, nonce: id(9), link_token: "synthetic-link" };
    if (path.endsWith("/execute"))
      return { ...attempt, state: "binding_review", manifest };
    if (path.endsWith("/bindings")) {
      bindings = JSON.parse(body!);
      enrolled = true;
      return {
        ...attempt,
        state: "active",
        manifest,
        sync_item_id: id(6),
        streams: [
          {
            stream_id: id(7),
            source_account_id: id(8),
            namespace_id: id(10),
            binding_revision: "1",
            overlap_review_required: false,
          },
        ],
      };
    }
    throw Error("unexpected");
  }) as unknown as API;
  const p = {
    ...props(api),
    link: { open: vi.fn(async () => ({ publicToken: "synthetic-public" })) },
  };
  render(<Connections {...p} />);
  await screen.findByText("No connected Items in this book.");
  const start = screen.getByRole("button", { name: "Connect account" });
  fireEvent.click(start);
  fireEvent.click(start);
  await screen.findByRole("group", {
    name: "Review verified account bindings",
  });
  expect(p.link.open).toHaveBeenCalledOnce();
  fireEvent.change(screen.getByLabelText("Binding date"), {
    target: { value: "2026-01-02" },
  });
  fireEvent.change(
    screen.getByLabelText("Ledger account for synthetic-account"),
    { target: { value: id(3) } },
  );
  fireEvent.change(
    screen.getByLabelText("Currency version for synthetic-account"),
    { target: { value: id(4) } },
  );
  fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirm selected bindings" }),
  );
  await screen.findByText("Historical attempt status: active");
  expect(bindings).toEqual({
    consent_grants: true,
    bindings: [
      {
        account_id: "synthetic-account",
        ledger_account_id: id(3),
        currency_version_id: id(4),
        source_account_id: null,
        expected_binding_revision: null,
        binding_date: "2026-01-02",
        binding_evidence_ref: expect.stringMatching(/^[0-9a-f-]{36}$/),
      },
    ],
  });
  fireEvent.click(
    await screen.findByRole("button", {
      name: "Review transactions for synthetic-account",
    }),
  );
  expect(p.onReview).toHaveBeenCalledWith(id(8));
  expect(localStorage.length).toBe(0);
  expect(sessionStorage.length).toBe(0);
});
it("TestConnectionsAutomaticBinding proposes distinct accounts for every supported provider account", async () => {
  let submitted: components["schemas"]["ConnectionBindingRequest"] | undefined;
  const accounts = ["checking", "savings", "money market"].map(
    (subtype, index) => ({
      account_id: `provider-${index + 1}`,
      type: "depository" as const,
      subtype,
      iso_currency_code: "USD",
    }),
  );
  const reviewed = {
    ...attempt,
    state: "binding_review" as const,
    manifest: {
      item_id: "synthetic-item",
      consent_expires_at: "2030-01-01T00:00:00Z",
      accounts: [
        ...accounts,
        {
          account_id: "provider-credit",
          type: "credit" as const,
          subtype: "credit card",
          iso_currency_code: "USD",
        },
      ],
    },
  };
  const api = vi.fn(async (path: string, body?: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.endsWith(`/connection-attempts/${attempt.attempt_id}`))
      return reviewed;
    if (path.endsWith("/bindings")) {
      submitted = JSON.parse(body!);
      return {
        ...reviewed,
        state: "active",
        sync_item_id: id(6),
        streams: accounts.map((_, index) => ({
          stream_id: id(20 + index),
          source_account_id: id(30 + index),
          namespace_id: id(40 + index),
          binding_revision: "1",
          overlap_review_required: false,
        })),
      };
    }
    throw Error("unexpected");
  }) as unknown as API;
  render(<Connections {...props(api)} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.change(screen.getByLabelText("Known connection attempt ID"), {
    target: { value: attempt.attempt_id },
  });
  fireEvent.click(screen.getByRole("button", { name: "Inspect attempt" }));
  await screen.findByRole("group", { name: "Review verified account bindings" });
  fireEvent.change(screen.getByLabelText("Binding date"), {
    target: { value: "2026-10-19" },
  });
  expect(
    screen.getByText("This account cannot be bound in this profile."),
  ).not.toBeNull();
  fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirm selected bindings" }),
  );
  await waitFor(() => expect(submitted).toBeDefined());
  expect(submitted!.bindings).toHaveLength(3);
  expect(new Set(submitted!.bindings.map((binding) => binding.ledger_account_id)).size).toBe(3);
  expect(new Set(submitted!.bindings.map((binding) => binding.new_ledger_account?.code)).size).toBe(3);
  expect(
    submitted!.bindings.every(
      (binding) =>
        binding.new_ledger_account?.account_id === binding.ledger_account_id &&
        binding.new_ledger_account.category === "asset" &&
        binding.new_ledger_account.normal_side === "debit" &&
        binding.new_ledger_account.monetary,
    ),
  ).toBe(true);
});

// Lost Begin has no rediscoverable ID in this profile, so a new key cannot be an implicit retry.
it("TestConnectionsScopeAndRecovery blocks new enrollment after unknown Begin", async () => {
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    throw new APIError(0, true);
  }) as unknown as API;
  render(<Connections {...props(api)} link={{ open: vi.fn() }} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.click(screen.getByRole("button", { name: "Connect account" }));
  await screen.findByRole("alert");
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  expect(
    vi
      .mocked(api)
      .mock.calls.filter(([path]) => path.endsWith("/connection-attempts")),
  ).toHaveLength(1);
});
// Status reconciliation may read a stored version, but never retransmits lost Link credentials.
it("TestConnectionsScopeAndRecovery reconciles known recorded credential only", async () => {
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    return {
      ...attempt,
      state: path.endsWith("/reconcile")
        ? "binding_review"
        : "credential_recovery_required",
      manifest,
    };
  }) as unknown as API;
  render(<Connections {...props(api)} link={{ open: vi.fn() }} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.change(screen.getByLabelText("Known connection attempt ID"), {
    target: { value: id(5) },
  });
  fireEvent.click(screen.getByRole("button", { name: "Inspect attempt" }));
  await screen.findByText(
    "Historical attempt status: credential recovery required",
  );
  fireEvent.click(screen.getByRole("button", { name: "Try connecting again" }));
  await screen.findByRole("group", {
    name: "Review verified account bindings",
  });
  expect(
    vi
      .mocked(api)
      .mock.calls.find(([path]) => path.endsWith("/reconcile"))
      ?.slice(1),
  ).toEqual(["{}", undefined]);
  expect(
    vi.mocked(api).mock.calls.some(([path]) => path.endsWith("/execute")),
  ).toBe(false);
});

// A confirmed absent credential permits a guarded new session, never exchange replay.
it.each([true, false])(
  "resolves a failed setup with retry=%s after confirming no stored credential",
  async (retry) => {
    const recovered = {
      ...attempt,
      state: "credential_recovery_required",
      manifest,
    };
    let abandoned = false;
    const api = vi.fn(async (path: string, body?: string) => {
      if (path.endsWith("/connections")) return { items: [] };
      if (path.includes("/connection-attempts?"))
        return {
          items: [
            {
              ...recovered,
              state: abandoned ? "abandoned" : recovered.state,
              expired_unclaimed: false,
              binding_expired: false,
            },
          ],
          unfinished_count: abandoned ? "0" : "1",
          next_cursor: null,
        };
      if (path.endsWith("/reconcile")) throw new APIError(409, false);
      if (path.endsWith("/abandon")) {
        expect(JSON.parse(body!)).toEqual({ confirm_unrecoverable: true });
        abandoned = true;
        return { ...recovered, state: "abandoned" };
      }
      if (path.endsWith("/connection-attempts"))
        return {
          attempt: { ...attempt, attempt_id: id(10) },
          nonce: id(9),
          link_token: "synthetic-link",
          expires_at: attempt.expires_at,
          hosted_link_url: "https://secure.plaid.com/hl/synthetic",
        };
      return recovered;
    }) as unknown as API;
    render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
    await screen.findByText("Unfinished attempts: 1");
    fireEvent.change(screen.getByLabelText("Known connection attempt ID"), {
      target: { value: attempt.attempt_id },
    });
    fireEvent.click(screen.getByRole("button", { name: "Inspect attempt" }));
    await screen.findByText(
      "Historical attempt status: credential recovery required",
    );
    fireEvent.click(
      screen.getByRole("button", {
        name: retry ? "Try connecting again" : "Cancel setup",
      }),
    );
    if (retry) await screen.findByRole("link", { name: "Open Plaid Sandbox" });
    else
      await screen.findByText(
        "Setup canceled. You can connect an account whenever you’re ready.",
      );
    expect(
      vi
        .mocked(api)
        .mock.calls.filter(([path]) => path.endsWith("/connection-attempts")),
    ).toHaveLength(retry ? 1 : 0);
    expect(screen.getByText("Unfinished attempts: 0")).toBeTruthy();
    expect(
      vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/reconcile")),
    ).toHaveLength(1);
    expect(
      vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/execute")),
    ).toHaveLength(0);
    expect(
      vi
        .mocked(api)
        .mock.calls.filter(([path]) => path.endsWith(`/${attempt.attempt_id}`)),
    ).toHaveLength(2);
    expect(
      vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/abandon")),
    ).toHaveLength(1);
  },
);

// A confirmed rejection needs a fresh review rather than an endless immutable retry envelope.
it("TestConnectionsScopeAndRecovery clears confirmed disconnect rejection", async () => {
  const api = vi.fn(async (_path: string, body?: string) => {
    if (!body) return listing;
    throw new APIError(409, false);
  }) as unknown as API;
  render(<Connections {...props(api)} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Disconnect synthetic-account" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Confirm disconnect" }));
  await screen.findByRole("alert");
  expect(
    screen.queryByRole("button", { name: "Resolve saved request" }),
  ).toBeNull();
  expect(
    (
      screen.getByRole("button", {
        name: "Refresh connections",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(false);
});
// A missing or unsupported state is unavailable, never an empty or active connection.
it("TestConnectionsScopeAndRecovery rejects malformed list state", async () => {
  const api = vi.fn().mockResolvedValue({
    items: [{ ...listing.items[0], state: "invented" }],
  }) as unknown as API;
  render(<Connections {...props(api)} />);
  await screen.findByRole("alert");
  expect(screen.queryByText("Provider Item synthetic-item")).toBeNull();
  expect(screen.queryByText("No connected Items in this book.")).toBeNull();
});

const status = {
  stream_revision: "3",
  stream_id: id(7),
  source_account_id: id(8),
  sync_item_id: id(6),
  state: "active",
  item_state: "active",
  item_generation: "1",
  binding_revision: "2",
  requested_generation: "9007199254740993",
  completed_generation: "0",
  queued: true,
  lease_active: false,
  last_completed_at: null,
  provider_updated_at: null,
  readiness: "NOT_READY",
  last_error: null,
};
const emptyAttempts = { items: [], next_cursor: null, unfinished_count: "0" };
// Discovery is durable authority; an unfinished attempt blocks a new Begin after remount.
it("TestConnectionsScopeAndRecovery discovers unfinished attempts after reload", async () => {
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections")
      ? listing
      : path.includes("/connection-attempts?")
        ? {
            items: [
              {
                ...attempt,
                state: "credential_recovery_required",
                manifest,
                expired_unclaimed: false,
                binding_expired: false,
              },
            ],
            next_cursor: null,
            unfinished_count: "1",
          }
        : { ...attempt, state: "credential_recovery_required", manifest },
  ) as unknown as API;
  const p = { ...props(api, false), link: { open: vi.fn() } };
  const first = render(<Connections {...p} />);
  await screen.findByText("Unfinished attempts: 1");
  first.unmount();
  render(<Connections {...p} />);
  await screen.findByText("Unfinished attempts: 1");
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  fireEvent.click(
    screen.getByRole("button", { name: `Review attempt ${id(5)}` }),
  );
  await screen.findByText(
    "Historical attempt status: credential recovery required",
  );
  expect(p.link.open).not.toHaveBeenCalled();
  expect(localStorage.length + sessionStorage.length).toBe(0);
});
it("TestConnectionsDiscovery waits for complete authenticated result and rejects invalid count", async () => {
  const discovery = deferred<unknown>();
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections") ? listing : discovery.promise,
  ) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  discovery.resolve({ ...emptyAttempts, unfinished_count: 0 });
  await screen.findByRole("alert");
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
});
it("TestConnectionsDiscovery shows expired unclaimed without replay and paginates", async () => {
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return listing;
    if (path.includes("before=")) return emptyAttempts;
    if (path.includes("/connection-attempts?"))
      return {
        items: [
          { ...attempt, expired_unclaimed: true, binding_expired: false },
        ],
        next_cursor: id(5),
        unfinished_count: "0",
      };
    return attempt;
  }) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  await screen.findByText("Unfinished attempts: 0");
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(false);
  fireEvent.click(
    screen.getByRole("button", { name: `Review attempt ${id(5)}` }),
  );
  await screen.findByText(/Expired before claim\. This attempt cannot/);
  fireEvent.click(
    screen.getByRole("button", { name: "Older connection attempts" }),
  );
  await screen.findByText("No attempts on this page.");
  expect(vi.mocked(api).mock.calls.some(([p]) => p.endsWith("execute"))).toBe(
    false,
  );
});
// Request receipt is never rendered as a provider timestamp or posted amount.
it("TestConnectionsStreamStatus shows exact generations and honest unknown timestamps", async () => {
  const api = vi.fn(async (path: string) =>
    path.endsWith("/status") ? status : listing,
  ) as unknown as API;
  render(<Connections {...props(api)} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: "View sync status for synthetic-account",
    }),
  );
  await screen.findByText(
    "Requested generation 9007199254740993; completed generation 0.",
  );
  expect(
    screen.getByText("Last complete publication: None recorded"),
  ).toBeTruthy();
  expect(screen.getByText(/Last provider update: Unknown/)).toBeTruthy();
  expect(screen.getByText(/Acquisition does not post journals/)).toBeTruthy();
});
it("TestConnectionsStreamStatus confirms one account and resolves unknown request unchanged", async () => {
  const mutations: unknown[][] = [];
  const api = vi.fn(async (path: string, body?: string, key?: string) => {
    if (path.endsWith("/operations")) {
      mutations.push([path, body, key]);
      if (mutations.length === 1) throw new APIError(0, true);
      return { ...status, requested_generation: "9007199254740994" };
    }
    return path.endsWith("/status") ? status : listing;
  }) as unknown as API;
  render(<Connections {...props(api)} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: "View sync status for synthetic-account",
    }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Request sync now" }),
  );
  expect(screen.getByText(/for synthetic-account only/)).toBeTruthy();
  expect(mutations).toHaveLength(0);
  fireEvent.click(screen.getByRole("button", { name: "Confirm sync request" }));
  await screen.findByRole("alert");
  fireEvent.click(
    screen.getByRole("button", { name: "Resolve saved request" }),
  );
  await screen.findByText(/Operation recorded\./);
  expect(mutations).toHaveLength(2);
  expect(mutations[1]).toEqual(mutations[0]);
  expect(JSON.parse(mutations[0][1] as string)).toEqual({
    action: "request",
    expected_item_generation: "1",
    expected_binding_revision: "2",
    expected_stream_revision: "3",
  });
  // A replayed historical receipt must not overwrite the fresh status read.
  expect(
    screen.getByText(
      "Requested generation 9007199254740993; completed generation 0.",
    ),
  ).toBeTruthy();
});
it("TestConnectionsStreamStatus rejects wrong stream and suppresses response after lock", async () => {
  const reply = deferred<unknown>();
  const api = vi.fn(async (path: string) =>
    path.endsWith("/status") ? reply.promise : listing,
  ) as unknown as API;
  const p = props(api);
  const view = render(<Connections {...p} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: "View sync status for synthetic-account",
    }),
  );
  view.rerender(<Connections {...p} locked />);
  reply.resolve({ ...status, stream_id: id(99) });
  await Promise.resolve();
  expect(
    screen.queryByRole("region", { name: "Account sync status" }),
  ).toBeNull();
});

// Pause/resume uses stream revision without inventing a change to the source binding.
it.each(["pause", "resume"])(
  "TestConnectionsStreamStatus pins explicit %s to one stream revision",
  async (action) => {
    let current = {
      ...status,
      state: action === "pause" ? "active" : "paused",
    };
    const sent: unknown[] = [];
    const api = vi.fn(async (path: string, body?: string) => {
      if (path.endsWith("/operations")) {
        sent.push(JSON.parse(body!));
        current = {
          ...current,
          state: action === "pause" ? "paused" : "active",
          stream_revision: "4",
        };
        return current;
      }
      return path.endsWith("/status") ? current : listing;
    }) as unknown as API;
    render(<Connections {...props(api)} />);
    fireEvent.click(
      await screen.findByRole("button", {
        name: "View sync status for synthetic-account",
      }),
    );
    fireEvent.click(
      await screen.findByRole("button", {
        name: action === "pause" ? "Pause account sync" : "Resume account sync",
      }),
    );
    expect(sent).toHaveLength(0);
    fireEvent.click(screen.getByRole("button", { name: `Confirm ${action}` }));
    await screen.findByText(/Operation recorded\./);
    expect(sent).toEqual([
      {
        action,
        expected_item_generation: "1",
        expected_binding_revision: "2",
        expected_stream_revision: "3",
      },
    ]);
    expect(
      (
        screen.getByRole("button", {
          name:
            action === "pause" ? "Resume account sync" : "Pause account sync",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(false);
  },
);
it("TestConnectionsStreamStatus clears stale operation after confirmed conflict", async () => {
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/operations")) throw new APIError(409, false);
    return path.endsWith("/status") ? status : listing;
  }) as unknown as API;
  render(<Connections {...props(api)} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: "View sync status for synthetic-account",
    }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Pause account sync" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Confirm pause" }));
  await screen.findByRole("alert");
  expect(screen.queryByRole("button", { name: "Confirm pause" })).toBeNull();
  expect(
    screen.queryByRole("button", { name: "Resolve saved request" }),
  ).toBeNull();
});
it("TestConnectionsStreamStatus revocation clears discovery and blocks new enrollment", async () => {
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/status")) throw new APIError(403, false);
    return listing;
  }) as unknown as API;
  const p = props(api);
  render(<Connections {...p} link={{ open: vi.fn() }} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: "View sync status for synthetic-account",
    }),
  );
  await screen.findByRole("alert");
  expect(p.denied).toHaveBeenCalledOnce();
  expect(screen.queryByText("Provider Item synthetic-item")).toBeNull();
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
});

// Known expired publication is distinct from an unknown remote outcome and never replays exchange.
it("TestConnectionsDiscovery permits fresh enrollment after expired binding inspection", async () => {
  const expired = {
    ...attempt,
    state: "binding_review",
    manifest,
    expires_at: "2000-01-01T00:00:00Z",
    expired_unclaimed: false,
    binding_expired: true,
  };
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections")
      ? listing
      : path.includes("/connection-attempts?")
        ? { items: [expired], next_cursor: null, unfinished_count: "0" }
        : expired,
  ) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  fireEvent.click(
    await screen.findByRole("button", { name: `Review attempt ${id(5)}` }),
  );
  await screen.findByText(/Binding review expired; start a fresh connection/);
  await waitFor(() =>
    expect(
      (
        screen.getByRole("button", {
          name: "Connect account",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(false),
  );
  expect(
    screen.queryByRole("group", { name: "Review verified account bindings" }),
  ).toBeNull();
  expect(screen.queryByText(/Expired before claim\. This attempt/)).toBeNull();
  expect(vi.mocked(api).mock.calls.some(([p]) => p.endsWith("/execute"))).toBe(
    false,
  );
});
it("TestConnectionsDiscovery keeps expired unknown claimed work blocking enrollment", async () => {
  const unknown = {
    ...attempt,
    state: "exchange_claimed",
    expires_at: "2000-01-01T00:00:00Z",
    expired_unclaimed: false,
    binding_expired: false,
  };
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections")
      ? listing
      : path.includes("/connection-attempts?")
        ? { items: [unknown], next_cursor: null, unfinished_count: "1" }
        : unknown,
  ) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  fireEvent.click(
    await screen.findByRole("button", { name: `Review attempt ${id(5)}` }),
  );
  await screen.findByText("Historical attempt status: exchange claimed");
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
  expect(screen.queryByText(/Binding review expired;/)).toBeNull();
});

// Hosted navigation is explicit, memory-only and independent of SDK callbacks.
it("Hosted completion stays pending, then exposes verified binding review", async () => {
  let checks = 0;
  const api = vi.fn(async (path: string, body?: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.endsWith("/complete")) {
      expect(JSON.parse(body!)).toEqual({});
      return ++checks === 1
        ? attempt
        : { ...attempt, state: "binding_review", manifest };
    }
    return {
      attempt,
      nonce: id(9),
      link_token: "synthetic-link",
      expires_at: attempt.expires_at,
      hosted_link_url: "https://secure.plaid.com/hl/synthetic",
    };
  }) as unknown as API;
  const link = { open: vi.fn() };
  render(<Connections {...props(api)} link={link} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.click(screen.getByRole("button", { name: "Connect account" }));
  const anchor = await screen.findByRole("link", {
    name: "Open Plaid Sandbox",
  });
  expect(anchor.getAttribute("rel")).toBe("noopener noreferrer");
  expect(anchor.getAttribute("referrerpolicy")).toBe("no-referrer");
  expect(link.open).not.toHaveBeenCalled();
  fireEvent.click(
    screen.getByRole("button", { name: "Check bank setup completion" }),
  );
  await screen.findByText(/Plaid has not reported completion yet/);
  fireEvent.click(
    screen.getByRole("button", { name: "Check bank setup completion" }),
  );
  await screen.findByRole("group", {
    name: "Review verified account bindings",
  });
  expect(screen.queryByRole("link", { name: "Open Plaid Sandbox" })).toBeNull();
  expect(checks).toBe(2);
});

// A reload loses the browser handoff but still offers server-side Plaid completion.
it("checks a discovered Hosted attempt after reload without a browser token", async () => {
  const api = vi.fn(async (path: string, body?: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.includes("/connection-attempts?"))
      return {
        items: [
          { ...attempt, expired_unclaimed: false, binding_expired: false },
        ],
        next_cursor: null,
        unfinished_count: "1",
      };
    if (path.endsWith(`/${attempt.attempt_id}`)) return attempt;
    if (path.endsWith(`/${attempt.attempt_id}/complete`)) {
      expect(body).toBe("{}");
      return { ...attempt, state: "binding_review", manifest };
    }
    throw Error("unexpected request");
  }) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: `Review attempt ${attempt.attempt_id}`,
    }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Check bank setup completion" }),
  );
  await screen.findByRole("group", {
    name: "Review verified account bindings",
  });
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/complete")),
  ).toHaveLength(1);
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/execute")),
  ).toHaveLength(0);
});

// Invalid navigation and unknown exchange outcomes must never retain a runnable handoff.
it.each(["unsafe URL", "uncertain reply", "scope change", "cancel"])(
  "Hosted handoff clears on %s",
  async (scenario) => {
    const api = vi.fn(async (path: string) => {
      if (path.endsWith("/connections")) return { items: [] };
      if (path.endsWith("/abandon")) return { ...attempt, state: "abandoned" };
      if (path.endsWith("/complete")) throw new APIError(0, true);
      return {
        attempt,
        nonce: id(9),
        link_token: "synthetic-link",
        expires_at: attempt.expires_at,
        hosted_link_url:
          scenario === "unsafe URL"
            ? "https://evil.test/hl/token"
            : "https://secure.plaid.com/hl/synthetic",
      };
    }) as unknown as API;
    const p = { ...props(api), link: { open: vi.fn() } };
    const view = render(<Connections {...p} />);
    await screen.findByText("No connected Items in this book.");
    fireEvent.click(screen.getByRole("button", { name: "Connect account" }));
    if (scenario === "unsafe URL") await screen.findByRole("alert");
    else {
      await screen.findByRole("link", { name: "Open Plaid Sandbox" });
      if (scenario === "scope change")
        view.rerender(<Connections {...p} locked />);
      else if (scenario === "cancel")
        fireEvent.click(
          screen.getByRole("button", { name: "Cancel bank setup" }),
        );
      else {
        fireEvent.click(
          screen.getByRole("button", { name: "Check bank setup completion" }),
        );
        await screen.findByRole("alert");
      }
    }
    await waitFor(() =>
      expect(
        screen.queryByRole("link", { name: "Open Plaid Sandbox" }),
      ).toBeNull(),
    );
    expect(p.link.open).not.toHaveBeenCalled();
  },
);

// An uncertain reconciliation must retain the attempt and never begin again.
it("does not restart after an uncertain reconciliation", async () => {
  const recovery = {
    ...attempt,
    state: "credential_recovery_required",
    manifest,
    expired_unclaimed: false,
    binding_expired: false,
  };
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.includes("/connection-attempts?"))
      return {
        items: [recovery],
        unfinished_count: "1",
        next_cursor: null,
      };
    if (path.endsWith("/reconcile")) throw new APIError(503, true);
    return recovery;
  }) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: `Review attempt ${attempt.attempt_id}`,
    }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Try connecting again" }),
  );
  await screen.findByText(/This operation could not be confirmed/);
  expect(screen.getByText("Unfinished attempts: 1")).toBeTruthy();
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/abandon")),
  ).toHaveLength(0);
  expect(
    vi
      .mocked(api)
      .mock.calls.filter(([path]) => path.endsWith("/connection-attempts")),
  ).toHaveLength(0);
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/execute")),
  ).toHaveLength(0);
});

// A server conflict can mean a credential was stored concurrently; never restart.
it("stops retry when the stored-credential guard rejects abandonment", async () => {
  const recovery = {
    ...attempt,
    state: "credential_recovery_required",
    manifest,
  };
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.includes("/connection-attempts?"))
      return {
        items: [
          { ...recovery, expired_unclaimed: false, binding_expired: false },
        ],
        unfinished_count: "1",
        next_cursor: null,
      };
    if (path.endsWith("/reconcile") || path.endsWith("/abandon"))
      throw new APIError(409, false);
    return recovery;
  }) as unknown as API;
  render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: `Review attempt ${attempt.attempt_id}`,
    }),
  );
  fireEvent.click(
    await screen.findByRole("button", { name: "Try connecting again" }),
  );
  await screen.findByText(/Connection status changed/);
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/abandon")),
  ).toHaveLength(1);
  expect(
    vi
      .mocked(api)
      .mock.calls.filter(([path]) => path.endsWith("/connection-attempts")),
  ).toHaveLength(0);
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/execute")),
  ).toHaveLength(0);
});

// Item transport failure cannot erase independently successful attempt discovery.
it("keeps durable recovery discovery when connection listing fails", async () => {
  const recovered = {
    ...attempt,
    state: "credential_recovery_required",
    manifest,
    expired_unclaimed: false,
    binding_expired: false,
  };
  const api = vi.fn(async (path: string) => {
    if (path.includes("/connection-attempts?"))
      return { items: [recovered], next_cursor: null, unfinished_count: "1" };
    throw new APIError(0, true);
  }) as unknown as API;
  render(<Connections {...props(api, false)} />);
  await screen.findByRole("button", {
    name: `Review attempt ${attempt.attempt_id}`,
  });
  expect(screen.getByText("Unfinished attempts: 1")).toBeTruthy();
  expect(screen.queryByText(/Attempt discovery unavailable/)).toBeNull();
});

// A denied historical attempt must not eject an otherwise authorized book.
it("keeps the selected Connections scope after an attempt-only denial", async () => {
  const recovered = {
    ...attempt,
    state: "credential_recovery_required",
    manifest,
    expired_unclaimed: false,
    binding_expired: false,
  };
  const denied = vi.fn();
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.includes("/connection-attempts?"))
      return { items: [recovered], next_cursor: null, unfinished_count: "1" };
    if (path.endsWith(`/${attempt.attempt_id}`)) throw new APIError(403, false);
    throw new Error("unexpected request");
  }) as unknown as API;
  render(<Connections {...props(api, false)} denied={denied} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: `Review attempt ${attempt.attempt_id}`,
    }),
  );
  await screen.findByText(/This attempt is unavailable/);
  expect(screen.getByText("Unfinished attempts: 1")).toBeTruthy();
  expect(denied).not.toHaveBeenCalled();
  expect(
    (
      screen.getByRole("button", {
        name: "Connect account",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
});

// Revoked book access still clears the financial workspace after revalidation.
it("clears the scope when attempt and discovery access are both denied", async () => {
  const denied = vi.fn();
  let discoveryReads = 0;
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.includes("/connection-attempts?")) {
      if (++discoveryReads > 1) throw new APIError(403, false);
      return {
        items: [
          { ...attempt, expired_unclaimed: false, binding_expired: false },
        ],
        next_cursor: null,
        unfinished_count: "1",
      };
    }
    if (path.endsWith(`/${attempt.attempt_id}`)) throw new APIError(403, false);
    throw new Error("unexpected request");
  }) as unknown as API;
  render(<Connections {...props(api, false)} denied={denied} />);
  fireEvent.click(
    await screen.findByRole("button", {
      name: `Review attempt ${attempt.attempt_id}`,
    }),
  );
  await waitFor(() => expect(denied).toHaveBeenCalledOnce());
  expect(screen.queryByText("Unfinished attempts: 1")).toBeNull();
});

// A committed exchange may return 409 during storage; recovery uses GET only.
it("inspects the exact Hosted attempt after conflict without repeating exchange", async () => {
  const recovered = {
    ...attempt,
    state: "credential_recovery_required",
    manifest,
  };
  const api = vi.fn(async (path: string) => {
    if (path.endsWith("/connections")) return { items: [] };
    if (path.endsWith("/complete")) throw new APIError(409, false);
    if (path.endsWith(`/${attempt.attempt_id}`)) return recovered;
    return {
      attempt,
      nonce: id(9),
      link_token: "synthetic-link",
      expires_at: attempt.expires_at,
      hosted_link_url: "https://secure.plaid.com/hl/synthetic",
    };
  }) as unknown as API;
  render(<Connections {...props(api)} link={{ open: vi.fn() }} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.click(screen.getByRole("button", { name: "Connect account" }));
  await screen.findByRole("link", { name: "Open Plaid Sandbox" });
  fireEvent.click(
    screen.getByRole("button", { name: "Check bank setup completion" }),
  );
  await screen.findByText(/Bank setup needs credential recovery/);
  expect(
    screen.getByText("Historical attempt status: credential recovery required"),
  ).toBeTruthy();
  expect(
    screen.queryByRole("button", { name: "Check bank setup completion" }),
  ).toBeNull();
  expect(
    vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/complete")),
  ).toHaveLength(1);
  expect(
    (
      screen.getByRole("button", {
        name: "Try connecting again",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(false);
});

// Reloaded unclaimed setups can be canceled or restarted without IDs or operator SQL.
it.each([false, true])(
  "cancels a discovered setup, retry=%s",
  async (retry) => {
    let abandoned = false;
    const api = vi.fn(async (path: string, body?: string) => {
      if (path.endsWith("/connections")) return { items: [] };
      if (path.includes("/connection-attempts?"))
        return {
          items: [
            {
              ...attempt,
              state: abandoned ? "abandoned" : "link_ready",
              expired_unclaimed: false,
              binding_expired: false,
            },
          ],
          unfinished_count: abandoned ? "0" : "1",
          next_cursor: null,
        };
      if (path.endsWith("/abandon")) {
        expect(JSON.parse(body!)).toEqual({ confirm_unrecoverable: true });
        abandoned = true;
        return { ...attempt, state: "abandoned" };
      }
      if (path.endsWith("/connection-attempts"))
        return {
          attempt: { ...attempt, attempt_id: id(10) },
          nonce: id(9),
          link_token: "synthetic-link",
          expires_at: attempt.expires_at,
          hosted_link_url: "https://secure.plaid.com/hl/synthetic",
        };
      throw Error("unexpected request");
    }) as unknown as API;
    render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
    fireEvent.click(
      await screen.findByRole("button", {
        name: retry ? "Try again" : "Cancel setup",
      }),
    );
    if (retry) await screen.findByRole("link", { name: "Open Plaid Sandbox" });
    else
      await waitFor(() =>
        expect(
          (
            screen.getByRole("button", {
              name: "Connect account",
            }) as HTMLButtonElement
          ).disabled,
        ).toBe(false),
      );
    expect(
      vi
        .mocked(api)
        .mock.calls.filter(([path]) => path.endsWith("/connection-attempts")),
    ).toHaveLength(retry ? 1 : 0);
    expect(
      vi.mocked(api).mock.calls.filter(([path]) => path.endsWith("/execute")),
    ).toHaveLength(0);
  },
);

// Unknown cancellation outcomes only allow retry after an authoritative abandoned read.
it.each(["abandoned", "exchange_claimed"])(
  "resolves lost cancellation reply as %s",
  async (state) => {
    let requested = false;
    const api = vi.fn(async (path: string) => {
      if (path.endsWith("/connections")) return { items: [] };
      if (path.includes("/connection-attempts?"))
        return {
          items: [
            {
              ...attempt,
              state: requested ? state : "link_ready",
              expired_unclaimed: false,
              binding_expired: false,
            },
          ],
          unfinished_count: requested && state === "abandoned" ? "0" : "1",
          next_cursor: null,
        };
      if (path.endsWith("/abandon")) {
        requested = true;
        throw new APIError(503, true);
      }
      if (path.endsWith(`/${attempt.attempt_id}`)) return { ...attempt, state };
      if (path.endsWith("/connection-attempts"))
        return {
          attempt: { ...attempt, attempt_id: id(10) },
          nonce: id(9),
          link_token: "synthetic-link",
          expires_at: attempt.expires_at,
          hosted_link_url: "https://secure.plaid.com/hl/synthetic",
        };
      throw Error("unexpected request");
    }) as unknown as API;
    render(<Connections {...props(api, false)} link={{ open: vi.fn() }} />);
    fireEvent.click(await screen.findByRole("button", { name: "Try again" }));
    if (state === "abandoned")
      await screen.findByRole("link", { name: "Open Plaid Sandbox" });
    else
      await screen.findByText(
        "Setup could not be canceled. Check its status before trying again.",
      );
    expect(
      vi
        .mocked(api)
        .mock.calls.filter(([path]) => path.endsWith("/connection-attempts")),
    ).toHaveLength(state === "abandoned" ? 1 : 0);
  },
);
