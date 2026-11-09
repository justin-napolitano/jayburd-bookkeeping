import { useId, useRef, useState } from "react";
import type { BookView } from "./BookWorkspace";
import styles from "./FinanceNavigation.module.css";

export type FinanceTool = "Position" | "All transactions" | BookView;
const daily: { tool: FinanceTool; label: string }[] = [
  { tool: "Money overview", label: "Overview" },
  { tool: "All transactions", label: "Transactions" },
  { tool: "Connections", label: "Accounts" },
];
const bookkeeping: { tool: FinanceTool; label: string }[] = [
  { tool: "Review transactions", label: "Review accounting" },
  { tool: "Account activity", label: "Posted entries" },
  { tool: "Position", label: "Position" },
  { tool: "Reports", label: "Reports" },
  { tool: "Accounts", label: "Chart of accounts" },
];

// Presentation labels preserve the existing tool identities and authorized receiving views.
export function financeToolLabel(tool: FinanceTool): string {
  return (
    [...daily, ...bookkeeping].find((item) => item.tool === tool)?.label ??
    "Book overview"
  );
}

// One responsive navigation surface; mobile expansion never enables a locked destination.
export function FinanceNavigation({
  tool,
  locked,
  loading,
  onNavigate,
  onRefresh,
  onSignOut,
}: {
  tool: FinanceTool;
  locked: boolean;
  loading: boolean;
  onNavigate: (tool: FinanceTool) => void;
  onRefresh: () => void;
  onSignOut?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const panelID = useId(),
    dailyID = useId(),
    bookID = useId();
  const menu = useRef<HTMLButtonElement>(null);
  const choose = (next: FinanceTool) => {
    if (locked) return;
    setOpen(false);
    onNavigate(next);
  };
  const destinations = (items: typeof daily) =>
    items.map((item) => (
      <button
        key={item.tool}
        type="button"
        className={styles.destination}
        disabled={locked}
        aria-current={tool === item.tool ? "page" : undefined}
        onClick={() => choose(item.tool)}
      >
        {item.label}
      </button>
    ));
  return (
    <aside
      className={`finance-navigation ${styles.sidebar}`}
      aria-label="Workspace navigation"
      onKeyDown={(event) => {
        if (event.key === "Escape" && open) {
          event.preventDefault();
          setOpen(false);
          menu.current?.focus();
        }
      }}
    >
      <div className={styles.header}>
        <div className={styles.identity}>
          <span className={styles.brand}>Jayburd</span>
          <span className={styles.current}>{financeToolLabel(tool)}</span>
        </div>
        <button
          ref={menu}
          className={styles.menu}
          type="button"
          aria-expanded={open}
          aria-controls={panelID}
          onClick={() => setOpen((value) => !value)}
        >
          {open ? "Close menu" : "Menu"}
          <span aria-hidden="true">{open ? "−" : "+"}</span>
        </button>
      </div>
      <div id={panelID} className={styles.panel} data-open={open}>
        <nav aria-label="Finance tools" className={styles.navigation}>
          <div role="group" aria-labelledby={dailyID} className={styles.group}>
            <p id={dailyID} className={styles.groupTitle}>
              Banking
            </p>
            {destinations(daily)}
          </div>
          <div role="group" aria-labelledby={bookID} className={styles.group}>
            <p id={bookID} className={styles.groupTitle}>
              Accounting
            </p>
            {destinations(bookkeeping)}
          </div>
        </nav>
        {locked && (
          <p className={styles.lockNotice}>
            Finish or recover your saved changes to continue.
          </p>
        )}
        <div className={styles.utilities}>
          <button
            type="button"
            disabled={loading || locked}
            onClick={() => {
              setOpen(false);
              onRefresh();
            }}
          >
            Refresh books
          </button>
          {onSignOut && (
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                onSignOut();
              }}
            >
              Sign out
            </button>
          )}
        </div>
      </div>
    </aside>
  );
}
