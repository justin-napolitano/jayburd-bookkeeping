import { expect, it } from "vitest";
import { accountLabels } from "./accountLabels";

it("distinguishes duplicate unnamed accounts and keeps fixed-set labels after reorder", () => {
  const first = accountLabels([
    "private-card-b",
    "private-card-a",
    "private-card-b",
  ]);
  const reordered = accountLabels(["private-card-a", "private-card-b"]);
  expect(first("private-card-a")).toBe("Bank account 1");
  expect(first("private-card-b")).toBe("Bank account 2");
  expect(reordered("private-card-a")).toBe(first("private-card-a"));
  expect(reordered("private-card-b")).toBe(first("private-card-b"));
  expect(first("unknown")).toBe("Bank account unavailable");
});

it("never treats names, masks or identifier fragments as display metadata", () => {
  const label = accountLabels(["<script>private-1234</script>"]);
  expect(label("<script>private-1234</script>")).toBe("Bank account 1");
  expect(accountLabels([])("private-1234")).toBe("Bank account unavailable");
});
