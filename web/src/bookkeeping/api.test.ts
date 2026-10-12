import { afterEach, describe, expect, it, vi } from "vitest";
import { APIError, createAPI } from "./api";

/** Synthetic fetch outcomes cannot reach an API or identity provider. */
function fixture() {
  const fetch = vi.fn<typeof globalThis.fetch>();
  vi.stubGlobal("fetch", fetch);
  const getToken = vi.fn(async () => "synthetic-token");
  const rejectSession = vi.fn();
  return {
    fetch,
    getToken,
    rejectSession,
    api: createAPI(getToken, rejectSession),
  };
}

/** Restore global transport after each test so fixtures cannot bleed across suites. */
afterEach(() => vi.unstubAllGlobals());

describe("API client boundary", () => {
  it("uses fresh bearer authorization, omits cookies and forbids redirects", async () => {
    const { fetch, getToken, api } = fixture();
    fetch.mockImplementation(
      async () => new Response('{"entities":[]}', { status: 200 }),
    );
    getToken
      .mockResolvedValueOnce("synthetic-first-token")
      .mockResolvedValueOnce("synthetic-second-token");
    expect(await api("/v1/workspace")).toEqual({ entities: [] });
    await api("/v1/workspace");
    expect(getToken).toHaveBeenCalledTimes(2);
    expect(fetch.mock.calls[0]).toEqual([
      "/v1/workspace",
      expect.objectContaining({
        method: "GET",
        cache: "no-store",
        credentials: "omit",
        redirect: "error",
        headers: { Authorization: "Bearer synthetic-first-token" },
        body: undefined,
      }),
    ]);
    expect(fetch.mock.calls[1][1]?.headers).toEqual({
      Authorization: "Bearer synthetic-second-token",
    });
    expect(fetch.mock.calls[0][1]?.signal).toBeInstanceOf(AbortSignal);
  });

  it("carries the exact reviewed body and idempotency key without rewriting amounts", async () => {
    const { fetch, api } = fixture();
    const body = '{"amount":"1.2300","description":"synthetic"}';
    fetch.mockImplementation(
      async () =>
        new Response('{"journal_id":"synthetic-journal"}', { status: 201 }),
    );
    await api("/v1/setup", body, "synthetic-request-key");
    expect(fetch).toHaveBeenCalledWith(
      "/v1/setup",
      expect.objectContaining({
        method: "POST",
        body,
        headers: {
          Authorization: "Bearer synthetic-token",
          "Content-Type": "application/json",
          "Idempotency-Key": "synthetic-request-key",
        },
      }),
    );
  });

  it("rejects token failure before transport and clears the session without exposing provider text", async () => {
    const { fetch, getToken, rejectSession, api } = fixture();
    getToken.mockRejectedValue(new Error("private synthetic provider detail"));
    await expect(api("/v1/workspace")).rejects.toMatchObject({
      status: 401,
      uncertain: false,
      message: "Your session needs a new sign-in.",
    });
    expect(fetch).not.toHaveBeenCalled();
    expect(rejectSession).toHaveBeenCalledTimes(1);
  });

  it("invalidates the session on backend 401 and never reads its error payload", async () => {
    const { fetch, rejectSession, api } = fixture();
    const response = new Response(
      '{"detail":"private synthetic backend detail"}',
      { status: 401 },
    );
    const read = vi.spyOn(response, "json");
    fetch.mockResolvedValue(response);
    await expect(api("/v1/workspace")).rejects.toMatchObject({
      status: 401,
      uncertain: false,
      message: "Your session needs a new sign-in.",
    });
    expect(rejectSession).toHaveBeenCalledTimes(1);
    expect(read).not.toHaveBeenCalled();
  });

  it.each([403, 404, 409, 422])(
    "redacts backend status %i and does not invalidate a valid session",
    async (status) => {
      const { fetch, rejectSession, api } = fixture();
      fetch.mockResolvedValue(
        new Response("private synthetic backend detail", { status }),
      );
      const failure = await api("/v1/setup", "{}", "synthetic-key").catch(
        (error: unknown) => error,
      );
      expect(failure).toBeInstanceOf(APIError);
      expect(failure).toMatchObject({ status, uncertain: false });
      expect((failure as Error).message).not.toContain("private synthetic");
      expect(rejectSession).not.toHaveBeenCalled();
    },
  );

  it.each([408, 429, 500, 503])(
    "preserves uncertainty for backend status %i",
    async (status) => {
      const { fetch, api } = fixture();
      fetch.mockResolvedValue(
        new Response("private synthetic backend detail", { status }),
      );
      await expect(
        api("/v1/setup", "{}", "synthetic-key"),
      ).rejects.toMatchObject({ status, uncertain: true });
      expect(fetch).toHaveBeenCalledTimes(1); // Retry needs explicit caller review, never automatic transport replay.
    },
  );

  it("treats an unreadable success response as uncertain rather than inviting a new request", async () => {
    const { fetch, api } = fixture();
    fetch.mockResolvedValue(
      new Response("invalid synthetic JSON", { status: 201 }),
    );
    await expect(api("/v1/setup", "{}", "synthetic-key")).rejects.toMatchObject(
      { status: 0, uncertain: true },
    );
  });

  it.each(["network failure", "redirect rejected", "deadline exceeded"])(
    "treats %s as uncertain and hides raw transport errors",
    async (reason) => {
      const { fetch, rejectSession, api } = fixture();
      fetch.mockRejectedValue(new TypeError(`private synthetic ${reason}`));
      const failure = await api("/v1/setup", "{}", "synthetic-key").catch(
        (error: unknown) => error,
      );
      expect(failure).toMatchObject({ status: 0, uncertain: true });
      expect((failure as Error).message).not.toContain("private synthetic");
      expect(fetch.mock.calls[0][1]?.redirect).toBe("error");
      expect(rejectSession).not.toHaveBeenCalled();
    },
  );
  it("sends command transitions without inventing an idempotency header", async () => {
    const { fetch, api } = fixture();
    fetch.mockResolvedValue(new Response("{}", { status: 200 }));
    await api("/v1/commands/synthetic-command/execute", "{}");
    expect(fetch.mock.calls[0][1]?.headers).not.toHaveProperty(
      "Idempotency-Key",
    );
    expect(fetch.mock.calls[0][1]?.body).toBe("{}");
  });
});
