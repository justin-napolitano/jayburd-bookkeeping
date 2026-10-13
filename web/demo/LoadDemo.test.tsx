import React from "react";
import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { LoadDemo, type DemoAcquisition } from "./LoadDemo";
import { APIError, type API } from "../src/bookkeeping/api";
afterEach(cleanup);
const path = "/v1/demo/acquisition";
const initial: DemoAcquisition = {
  profile: "synthetic-acquisition-v1",
  total_count: "3000",
  imported_count: "0",
  complete: false,
  blocked: false,
  pending_command_id: null,
  batches: [],
};
const complete = { ...initial, imported_count: "3000", complete: true };
// Deferred stage completion exposes Pause and unmount races at the request boundary.
function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { resolve, promise };
}
it("mount only discovers and explicit loading advances server steps", async () => {
  const api = vi
      .fn()
      .mockResolvedValueOnce(initial)
      .mockResolvedValueOnce(initial)
      .mockResolvedValueOnce({ ...initial, imported_count: "1000" })
      .mockResolvedValueOnce(complete),
    done = vi.fn(),
    active = vi.fn();
  render(
    <LoadDemo
      api={api as API}
      locked={false}
      onComplete={done}
      onActiveChange={active}
    />,
  );
  await screen.findByText("0 of 3000 transactions imported");
  expect(api.mock.calls).toEqual([[path]]);
  fireEvent.click(screen.getByRole("button", { name: "Load demo books" }));
  await screen.findByText(/3000 of 3000/);
  expect(api.mock.calls).toEqual([
    [path],
    [path],
    [`${path}/step`, "{}"],
    [`${path}/step`, "{}"],
  ]);
  expect(done).toHaveBeenCalledOnce();
  expect(active.mock.calls).toEqual([[true], [false]]);
});
it("pause finishes current step; resume reconciles before writing", async () => {
  const step = deferred<DemoAcquisition>(),
    api = vi
      .fn()
      .mockResolvedValueOnce(initial)
      .mockResolvedValueOnce(initial)
      .mockImplementationOnce(() => step.promise)
      .mockResolvedValueOnce({ ...initial, imported_count: "1000" })
      .mockResolvedValueOnce(complete);
  render(
    <LoadDemo
      api={api as API}
      locked={false}
      onComplete={vi.fn()}
      onActiveChange={vi.fn()}
    />,
  );
  await screen.findByText(/0 of 3000/);
  fireEvent.click(screen.getByRole("button", { name: "Load demo books" }));
  await waitFor(() => expect(api).toHaveBeenCalledTimes(3));
  fireEvent.click(
    screen.getByRole("button", { name: "Pause after this step" }),
  );
  step.resolve({ ...initial, imported_count: "1000" });
  fireEvent.click(
    await screen.findByRole("button", { name: "Resume demo loading" }),
  );
  await screen.findByText(/3000 of 3000/);
  expect(api.mock.calls[3]).toEqual([path]);
  expect(api).toHaveBeenCalledTimes(5);
});
it("unknown step reconciles without another automatic mutation", async () => {
  const api = vi
    .fn()
    .mockResolvedValueOnce(initial)
    .mockResolvedValueOnce(initial)
    .mockRejectedValueOnce(new APIError(503, true))
    .mockResolvedValueOnce({ ...initial, imported_count: "1000" });
  render(
    <LoadDemo
      api={api as API}
      locked={false}
      onComplete={vi.fn()}
      onActiveChange={vi.fn()}
    />,
  );
  await screen.findByText(/0 of 3000/);
  fireEvent.click(screen.getByRole("button", { name: "Load demo books" }));
  await screen.findByText(/1000 of 3000/);
  expect(api.mock.calls.at(-1)).toEqual([path]);
  expect(api.mock.calls.filter((c) => c.length === 2)).toHaveLength(1);
  expect(screen.getByRole("alert").textContent).toMatch(/unconfirmed/);
});
it("foreign pending command blocks writes and completion", async () => {
  const api = vi
      .fn()
      .mockResolvedValue({
        ...initial,
        blocked: true,
        pending_command_id: "other",
      }),
    done = vi.fn();
  render(
    <LoadDemo
      api={api as API}
      locked={false}
      onComplete={done}
      onActiveChange={vi.fn()}
    />,
  );
  await screen.findByText(/Another command needs attention/);
  fireEvent.click(screen.getByRole("button", { name: "Load demo books" }));
  await waitFor(() => expect(api).toHaveBeenCalledTimes(2));
  expect(done).not.toHaveBeenCalled();
  expect(api.mock.calls.every((c) => c.length === 1)).toBe(true);
});
it("unmount prevents the next mutation after a late reply", async () => {
  const step = deferred<DemoAcquisition>(),
    done = vi.fn(),
    api = vi
      .fn()
      .mockResolvedValueOnce(initial)
      .mockResolvedValueOnce(initial)
      .mockImplementationOnce(() => step.promise);
  const view = render(
    <LoadDemo
      api={api as API}
      locked={false}
      onComplete={done}
      onActiveChange={vi.fn()}
    />,
  );
  await screen.findByText(/0 of 3000/);
  fireEvent.click(screen.getByRole("button", { name: "Load demo books" }));
  await waitFor(() => expect(api).toHaveBeenCalledTimes(3));
  view.unmount();
  step.resolve(initial);
  await new Promise((r) => setTimeout(r, 0));
  expect(api).toHaveBeenCalledTimes(3);
  expect(done).not.toHaveBeenCalled();
});
it("bounds nonconverging server progress to64 mutations per click", async () => {
  const api = vi.fn().mockResolvedValue(initial);
  render(
    <LoadDemo
      api={api as API}
      locked={false}
      onComplete={vi.fn()}
      onActiveChange={vi.fn()}
    />,
  );
  await screen.findByText(/0 of 3000/);
  fireEvent.click(screen.getByRole("button", { name: "Load demo books" }));
  await screen.findByRole("button", { name: "Resume demo loading" });
  expect(api.mock.calls.filter((c) => c.length === 2)).toHaveLength(64);
});
