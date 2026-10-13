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
// Download is optional for injected test clients; production clients own fresh authentication.
export type API = (<T>(
  path: string,
  body?: string,
  key?: string,
) => Promise<T>) & {
  download?: (path: string) => Promise<Blob>;
};
// A bounded same-origin request carries a fresh memory-only token and no cookies.
export function createAPI(
  getToken: () => Promise<string>,
  rejectSession: () => void,
): API {
  // Share authentication and safe error handling across JSON and bounded CSV reads.
  const request = async (
    path: string,
    body?: string,
    key?: string,
  ): Promise<Response> => {
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
            : {
                "Content-Type": "application/json",
                ...(key ? { "Idempotency-Key": key } : {}),
              }),
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
    return response;
  };
  const client: API = async <T>(
    path: string,
    body?: string,
    key?: string,
  ): Promise<T> => {
    const response = await request(path, body, key);
    try {
      return (await response.json()) as T;
    } catch {
      throw new APIError(0, true);
    }
  };
  // Buffer a bounded response before offering any bytes to the user as a download.
  client.download = async (path: string): Promise<Blob> => {
    const response = await request(path);
    if (
      response.status !== 200 ||
      response.headers.get("Content-Type") !==
        "text/csv; charset=utf-8; header=present" ||
      !response.body
    ) {
      void response.body?.cancel();
      throw new APIError(0, false);
    }
    const reader = response.body.getReader();
    const chunks: Uint8Array<ArrayBuffer>[] = [];
    let size = 0;
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        size += value.byteLength;
        if (size > 10 * 1024 * 1024) throw new Error("limit");
        chunks.push(new Uint8Array(value));
      }
      return new Blob(chunks, {
        type: "text/csv; charset=utf-8; header=present",
      });
    } catch {
      throw new APIError(0, false);
    } finally {
      void reader.cancel().catch(() => {});
      reader.releaseLock();
    }
  };
  return client;
}
