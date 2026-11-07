import { describe, expect, it, vi } from "vitest";
import { APIError, type API } from "../bookkeeping/api";
import {
  createCollectionJournalReader,
  validCollectionJournal,
} from "./journal";
const id = (n: number) =>
  `82000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
const identity = { entity_id: id(1), book_id: id(2), journal_id: id(3) };
const journal = {
  ...identity,
  description: "Fictional posted purchase",
  accounting_date: "2026-10-01",
  book_revision: "4",
  functional_currency: "USD",
  functional_currency_version_id: id(4),
  lines: [
    {
      line_number: 1,
      account_id: id(5),
      side: "debit",
      transaction_currency: "EUR",
      transaction_currency_version_id: id(6),
      transaction_amount: "9007199254740993.01",
      functional_amount: "9907919180215092.311",
    },
    {
      line_number: 2,
      account_id: id(7),
      side: "credit",
      transaction_currency: "EUR",
      transaction_currency_version_id: id(6),
      transaction_amount: "9007199254740993.01",
      functional_amount: "9907919180215092.311",
    },
  ],
};
describe("authorized linked journal read", () => {
  it("reads only the exact existing authorized route and preserves original strings", async () => {
    const api = vi.fn().mockResolvedValue(journal) as unknown as API;
    const result = await createCollectionJournalReader(api)(identity);
    expect(api).toHaveBeenCalledExactlyOnceWith(
      `/v1/entities/${identity.entity_id}/books/${identity.book_id}/journals/${identity.journal_id}`,
    );
    expect(result.lines[0].transaction_amount).toBe("9007199254740993.01");
  });
  it("rejects mismatched scope, malformed/duplicate lines and raw diagnostics", async () => {
    for (const bad of [
      { ...journal, book_id: id(8) },
      { ...journal, lines: [journal.lines[0], journal.lines[0]] },
      {
        ...journal,
        lines: [
          { ...journal.lines[0], transaction_amount: 9007199254740993 },
          journal.lines[1],
        ],
      },
    ])
      expect(validCollectionJournal(bad, identity)).toBe(false);
    const api = vi
      .fn()
      .mockRejectedValue(new APIError(403, false, "read")) as unknown as API;
    await expect(
      createCollectionJournalReader(api)(identity),
    ).rejects.toMatchObject({ kind: "denied" });
    const wrong = vi
      .fn()
      .mockResolvedValue({ ...journal, journal_id: id(9) }) as unknown as API;
    await expect(
      createCollectionJournalReader(wrong)(identity),
    ).rejects.toMatchObject({ kind: "unavailable" });
  });
});
