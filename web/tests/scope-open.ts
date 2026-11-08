import type { Locator, Page } from "@playwright/test";

function systemLine(scope: Page | Locator, label: string) {
  const page = "goto" in scope ? scope : scope.page();
  return scope.locator(".sw-line").filter({ has: page.getByRole("checkbox", { name: label, exact: true }) });
}

/**
 * Opens a Profile system's pop-up (its dot): sources, then Works, Supplied
 * by and "Parts and split…". There is no Works or By column any more.
 */
export async function openSystemPop(scope: Page | Locator, label: string) {
  const line = systemLine(scope, label);
  const details = line.locator("details.sw-evpop");
  if (!(await details.evaluate((d) => (d as HTMLDetailsElement).open))) await line.locator("summary.sw-evmark").click();
  return line;
}

/** Opens a Profile system's work items from "Parts and split…" in its pop-up. */
export async function openSystemWork(scope: Page | Locator, label: string) {
  const line = await openSystemPop(scope, label);
  // Saved work loads after the profile; the button appears once it has.
  await line.getByRole("button", { name: "Parts and split…", exact: true }).click();
  return line;
}
