// @vitest-environment node
import { describe, expect, it, vi } from "vitest";
import { createBridge } from "./bridge";

const env = {
  BOOKKEEPING_API_MODE: "cloud-run",
  BOOKKEEPING_API_ORIGIN: "https://api-synthetic.run.app",
  BOOKKEEPING_PUBLIC_ORIGIN: "https://books.example.invalid",
};
const workspace = "00000000-0000-4000-8000-000000000001";
const path = `/v1/collections/${workspace}/automation`;
const raw =
  '{ "key":"synthetic.settings", "expected_revision":"0", "enabled":true }';
const headers = {
  Authorization: "Bearer synthetic.user.signature",
  Origin: env.BOOKKEEPING_PUBLIC_ORIGIN,
};

// Only synthetic metadata and transport are mocked; the production bridge owns all route/framing checks.
function transport() {
  return vi
    .fn<typeof fetch>()
    .mockResolvedValueOnce(
      new Response("synthetic.service.signature", {
        headers: { "Metadata-Flavor": "Google" },
      }),
    )
    .mockResolvedValueOnce(Response.json({ enabled: true, revision: "1" }));
}

describe("automatic category settings bridge", () => {
  it.each(["GET", "POST"])(
    "forwards only the exact %s route and original bytes",
    async (method) => {
      const fetcher = transport();
      const response = await createBridge(
        fetcher,
        () => env,
      )(
        new Request(env.BOOKKEEPING_PUBLIC_ORIGIN + path, {
          method,
          headers: {
            ...headers,
            ...(method === "POST"
              ? { "Content-Type": "application/json" }
              : {}),
          },
          body: method === "POST" ? raw : undefined,
        }),
      );
      expect(response.status).toBe(200);
      expect(response.headers.get("Cache-Control")).toBe("no-store");
      expect(fetcher).toHaveBeenCalledTimes(2);
      const [url, init] = fetcher.mock.calls[1];
      expect(url).toBe(env.BOOKKEEPING_API_ORIGIN + path);
      expect(init?.method).toBe(method);
      expect(new Headers(init?.headers).has("Idempotency-Key")).toBe(false);
      expect(
        method === "POST"
          ? new TextDecoder().decode(init?.body as Uint8Array)
          : init?.body,
      ).toBe(method === "POST" ? raw : undefined);
    },
  );

  it.each([
    ["version2", path.replace("/v1/", "/v2/"), "POST", {}, raw, 404],
    ["version3", path.replace("/v1/", "/v3/"), "GET", {}, undefined, 404],
    ["query", path + "?enabled=true", "POST", {}, raw, 404],
    ["empty query", path + "?", "GET", {}, undefined, 404],
    ["trusted results", path + "/execute", "POST", {}, raw, 404],
    ["unsupported method", path, "PUT", {}, raw, 404],
    [
      "noncanonical workspace",
      path.replace(
        workspace,
        workspace.toUpperCase().replace("000001", "00000A"),
      ),
      "GET",
      {},
      undefined,
      404,
    ],
    [
      "idempotency header",
      path,
      "POST",
      { "Idempotency-Key": "another-key" },
      raw,
      400,
    ],
    ["encoding", path, "POST", { "Content-Encoding": "gzip" }, raw, 400],
    ["read length", path, "GET", { "Content-Length": "1" }, undefined, 400],
    [
      "foreign origin",
      path,
      "POST",
      { Origin: "https://foreign.invalid" },
      raw,
      403,
    ],
    [
      "content type",
      path,
      "POST",
      { "Content-Type": "application/json; charset=utf-8" },
      raw,
      400,
    ],
    ["body limit", path, "POST", {}, "x".repeat(1025), 413],
  ] as const)(
    "rejects %s before upstream access",
    async (_name, route, method, extra, body, status) => {
      const fetcher = vi.fn<typeof fetch>();
      const response = await createBridge(
        fetcher,
        () => env,
      )(
        new Request(env.BOOKKEEPING_PUBLIC_ORIGIN + route, {
          method,
          headers: { ...headers, "Content-Type": "application/json", ...extra },
          body,
        }),
      );
      expect(response.status).toBe(status);
      expect(fetcher).not.toHaveBeenCalled();
    },
  );

  it("never retries a settings write after an unknown upstream outcome", async () => {
    const fetcher = transport();
    fetcher
      .mockReset()
      .mockResolvedValueOnce(
        new Response("synthetic.service.signature", {
          headers: { "Metadata-Flavor": "Google" },
        }),
      )
      .mockRejectedValueOnce(new Error("synthetic lost committed response"));
    const response = await createBridge(
      fetcher,
      () => env,
    )(
      new Request(env.BOOKKEEPING_PUBLIC_ORIGIN + path, {
        method: "POST",
        headers: { ...headers, "Content-Type": "application/json" },
        body: raw,
      }),
    );
    expect(response.status).toBe(502);
    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(
      new TextDecoder().decode(fetcher.mock.calls[1][1]?.body as Uint8Array),
    ).toBe(raw);
  });
});
