import { Bookkeeping } from "../src/bookkeeping/Bookkeeping";
// This synthetic token exists only in the Playwright bundle, never in app routes.
const getToken = async () => "synthetic-test-token";
const rejectSession = () => {};
export function Harness() {
  return <Bookkeeping getToken={getToken} rejectSession={rejectSession} />;
}
