// Demo cents stay integers; the ledger independently validates posting policy.
export const depositCents = (index: number): bigint => BigInt((index % 99) + 1);
// Closed-form expectation is independent of the per-journal generator.
export function expectedCents(count: number): bigint {
  const rest = BigInt(count % 99);
  return BigInt(Math.floor(count / 99)) * 4950n + (rest * (rest + 1n)) / 2n;
}
// Decimal serialization never routes money through floating point.
export const decimal = (value: bigint): string =>
  `${value / 100n}.${String(value % 100n).padStart(2, "0")}`;
// SQL may return trailing scale, but fractional cents must fail verification.
export function parseCents(value: string): bigint {
  if (!/^\d+(\.\d+)?$/.test(value)) throw new Error("Invalid receipt amount");
  const [whole, fraction = ""] = value.split(".");
  if (/[1-9]/.test(fraction.slice(2)))
    throw new Error("Fractional cent receipt");
  return BigInt(whole) * 100n + BigInt(fraction.slice(0, 2).padEnd(2, "0"));
}
