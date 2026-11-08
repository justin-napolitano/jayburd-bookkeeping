export const id = (n: number) =>
  `84000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
export const filterBook = {
  entity_id: id(1),
  book_id: id(2),
  name: "Fictional travel book",
  entity_name: "Fictional owner entity",
};
export const filterCurrency = {
  currency_version_id: id(3),
  code: "USD",
  numeric_code: "840",
  minor_unit: 2,
  valid_from: "2000-01-01",
  valid_to: null,
};
