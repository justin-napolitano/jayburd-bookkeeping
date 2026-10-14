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
function props(api: API): ConnectionsProps {
  return {
    api,
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
    (
      screen.getByRole("button", {
        name: "Reconcile recorded credential",
      }) as HTMLButtonElement
    ).disabled,
  ).toBe(true);
});
it("TestConnectionsScopeAndRecovery rejects mismatched attempt identity", async () => {
  const api = vi.fn(async (path: string) =>
    path.endsWith("/connections")
      ? { items: [] }
      : { ...attempt, entity_id: id(99) },
  ) as unknown as API;
  render(<Connections {...props(api)} />);
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
  render(<Connections {...props(api)} />);
  await screen.findByText("No connected Items in this book.");
  fireEvent.change(screen.getByLabelText("Known connection attempt ID"), {
    target: { value: id(5) },
  });
  fireEvent.click(screen.getByRole("button", { name: "Inspect attempt" }));
  await screen.findByText(
    "Historical attempt status: credential recovery required",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Reconcile recorded credential" }),
  );
  await screen.findByRole("group", {
    name: "Review verified account bindings",
  });
  expect(
    vi
      .mocked(api)
      .mock.calls.find(([path]) => path.endsWith("/reconcile"))
      ?.slice(1),
  ).toEqual(["{}"]);
  expect(
    vi.mocked(api).mock.calls.some(([path]) => path.endsWith("/execute")),
  ).toBe(false);
});

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
  const api = vi
    .fn()
    .mockResolvedValue({
      items: [{ ...listing.items[0], state: "invented" }],
    }) as unknown as API;
  render(<Connections {...props(api)} />);
  await screen.findByRole("alert");
  expect(screen.queryByText("Provider Item synthetic-item")).toBeNull();
  expect(screen.queryByText("No connected Items in this book.")).toBeNull();
});
