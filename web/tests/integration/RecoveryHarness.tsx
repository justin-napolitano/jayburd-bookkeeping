import { Bookkeeping } from "../../src/bookkeeping/Bookkeeping";

const preparedRecoveryToken = async () => "synthetic.prepared.proof";
const committedRecoveryToken = async () => "synthetic.committed.proof";
const rejectRecoveryIdentity = () => {
  throw new Error("Fictional recovery identity unexpectedly rejected");
};

/** Both identities exist only in the disposable integration binary and test browser bundle. */
export function RecoveryHarness({
  committed = false,
}: {
  committed?: boolean;
}) {
  return (
    <Bookkeeping
      getToken={committed ? committedRecoveryToken : preparedRecoveryToken}
      rejectSession={rejectRecoveryIdentity}
    />
  );
}
