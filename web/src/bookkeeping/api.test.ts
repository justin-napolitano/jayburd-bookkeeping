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
  it("never dispatches a mutation through the reviewed query helper", async () => {
    const { fetch, api, getToken } = fixture();
    await expect(
      api.query!("/v1/commands/synthetic/execute", "{}"),
    ).rejects.toMatchObject({ status: 400, uncertain: false });
    expect(fetch).not.toHaveBeenCalled();
    expect(getToken).not.toHaveBeenCalled();
  });
  it.each(["review/query", "labels/preview"])(
    "sends private collection %s without a retry header",
    async (suffix) => {
      const { fetch, api } = fixture();
      fetch.mockResolvedValue(Response.json({ profile: "synthetic" }));
      const path = `/v1/collections/00000000-0000-4000-8000-000000000001/${suffix}`;
      await api.query!(path, '{"snapshot":null}');
      expect(fetch.mock.calls[0][0]).toBe(path);
      expect(fetch.mock.calls[0][1]?.headers).not.toHaveProperty("Idempotency-Key");
    },
  );
  it.each(["label-commands", "label-commands/00000000-0000-4000-8000-000000000002/execute"])(
    "rejects private command %s through the read helper",
    async (suffix) => {
      const { fetch, api } = fixture();
      await expect(api.query!(`/v1/collections/00000000-0000-4000-8000-000000000001/${suffix}`, "{}")).rejects.toMatchObject({status:400, uncertain:false});
      expect(fetch).not.toHaveBeenCalled();
    },
  );
  it.each(["review-items", "source-activity"])(
    "sends private %s POST reads with read availability semantics and no retry key",
    async (resource) => {
      const { fetch, api } = fixture();
      fetch.mockResolvedValue(new Response("private error", { status: 503 }));
      await expect(
        api.query!(
          `/v1/entities/e/books/b/${resource}/query`,
          '{"search":"fictional"}',
        ),
      ).rejects.toMatchObject({
        uncertain: true,
        message: "Data is unavailable right now. Retry the read.",
      });
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(fetch.mock.calls[0][0]).toBe(
        `/v1/entities/e/books/b/${resource}/query`,
      );
      expect(fetch.mock.calls[0][1]).toMatchObject({
        method: "POST",
        body: '{"search":"fictional"}',
      });
      expect(fetch.mock.calls[0][1]?.headers).not.toHaveProperty(
        "Idempotency-Key",
      );
    },
  );
  it("projects only the approved problem code from a bounded read error", async () => {
    const { fetch, api } = fixture();
    fetch.mockResolvedValue(
      new Response(
        JSON.stringify({
          code: "unsupported_policy",
          detail: "private diagnostic",
        }),
        {
          status: 422,
          headers: {
            "Content-Type": "application/problem+json; charset=utf-8",
          },
        },
      ),
    );
    const failure = await api("/v1/categories").catch(
      (error: unknown) => error,
    );
    expect(failure).toMatchObject({
      status: 422,
      uncertain: false,
      code: "unsupported_policy",
    });
    expect(JSON.stringify(failure)).not.toContain("private diagnostic");
    expect((failure as Error).message).not.toContain("private diagnostic");
  });

  it.each([
    "unknown",
    "invalid",
    "oversized",
    "wrong type",
    "array",
    "interrupted",
  ])("ignores %s read problem bodies", async (reason) => {
    const { fetch, api } = fixture();
    let cancelled = false;
    const body =
      reason === "oversized"
        ? new ReadableStream({
            start(controller) {
              controller.enqueue(new Uint8Array(2049));
            },
            cancel() {
              cancelled = true;
            },
          })
        : reason === "interrupted"
          ? new ReadableStream({
              start(controller) {
                controller.error(new Error("private"));
              },
            })
          : reason === "invalid"
            ? "invalid JSON"
            : reason === "array"
              ? '[{"code":"unsupported_policy"}]'
              : JSON.stringify({
                  code:
                    reason === "unknown"
                      ? "private_unknown"
                      : "unsupported_policy",
                });
    fetch.mockResolvedValue(
      new Response(body, {
        status: 422,
        headers: {
          "Content-Type":
            reason === "wrong type" ? "text/html" : "application/problem+json",
        },
      }),
    );
    await expect(api("/v1/categories")).rejects.toMatchObject({
      status: 422,
      uncertain: false,
      code: undefined,
    });
    if (reason === "oversized") expect(cancelled).toBe(true);
  });

  it.each([401, 403, 404, 503])(
    "never consumes authority/availability error bodies for %i",
    async (status) => {
      const { fetch, api } = fixture();
      const response = new Response('{"code":"unsupported_policy"}', {
        status,
        headers: { "Content-Type": "application/problem+json" },
      });
      fetch.mockResolvedValue(response);
      await expect(api("/v1/categories")).rejects.toBeInstanceOf(APIError);
      expect(response.bodyUsed).toBe(false);
    },
  );

  it("never consumes mutation problem bodies", async () => {
    const { fetch, api } = fixture();
    const response = new Response('{"code":"unsupported_policy"}', {
      status: 422,
      headers: { "Content-Type": "application/problem+json" },
    });
    fetch.mockResolvedValue(response);
    await expect(api("/v1/draft", "{}", "synthetic-key")).rejects.toMatchObject(
      { code: undefined },
    );
    expect(response.bodyUsed).toBe(false);
  });
  it.each([408, 429, 500, 503])(
    "describes GET status %i as unavailable while preserving recovery uncertainty",
    async (status) => {
      const { fetch, api } = fixture();
      fetch.mockResolvedValue(new Response("private detail", { status }));
      await expect(api("/v1/workspace")).rejects.toMatchObject({
        status,
        uncertain: true,
        message: "Data is unavailable right now. Retry the read.",
      });
      expect(fetch).toHaveBeenCalledTimes(1);
    },
  );

  it.each(["network", "timeout", "invalid JSON"])(
    "keeps GET %s failure read-specific without assuming pending work is absent",
    async (reason) => {
      const { fetch, api } = fixture();
      if (reason === "invalid JSON")
        fetch.mockResolvedValue(new Response("private invalid JSON"));
      else
        fetch.mockRejectedValue(
          new DOMException(
            "private transport detail",
            reason === "timeout" ? "TimeoutError" : "NetworkError",
          ),
        );
      await expect(api("/v1/commands/pending")).rejects.toMatchObject({
        status: 0,
        uncertain: true,
        message: "Data is unavailable right now. Retry the read.",
      });
      expect(fetch).toHaveBeenCalledTimes(1);
    },
  );

  it.each([403, 404])("keeps GET authority fencing for %i", async (status) => {
    const { fetch, api, rejectSession } = fixture();
    fetch.mockResolvedValue(new Response("private detail", { status }));
    await expect(api("/v1/workspace")).rejects.toMatchObject({
      status,
      uncertain: false,
      message: "This scope is unavailable or access was denied.",
    });
    expect(rejectSession).not.toHaveBeenCalled();
  });

  it.each([409, 422])(
    "keeps rejected GET %i free of mutation guidance",
    async (status) => {
      const { fetch, api } = fixture();
      fetch.mockResolvedValue(new Response("private detail", { status }));
      await expect(api("/v1/workspace")).rejects.toMatchObject({
        status,
        uncertain: false,
        message: "This data could not be read. Check the selection and retry.",
      });
    },
  );
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
      ).rejects.toMatchObject({
        status,
        uncertain: true,
        message:
          "The outcome is unknown. Retry this unchanged request to find its committed result.",
      });
      expect(fetch).toHaveBeenCalledTimes(1); // Retry needs explicit caller review, never automatic transport replay.
    },
  );

  it("treats an unreadable success response as uncertain rather than inviting a new request", async () => {
    const { fetch, api } = fixture();
    fetch.mockResolvedValue(
      new Response("invalid synthetic JSON", { status: 201 }),
    );
    await expect(api("/v1/setup", "{}", "synthetic-key")).rejects.toMatchObject(
      {
        status: 0,
        uncertain: true,
        message:
          "The outcome is unknown. Retry this unchanged request to find its committed result.",
      },
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
      expect(failure).toMatchObject({
        status: 0,
        uncertain: true,
        message:
          "The outcome is unknown. Retry this unchanged request to find its committed result.",
      });
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

// Downloads share bearer refresh and reject oversized or non-CSV payloads before returning bytes.
describe("review CSV client", () => {
  it("downloads exact bytes with fresh header authentication", async () => {
    const { fetch, api, getToken } = fixture();
    fetch.mockResolvedValue(
      new Response('"amount"\r\n"0.01"\r\n', {
        headers: { "Content-Type": "text/csv; charset=utf-8; header=present" },
      }),
    );
    const blob = await api.download!(
      "/v1/reports/test/export.csv?snapshot_revision=1",
    );
    const text = await new Promise<string>((resolve) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result));
      reader.readAsText(blob);
    });
    expect(text).toBe('"amount"\r\n"0.01"\r\n');
    expect(getToken).toHaveBeenCalledOnce();
    expect(fetch.mock.calls[0][0]).not.toContain("token");
    expect(fetch.mock.calls[0][1]).toMatchObject({
      cache: "no-store",
      credentials: "omit",
      redirect: "error",
      headers: { Authorization: "Bearer synthetic-token" },
    });
  });
  it.each(["application/json", "text/html", "text/csv"])(
    "rejects unapproved content type %s",
    async (content) => {
      const { fetch, api } = fixture();
      fetch.mockResolvedValue(
        new Response("private", { headers: { "Content-Type": content } }),
      );
      await expect(api.download!("/v1/export.csv")).rejects.toMatchObject({
        uncertain: false,
        message: "This data could not be read. Check the selection and retry.",
      });
    },
  );
  it("cancels at the streaming byte cap rather than returning a partial file", async () => {
    const { fetch, api } = fixture();
    let cancelled = false;
    fetch.mockResolvedValue(
      new Response(
        new ReadableStream({
          start(controller) {
            controller.enqueue(new Uint8Array(10 * 1024 * 1024 + 1));
          },
          cancel() {
            cancelled = true;
          },
        }),
        {
          headers: {
            "Content-Type": "text/csv; charset=utf-8; header=present",
          },
        },
      ),
    );
    await expect(api.download!("/v1/export.csv")).rejects.toBeInstanceOf(
      APIError,
    );
    expect(cancelled).toBe(true);
  });
});
