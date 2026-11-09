import type { Page } from "playwright-core";

// The same navigation is persistent on desktop and explicitly expanded on mobile.
export async function openFinanceMenu(page: Page) {
  const menu = page.getByRole("button", { name: "Menu", exact: true });
  if (await menu.isVisible()) await menu.click();
}

// Hidden destinations remain inspectable for current-page and recovery-lock assertions.
export function financeNavItem(page: Page, name: string) {
  return page
    .getByRole("navigation", { name: "Finance tools", includeHidden: true })
    .getByRole("button", { name, exact: true, includeHidden: true });
}

export async function chooseFinanceTool(page: Page, name: string) {
  await openFinanceMenu(page);
  await financeNavItem(page, name).click();
}
