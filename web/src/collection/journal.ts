import { APIError, type API } from "../bookkeeping/api";
import type { Journal } from "../bookkeeping/types";
import { CollectionFailure } from "./types";

// An anchor is a current authorized server projection, never a client-supplied grant or URL.
export type JournalIdentity = {
  entity_id: string;
  book_id: string;
  journal_id: string;
};
export type JournalReader = (identity: JournalIdentity) => Promise<Journal>;
const record = (value: unknown): value is Record<string, unknown> =>
  !!value && typeof value === "object" && !Array.isArray(value);
const uuid = (value: unknown): value is string =>
  typeof value === "string" &&
  /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/.test(value) &&
  value !== "00000000-0000-0000-0000-000000000000";
const text = (value: unknown, bound: number) =>
  typeof value === "string" && [...value].length <= bound;
const amount = (value: unknown) =>
  typeof value === "string" &&
  /^(0|[1-9][0-9]{0,28})(\.[0-9]{1,9})?$/.test(value);
const currency = (value: unknown) =>
  typeof value === "string" && /^[A-Z]{3}$/.test(value);

// Bound the existing immutable journal projection before rendering any original-unit values.
export function validCollectionJournal(
  value: unknown,
  identity: JournalIdentity,
): value is Journal {
  if (
    !record(value) ||
    value.entity_id !== identity.entity_id ||
    value.book_id !== identity.book_id ||
    value.journal_id !== identity.journal_id ||
    !text(value.description, 500) ||
    typeof value.accounting_date !== "string" ||
    !/^\d{4}-\d{2}-\d{2}$/.test(value.accounting_date) ||
    typeof value.book_revision !== "string" ||
    !/^(0|[1-9][0-9]{0,18})$/.test(value.book_revision) ||
    !currency(value.functional_currency) ||
    !uuid(value.functional_currency_version_id) ||
    !Array.isArray(value.lines) ||
    value.lines.length > 1000 ||
    value.lines.length < 2
  )
    return false;
  const numbers = new Set<number>();
  return value.lines.every((line) => {
    if (
      !record(line) ||
      !uuid(line.account_id) ||
      !["debit", "credit"].includes(String(line.side)) ||
      !currency(line.transaction_currency) ||
      !uuid(line.transaction_currency_version_id) ||
      !amount(line.transaction_amount) ||
      !amount(line.functional_amount) ||
      !Number.isInteger(line.line_number) ||
      Number(line.line_number) < 1 ||
      Number(line.line_number) > 1000 ||
      numbers.has(Number(line.line_number))
    )
      return false;
    numbers.add(Number(line.line_number));
    return true;
  });
}

// Reuse the existing authenticated book read; private authority never grants independent journal access.
export function createCollectionJournalReader(api: API): JournalReader {
  return async (identity) => {
    if (
      ![identity.entity_id, identity.book_id, identity.journal_id].every(uuid)
    )
      throw new CollectionFailure("unavailable");
    try {
      const value = await api<unknown>(
        `/v1/entities/${identity.entity_id}/books/${identity.book_id}/journals/${identity.journal_id}`,
      );
      if (!validCollectionJournal(value, identity))
        throw new CollectionFailure("unavailable");
      return value;
    } catch (reason) {
      if (reason instanceof CollectionFailure) throw reason;
      if (reason instanceof APIError && [401, 403, 404].includes(reason.status))
        throw new CollectionFailure("denied");
      throw new CollectionFailure("unavailable");
    }
  };
}
