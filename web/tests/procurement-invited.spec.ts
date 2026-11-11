import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

// docs/plans/2026-11-11-procurement-invited-firms.md: the PM names the firms
// the RFP goes to before any quote arrives, and a quote dropped on an invited
// firm's cell fills that cell.
test("firms named before their quotes are invited, and a quote fills its firm's cell", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Invited firms");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const rows = `/api/projects/${project}/procurement/rows`;
  const planner = await (await page.request.post(rows, { headers: origin, data: { discipline_id: "consultant.town_planner" } })).json();
  const architect = await (await page.request.post(rows, { headers: origin, data: { discipline_id: "consultant.architect" } })).json();
  // A firm already in the directory, named on another row.
  await page.request.post(`/api/projects/${project}/procurement/packages/${architect.id}/invitations`, { headers: origin, data: { name: "Beta Planning Pty Ltd" } });

  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  const row = page.getByRole("row", { name: /^Town Planner/ });
  const slots = row.locator("td.firm-slot");
  await expect(row.getByRole("img", { name: "Drop a proposal", exact: true })).toHaveCount(3);

  // Type a new firm's name and press Enter.
  await row.getByRole("button", { name: "Name firm 1 for Town Planner" }).click();
  const field = row.getByLabel("Firm 1 for Town Planner", { exact: true });
  await expect(field).toBeFocused();
  await field.fill("Alpah Planing");
  await field.press("Enter");
  await expect(slots.nth(0)).toContainText("Alpah Planing");
  await expect(slots.nth(0)).toContainText("Invited");

  // A misspelt name is corrected in place: click it, the name is the field.
  await row.getByRole("button", { name: "Change Alpah Planing", exact: true }).click();
  const fix = row.getByLabel("Firm 1 for Town Planner", { exact: true });
  await expect(fix).toBeFocused();
  await expect(fix).toHaveValue("Alpah Planing");
  await fix.press("Escape");
  await expect(slots.nth(0)).toContainText("Alpah Planing");
  await row.getByRole("button", { name: "Change Alpah Planing", exact: true }).click();
  await fix.fill("Alpha Planning");
  await fix.press("Enter");
  await expect(slots.nth(0)).toContainText("Alpha Planning");
  await expect(slots.nth(0)).toContainText("Invited");
  const typo = await (await page.request.get(`/api/firms?q=${encodeURIComponent("Alpah")}`)).json();
  expect(typo.firms ?? []).toHaveLength(0);

  // Choose a directory firm from the suggestions.
  await row.getByRole("button", { name: "Name firm 2 for Town Planner" }).click();
  await row.getByLabel("Firm 2 for Town Planner", { exact: true }).fill("beta");
  const suggestions = page.getByRole("group", { name: "Firms in your directory" });
  await expect(suggestions.getByRole("button", { name: /Beta Planning Pty Ltd/ })).toBeVisible();
  await page.screenshot({ path: "../.tools/procurement-invited-naming.png" });
  await suggestions.getByRole("button", { name: /Beta Planning Pty Ltd/ }).click();
  await expect(slots.nth(1)).toContainText("Beta Planning Pty Ltd");
  await expect(slots.nth(1)).toContainText("Invited");

  // Escape names nobody.
  await row.getByRole("button", { name: "Name firm 3 for Town Planner" }).click();
  await row.getByLabel("Firm 3 for Town Planner", { exact: true }).fill("Never Mind");
  await row.getByLabel("Firm 3 for Town Planner", { exact: true }).press("Escape");
  await expect(row.getByLabel("Firm 3 for Town Planner", { exact: true })).toHaveCount(0);
  await expect(row.getByRole("button", { name: "Name firm 3 for Town Planner" })).toBeAttached();
  // Nothing to price yet: the row still offers Prices, not Review.
  await expect(row.getByRole("button", { name: "Prices for Town Planner" })).toBeVisible();

  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(slots.nth(0)).toContainText("Alpha Planning");
  await expect(slots.nth(1)).toContainText("Beta Planning Pty Ltd");
  await page.screenshot({ path: "../.tools/procurement-invited.png" });

  // Beta's quote, printed under another name, dropped on Beta's cell.
  const schedule = await (await page.request.get(`/api/projects/${project}/procurement/packages/${planner.id}/prices`)).json();
  const lines = schedule.items.filter((i: { base: boolean }) => i.base).map((i: { label: string }) => `${i.label}~100000`).join(",");
  const pdf = (await (await page.request.get(`/__e2e/proposal.pdf?firm=Printed%20Planners&abn=7&lines=${encodeURIComponent(lines)}`)).body()).toString("base64");
  await slots.nth(1).evaluate((el, b64) => {
    const dt = new DataTransfer();
    dt.items.add(new File([Uint8Array.from(atob(b64), c => c.charCodeAt(0))], "Fee proposal (Beta).pdf", { type: "application/pdf" }));
    for (const type of ["dragenter", "dragover", "drop"]) el.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
  }, pdf);
  await expect(slots.nth(1).locator('button.firm-chip:not([data-status="invited"])')).toContainText("Beta Planning Pty Ltd", { timeout: 20_000 });
  await expect(slots.nth(1)).not.toContainText("Invited");
  await expect(slots.nth(0)).toContainText("Invited");
  await expect(row.getByRole("button", { name: "Review Town Planner" })).toBeVisible({ timeout: 20_000 });

  // Removing Alpha's invitation leaves Beta's quote, now in Firm 1.
  await row.getByRole("button", { name: "Remove invitation for Alpha Planning" }).click();
  await expect(row.locator('.firm-chip[data-status="invited"]')).toHaveCount(0);
  await expect(slots.nth(0)).toContainText("Beta Planning Pty Ltd");
});
