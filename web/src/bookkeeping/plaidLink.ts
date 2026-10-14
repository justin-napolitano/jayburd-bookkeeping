import type { ConnectionLink } from "./Connections";

const sdkURL = "https://cdn.plaid.com/link/v2/stable/link-initialize.js";
const loadTimeout = 10_000;
const sessionTimeout = 5 * 60_000;
// The reviewed SDK boundary uses only token handoff and lifecycle callbacks.
type Handler = {
  open(): void;
  exit(input: { force: true }): void;
  destroy(): void;
};
type SDK = {
  create(input: {
    token: string;
    onSuccess(publicToken: unknown): void;
    onExit(error: unknown): void;
    onLoad(): void;
  }): Handler;
};
// A shape check is not provider authentication; the server still verifies session association.
function validToken(value: unknown): value is string {
  return (
    typeof value === "string" &&
    value.length > 0 &&
    value.length <= 4096 &&
    /^[!-~]+$/.test(value)
  );
}
// createPlaidLink is inert until an explicit open call; no token enters persistence or logs.
// Each instance permits one active session and fences every callback after settlement.
export function createPlaidLink(): ConnectionLink {
  let active = false;
  let loaded = false;
  return {
    open({ linkToken, signal }) {
      if (signal.aborted) return Promise.resolve(null);
      if (active || !validToken(linkToken) || typeof window === "undefined") {
        return Promise.reject(new Error("Bank setup is unavailable."));
      }
      active = true;
      return new Promise((resolve, reject) => {
        let settled = false;
        let handler: Handler | undefined;
        let script: HTMLScriptElement | undefined;
        let loadTimer: ReturnType<typeof setTimeout> | undefined;
        let sessionTimer: ReturnType<typeof setTimeout> | undefined;
        // Mark terminal before SDK cleanup, whose exit method can itself invoke callbacks.
        const finish = (
          result: { publicToken: string } | null,
          failed = false,
          close = false,
        ) => {
          if (settled) return;
          settled = true;
          active = false;
          clearTimeout(loadTimer);
          clearTimeout(sessionTimer);
          signal.removeEventListener("abort", abort);
          if (script) {
            script.onload = null;
            script.onerror = null;
            if (!loaded) script.remove();
          }
          if (handler) {
            if (close) {
              try {
                handler.exit({ force: true });
              } catch {
                /* Ignore SDK diagnostics. */
              }
            }
            try {
              handler.destroy();
            } catch {
              /* Never surface vendor payloads. */
            }
          }
          if (failed)
            reject(
              new Error(
                "Bank setup could not complete. Inspect the existing attempt before trying again.",
              ),
            );
          else resolve(result);
        };
        const abort = () => finish(null, false, true);
        // initialize never forwards account metadata or vendor error objects to application state.
        const initialize = () => {
          if (settled || signal.aborted) {
            abort();
            return;
          }
          const sdk = (window as Window & { Plaid?: SDK }).Plaid;
          if (!sdk || typeof sdk.create !== "function") {
            finish(null, true);
            return;
          }
          try {
            handler = sdk.create({
              token: linkToken,
              onSuccess: (publicToken) => {
                if (settled || signal.aborted) return;
                if (!validToken(publicToken)) {
                  finish(null, true, true);
                  return;
                }
                finish({ publicToken });
              },
              onExit: (error) => finish(null, error != null),
              onLoad: () => {
                if (!settled) clearTimeout(loadTimer);
              },
            });
            // Defensive cleanup also covers SDK implementations invoking callbacks synchronously in create.
            if (settled) {
              try {
                handler.destroy();
              } catch {
                /* Redacted. */
              }
              return;
            }
            sessionTimer = setTimeout(
              () => finish(null, true, true),
              sessionTimeout,
            );
            handler.open();
          } catch {
            finish(null, true, true);
          }
        };
        signal.addEventListener("abort", abort, { once: true });
        loadTimer = setTimeout(() => finish(null, true, true), loadTimeout);
        if (loaded) {
          initialize();
          return;
        }
        script = document.createElement("script");
        script.src = sdkURL;
        script.async = true;
        script.referrerPolicy = "no-referrer";
        script.onload = () => {
          if (!settled) {
            loaded = true;
            initialize();
          }
        };
        script.onerror = () => finish(null, true);
        document.head.appendChild(script);
      });
    },
  };
}
