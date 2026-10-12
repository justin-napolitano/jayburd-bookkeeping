import { describe, expect, it } from "vitest";
import { decimal, depositCents, expectedCents, parseCents } from "./volume";
describe("volume exact-cent oracle", () => {
  it("predicts thousands independently from the generator", () => {
    expect(expectedCents(1000)).toBe(49555n);
    expect(expectedCents(5000)).toBe(248775n);
    expect(expectedCents(10000)).toBe(499951n);
    for (const count of [1000, 5000, 10000]) {
      let total = 0n;
      for (let i = 0; i < count; i++) total += depositCents(i);
      expect(total).toBe(expectedCents(count));
    }
  });
  it("serializes exact cents and rejects fractions", () => {
    expect(decimal(49555n)).toBe("495.55");
    expect(parseCents("495.550000000")).toBe(49555n);
    expect(() => parseCents("0.001")).toThrow();
    expect(depositCents(98)).toBe(99n);
    expect(depositCents(99)).toBe(1n);
  });
});
