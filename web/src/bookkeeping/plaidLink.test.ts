import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { createPlaidLink } from "./plaidLink";

type Callbacks = {
  token: string;
  onLoad(): void;
  onSuccess(value: unknown): void;
  onExit(error: unknown): void;
};
let callbacks: Callbacks;
const handler = { open: vi.fn(), exit: vi.fn(), destroy: vi.fn() };
// install supplies a fictional SDK without any external script execution.
function install() {
  const create = vi.fn((input: Callbacks) => {
    callbacks = input;
    return handler;
  });
  Object.assign(window, { Plaid: { create } });
  document
    .querySelector<HTMLScriptElement>("script")
    ?.dispatchEvent(new Event("load"));
  return create;
}
beforeEach(() => {
  vi.useFakeTimers();
  document.head.innerHTML = "";
  vi.clearAllMocks();
});
afterEach(() => {
  vi.useRealTimers();
  delete (window as Window & { Plaid?: unknown }).Plaid;
  document.head.innerHTML = "";
});

it("is inert until explicit open and returns only memory-only public token", async () => {
  const link = createPlaidLink();
  expect(document.scripts.length).toBe(0);
  const result = link.open({
    linkToken: "synthetic-link",
    signal: new AbortController().signal,
  });
  expect(document.scripts[0].src).toBe(
    "https://cdn.plaid.com/link/v2/stable/link-initialize.js",
  );
  const sdk = install();
  expect(sdk).toHaveBeenCalledOnce();
  expect(callbacks.token).toBe("synthetic-link");
  callbacks.onLoad();
  callbacks.onSuccess("synthetic-public");
  await expect(result).resolves.toEqual({ publicToken: "synthetic-public" });
  callbacks.onSuccess("late-private-value");
  callbacks.onExit({ secret: "ignored" });
  expect(handler.destroy).toHaveBeenCalledOnce();
  expect(localStorage.length).toBe(0);
  expect(sessionStorage.length).toBe(0);
});
it("aborts load without creating a handler or accepting late script callbacks", async () => {
  const controller = new AbortController();
  const result = createPlaidLink().open({
    linkToken: "synthetic-link",
    signal: controller.signal,
  });
  const script = document.scripts[0];
  controller.abort();
  install();
  script.dispatchEvent(new Event("load"));
  await expect(result).resolves.toBeNull();
  expect(handler.open).not.toHaveBeenCalled();
  expect(document.scripts.length).toBe(0);
});
it("fences scope cancellation before cleanup callbacks and destroys the iframe", async () => {
  const controller = new AbortController();
  const result = createPlaidLink().open({
    linkToken: "synthetic-link",
    signal: controller.signal,
  });
  install();
  callbacks.onLoad();
  handler.exit.mockImplementationOnce(() =>
    callbacks.onSuccess("stale-public"),
  );
  controller.abort();
  callbacks.onSuccess("stale-public");
  await expect(result).resolves.toBeNull();
  expect(handler.exit).toHaveBeenCalledWith({ force: true });
  expect(handler.destroy).toHaveBeenCalledOnce();
});
it("bounds load and session durations with redacted errors", async () => {
  const link = createPlaidLink();
  const first = link.open({
    linkToken: "synthetic-link",
    signal: new AbortController().signal,
  });
  const assertion = expect(first).rejects.toThrow(
    "Bank setup could not complete",
  );
  await vi.advanceTimersByTimeAsync(10_000);
  await assertion;
  const second = link.open({
    linkToken: "synthetic-next",
    signal: new AbortController().signal,
  });
  install();
  callbacks.onLoad();
  const session = expect(second).rejects.toThrow(
    "Bank setup could not complete",
  );
  await vi.advanceTimersByTimeAsync(300_000);
  await session;
  expect(handler.exit).toHaveBeenCalledOnce();
  expect(handler.destroy).toHaveBeenCalledOnce();
});
it("rejects simultaneous open and malformed success without token disclosure", async () => {
  const link = createPlaidLink();
  const controller = new AbortController();
  const first = link.open({
    linkToken: "synthetic-link",
    signal: controller.signal,
  });
  await expect(
    link.open({ linkToken: "synthetic-link", signal: controller.signal }),
  ).rejects.toThrow("Bank setup is unavailable");
  install();
  const done = expect(first).rejects.toThrow("Bank setup could not complete");
  callbacks.onSuccess(null);
  await done;
});
it("treats normal exit as cancellation and discards vendor errors", async () => {
  const first = createPlaidLink().open({
    linkToken: "synthetic-link",
    signal: new AbortController().signal,
  });
  install();
  callbacks.onExit(null);
  await expect(first).resolves.toBeNull();
  document.head.innerHTML = "";
  const second = createPlaidLink().open({
    linkToken: "synthetic-link",
    signal: new AbortController().signal,
  });
  install();
  const done = expect(second).rejects.not.toThrow("sensitive-vendor-error");
  callbacks.onExit({ message: "sensitive-vendor-error" });
  await done;
});

it("handles synchronous completion during SDK creation without leaking its handler", async () => {
  const result = createPlaidLink().open({
    linkToken: "synthetic-link",
    signal: new AbortController().signal,
  });
  Object.assign(window, {
    Plaid: {
      create(input: Callbacks) {
        input.onSuccess("synthetic-public");
        return handler;
      },
    },
  });
  document.scripts[0].dispatchEvent(new Event("load"));
  await expect(result).resolves.toEqual({ publicToken: "synthetic-public" });
  expect(handler.open).not.toHaveBeenCalled();
  expect(handler.destroy).toHaveBeenCalledOnce();
});
it("does not load for invalid tokens or an already aborted lifetime", async () => {
  const controller = new AbortController();
  controller.abort();
  await expect(
    createPlaidLink().open({
      linkToken: "synthetic-link",
      signal: controller.signal,
    }),
  ).resolves.toBeNull();
  await expect(
    createPlaidLink().open({
      linkToken: "private\ninvalid",
      signal: new AbortController().signal,
    }),
  ).rejects.toThrow("Bank setup is unavailable");
  expect(document.scripts.length).toBe(0);
});
it("rejects script failure without retaining failed script handlers", async () => {
  const result = createPlaidLink().open({
    linkToken: "synthetic-link",
    signal: new AbortController().signal,
  });
  const done = expect(result).rejects.toThrow("Bank setup could not complete");
  document.scripts[0].dispatchEvent(new Event("error"));
  await done;
  expect(document.scripts.length).toBe(0);
});
