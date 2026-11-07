import type { Catalog, Selector } from "./types";
const decimal = /^(0|[1-9][0-9]{0,28})(\.[0-9]{1,9})?$/;
const scaled = (value: string) => {
  const [whole, fraction = ""] = value.split(".");
  return BigInt(whole + fraction.padEnd(9, "0"));
};
// Feedback preserves the owner's exact input; it never rounds, converts or executes a query locally.
export function collectionFilterError(
  selector: Selector,
  catalog: Catalog,
): string | null {
  if (
    "value" in selector.merchant &&
    (!selector.merchant.value.length ||
      [...selector.merchant.value].length > 100)
  )
    return "Enter merchant text with 1–100 characters.";
  if (!selector.amount) return null;
  const amount = selector.amount,
    currency = catalog.currencies.find(
      (value) => value.currency_version_id === amount.currency_version_id,
    );
  if (!currency)
    return "Choose a current dated original currency before filtering amounts.";
  for (const value of [amount.minimum, amount.maximum]) {
    if (value === null) continue;
    if (value.length > 39 || !decimal.test(value))
      return "Enter original amounts as nonnegative decimal strings with at most 29 integer and 9 fractional digits.";
    const fraction = (value.split(".")[1] ?? "").replace(/0+$/, "");
    if (fraction.length > currency.minor_unit)
      return `Use the dated ${currency.code} precision of ${currency.minor_unit} decimal places; amounts are never rounded.`;
  }
  if (
    amount.minimum !== null &&
    amount.maximum !== null &&
    scaled(amount.minimum) > scaled(amount.maximum)
  )
    return "Minimum original amount must not exceed maximum original amount.";
  return null;
}
