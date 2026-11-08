import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";

// This token and actor exist only in the disposable integration binary.
const getRuleToken = async () => "synthetic.rules.proof";
const rejectRuleIdentity = () => {
  throw new Error("Fictional rules identity unexpectedly rejected");
};

/** Exercise the actual product composition and authenticated Next bridge. */
export function RulesHarness() {
  return (
    <Bookkeeping getToken={getRuleToken} rejectSession={rejectRuleIdentity} />
  );
}
