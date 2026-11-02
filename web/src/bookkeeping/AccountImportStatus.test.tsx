import { afterEach, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import { AccountImportStatus } from "./AccountImportStatus";
import { APIError, type API } from "./api";
import {
  pickerOptions,
  pickerStatus,
} from "../../tests/registration-options.fixture";
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});
// Fake clocks prove the hard read bound without waiting or invoking acquisition/provider operations.
it("bounds import reads at five and cancels future refresh after unmount", async () => {
  vi.useFakeTimers();
  const api = vi.fn().mockResolvedValue(pickerStatus) as API;
  const props = {
    api,
    entity: pickerOptions.entity_id,
    book: pickerOptions.book_id,
    item: pickerOptions.sync_item_id,
    account: pickerStatus,
    locked: false,
    denied: vi.fn(),
  };
  const view = render(<AccountImportStatus {...props} />);
  await act(async () => {
    await vi.advanceTimersByTimeAsync(16000);
  });
  expect(api).toHaveBeenCalledTimes(5);
  expect(screen.getByText("Importing")).toBeTruthy();
  view.unmount();
  await act(async () => {
    await vi.advanceTimersByTimeAsync(30000);
  });
  expect(api).toHaveBeenCalledTimes(5);
});

it("settled history completion stops refresh; scope denial never retains an earlier success", async () => {
  const api = vi
    .fn()
    .mockResolvedValue({
      ...pickerStatus,
      completed_generation: "1",
      readiness: "HISTORICAL_UPDATE_COMPLETE",
    }) as API;
  const props = {
    api,
    entity: pickerOptions.entity_id,
    book: pickerOptions.book_id,
    item: pickerOptions.sync_item_id,
    account: pickerStatus,
    locked: false,
    denied: vi.fn(),
  };
  const view = render(<AccountImportStatus {...props} />);
  await screen.findByText("Import complete");
  expect(api).toHaveBeenCalledOnce();
  view.rerender(
    <AccountImportStatus
      {...props}
      api={vi.fn().mockRejectedValue(new APIError(403, false)) as API}
    />,
  );
  await screen.findByText(/Import status unavailable/);
  expect(screen.queryByText("Import complete")).toBeNull();
  expect(props.denied).toHaveBeenCalledOnce();
});
