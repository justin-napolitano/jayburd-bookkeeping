import { createBridge } from "../../../server/bridge";
export const runtime = "nodejs";
export const dynamic = "force-dynamic";
// Versioned private routes use the same exact allowlist, verified identity and bounded transport.
const bridge = createBridge();
/** Forward only reviewed v2 reads; the bridge rejects every unlisted path. */
export function GET(request: Request) {
  return bridge(request);
}
/** Preserve immutable preview/command bytes without retrying a write. */
export function POST(request: Request) {
  return bridge(request);
}
