import { createBridge } from "../../../server/bridge";
export const runtime = "nodejs";
export const dynamic = "force-dynamic";
const bridge = createBridge();
/** Forward only exact reviewed v3 reads in the shared no-store bridge. */
export function GET(request: Request) {
  return bridge(request);
}
/** Bound private selectors and durable rule intents without URL leakage or write retries. */
export function POST(request: Request) {
  return bridge(request);
}
