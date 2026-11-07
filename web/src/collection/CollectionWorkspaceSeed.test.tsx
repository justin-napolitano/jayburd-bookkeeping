import { useLayoutEffect } from "react";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { API } from "../bookkeeping/api";
import { APIError } from "../bookkeeping/api";
import {
  collectionCatalog,
  collectionPage,
  collectionPreview,
  collectionQuery,
  collectionRow,
} from "../../tests/collection-fixture";
import { CollectionWorkspace } from "./CollectionWorkspace";
import type { LabelCommand, Preview, PreviewRequest, Query } from "./types";
afterEach(cleanup);
const workspace = "88000000-0000-4000-8000-000000000001",
  source = "88000000-0000-4000-8000-000000000002";
const row = collectionRow(0, {
  workspace_id: workspace,
  source_id: source,
  description: "Fictional previous-session canary",
});
function fixture() {
  let omit = false,
    preview: Preview | null = null,
    command: LabelCommand | null = null;
  const api = vi.fn(async (path: string, body?: string) => {
    if (path === "/v1/collection")
      return {
        profile: "private-source-workspace-v1",
        workspace_id: workspace,
        workspace_revision: "1",
        capabilities: ["read", "classify"],
      };
    if (path.endsWith("/taxonomy"))
      return {
        profile: "private-collection-catalog-v3",
        workspace_id: workspace,
        authority_revision: "1",
        ...collectionCatalog,
        sources: omit
          ? []
          : [{ source_id: source, name: "Fictional checking" }],
      };
    const wrap = () => ({
      profile: "private-collection-command-v3",
      workspace_id: workspace,
      command,
    });
    if (path.endsWith("/pending")) return wrap();
    if (path.endsWith("/label-commands")) {
      const request = JSON.parse(body!);
      command = {
        workspace_id: workspace,
        command_id: "88000000-0000-4000-8000-000000000003",
        prepare_key: request.prepare_key,
        state: "prepared",
        acknowledged: false,
        preview: preview!,
      };
      return wrap();
    }
    if (path.endsWith("/execute")) {
      command = { ...command!, state: "committed" };
      return wrap();
    }
    if (path.endsWith("/acknowledge")) {
      command = { ...command!, acknowledged: true };
      const result = structuredClone(wrap());
      command = null;
      omit = true;
      return result;
    }
    throw Error("Unexpected fictional operation");
  }) as unknown as API;
  api.query = vi.fn(async (path: string, body: string) => {
    if (path.endsWith("/review/query"))
      return collectionPage(
        {
          ...collectionQuery(),
          ...JSON.parse(body),
          workspace_id: workspace,
        } as Query,
        omit ? [] : [row],
      );
    preview = collectionPreview(
      { ...JSON.parse(body), workspace_id: workspace } as PreviewRequest,
      [row],
    );
    return preview;
  }) as API["query"];
  return api;
}
describe("private coordinator session and seed boundaries", () => {
  it("hides old financial rows synchronously before a replacement API passive effect runs", async () => {
    const old = fixture(),
      next = vi.fn(() => new Promise(() => {})) as unknown as API,
      observed: (string | null)[] = [];
    const lock = vi.fn();
    function Witness({ api }: { api: API }) {
      useLayoutEffect(() => {
        if (api === next)
          observed.push(
            screen.queryByText(row.description)?.textContent ?? null,
          );
      }, [api]);
      return (
        <CollectionWorkspace
          api={api}
          active
          bookLocked={false}
          onLockChange={lock}
        />
      );
    }
    const view = render(<Witness api={old} />);
    await screen.findByText(row.description);
    view.rerender(<Witness api={next} />);
    expect(observed).toEqual([null]);
    expect(screen.queryByText(row.description)).toBeNull();
    expect(lock).toHaveBeenLastCalledWith(true);
  });
  it("does not repopulate an old delayed row after a new API reports feature absence", async () => {
    const old = fixture();
    let finish!: (value: unknown) => void;
    old.query = vi.fn(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    ) as API["query"];
    const next = vi
      .fn()
      .mockRejectedValue(new APIError(404, false, "read")) as unknown as API;
    const lock = vi.fn();
    const view = render(
      <CollectionWorkspace
        api={old}
        active
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await waitFor(() => expect(old.query).toHaveBeenCalledTimes(1));
    view.rerender(
      <CollectionWorkspace
        api={next}
        active
        bookLocked={false}
        onLockChange={lock}
      />,
    );
    await screen.findByText(
      "Private transactions are unavailable for this session.",
    );
    await act(async () =>
      finish(
        collectionPage({ ...collectionQuery(), workspace_id: workspace }, [
          row,
        ]),
      ),
    );
    expect(screen.queryByText(row.description)).toBeNull();
  });
  it("discards a confirmed correction seed when fresh catalog discovery omits its source", async () => {
    const api = fixture();
    render(
      <CollectionWorkspace
        api={api}
        active
        bookLocked={false}
        onLockChange={() => {}}
      />,
    );
    await screen.findByText(row.description);
    await userEvent.click(screen.getByRole("checkbox"));
    await userEvent.selectOptions(
      screen.getByLabelText("Label selected transactions", { exact: true }),
      "uncategorized",
    );
    await userEvent.click(
      screen.getByRole("button", { name: /^Preview labels for 1 selected$/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: /^Save label review$/ }),
    );
    await userEvent.click(
      await screen.findByRole("button", {
        name: /^Confirm labels for 1 selected$/,
      }),
    );
    await userEvent.click(
      await screen.findByRole("button", {
        name: /^Acknowledge and create rule$/,
      }),
    );
    await screen.findByText("0 of 0 matching transactions");
    expect(screen.queryByLabelText("Rule 1 name", { exact: true })).toBeNull();
    expect(screen.queryByText(row.description)).toBeNull();
  });
});
