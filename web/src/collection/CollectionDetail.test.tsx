import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  collectionFixture,
  collectionRow,
  collectionSnapshot,
} from "../../tests/collection-fixture";
import { CollectionDetail } from "./CollectionDetail";
import {
  CollectionFailure,
  selectionOf,
  validSourceDetail,
  type DetailRequest,
  type SourceDetail,
} from "./types";
afterEach(cleanup);
const row = collectionRow();
const request: DetailRequest = {
  profile: "private-collection-detail-v1",
  snapshot: collectionSnapshot,
  ...selectionOf(row),
};
const projection = (
  state: SourceDetail["links"]["state"] = "unavailable",
): SourceDetail => ({
  profile: request.profile,
  workspace_id: request.workspace_id,
  snapshot: request.snapshot,
  row: {
    ...row,
    journal_state: state === "unavailable" ? "unavailable" : "unposted",
    assignment:
      state === "available"
        ? {
            state: "assigned",
            books: [
              {
                book_id: "83000000-0000-4000-8000-000000000001",
                name: "Fictional book",
              },
            ],
          }
        : { state: state },
  },
  links: { state, journals: [] },
});
describe("current source detail", () => {
  it.each(["available", "unassigned", "unavailable"] as const)(
    "keeps %s book references separate without inferring a journal",
    async (state) => {
      const fixture = collectionFixture();
      fixture.port.detail = vi.fn().mockResolvedValue(projection(state));
      const denied = vi.fn(),
        back = vi.fn();
      render(
        <CollectionDetail
          request={request}
          port={fixture.port}
          onDenied={denied}
          onBack={back}
        />,
      );
      await screen.findByRole("heading", { name: "Transaction detail" });
      await waitFor(() =>
        expect(document.activeElement).toBe(
          screen.getByRole("heading", { name: "Transaction detail" }),
        ),
      );
      expect(
        screen.getByText(
          state === "available"
            ? "No posted journal is linked in the currently readable book."
            : state === "unassigned"
              ? "This source is unassigned to a book."
              : "Book entries are unavailable with your current access.",
        ),
      ).toBeTruthy();
      expect(
        screen.queryByRole("button", { name: /View posted journal/ }),
      ).toBeNull();
      await userEvent.click(
        screen.getByRole("button", { name: "Back to transactions" }),
      );
      expect(back).toHaveBeenCalledTimes(1);
      expect(denied).not.toHaveBeenCalled();
    },
  );
  it("rejects changed snapshots/identities and forbidden anchors in unavailable projection", () => {
    expect(
      validSourceDetail(
        {
          ...projection(),
          snapshot: { ...collectionSnapshot, map_revision: "9" },
        },
        request,
      ),
    ).toBe(false);
    expect(
      validSourceDetail({ ...projection(), row: collectionRow(2) }, request),
    ).toBe(false);
    expect(
      validSourceDetail(
        {
          ...projection(),
          links: {
            state: "unavailable",
            journals: [
              {
                entity_id: "x",
                book_id: "b",
                journal_id: "j",
                source_revision: "1",
              },
            ],
          },
        },
        request,
      ),
    ).toBe(false);
  });
  it("destroys rendered detail on current denial and ignores an old-scope late response", async () => {
    let finish!: (value: SourceDetail) => void;
    const fixture = collectionFixture();
    fixture.port.detail = vi
      .fn()
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            finish = resolve;
          }),
      )
      .mockRejectedValueOnce(new CollectionFailure("denied"));
    const denied = vi.fn();
    const view = render(
      <CollectionDetail
        request={request}
        port={fixture.port}
        onDenied={denied}
        onBack={() => {}}
      />,
    );
    const next = { ...request, observation_id: "fictional-observation-next" };
    view.rerender(
      <CollectionDetail
        request={next}
        port={fixture.port}
        onDenied={denied}
        onBack={() => {}}
      />,
    );
    await waitFor(() => expect(denied).toHaveBeenCalledTimes(1));
    await act(async () => finish(projection()));
    expect(screen.queryByText(row.description)).toBeNull();
  });
});
