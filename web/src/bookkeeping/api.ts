// Errors expose only fixed UI messages, never provider or backend raw detail.
export class APIError extends Error {
  constructor(
    public status: number,
    public uncertain: boolean,
    operation: "read" | "mutation" = "mutation",
    public code?: "unsupported_policy",
  ) {
    super(
      status === 401
        ? "Your session needs a new sign-in."
        : status === 403 || status === 404
          ? "This scope is unavailable or access was denied."
          : operation === "read"
            ? uncertain
              ? "Data is unavailable right now. Retry the read."
              : "This data could not be read. Check the selection and retry."
            : status === 409
              ? "The request conflicts with current books. Refresh and review before starting a new request."
              : uncertain
                ? "The outcome is unknown. Retry this unchanged request to find its committed result."
                : "The server rejected this request. Check the fields and accounting policy.",
    );
  }
}

// Retain one reviewed problem code only; never expose raw diagnostics or buffer an unbounded error.
async function readProblemCode(
  response: Response,
): Promise<"unsupported_policy" | undefined> {
  if (
    response.status !== 422 ||
    response.headers.get("Content-Type")?.split(";")[0].trim().toLowerCase() !==
      "application/problem+json" ||
    !response.body
  )
    return;
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > 2048) return;
      chunks.push(value);
    }
    const bytes = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) {
      bytes.set(chunk, offset);
      offset += chunk.byteLength;
    }
    const value: unknown = JSON.parse(
      new TextDecoder("utf-8", { fatal: true }).decode(bytes),
    );
    if (
      value !== null &&
      typeof value === "object" &&
      !Array.isArray(value) &&
      "code" in value &&
      value.code === "unsupported_policy"
    )
      return "unsupported_policy";
  } catch {
    // A malformed/interrupted problem never replaces the original status or authority decision.
  } finally {
    void reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}
// Download is optional for injected test clients; production clients own fresh authentication.
export type API = (<T>(
  path: string,
  body?: string,
  key?: string,
) => Promise<T>) & {
  download?: (path: string) => Promise<Blob>;
  // Explicit reviewed POST read; ordinary POST calls retain commit uncertainty.
  query?: <T>(path: string, body: string) => Promise<T>;
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
    readOnly = false,
  ): Promise<Response> => {
    // Wording follows the actual method; uncertainty remains authoritative for recovery gates.
    const operation = body === undefined || readOnly ? "read" : "mutation";
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
      throw new APIError(0, true, operation);
    }
    if (response.status === 401) rejectSession();
    if (!response.ok)
      throw new APIError(
        response.status,
        response.status >= 500 ||
          response.status === 408 ||
          response.status === 429,
        operation,
        operation === "read" ? await readProblemCode(response) : undefined,
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
      throw new APIError(0, true, body === undefined ? "read" : "mutation");
    }
  };
  // Query transports private filters without placing their values in a URL.
  client.query = async <T>(path: string, body: string): Promise<T> => {
    if (
      !/^\/v1\/entities\/[^/?#]+\/books\/[^/?#]+\/review-items\/query$/.test(
        path,
      )
    )
      throw new APIError(400, false, "read");
    const response = await request(path, body, undefined, true);
    try {
      return (await response.json()) as T;
    } catch {
      throw new APIError(0, true, "read");
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
      throw new APIError(0, false, "read");
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
      throw new APIError(0, false, "read");
    } finally {
      void reader.cancel().catch(() => {});
      reader.releaseLock();
    }
  };
  return client;
}
