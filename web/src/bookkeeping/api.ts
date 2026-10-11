// Errors expose only fixed UI messages, never provider or backend raw detail.
export class APIError extends Error {
  constructor(
    public status: number,
    public uncertain: boolean,
  ) {
    super(
      status === 401
        ? "Your session needs a new sign-in."
        : status === 403 || status === 404
          ? "This scope is unavailable or access was denied."
          : status === 409
            ? "The request conflicts with current books. Refresh and review before starting a new request."
            : uncertain
              ? "The outcome is unknown. Retry this unchanged request to find its committed result."
              : "The server rejected this request. Check the fields and accounting policy.",
    );
  }
}
export type API = <T>(path: string, body?: string, key?: string) => Promise<T>;
// A bounded same-origin request carries a fresh memory-only token and no cookies.
export function createAPI(
  getToken: () => Promise<string>,
  rejectSession: () => void,
): API {
  return async <T>(path: string, body?: string, key?: string): Promise<T> => {
    let token: string;
    try {
      token = await getToken();
    } catch {
      rejectSession();
      throw new APIError(401, false);
    }
    let response: Response;
    try {
      response = await fetch(path, {
        method: body === undefined ? "GET" : "POST",
        cache: "no-store",
        credentials: "omit",
        redirect: "error",
        signal: AbortSignal.timeout(15000),
        headers: {
          Authorization: `Bearer ${token}`,
          ...(body === undefined
            ? {}
            : { "Content-Type": "application/json", "Idempotency-Key": key! }),
        },
        body,
      });
    } catch {
      throw new APIError(0, true);
    }
    if (response.status === 401) rejectSession();
    if (!response.ok)
      throw new APIError(
        response.status,
        response.status >= 500 ||
          response.status === 408 ||
          response.status === 429,
      );
    try {
      return (await response.json()) as T;
    } catch {
      throw new APIError(0, true);
    }
  };
}
