import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
const { api } = vi.hoisted(() => ({ api: vi.fn() }));
vi.mock("../src/bookkeeping/api", () => ({ createAPI: () => api }));
import { VolumeViewer } from "./VolumeViewer";
afterEach(() => {
  cleanup();
  api.mockReset();
});
const workspace = {
  currencies: [
    {
      alphabetic_code: "USD",
      minor_unit: 2,
      valid_from: "2020-01-01",
      valid_to: null,
      currency_version_id: "currency",
    },
  ],
};
describe("volume command scheduling", () => {
  it("retries the unchanged registration body and key after an unknown outcome", async () => {
    api
      .mockResolvedValueOnce({ command: null })
      .mockResolvedValueOnce(workspace)
      .mockRejectedValueOnce(new Error("Unknown registration outcome"))
      .mockRejectedValueOnce(new Error("Still unknown"));
    render(<VolumeViewer onChanged={() => {}} />);
    fireEvent.click(screen.getByRole("button", { name: "Start volume run" }));
    fireEvent.click(
      await screen.findByRole("button", {
        name: "Retry same volume operation",
      }),
    );
    await screen.findByText("Still unknown");
    expect(api).toHaveBeenCalledTimes(4);
    expect(api.mock.calls[3]).toEqual(api.mock.calls[2]);
    expect(
      screen.getByRole("button", { name: "Start volume run" }),
    ).toHaveProperty("disabled", true);
  });
  it("does not schedule a new request after unmounting during discovery", async () => {
    let resolve!: (value: unknown) => void;
    api.mockImplementationOnce(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const view = render(<VolumeViewer onChanged={() => {}} />);
    fireEvent.click(screen.getByRole("button", { name: "Start volume run" }));
    await waitFor(() => expect(api).toHaveBeenCalledTimes(1));
    view.unmount();
    resolve({ command: null });
    await new Promise((r) => setTimeout(r, 0));
    expect(api).toHaveBeenCalledTimes(1);
  });
});
