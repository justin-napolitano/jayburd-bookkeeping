import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { APIError, type API } from "../bookkeeping/api";
import { CollectionOnboarding } from "./CollectionOnboarding";
import {
  createOnboardingClient,
  validOnboardingCatalog,
  validOnboardingCommand,
  validOnboardingPending,
  validOnboardingPreview,
  type OnboardingCommand,
} from "./onboarding";
import {
  commitOnboarding,
  onboardingCatalog,
  onboardingCommand,
  onboardingIDs,
  onboardingPending,
  onboardingPreview,
} from "./onboarding-fixture";
afterEach(cleanup);
const root = "/v1/collection-onboarding";
function fixture(adopt = false) {
  let saved: OnboardingCommand | null = null,
    ready = adopt;
  const preview = onboardingPreview(adopt);
  const request = vi.fn(async (path: string, body?: string) => {
    if (path === root) return onboardingCatalog(ready);
    if (path.endsWith("/pending"))
      return onboardingPending(saved?.acknowledged ? null : saved);
    if (path.endsWith("/previews")) return preview;
    if (path === root + "/commands") {
      const input = JSON.parse(body!);
      saved ??= onboardingCommand(preview, input.prepare_key);
      return saved;
    }
    if (path.endsWith("/execute")) {
      saved = commitOnboarding(saved!);
      ready = true;
      return saved;
    }
    if (path.endsWith("/cancel")) {
      saved = { ...saved!, state: "cancelled", result: { cancelled: true } };
      return saved;
    }
    if (path.endsWith("/acknowledge")) {
      saved = { ...saved!, acknowledged: true };
      return saved;
    }
    throw Error("unexpected synthetic route");
  });
  return {
    request,
    preview,
    get saved() {
      return saved;
    },
    set saved(v) {
      saved = v;
    },
  };
}
function mount(
  request: ReturnType<typeof vi.fn>,
  extra: Partial<React.ComponentProps<typeof CollectionOnboarding>> = {},
) {
  const lock = vi.fn(),
    changed = vi.fn();
  return {
    lock,
    changed,
    view: render(
      <CollectionOnboarding
        api={request as unknown as API}
        active
        externalLocked={false}
        onLockChange={lock}
        onChanged={changed}
        {...extra}
      />,
    ),
  };
}
async function saveSetup() {
  await userEvent.click(
    await screen.findByRole("button", { name: "Set up private transactions" }),
  );
  await userEvent.click(
    await screen.findByRole("button", { name: "Save setup review" }),
  );
  await screen.findByRole("button", { name: "Create private workspace" });
}
describe("private onboarding authority and recovery", () => {
  it("uses only explicit setup/save/execute/ack and refreshes real discovery", async () => {
    const f = fixture(),
      { changed, lock } = mount(f.request);
    await screen.findByRole("button", { name: "Set up private transactions" });
    expect(
      f.request.mock.calls.every(([p]) => !String(p).endsWith("/commands")),
    ).toBe(true);
    await saveSetup();
    expect(f.saved?.state).toBe("prepared");
    expect(changed).not.toHaveBeenCalled();
    expect(lock).toHaveBeenLastCalledWith(true);
    expect(document.activeElement).toBe(
      screen.getByRole("heading", { name: "Saved private setup review" }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Create private workspace" }),
    );
    await screen.findByText(
      "Private workspace created. No journal was created.",
    );
    expect(changed).not.toHaveBeenCalled();
    await userEvent.click(
      screen.getByRole("button", { name: "Acknowledge setup result" }),
    );
    await screen.findByRole("button", {
      name: "Review connection for Fictional records",
    });
    expect(changed).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    const body = JSON.parse(
      f.request.mock.calls.find(([p]) => p === root + "/commands")![1]!,
    );
    expect(Object.keys(body).sort()).toEqual(["prepare_key", "preview_id"]);
    expect(f.request.mock.calls.every((c) => c.length < 3)).toBe(true);
  });
  it("reviews the complete authorized connection and discloses the acquisition pause", async () => {
    const f = fixture(true);
    mount(f.request);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Review connection for Fictional records",
      }),
    );
    expect(
      await screen.findByText(
        /Existing automatic bank syncing for this connection will pause/,
      ),
    ).toBeTruthy();
    expect(
      screen.getByRole("list", { name: "Reviewed source accounts" })
        .textContent,
    ).toContain("Fictional checking");
    await userEvent.click(
      screen.getByRole("button", { name: "Save setup review" }),
    );
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Adopt reviewed connection history",
      }),
    );
    await screen.findByText(
      "Existing connection history adopted. Existing automatic bank syncing paused. No journal was created.",
    );
    expect(
      f.request.mock.calls.find(([p]) => p.endsWith("/previews"))?.[1],
    ).toBe(
      JSON.stringify({
        operation: "adopt",
        candidate_id: onboardingIDs.candidate,
        snapshot: "a".repeat(64),
      }),
    );
  });
  it("treats only initial unsupported404 as unavailable feature, without setup or implicit grants", async () => {
    const request = vi.fn().mockRejectedValue(new APIError(404, false, "read"));
    const { lock } = mount(request);
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    expect(screen.queryByText("Set up private transactions")).toBeNull();
    expect(request.mock.calls).toEqual([[root + "/commands/pending"]]);
  });
  it("locks malformed/network/unauthorized discovery, never replacing it with absence", async () => {
    for (const response of [
      { ...onboardingPending(), private_canary: "never show" },
      new APIError(503, true, "read"),
      new APIError(401, false, "read"),
    ]) {
      const request =
        response instanceof Error
          ? vi.fn().mockRejectedValue(response)
          : vi.fn().mockResolvedValue(response);
      const { lock } = mount(request);
      await screen.findByRole("alert");
      expect(lock).toHaveBeenLastCalledWith(true);
      expect(document.body.textContent).not.toContain("never show");
      cleanup();
    }
  });
  it("keeps exact prepare key after lost reply and expiry; pending:none is not a receipt", async () => {
    const f = fixture();
    const original = f.request.getMockImplementation()!;
    let lost = false;
    f.request.mockImplementation(async (path, body) => {
      const value = await original(path, body);
      if (path === root + "/commands" && !lost) {
        lost = true;
        throw new APIError(503, true);
      }
      if (path.endsWith("/pending") && lost) return onboardingPending();
      return value;
    });
    const { lock } = mount(f.request);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Set up private transactions",
      }),
    );
    await userEvent.click(
      screen.getByRole("button", { name: "Save setup review" }),
    );
    await screen.findByText(/outcome is unconfirmed/i);
    await userEvent.click(
      screen.getByRole("button", { name: "Check saved private setup" }),
    );
    expect(lock).toHaveBeenLastCalledWith(true);
    const now = Date.now;
    vi.spyOn(Date, "now").mockReturnValue(now() + 3600000);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Retry saving the same setup review",
      }),
    );
    await screen.findByRole("button", { name: "Create private workspace" });
    const saves = f.request.mock.calls.filter(
      ([p]) => p === root + "/commands",
    );
    expect(saves).toHaveLength(2);
    expect(saves[0][1]).toBe(saves[1][1]);
    vi.restoreAllMocks();
  });
  it("restores committed lost execute on full remount and still requires explicit acknowledgement", async () => {
    const f = fixture();
    const original = f.request.getMockImplementation()!;
    f.request.mockImplementation(async (path, body) => {
      const value = await original(path, body);
      if (path.endsWith("/execute")) throw new APIError(503, true);
      return value;
    });
    const first = mount(f.request);
    await saveSetup();
    await userEvent.click(
      screen.getByRole("button", { name: "Create private workspace" }),
    );
    await screen.findByRole("alert");
    first.view.unmount();
    const second = mount(f.request, { active: false });
    await screen.findByText(
      "Private workspace created. No journal was created.",
    );
    expect(second.changed).not.toHaveBeenCalled();
    expect(
      f.request.mock.calls.filter(([p]) => p.endsWith("/execute")),
    ).toHaveLength(1);
    await userEvent.click(
      screen.getByRole("button", { name: "Acknowledge setup result" }),
    );
    await waitFor(() => expect(second.changed).toHaveBeenCalledTimes(1));
  });
  it("retains lost acknowledgement after none and resolves only its exact same terminal receipt", async () => {
    const f = fixture();
    f.saved = commitOnboarding(onboardingCommand());
    const original = f.request.getMockImplementation()!;
    let lost = false;
    f.request.mockImplementation(async (path, body) => {
      const value = await original(path, body);
      if (path.endsWith("/acknowledge") && !lost) {
        lost = true;
        throw new APIError(503, true);
      }
      return value;
    });
    const { changed, lock } = mount(f.request);
    await userEvent.click(
      await screen.findByRole("button", { name: "Acknowledge setup result" }),
    );
    await screen.findByRole("alert");
    await userEvent.click(
      screen.getByRole("button", { name: "Check saved private setup" }),
    );
    expect(changed).not.toHaveBeenCalled();
    expect(lock).toHaveBeenLastCalledWith(true);
    await userEvent.click(
      screen.getByRole("button", { name: "Retry acknowledging setup result" }),
    );
    await waitFor(() => expect(changed).toHaveBeenCalledTimes(1));
  });
  it("rejects switched terminal acknowledgement and changed immutable previews", async () => {
    const f = fixture();
    f.saved = commitOnboarding(onboardingCommand());
    const original = f.request.getMockImplementation()!;
    f.request.mockImplementation(async (path, body) =>
      path.endsWith("/acknowledge")
        ? {
            ...f.saved!,
            state: "cancelled",
            result: { cancelled: true },
            acknowledged: true,
          }
        : original(path, body),
    );
    const { changed, lock } = mount(f.request);
    await userEvent.click(
      await screen.findByRole("button", { name: "Acknowledge setup result" }),
    );
    await screen.findByRole("alert");
    expect(changed).not.toHaveBeenCalled();
    expect(lock).toHaveBeenLastCalledWith(true);
  });
  it("does not disclose old catalog or accept a late old response on API replacement", async () => {
    let finish!: (v: unknown) => void;
    const old = vi.fn(async (p: string) =>
      p.endsWith("/pending")
        ? onboardingPending()
        : new Promise((resolve) => {
            finish = resolve;
          }),
    );
    const next = vi.fn().mockResolvedValue(onboardingPending());
    const { view, lock } = mount(old);
    await waitFor(() => expect(finish).toBeTypeOf("function"));
    view.rerender(
      <CollectionOnboarding
        api={next as unknown as API}
        active={false}
        externalLocked={false}
        onLockChange={lock}
      />,
    );
    await act(async () => finish(onboardingCatalog(true)));
    expect(document.body.textContent).not.toContain("Fictional records");
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
  });
});
describe("closed onboarding wire", () => {
  it("accepts SQL microsecond timestamps, explicit cancellation and newly allocated adoption aliases", () => {
    const p = onboardingPreview(true);
    p.expires_at = "2026-11-08T05:40:00.123456Z";
    expect(validOnboardingPreview(p)).toBe(true);
    expect(
      validOnboardingPreview({
        ...p,
        expires_at: "2026-02-30T05:40:00.123456Z",
      }),
    ).toBe(false);
    expect(
      validOnboardingCommand({
        ...onboardingCommand(p),
        state: "cancelled",
        result: { cancelled: true },
      }),
    ).toBe(true);
    const receipt = commitOnboarding(onboardingCommand(p));
    (
      receipt.result!.sources as { source_account_id: string }[]
    )[0].source_account_id = "81000000-0000-4000-8000-000000000099";
    expect(validOnboardingCommand(receipt)).toBe(true);
    expect(
      validOnboardingCommand({
        ...receipt,
        result: { ...receipt.result, collection_item_id: onboardingIDs.source },
      }),
    ).toBe(false);
  });
  it("does not grant adoption to a read-only workspace or accept forged setup capabilities", () => {
    const c = onboardingCatalog(true);
    if (c.workspace.state === "ready") c.workspace.capabilities = ["read"];
    expect(validOnboardingCatalog(c)).toBe(false);
    c.can_adopt = false;
    c.candidates = [];
    expect(validOnboardingCatalog(c)).toBe(true);
    const p = onboardingPreview();
    p.review = { capabilities: ["read", "read", "classify"] };
    expect(validOnboardingPreview(p)).toBe(false);
    expect(
      validOnboardingPending({
        ...onboardingPending(),
        state: "blocked",
        recovery_id: onboardingIDs.recovery,
      }),
    ).toBe(false);
    expect(
      validOnboardingCommand({ ...onboardingCommand(), acknowledged: true }),
    ).toBe(false);
  });
  it("uses the reviewed POST read transport and rejects foreign response identity", async () => {
    const api = vi.fn() as unknown as API;
    api.query = vi.fn().mockResolvedValue(onboardingPreview());
    await createOnboardingClient(api).preview({ operation: "setup" });
    expect(api).not.toHaveBeenCalled();
    expect(api.query).toHaveBeenCalledWith(
      root + "/previews",
      '{"operation":"setup"}',
    );
    api.query = vi
      .fn()
      .mockResolvedValue({ ...onboardingPreview(), extra: "private canary" });
    await expect(
      createOnboardingClient(api).preview({ operation: "setup" }),
    ).rejects.toMatchObject({ kind: "unavailable" });
  });
});
