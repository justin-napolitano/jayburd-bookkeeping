import React from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { ActivityOverview } from "./ActivityOverview";
import { APIError, type API } from "./api";
import {
  activityBook,
  activityEntity,
  activityCurrencies,
  activitySources,
  activityPage,
  type ActivityQuery,
} from "../../tests/activity-fixture";
afterEach(cleanup);

it("shows separate currency versions and a real empty period without a fabricated balance", async () => {
  const { props, query } = setup();
  query.mockImplementationOnce(async (_path, body) => {
    const p = activityPage(JSON.parse(body));
    return {
      ...p,
      units: [
        ...p.units,
        {
          ...p.units[0],
          currency_version_id: "jpy",
          currency: "JPY",
          minor_unit: 0,
          inflow: "200",
          outflow: "100",
          net_inflow: "100",
        },
      ],
    };
  });
  render(<ActivityOverview {...props} />);
  await screen.findByText("200");
  expect(screen.getByRole("heading", { name: /JPY/ })).toBeTruthy();
  expect(screen.getByRole("heading", { name: /USD/ })).toBeTruthy();
  query.mockImplementationOnce(async (_path, body) => ({
    ...activityPage(JSON.parse(body)),
    units: [],
    available_from: null,
    available_through: null,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Refresh overview" }));
  await screen.findByText("No eligible transactions match this period.");
  expect(screen.getByText(/Bank balance unavailable/)).toBeTruthy();
});

it("ignores a late response after the account changes", async () => {
  const { props, api, query } = setup();
  vi.mocked(api).mockResolvedValue({
    ...activitySources,
    items: [
      ...activitySources.items,
      {
        ...activitySources.items[0],
        source_account_id: "second",
        display_name: "Fictional savings",
      },
    ],
  });
  let resolve: (value: ReturnType<typeof activityPage>) => void = () => {};
  query.mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r;
      }),
  );
  render(<ActivityOverview {...props} />);
  await screen.findByRole("option", { name: /Fictional savings/ });
  fireEvent.change(screen.getByLabelText("Bank account"), {
    target: { value: "s" },
  });
  await waitFor(() => expect(query).toHaveBeenCalledTimes(1));
  const old = JSON.parse(query.mock.calls[0][1]);
  query.mockRejectedValueOnce(new APIError(503, true, "read"));
  fireEvent.change(screen.getByLabelText("Bank account"), {
    target: { value: "second" },
  });
  await screen.findByRole("alert");
  resolve(activityPage(old));
  await waitFor(() =>
    expect(screen.queryByText("9,007,199,254,740,993.01")).toBeNull(),
  );
});
// Only read transports exist in this fixture; a mutation is an immediate test failure.
function setup() {
  const query = vi.fn(async (_path: string, body: string) =>
    activityPage(JSON.parse(body)),
  );
  const api = Object.assign(
    vi.fn(async () => activitySources),
    { query },
  ) as unknown as API;
  const denied = vi.fn();
  const props = {
    api,
    entity: activityEntity,
    book: activityBook,
    currencies: activityCurrencies,
    locked: false,
    denied,
    onReview: vi.fn(),
    onAccounts: vi.fn(),
  };
  return { query, api, props, denied };
}
it("starts with every imported date, preserves decimal strings and pins exact merchant continuation", async () => {
  const { props, query } = setup();
  render(<ActivityOverview {...props} />);
  await screen.findByText("9,007,199,254,740,993.01");
  fireEvent.change(screen.getByLabelText("Bank account"), {
    target: { value: "s" },
  });
  expect(screen.getByText("9,007,199,254,740,993.01")).toBeTruthy();
  expect(query.mock.calls[0][1]).toContain('"starts_on":"0001-01-01"');
  expect(query.mock.calls[0][1]).toContain('"through_date":"9999-12-31"');
  expect(screen.getByText(/Bank balance unavailable/)).toBeTruthy();
  fireEvent.click(
    screen.getByRole("button", { name: /View USD .* merchants/ }),
  );
  await screen.findByText("Fictional merchant & cafe");
  fireEvent.click(screen.getByRole("button", { name: "View transactions" }));
  await screen.findByRole("caption");
  const request = JSON.parse(query.mock.calls.at(-1)![1]) as ActivityQuery;
  expect(request.merchant).toEqual({
    status: "attributed",
    label: "Fictional merchant & cafe",
  });
  expect(request.snapshot?.filter_digest).toBe("fictional-pin");
  fireEvent.click(screen.getByRole("button", { name: "Next page" }));
  await screen.findByText("Fictional payment");
  expect(JSON.parse(query.mock.calls.at(-1)![1])).toEqual({
    ...request,
    cursor: "fictional-continuation",
  });
});
it("keeps date edits unapplied and discards stale totals on conflict", async () => {
  const { props, query } = setup();
  render(<ActivityOverview {...props} />);
  await screen.findByText("9,007,199,254,740,993.01");
  fireEvent.change(screen.getByLabelText("Period"), {
    target: { value: "custom" },
  });
  fireEvent.change(screen.getByLabelText("From"), {
    target: { value: "2026-10-01" },
  });
  expect(query).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole("button", { name: "Apply period" }));
  await waitFor(() => expect(query).toHaveBeenCalledTimes(2));
  expect(JSON.parse(query.mock.calls[1][1]).starts_on).toBe("2026-10-01");
  await screen.findByText("9,007,199,254,740,993.01");
  query.mockRejectedValueOnce(new APIError(409, false, "read"));
  fireEvent.click(
    screen.getByRole("button", { name: /View USD .* transactions/ }),
  );
  await screen.findByRole("alert");
  expect(screen.queryByText("9,007,199,254,740,993.01")).toBeNull();
});
it("destroys old scope requests on locking and reauthorizes denied reads", async () => {
  const { props, query, denied } = setup();
  let resolve: (value: ReturnType<typeof activityPage>) => void = () => {};
  query.mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r;
      }),
  );
  const view = render(<ActivityOverview {...props} />);
  await waitFor(() => expect(query).toHaveBeenCalledTimes(1));
  const old = JSON.parse(query.mock.calls[0][1]);
  view.rerender(<ActivityOverview {...props} locked />);
  resolve(activityPage(old));
  expect(screen.queryByText("9,007,199,254,740,993.01")).toBeNull();
  query.mockRejectedValueOnce(new APIError(403, false, "read"));
  view.rerender(<ActivityOverview {...props} />);
  await waitFor(() => expect(denied).toHaveBeenCalledTimes(1));
});
it("rejects a financial response for a different source instead of presenting its numbers", async () => {
  const { props, query } = setup();
  query.mockImplementationOnce(async (_path, body) => ({
    ...activityPage(JSON.parse(body)),
    source_account_id: "other",
  }));
  render(<ActivityOverview {...props} />);
  await screen.findByRole("alert");
  expect(screen.queryByText("9,007,199,254,740,993.01")).toBeNull();
});

it("focuses committed drill headings even when a frame runs before React commits", async () => {
  // A frame may beat an asynchronous state commit; it cannot be the focus prerequisite.
  vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
    callback(0);
    return 1;
  });
  try {
    const { props } = setup();
    render(<ActivityOverview {...props} />);
    await screen.findByText("9,007,199,254,740,993.01");
    fireEvent.click(
      screen.getByRole("button", { name: /View USD .* merchants/ }),
    );
    const merchants = await screen.findByRole("heading", {
      name: "USD merchant activity",
    });
    await waitFor(() => expect(document.activeElement).toBe(merchants));
    fireEvent.click(screen.getByRole("button", { name: "View transactions" }));
    const transactions = await screen.findByRole("heading", {
      name: "Fictional merchant & cafe",
    });
    await waitFor(() => expect(document.activeElement).toBe(transactions));
    fireEvent.click(screen.getByRole("button", { name: "Back to merchants" }));
    const returned = await screen.findByRole("heading", {
      name: "USD merchant activity",
    });
    await waitFor(() => expect(document.activeElement).toBe(returned));
    fireEvent.click(screen.getByRole("button", { name: "Close details" }));
    const summary = await screen.findByRole("heading", {
      name: "Money in and out",
    });
    await waitFor(() => expect(document.activeElement).toBe(summary));
  } finally {
    vi.unstubAllGlobals();
  }
});
