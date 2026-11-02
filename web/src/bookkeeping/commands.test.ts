import { act, renderHook, waitFor, cleanup } from "@testing-library/react";
import { afterEach, describe, it, expect, vi } from "vitest";
import { useCommands } from "./commands";
import { APIError, type API } from "./api";
import type { DurableCommand, RegisterCommand } from "./types";
import { savedRegistration } from "../../tests/registration.fixture";
afterEach(cleanup);

it("does not return an acknowledged old account plan after the API session changes during discovery", async () => {
  let resolveOld!: (value: {command:null}) => void;
  let reads=0,waiting=false;
  const oldAPI=vi.fn(async(path:string)=>{
    if(path.endsWith("/pending")) {
      if(++reads===1) return {command:null};
      waiting=true;return new Promise<{command:null}>((resolve)=>{resolveOld=resolve;});
    }
    return {...savedRegistration,state:"committed",acknowledged:true};
  }) as unknown as API;
  const nextAPI=vi.fn(async()=>({command:null})) as unknown as API;
  const hook=renderHook(({api})=>useCommands(api),{initialProps:{api:oldAPI}});
  await waitFor(()=>expect(hook.result.current.ready).toBe(true));
  let outcome:ReturnType<typeof hook.result.current.prepareRegistration>;
  act(()=>{outcome=hook.result.current.prepareRegistration(savedRegistration.entity_id!,savedRegistration.book_id!,"00000000-0000-4000-8000-000000000004",["synthetic-checking"]);});
  await waitFor(()=>expect(waiting).toBe(true));
  hook.rerender({api:nextAPI});
  await waitFor(()=>expect(hook.result.current.ready).toBe(true));
  await act(async()=>{resolveOld({command:null});expect(await outcome!).toBeUndefined();});
  expect(hook.result.current.command).toBeNull();
});

it("retains the exact account-selection path, key and body after an unknown preparation reply", async () => {
  const requests: { path: string; body?: string; key?: string }[] = [];
  let saved = false;
  const api = vi.fn(async (path: string, body?: string, key?: string) => {
    if (path.endsWith("/pending"))
      return { command: saved ? savedRegistration : null };
    requests.push({ path, body, key });
    if (requests.length === 1) throw new APIError(0, true);
    saved = true;
    return savedRegistration;
  }) as unknown as API;
  let hook = renderHook(() => useCommands(api));
  await waitFor(() => expect(hook.result.current.ready).toBe(true));
  await act(async () => {
    await hook.result.current.prepareRegistration(
      savedRegistration.entity_id!,
      savedRegistration.book_id!,
      "00000000-0000-4000-8000-000000000004",
      ["synthetic-checking"],
    );
  });
  expect(hook.result.current.locked).toBe(true);
  expect(hook.result.current.registration).not.toBeNull();
  await act(async () => {
    hook.result.current.retryRegistration();
  });
  await waitFor(() =>
    expect(hook.result.current.command?.command_id).toBe(
      savedRegistration.command_id,
    ),
  );
  expect(requests).toHaveLength(2);
  expect(requests[0]).toEqual(requests[1]);
  expect(requests[0].path).toMatch(/\/connection-attempts\/.*\/registration$/);
  expect(JSON.parse(requests[0].body!)).toEqual({
    account_ids: ["synthetic-checking"],
  });
  hook.unmount();
  hook = renderHook(() => useCommands(api));
  await waitFor(() =>
    expect(hook.result.current.command?.operation).toBe("register_accounts"),
  );
  expect(requests).toHaveLength(2);
  expect(localStorage.length).toBe(0);
  expect(sessionStorage.length).toBe(0);
});
const body = {
  entity_name: "Synthetic LLC",
  entity_kind: "legal",
  book_name: "Book",
  starts_on: "2026-01-01",
  ends_before: "2027-01-01",
  functional_currency_version_id: "currency",
  accounts: [],
} as const;
const input = {
  operation: "setup",
  entity_id: null,
  book_id: null,
  body: { ...body, accounts: [] },
} satisfies RegisterCommand;
const prepared: DurableCommand = {
  ...input,
  command_id: "old-command",
  state: "prepared",
  result: null,
  acknowledged: false,
  created_at: "2026-01-01T00:00:00Z",
};
// API replacement models session composition changes while an old network operation remains unresolved.
describe("command session boundary", () => {
  it("clears only explicit inaccessible recovery and reloads the command gate", async () => {
    let blocked = true;
    const api = vi.fn(async (path: string) => {
      if (path === "/v1/commands/pending/clear") {
        blocked = false;
        return { cleared: true };
      }
      if (path.endsWith("/pending") && blocked) throw new APIError(403, false);
      return { command: null };
    }) as unknown as API;
    const hook = renderHook(() => useCommands(api));
    await waitFor(() => expect(hook.result.current.accessDenied).toBe(true));
    expect(hook.result.current.locked).toBe(true);
    await act(async () => hook.result.current.clearInaccessible());
    await waitFor(() => expect(hook.result.current.ready).toBe(true));
    expect(hook.result.current.accessDenied).toBe(false);
    expect(hook.result.current.locked).toBe(false);
    expect(api).toHaveBeenNthCalledWith(2, "/v1/commands/pending/clear", "{}");
  });

  it("clears a prior denial only after a later discovery succeeds", async () => {
    let blocked = true;
    const api = vi.fn(async () => {
      if (blocked) throw new APIError(403, false);
      return { command: null };
    }) as unknown as API;
    const hook = renderHook(() => useCommands(api));
    await waitFor(() => expect(hook.result.current.accessDenied).toBe(true));
    expect(hook.result.current.error).toContain("access was denied");
    blocked = false;
    await act(async () => hook.result.current.reload());
    await waitFor(() => expect(hook.result.current.ready).toBe(true));
    expect(hook.result.current.accessDenied).toBe(false);
    expect(hook.result.current.error).toBe("");
    expect(hook.result.current.locked).toBe(false);
  });

  it("preserves a rejected setup error when pending discovery finds no work", async () => {
    const api = vi.fn(async (path: string) => {
      if (path === "/v1/commands") throw new APIError(403, false);
      return { command: null };
    }) as unknown as API;
    const hook = renderHook(() => useCommands(api));
    await waitFor(() => expect(hook.result.current.ready).toBe(true));
    await act(async () => hook.result.current.prepare(input));
    await waitFor(() => expect(hook.result.current.busy).toBe(false));
    expect(hook.result.current.error).toContain("access was denied");
    expect(hook.result.current.accessDenied).toBe(false);
    expect(hook.result.current.command).toBeNull();
  });

  it.each(["register", "execute"] as const)(
    "ignores late %s completion after API replacement",
    async (operation) => {
      let finish!: (value: DurableCommand) => void;
      const oldAPI = vi.fn((path: string) =>
        path.endsWith("/pending")
          ? Promise.resolve({
              command: operation === "execute" ? prepared : null,
            })
          : new Promise<DurableCommand>((resolve) => {
              finish = resolve;
            }),
      ) as unknown as API;
      const newAPI = vi.fn(async () => ({ command: null })) as unknown as API;
      const hook = renderHook(({ api }: { api: API }) => useCommands(api), {
        initialProps: { api: oldAPI },
      });
      await waitFor(() => expect(hook.result.current.ready).toBe(true));
      act(() => {
        if (operation === "register") hook.result.current.prepare(input);
        else void hook.result.current.transition("execute");
      });
      expect(hook.result.current.busy).toBe(true);
      hook.rerender({ api: newAPI });
      await waitFor(() => expect(hook.result.current.ready).toBe(true));
      await act(async () => {
        finish({
          ...prepared,
          state: operation === "execute" ? "committed" : "prepared",
        });
      });
      expect(hook.result.current.command).toBeNull();
      expect(hook.result.current.registration).toBeNull();
      expect(hook.result.current.busy).toBe(false);
      expect(hook.result.current.locked).toBe(false);
      expect(newAPI).toHaveBeenCalledTimes(1);
    },
  );
});
