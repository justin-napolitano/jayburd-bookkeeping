import { ActivityOverview } from "../src/bookkeeping/ActivityOverview";
import { createAPI } from "../src/bookkeeping/api";
import {
  activityBook,
  activityEntity,
  activityCurrencies,
} from "./activity-fixture";
const api = createAPI(
  async () => "synthetic-test-token",
  () => {},
);
// A test-only story injects fictional identity into the real read-only transport.
export function ActivityOverviewHarness() {
  return (
    <ActivityOverview
      api={api}
      entity={activityEntity}
      book={activityBook}
      currencies={activityCurrencies}
      locked={false}
      denied={() => {}}
      onReview={() => {}}
      onAccounts={() => {}}
    />
  );
}
