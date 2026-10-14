import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";
import type { ConnectionLink } from "../../src/bookkeeping/Connections";
// The synthetic provider callback exists only in this test bundle; no external SDK is loaded.
const link: ConnectionLink = {
  async open({ linkToken, signal }) {
    if (signal.aborted || linkToken !== "synthetic-link")
      throw new Error("fixture handoff");
    return { publicToken: "synthetic-public" };
  },
};
const getToken = async () => "synthetic.browser.proof";
// Identity rejection is observable and cannot create another principal.
const rejectSession = () => {
  throw new Error("Synthetic identity rejected");
};
// Use the production workspace and bridge with explicitly injected test-only identity/Link.
export function ConnectionsHarness() {
  return (
    <Bookkeeping
      getToken={getToken}
      rejectSession={rejectSession}
      connectionLink={link}
    />
  );
}
