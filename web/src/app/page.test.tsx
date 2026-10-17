import { afterEach, expect, it, vi } from "vitest";
import { headers } from "next/headers";
import { redirect } from "next/navigation";
import Page from "./page";

vi.mock("next/headers", () => ({ headers: vi.fn() }));
vi.mock("next/navigation", () => ({
  redirect: vi.fn(() => {
    throw new Error("redirected");
  }),
}));
afterEach(() => {
  vi.unstubAllEnvs();
  vi.clearAllMocks();
});

// The public app URL is fixed by deployment metadata, never by a request header.
it("redirects a Cloud Run alias to the configured public origin", async () => {
  vi.stubEnv("BOOKKEEPING_API_MODE", "cloud-run");
  vi.stubEnv(
    "BOOKKEEPING_PUBLIC_ORIGIN",
    "https://jayburd-dev-web-9088794002.us-central1.run.app",
  );
  vi.mocked(headers).mockResolvedValue(
    new Headers({ host: "jayburd-dev-web-ummxu4dlsa-uc.a.run.app" }),
  );
  await expect(Page()).rejects.toThrow("redirected");
  expect(redirect).toHaveBeenCalledWith(
    "https://jayburd-dev-web-9088794002.us-central1.run.app",
  );
});

it("serves the configured origin without a redirect", async () => {
  vi.stubEnv("BOOKKEEPING_API_MODE", "cloud-run");
  vi.stubEnv(
    "BOOKKEEPING_PUBLIC_ORIGIN",
    "https://jayburd-dev-web-9088794002.us-central1.run.app",
  );
  vi.mocked(headers).mockResolvedValue(
    new Headers({ host: "jayburd-dev-web-9088794002.us-central1.run.app" }),
  );
  expect(await Page()).toBeTruthy();
  expect(redirect).not.toHaveBeenCalled();
});

it("does not require a public origin in a local app", async () => {
  vi.stubEnv("BOOKKEEPING_API_MODE", "loopback");
  expect(await Page()).toBeTruthy();
  expect(headers).not.toHaveBeenCalled();
});
