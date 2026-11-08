import { useId, type ReactNode } from "react";
import type { Catalog, Selector, SortField } from "./types";

const sortFields: [SortField, string][] = [
  ["date", "Date"],
  ["account", "Account"],
  ["description", "Description"],
  ["merchant", "Merchant"],
  ["amount", "Original amount"],
  ["currency", "Original currency"],
  ["book", "Book"],
  ["source_state", "Source status"],
  ["journal_state", "Journal status"],
  ["topic", "Primary topic"],
  ["tags", "Strategic tags"],
  ["institution", "Institution availability"],
  ["legal_owner", "Legal owner availability"],
];

// Native labels keep stable accessible names independently of selected option text.
function Choice({
  label,
  value,
  onChange,
  children,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  children: ReactNode;
  disabled?: boolean;
}) {
  const id = useId();
  return (
    <label>
      <span id={id}>{label}</span>
      <select
        disabled={disabled}
        aria-labelledby={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        {children}
      </select>
    </label>
  );
}

// These controls only construct a bounded selector; SQL owns matching, ordering, totals and authority.
export function CollectionFilters({
  draft,
  catalog,
  onChange,
}: {
  draft: Selector;
  catalog: Catalog;
  onChange: (value: Selector) => void;
}) {
  const patch = (value: Partial<Selector>) => onChange({ ...draft, ...value });
  return (
    <>
      <Choice
        label="Merchant match"
        value={draft.merchant.kind}
        onChange={(value) =>
          patch({
            merchant:
              value === "exact" || value === "contains"
                ? { kind: value, value: "" }
                : { kind: value as "all" | "missing" },
          })
        }
      >
        <option value="all">All merchants</option>
        <option value="missing">Merchant not supplied</option>
        <option value="exact">Exact merchant</option>
        <option value="contains">Merchant contains</option>
      </Choice>
      <label>
        Merchant text
        <input
          value={"value" in draft.merchant ? draft.merchant.value : ""}
          disabled={!("value" in draft.merchant)}
          maxLength={200}
          onChange={(event) => {
            "value" in draft.merchant &&
              patch({
                merchant: { ...draft.merchant, value: event.target.value },
              });
          }}
        />
      </label>
      <Choice
        label="Original currency for amount filter"
        value={draft.amount?.currency_version_id ?? ""}
        onChange={(value) =>
          patch({
            amount: value
              ? {
                  currency_version_id: value,
                  minimum: null,
                  maximum: null,
                  direction: "all",
                }
              : null,
          })
        }
      >
        <option value="">No amount filter</option>
        {catalog.currencies.map((currency) => (
          <option
            key={currency.currency_version_id}
            value={currency.currency_version_id}
          >
            {currency.code} ({currency.valid_from} to{" "}
            {currency.valid_to ?? "present"})
          </option>
        ))}
      </Choice>
      <label>
        Minimum original amount
        <input
          inputMode="decimal"
          value={draft.amount?.minimum ?? ""}
          disabled={!draft.amount}
          maxLength={39}
          onChange={(event) =>
            draft.amount &&
            patch({
              amount: { ...draft.amount, minimum: event.target.value || null },
            })
          }
        />
      </label>
      <label>
        Maximum original amount
        <input
          inputMode="decimal"
          value={draft.amount?.maximum ?? ""}
          disabled={!draft.amount}
          maxLength={39}
          onChange={(event) =>
            draft.amount &&
            patch({
              amount: { ...draft.amount, maximum: event.target.value || null },
            })
          }
        />
      </label>
      <Choice
        disabled={!draft.amount}
        label="Amount direction"
        value={draft.amount?.direction ?? "all"}
        onChange={(value) =>
          draft.amount &&
          patch({
            amount: {
              ...draft.amount,
              direction: value as NonNullable<Selector["amount"]>["direction"],
            },
          })
        }
      >
        <option value="all">All directions</option>
        <option value="inflow">Inflow</option>
        <option value="outflow">Outflow</option>
        <option value="zero">Zero movement</option>
      </Choice>
      <Choice
        label="Specific book"
        value={
          draft.book.kind === "book"
            ? `book:${draft.book.entity_id}:${draft.book.book_id}`
            : draft.book.kind
        }
        onChange={(value) =>
          patch({
            book: value.startsWith("book:")
              ? {
                  kind: "book",
                  entity_id: value.split(":")[1],
                  book_id: value.split(":")[2],
                }
              : { kind: value as "all" | "unassigned" | "unavailable" },
          })
        }
      >
        <option value="all">All book states</option>
        <option value="unassigned">Unassigned</option>
        <option value="unavailable">Book metadata unavailable</option>
        {catalog.books.map((book) => (
          <option
            key={book.entity_id + "/" + book.book_id}
            value={`book:${book.entity_id}:${book.book_id}`}
          >
            {book.entity_name} / {book.name}
          </option>
        ))}
      </Choice>
      <Choice
        label="Source status"
        value={draft.status.source}
        onChange={(value) =>
          patch({
            status: {
              ...draft.status,
              source: value as Selector["status"]["source"],
            },
          })
        }
      >
        {["all", "booked", "pending", "removed", "held"].map((value) => (
          <option key={value} value={value}>
            {value === "all" ? "All source states" : value}
          </option>
        ))}
      </Choice>
      <Choice
        label="Journal status"
        value={draft.status.journal}
        onChange={(value) =>
          patch({
            status: {
              ...draft.status,
              journal: value as Selector["status"]["journal"],
            },
          })
        }
      >
        {[
          "all",
          "unposted",
          "posted",
          "correction_required",
          "unavailable",
        ].map((value) => (
          <option key={value} value={value}>
            {value === "all"
              ? "All journal states"
              : value.replaceAll("_", " ")}
          </option>
        ))}
      </Choice>
      <Choice
        label="Institution"
        value={draft.institution.kind}
        onChange={(value) =>
          patch({ institution: { kind: value as "all" | "unavailable" } })
        }
      >
        <option value="all">All institution states</option>
        <option value="unavailable">Institution unavailable</option>
      </Choice>
      <Choice
        label="Legal owner"
        value={draft.legal_owner.kind}
        onChange={(value) =>
          patch({ legal_owner: { kind: value as "all" | "unknown" } })
        }
      >
        <option value="all">All owner states</option>
        <option value="unknown">Legal owner unknown</option>
      </Choice>
      <p>
        Merchant text uses case-insensitive literal matching. Amounts are
        inclusive original magnitudes within one dated currency; end dates are
        exclusive. Amount sorting groups currencies and their versions before
        magnitude, with unavailable money last. Institution and legal owner
        metadata are unavailable; a book or account name does not establish
        ownership.
      </p>
      <Choice
        label="Sort transactions by"
        value={draft.sort.field}
        onChange={(value) =>
          patch({ sort: { ...draft.sort, field: value as SortField } })
        }
      >
        {sortFields.map(([field, label]) => (
          <option key={field} value={field}>
            {label}
          </option>
        ))}
      </Choice>
      <Choice
        label="Sort direction"
        value={draft.sort.order}
        onChange={(value) =>
          patch({ sort: { ...draft.sort, order: value as "asc" | "desc" } })
        }
      >
        <option value="asc">Ascending</option>
        <option value="desc">Descending</option>
      </Choice>
    </>
  );
}
