import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { APIError, type API } from "../bookkeeping/api";
import { CollectionRecovery } from "./CollectionRecovery";
import { createRecoveryClient, type RecoveryStatus } from "./recovery";

afterEach(cleanup);
const first = "71000000-0000-4000-8000-000000000001";
const second = "71000000-0000-4000-8000-000000000002";
const state = (
  value: RecoveryStatus["state"],
  recovery_id: string | null = value === "none" ? null : first,
): RecoveryStatus => ({
  profile: "private-collection-recovery-v1",
  state: value,
  recovery_id,
});
const receipt = (recovery_id = first) => ({
  profile: "private-collection-recovery-clear-v1",
  recovery_id,
  cleared: true,
});

describe("payload-free owner recovery", () => {
  it("allows initial feature-off404 but keeps other unknown discovery failures locked", async () => {
    const request = vi.fn().mockRejectedValue(new APIError(404, false, "read"));
    const lock = vi.fn();
    const view = render(
      <CollectionRecovery
        api={request as unknown as API}
        onLockChange={lock}
      />,
    );
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    expect(request.mock.calls).toEqual([["/v1/collection-recovery"]]);
    request.mockRejectedValue(new APIError(503, true, "read"));
    view.rerender(
      <CollectionRecovery
        api={request as unknown as API}
        refreshVersion={1}
        onLockChange={lock}
      />,
    );
    await screen.findByRole("alert");
    expect(lock).toHaveBeenLastCalledWith(true);
  });
  it("rejects extra payload and invalid identities without exposing service fields", async () => {
    for (const result of [
      { ...state("inaccessible"), description: "Private canary" },
      state("none", first),
      state("available", null),
    ]) {
      const api = vi.fn().mockResolvedValue(result) as unknown as API;
      await expect(createRecoveryClient(api).discover()).rejects.toMatchObject({
        kind: "unavailable",
      });
    }
    const api = vi
      .fn()
      .mockResolvedValue({
        ...receipt(),
        description: "Private canary",
      }) as unknown as API;
    await expect(createRecoveryClient(api).clear(first)).rejects.toMatchObject({
      kind: "uncertain",
    });
  });
  it("clears only on explicit owner action, hides opaque identity and confirms the exact receipt", async () => {
    let pending = true;
    const request = vi.fn(async (path: string) => {
      if (path.endsWith("/clear")) {
        pending = false;
        return receipt();
      }
      return state(pending ? "inaccessible" : "none");
    });
    const lock = vi.fn(),
      cleared = vi.fn();
    render(
      <CollectionRecovery
        api={request as unknown as API}
        onLockChange={lock}
        onCleared={cleared}
      />,
    );
    await screen.findByRole("button", {
      name: "Clear inaccessible private review",
    });
    expect(request).toHaveBeenCalledTimes(1);
    expect(document.body.textContent).not.toContain(first);
    expect(lock).toHaveBeenLastCalledWith(true);
    await userEvent.click(
      screen.getByRole("button", { name: "Clear inaccessible private review" }),
    );
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    expect(cleared).toHaveBeenCalledTimes(1);
    expect(request.mock.calls[1]).toEqual([
      "/v1/collection-recovery/clear",
      JSON.stringify({ recovery_id: first }),
    ]);
  });
  it("retains unknown clear(A) after none and retries A without clearing a newer claim B", async () => {
    let status = state("inaccessible"),
      attempts = 0;
    const bodies: unknown[] = [];
    const request = vi.fn(async (path: string, body?: string) => {
      if (!path.endsWith("/clear")) return status;
      bodies.push(JSON.parse(body!));
      if (++attempts === 1) {
        status = state("none");
        throw new APIError(503, true);
      }
      status = state("inaccessible", second);
      return receipt();
    });
    const lock = vi.fn(),
      cleared = vi.fn();
    render(
      <CollectionRecovery
        api={request as unknown as API}
        onLockChange={lock}
        onCleared={cleared}
      />,
    );
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Clear inaccessible private review",
      }),
    );
    await screen.findByText(/unconfirmed/);
    await userEvent.click(
      screen.getByRole("button", { name: "Check private recovery" }),
    );
    await screen.findByText(/unconfirmed/);
    expect(lock).toHaveBeenLastCalledWith(true);
    expect(cleared).not.toHaveBeenCalled();
    await userEvent.click(
      screen.getByRole("button", {
        name: "Retry clearing the same private review",
      }),
    );
    await screen.findByRole("button", {
      name: "Clear inaccessible private review",
    });
    expect(bodies).toEqual([{ recovery_id: first }, { recovery_id: first }]);
    expect(cleared).toHaveBeenCalledTimes(1);
    expect(lock).toHaveBeenLastCalledWith(true);
    expect(document.body.textContent).not.toContain(second);
  });
  it("does not treat a later404 as feature absence after known work or unknown clear", async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce(state("inaccessible"))
      .mockRejectedValue(new APIError(404, false, "read"));
    const lock = vi.fn();
    render(
      <CollectionRecovery
        api={request as unknown as API}
        onLockChange={lock}
      />,
    );
    await screen.findByRole("button", {
      name: "Clear inaccessible private review",
    });
    await userEvent.click(
      screen.getByRole("button", { name: "Check private recovery" }),
    );
    await screen.findByRole("alert");
    expect(lock).toHaveBeenLastCalledWith(true);
    expect(
      screen.queryByRole("button", {
        name: "Clear inaccessible private review",
      }),
    ).toBeNull();
  });
  it("routes restored available authority by state edge, preserves unknown A until its closure receipt", async () => {
    let current = state("available"),
      rejected = false;
    const request = vi.fn(async (path: string) => {
      if (path.endsWith("/clear")) {
        if (!rejected) {
          rejected = true;
          throw new APIError(503, true);
        }
        return receipt();
      }
      return current;
    });
    const available = vi.fn(),
      lock = vi.fn(),
      cleared = vi.fn(),
      changes = vi.fn();
    const props = {
      api: request as unknown as API,
      onLockChange: lock,
      onAvailable: available,
      onCleared: cleared,
      onStateChange: changes,
    };
    const view = render(<CollectionRecovery {...props} />);
    await waitFor(() => expect(available).toHaveBeenCalledTimes(1));
    view.rerender(<CollectionRecovery {...props} refreshVersion={1} />);
    await waitFor(() => expect(request).toHaveBeenCalledTimes(2));
    expect(available).toHaveBeenCalledTimes(1);
    current = state("inaccessible");
    view.rerender(<CollectionRecovery {...props} refreshVersion={2} />);
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Clear inaccessible private review",
      }),
    );
    await screen.findByText(/unconfirmed/);
    current = state("available");
    view.rerender(<CollectionRecovery {...props} refreshVersion={3} />);
    await waitFor(() => expect(available).toHaveBeenCalledTimes(2));
    await waitFor(() =>
      expect(changes).toHaveBeenLastCalledWith({
        state: "available",
        uncertain: true,
        canReview: true,
      }),
    );
    current = state("available", second);
    view.rerender(<CollectionRecovery {...props} refreshVersion={31} />);
    await waitFor(() =>
      expect(changes).toHaveBeenLastCalledWith({
        state: "available",
        uncertain: true,
        canReview: false,
      }),
    );
    expect(available).toHaveBeenCalledTimes(2);
    // Another context's ordinary acknowledgement records A's closure; none is still not a receipt here.
    current = state("none");
    view.rerender(<CollectionRecovery {...props} refreshVersion={4} />);
    await waitFor(() =>
      expect(changes).toHaveBeenLastCalledWith({
        state: "none",
        uncertain: true,
        canReview: false,
      }),
    );
    expect(lock).toHaveBeenLastCalledWith(true);
    await userEvent.click(
      screen.getByRole("button", {
        name: "Retry clearing the same private review",
      }),
    );
    await waitFor(() => expect(cleared).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
  });
  it("ignores an old session's late receipt after authenticated API replacement", async () => {
    let finish!: (value: unknown) => void;
    const old = vi.fn(async (path: string) =>
      path.endsWith("/clear")
        ? new Promise((resolve) => {
            finish = resolve;
          })
        : state("inaccessible"),
    );
    const next = vi.fn().mockResolvedValue(state("none"));
    const cleared = vi.fn(),
      lock = vi.fn();
    const view = render(
      <CollectionRecovery
        api={old as unknown as API}
        onLockChange={lock}
        onCleared={cleared}
      />,
    );
    await userEvent.click(
      await screen.findByRole("button", {
        name: "Clear inaccessible private review",
      }),
    );
    view.rerender(
      <CollectionRecovery
        api={next as unknown as API}
        onLockChange={lock}
        onCleared={cleared}
      />,
    );
    await waitFor(() => expect(lock).toHaveBeenLastCalledWith(false));
    await act(async () => finish(receipt()));
    expect(cleared).not.toHaveBeenCalled();
    expect(next.mock.calls).toEqual([["/v1/collection-recovery"]]);
  });
});
