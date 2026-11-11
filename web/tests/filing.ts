import { expect, type Page } from "@playwright/test";

/** Shows the open document's filing. It sits beside the page on a wide pane,
 * takes the page's place on a narrow one (closed until asked for), and is
 * shown outright for a file with no preview. */
export async function showFiling(page: Page) {
  const filing = page.locator(".document-workspace .document-filing");
  await expect(page.locator(".document-workspace")).toBeVisible();
  const toggle = page.getByRole("button", { name: /^Filing/ });
  if ((await toggle.count()) > 0 && (await toggle.getAttribute("aria-expanded")) !== "true") await toggle.click();
  await expect(filing).toBeVisible();
}

/** Leaves the open document by its Back button, named for the view behind it. */
export async function closeDocument(page: Page) {
  await page.locator(".document-workspace").getByRole("button", { name: /^Back to / }).click();
}
