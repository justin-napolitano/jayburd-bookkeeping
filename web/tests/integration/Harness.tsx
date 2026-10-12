import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";
// This fictional identity is confined to a test bundle; real Go authorization still runs.
const getToken = async () => "synthetic-browser-proof";
// A provider rejection fails the test UI instead of silently manufacturing another session.
const rejectSession = () => {
  throw new Error("Synthetic proof identity unexpectedly rejected");
};
/** Mount the real bookkeeping component with the explicit test-only auth seam. */
export function IntegrationHarness() {
  return <Bookkeeping getToken={getToken} rejectSession={rejectSession} />;
}
