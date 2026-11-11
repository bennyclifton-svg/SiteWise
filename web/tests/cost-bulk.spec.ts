import { expect, test } from "@playwright/test";

// Bulk actions on selected Cost Plan lines from the right-click menu, as in
// the document register and the Accommodation schedule: Shift-click selects
// lines without selecting text; exclude, duplicate, set category and delete
// act on the whole selection in one write.
test("cost lines: right-click menu and bulk actions", async ({ page }, info) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Cost bulk");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.getByRole("button", { name: "Costs", exact: true }).click();
  for (const name of ["Alpha allowance", "Beta allowance", "Gamma allowance"]) {
    await page.getByLabel("New Contingency item", { exact: true }).fill(name);
    await page.getByLabel("New Contingency item", { exact: true }).press("Enter");
    await expect(page.locator('.cost-row [data-col="item"]').getByText(name, { exact: true })).toBeVisible();
  }
  const row = (name: string) => page.locator(".cost-row").filter({ has: page.locator('[data-col="item"]').getByText(name, { exact: true }) });
  const menu = page.locator('.reg-menu[role="menu"]');

  // Shift-click selects the range and leaves no text selected.
  await row("Alpha allowance").locator('[data-col="code"]').click();
  await row("Gamma allowance").locator('[data-col="code"]').click({ modifiers: ["Shift"] });
  await expect(page.locator('.cost-row[aria-selected="true"]')).toHaveCount(3);
  expect(await page.evaluate(() => window.getSelection()?.toString() ?? "")).toBe("");

  // Right-click on a selected line acts on the selection.
  await row("Beta allowance").locator('[data-col="code"]').click({ button: "right" });
  await expect(menu).toContainText("3 lines");
  await expect(menu.getByRole("menuitem")).toHaveText(["Set category for 3 lines", "Exclude 3 from totals", "Duplicate 3 lines", "Delete 3 lines"]);
  await page.screenshot({ path: info.outputPath("cost-bulk-menu.png") });
  await menu.getByRole("menuitem", { name: "Exclude 3 from totals" }).click();
  await expect(row("Alpha allowance")).toHaveClass(/cost-excluded/);
  await expect(row("Gamma allowance")).toHaveClass(/cost-excluded/);

  // Set one category on two lines.
  await row("Alpha allowance").locator('[data-col="code"]').click();
  await row("Beta allowance").locator('[data-col="code"]').click({ modifiers: ["Control"] });
  await row("Alpha allowance").locator('[data-col="code"]').click({ button: "right" });
  await menu.getByRole("menuitem", { name: "Set category for 2 lines" }).click();
  const picker = page.getByRole("dialog", { name: "Category for 2 lines" });
  await picker.getByRole("combobox").selectOption("escalation");
  await picker.getByRole("button", { name: "Apply" }).click();
  await expect(row("Alpha allowance").locator('[data-col="category"]')).toContainText(/escalation/i);
  await expect(row("Beta allowance").locator('[data-col="category"]')).toContainText(/escalation/i);
  await expect(row("Gamma allowance").locator('[data-col="category"]')).toContainText(/contingency/i);

  // Duplicate one line: the copy sits straight after it.
  await row("Gamma allowance").locator('[data-col="code"]').click({ button: "right" });
  await menu.getByRole("menuitem", { name: "Duplicate", exact: true }).click();
  await expect(row("Gamma allowance (copy)")).toBeVisible();

  // Delete two lines with the Delete key, after one confirmation.
  await row("Alpha allowance").locator('[data-col="code"]').click();
  await row("Gamma allowance (copy)").locator('[data-col="code"]').click({ modifiers: ["Control"] });
  await expect(page.getByRole("button", { name: "Delete 2 selected lines" })).toBeVisible();
  let asked = "";
  page.once("dialog", (d) => { asked = d.message(); void d.accept(); });
  await page.keyboard.press("Delete");
  await expect(row("Alpha allowance")).toHaveCount(0);
  await expect(row("Gamma allowance (copy)")).toHaveCount(0);
  expect(asked).toContain("Delete 2 lines?");
  await expect(row("Beta allowance")).toBeVisible();
  await expect(row("Gamma allowance")).toBeVisible();
});
