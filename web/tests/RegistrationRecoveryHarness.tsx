import { useMemo } from "react";
import { createAPI } from "../src/bookkeeping/api";
import { useCommands } from "../src/bookkeeping/commands";
import { CommandPanel } from "../src/bookkeeping/CommandPanel";
import {
  registrationWorkspace,
  savedRegistration,
} from "./registration.fixture";
// This entrypoint is compiled only into synthetic component tests, never an application route.
export function RegistrationRecoveryHarness() {
  const api = useMemo(
    () =>
      createAPI(
        async () => "synthetic-registration-proof",
        () => {},
      ),
    [],
  );
  const recovery = useCommands(api);
  return (
    <>
      <button
        disabled={recovery.locked}
        onClick={() =>
          void recovery.prepareRegistration(
            savedRegistration.entity_id!,
            savedRegistration.book_id!,
            "00000000-0000-4000-8000-000000000004",
            ["synthetic-checking"],
          )
        }
      >
        Prepare selected account
      </button>
      <CommandPanel recovery={recovery} workspace={registrationWorkspace} />
    </>
  );
}
