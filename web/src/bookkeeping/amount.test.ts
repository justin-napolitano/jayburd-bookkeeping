import { describe, expect, it } from "vitest";
import { currencyAt, formatAmount } from "./amount";
import type { Currency } from "./types";

// Fixed expected values are independent of SQL and the synthetic generator.
describe("exact monetary display", () => {
  it.each([
    ["0", 2, "0.00"],
    ["0.010000000", 2, "0.01"],
    ["12.1", 2, "12.10"],
    ["1200.000000000", 0, "1200"],
    ["0.001000000", 3, "0.001"],
    ["12.1", 3, "12.100"],
    [
      "99999999999999999999999999999.990000000",
      2,
      "99999999999999999999999999999.99",
    ],
    ["0.000000001", 9, "0.000000001"],
    ["1.005000000", 2, "1.005"],
    ["1.100000000", 0, "1.1"],
  ])("preserves %s at scale %s as %s", (amount, scale, expected) => {
    expect(formatAmount(amount as string, scale as number)).toBe(expected);
  });
  it("never invents precision when metadata is missing or invalid", () => {
    for (const scale of [null, undefined, -1, 10, 1.5, NaN])
      expect(formatAmount("1.230000000", scale)).toBe("1.230000000");
    for (const amount of ["1e2", "NaN", "1,000.00", ""])
      expect(formatAmount(amount, 2)).toBe(amount);
  });
});

// Currency versions use inclusive starts and exclusive ends, matching the ledger.
describe("currency metadata selection", () => {
  const previous: Currency = {
    currency_version_id: "old",
    alphabetic_code: "XXX",
    minor_unit: 2,
    valid_from: "2020-01-01",
    valid_to: "2025-01-01",
  };
  const current: Currency = {
    ...previous,
    currency_version_id: "new",
    minor_unit: 3,
    valid_from: "2025-01-01",
    valid_to: null,
  };
  it("selects the accounting-date version, including its boundary", () => {
    expect(currencyAt([previous, current], "XXX", "2024-12-31")).toBe(previous);
    expect(currencyAt([previous, current], "XXX", "2025-01-01")).toBe(current);
  });
  it("does not guess for missing or ambiguous metadata", () => {
    expect(
      currencyAt([previous, current], "USD", "2025-01-01"),
    ).toBeUndefined();
    expect(currencyAt([previous], "XXX", "2019-12-31")).toBeUndefined();
    expect(currencyAt([current, current], "XXX", "2025-01-01")).toBeUndefined();
  });
});
