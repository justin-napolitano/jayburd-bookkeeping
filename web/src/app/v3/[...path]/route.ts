import { createBridge } from "../../../server/bridge";
export const runtime = "nodejs";
export const dynamic = "force-dynamic";
const bridge = createBridge();
/** Forward only the exact reviewed v3 read routes in the shared no-store bridge. */
export function GET(request: Request) {
  return bridge(request);
}
/** Private query/detail selectors stay in bounded bodies without URL leakage. */
export function POST(request: Request) {
  return bridge(request);
}
