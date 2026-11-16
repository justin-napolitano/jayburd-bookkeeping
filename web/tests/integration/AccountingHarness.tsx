import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";
// Only identity is synthetic: production Next forwarding, Go authorization/runner and SQL persist all outcomes.
const getToken = async () => "synthetic.accounting.proof";
const rejectSession = () => {
  throw new Error("Synthetic accounting identity unexpectedly rejected");
};
export function AccountingHarness() {
  return <Bookkeeping getToken={getToken} rejectSession={rejectSession} />;
}
