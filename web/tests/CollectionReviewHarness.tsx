import { useMemo, useState } from "react";
import { CollectionReview } from "../src/collection/CollectionReview";
import {
  collectionCatalog,
  collectionFixture,
  collectionRow,
  collectionSources,
  collectionWorkspace,
  heldCollectionRow,
} from "./collection-fixture";

// A browser-only fictional fixture exercises callbacks without a real server or auth bypass.
export function CollectionReviewHarness({
  scenario = "normal",
  held = false,
}: {
  scenario?: "normal" | "lost_prepare" | "pending";
  held?: boolean;
}) {
  const [counts, setCounts] = useState<Record<string, number>>({});
  const fixture = useMemo(
    () =>
      collectionFixture({
        scenario,
        ...(held ? { rows: [collectionRow(), heldCollectionRow()] } : {}),
        onCall: (name) =>
          setCounts((old) => ({ ...old, [name]: (old[name] ?? 0) + 1 })),
      }),
    [scenario, held],
  );
  return (
    <main>
      <h1>Fictional collection review</h1>
      <CollectionReview
        workspace={collectionWorkspace}
        catalog={collectionCatalog}
        sources={collectionSources}
        port={fixture.port}
        onDenied={() => {}}
      />
      <output aria-label="Fixture executions">{counts.execute ?? 0}</output>
      <output aria-label="Fixture acknowledgements">
        {counts.acknowledge ?? 0}
      </output>
    </main>
  );
}
