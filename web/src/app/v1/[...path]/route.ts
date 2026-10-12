import { createBridge } from '../../../server/bridge';
export const runtime = 'nodejs';
export const dynamic = 'force-dynamic';
// One bounded bridge per process; no request may select its target or credentials.
const bridge = createBridge();
/** Forward only explicitly allowlisted reads through the owned transport boundary. */
export function GET(request: Request) { return bridge(request); }
/** Forward immutable request bytes and idempotency key without retrying a write. */
export function POST(request: Request) { return bridge(request); }
