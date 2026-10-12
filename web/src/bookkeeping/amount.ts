import type { Currency } from "./types";

// Display only: preserve value exactly, including unexpected sub-minor-unit digits.
export function formatAmount(
  amount: string,
  minorUnit: number | null | undefined,
): string {
  if (
    !Number.isInteger(minorUnit) ||
    minorUnit == null ||
    minorUnit < 0 ||
    minorUnit > 9 ||
    !/^[0-9]+(?:\.[0-9]+)?$/.test(amount)
  )
    return amount;
  const [whole, supplied = ""] = amount.split(".");
  const fraction = supplied.replace(/0+$/, "").padEnd(minorUnit, "0");
  return fraction ? `${whole}.${fraction}` : whole;
}

// Currency metadata is versioned; never guess a scale from locale or a code alone.
export function currencyAt(
  currencies: Currency[],
  code: string,
  date: string,
): Currency | undefined {
  const matches = currencies.filter(
    (c) =>
      c.alphabetic_code === code &&
      c.valid_from <= date &&
      (c.valid_to === null || date < c.valid_to),
  );
  return matches.length === 1 ? matches[0] : undefined;
}
