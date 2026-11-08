import { describe, expect, it } from "vitest";
import {
  collectionCatalog,
  collectionQuery,
} from "../../tests/collection-fixture";
import { collectionFilterError } from "./filters";

const currency = {
  currency_version_id: "81000000-0000-4000-8000-000000000001",
  code: "USD",
  numeric_code: "840",
  minor_unit: 2,
  valid_from: "2000-01-01",
  valid_to: null,
};
const catalog = { ...collectionCatalog, currencies: [currency] };
const query = (minimum: string | null, maximum: string | null) =>
  collectionQuery({
    amount: {
      currency_version_id: currency.currency_version_id,
      minimum,
      maximum,
      direction: "all",
    },
  });
describe("original-unit filter feedback", () => {
  it("keeps large exact ranges and trailing zeros without floating-point rounding", () => {
    expect(
      collectionFilterError(
        query("9007199254740993.0100", "9007199254740993.02"),
        catalog,
      ),
    ).toBeNull();
    expect(
      collectionFilterError(
        query("9007199254740993.02", "9007199254740993.01"),
        catalog,
      ),
    ).toMatch(/Minimum/);
    expect(collectionFilterError(query("0.0000", null), catalog)).toBeNull();
  });
  it.each([
    "01.00",
    "-1",
    "1e3",
    "1,000.00",
    "NaN",
    "1.001",
    "100000000000000000000000000000",
  ])(
    "rejects unsupported original amount %s without changing input",
    (value) => {
      const request = query(value, null);
      expect(collectionFilterError(request, catalog)).not.toBeNull();
      expect(request.amount?.minimum).toBe(value);
    },
  );
  it("requires a current dated currency and codepoint-bounded merchant text", () => {
    expect(collectionFilterError(query("1", null), collectionCatalog)).toMatch(
      /dated/,
    );
    expect(
      collectionFilterError(
        collectionQuery({
          merchant: { kind: "exact", value: "😀".repeat(100) },
        }),
        catalog,
      ),
    ).toBeNull();
    expect(
      collectionFilterError(
        collectionQuery({
          merchant: { kind: "contains", value: "😀".repeat(101) },
        }),
        catalog,
      ),
    ).toMatch(/1–100/);
    expect(
      collectionFilterError(
        collectionQuery({ merchant: { kind: "exact", value: "" } }),
        catalog,
      ),
    ).not.toBeNull();
  });
});
