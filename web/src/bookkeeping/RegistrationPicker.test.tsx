import { afterEach, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import {
  RegistrationPicker,
  checkedRegistrationOptions,
} from "./RegistrationPicker";
import { APIError, type API } from "./api";
import { pickerOptions } from "../../tests/registration-options.fixture";
afterEach(cleanup);
// Inject only synthetic reads; every preparation is an observable explicit selection.
function fixture(api: API = vi.fn().mockResolvedValue(pickerOptions) as API) {
  return {
    api,
    entity: pickerOptions.entity_id,
    book: pickerOptions.book_id,
    attempt: pickerOptions.attempt_id,
    locked: false,
    prepare: vi.fn().mockResolvedValue(undefined),
    denied: vi.fn(),
    onReview: vi.fn(),
  };
}

it("selects only explicit eligible accounts and preserves registered source handoff without technical fields", async () => {
  const props = fixture();
  render(<RegistrationPicker {...props} />);
  await screen.findByText("Personal");
  const boxes = screen.getAllByRole("checkbox") as HTMLInputElement[];
  expect(boxes).toHaveLength(2);
  expect(boxes.every((box) => !box.checked)).toBe(true);
  expect(
    screen.queryByLabelText(/Account code|Currency version|Binding date/),
  ).toBeNull();
  const review = screen.getByRole("button", {
    name: "Review selected accounts",
  }) as HTMLButtonElement;
  expect(review.disabled).toBe(true);
  fireEvent.click(boxes[1]);
  fireEvent.click(review);
  await waitFor(() =>
    expect(props.prepare).toHaveBeenCalledWith(["private-card-b"]),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "View transactions for Checking 1" }),
  );
  expect(props.onReview).toHaveBeenCalledWith(
    pickerOptions.accounts[0].source_account_id,
  );
  expect(screen.queryByText("private-card-a")).toBeNull();
});

it("allows multiple accounts within server capacity and refresh clears the reviewed selection", async () => {
  const limited = { ...pickerOptions, remaining_sources: "1" };
  const props = fixture(vi.fn().mockResolvedValue(limited) as API);
  render(<RegistrationPicker {...props} />);
  await screen.findByText("Personal");
  const boxes = screen.getAllByRole("checkbox") as HTMLInputElement[];
  fireEvent.click(boxes[0]);
  expect(boxes[1].disabled).toBe(true);
  fireEvent.click(boxes[0]);
  expect(boxes[1].disabled).toBe(false);
  fireEvent.click(boxes[1]);
  fireEvent.click(
    screen.getByRole("button", { name: "Refresh account choices" }),
  );
  await waitFor(() =>
    expect(
      (
        screen.getByRole("button", {
          name: "Review selected accounts",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true),
  );
  expect(props.prepare).not.toHaveBeenCalled();
});

it("rejects foreign targets, duplicate identities, unknown reasons and malformed registered flags", () => {
  const props = fixture();
  for (const value of [
    { ...pickerOptions, book_id: pickerOptions.entity_id },
    { ...pickerOptions, remaining_sources: 1 },
    {
      ...pickerOptions,
      accounts: [pickerOptions.accounts[1], pickerOptions.accounts[1]],
    },
    {
      ...pickerOptions,
      accounts: [{ ...pickerOptions.accounts[1], reason: "toString" }],
    },
    {
      ...pickerOptions,
      accounts: [
        { ...pickerOptions.accounts[0], eligible: true, reason: null },
      ],
    },
    {
      ...pickerOptions,
      accounts: [
        {
          ...pickerOptions.accounts[1],
          eligible: false,
          reason: "already_registered",
        },
      ],
    },
  ]) {
    expect(() =>
      checkedRegistrationOptions(
        value as typeof pickerOptions,
        props.entity,
        props.book,
        props.attempt,
      ),
    ).toThrow();
  }
});

it("clears choices on denial and ignores a previous attempt's delayed reply", async () => {
  let resolve!: (value: typeof pickerOptions) => void;
  const delayed = new Promise<typeof pickerOptions>((r) => {
    resolve = r;
  });
  const api = vi
    .fn()
    .mockReturnValueOnce(delayed)
    .mockResolvedValueOnce({
      ...pickerOptions,
      attempt_id: pickerOptions.book_id,
      book_name: "Current",
    }) as API;
  const props = fixture(api),
    view = render(<RegistrationPicker {...props} />);
  view.rerender(
    <RegistrationPicker {...props} attempt={pickerOptions.book_id} />,
  );
  await screen.findByText("Current");
  resolve(pickerOptions);
  await waitFor(() => expect(screen.queryByText("Personal")).toBeNull());
  view.unmount();
  const denied = fixture(
    vi.fn().mockRejectedValue(new APIError(403, false)) as API,
  );
  render(<RegistrationPicker {...denied} />);
  await screen.findByRole("alert");
  expect(denied.denied).toHaveBeenCalledOnce();
  expect(screen.queryByRole("checkbox")).toBeNull();
});
