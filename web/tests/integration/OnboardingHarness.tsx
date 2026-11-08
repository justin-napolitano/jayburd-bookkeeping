import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";
// A fictional enrolled owner with real legacy history and no pre-created private workspace.
const getToken = async () => "synthetic.onboarding.proof";
const rejectSession = () => {
  throw new Error("Synthetic onboarding identity unexpectedly rejected");
};
export function OnboardingHarness() {
  return <Bookkeeping getToken={getToken} rejectSession={rejectSession} />;
}
