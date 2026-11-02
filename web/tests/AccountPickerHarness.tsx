import { useMemo, useState } from "react";
import { createAPI } from "../src/bookkeeping/api";
import { useCommands } from "../src/bookkeeping/commands";
import { CommandPanel } from "../src/bookkeeping/CommandPanel";
import { RegistrationPicker } from "../src/bookkeeping/RegistrationPicker";
import { registrationWorkspace } from "./registration.fixture";
import { pickerOptions } from "./registration-options.fixture";

// This test-only composition exercises real picker/recovery components with intercepted fictional HTTP.
export function AccountPickerHarness() {
  const api = useMemo(
    () =>
      createAPI(
        async () => "synthetic.picker.proof",
        () => {
          throw Error("synthetic session denied");
        },
      ),
    [],
  );
  const recovery = useCommands(api);
  const [source, setSource] = useState("");
  const workspace = {
    ...registrationWorkspace,
    currencies: [
      ...registrationWorkspace.currencies,
      {
        currency_version_id: "00000000-0000-4000-8000-000000000050",
        alphabetic_code: "EUR",
        minor_unit: 2,
        valid_from: "2000-01-01",
        valid_to: null,
      },
    ],
  };
  return (
    <main className="finance-workspace">
      <h2>Register accounts</h2>
      {!recovery.locked && (
        <RegistrationPicker
          api={api}
          entity={pickerOptions.entity_id}
          book={pickerOptions.book_id}
          attempt={pickerOptions.attempt_id}
          locked={recovery.locked}
          prepare={(ids) =>
            recovery.prepareRegistration(
              pickerOptions.entity_id,
              pickerOptions.book_id,
              pickerOptions.attempt_id,
              ids,
            )
          }
          denied={() => {
            throw Error("synthetic scope denied");
          }}
          onReview={setSource}
        />
      )}
      <CommandPanel recovery={recovery} workspace={workspace} />
      {source && <output aria-label="Reviewed source">{source}</output>}
    </main>
  );
}
