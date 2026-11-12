import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";

// Fictional identity is confined to this test bundle; the actual Go/SQL authorization remains active.
const getToken = async () => "synthetic.automatic.proof";
const rejectSession = () => {
  throw new Error("Synthetic automatic identity unexpectedly rejected");
};

export function AutomaticHarness() {
  return <Bookkeeping getToken={getToken} rejectSession={rejectSession} />;
}
