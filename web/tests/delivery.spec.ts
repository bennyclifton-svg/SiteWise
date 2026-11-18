import { expect, test, type Page } from "@playwright/test";

// The register (docs/plans/2026-11-17-rfis-review-comments.md, Task 5): one
// table, the grey new row, Go's numbers and due dates, a possible answer
// accepted with Undo, keyboard selection, a 409 retry, and a Kind filter
// that lists only kinds with rows. Synthetic project; no Jev, no model.

const headers = { Origin: "http://127.0.0.1:4173" };

/** Five working days after today in Sydney, Monday to Friday (no Program). */
function dueIn5(): string {
  const today = new Intl.DateTimeFormat("en-CA", { timeZone: "Australia/Sydney", year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
  const d = new Date(`${today}T00:00:00Z`);
  const working = (x: Date) => x.getUTCDay() !== 0 && x.getUTCDay() !== 6;
  for (let n = 0; n < 5;) { d.setUTCDate(d.getUTCDate() + 1); if (working(d)) n++; }
  return d.toISOString().slice(0, 10);
}

async function openRegister(page: Page, name: string) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.getByRole("button", { name: "Register", exact: true }).click();
  return new URL(page.url()).pathname.split("/").pop() as string;
}

async function raise(page: Page, title: string) {
  await page.getByLabel("Raise an RFI").fill(title);
  await page.getByLabel("Raise an RFI").press("Enter");
  await expect(page.locator("tr", { hasText: title })).toBeVisible();
}

test("an RFI typed in the grey row is numbered and due in five working days; a possible answer is accepted with Undo", async ({ page }) => {
  const project = await openRegister(page, "Synthetic register");
  await expect(page.getByText("Nothing is recorded as owed yet.")).toBeVisible();
  await expect(page.getByLabel("Kind")).toHaveCount(0);

  await raise(page, "Basement wall waterproofing");
  const row = page.locator("tr", { hasText: "Basement wall waterproofing" });
  await expect(row.locator("th")).toHaveText("RFI-001");
  await expect(row.locator('td[data-field="due"]')).toContainText(dueIn5());
  await expect(row.locator('td[data-field="status"]')).toContainText("draft");
  // Only the kind with rows is offered.
  await expect(page.getByLabel("Kind").locator("option")).toHaveText(["All kinds", "RFIs"]);

  // Undo puts the register back as it was: empty.
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(page.getByText("Nothing is recorded as owed yet.")).toBeVisible();
  await raise(page, "Basement wall waterproofing");
  await expect(row.locator("th")).toHaveText("RFI-001");

  // Sent by the PM (outside SiteWise until plan 1): Open. A reply citing it files.
  const view = await (await page.request.get(`/api/projects/${project}/delivery`)).json();
  const rfi = view.items[0];
  const opened = await page.request.post(`/api/projects/${project}/ops`, { headers, data: { area: "delivery", ops: [{ op: "delivery.edit", fields: { item: rfi.id, version: rfi.version, status: "open", owner_text: "Northline Structural" } }] } });
  expect(opened.ok()).toBeTruthy();
  expect((await page.request.post(`/__e2e/register-reply?project=${project}&text=${encodeURIComponent("Issued in response to RFI-001: sheet membrane, laps 150 mm.")}`)).ok()).toBeTruthy();
  await page.reload();
  await page.getByRole("button", { name: "Register", exact: true }).click();
  await expect(row.locator('td[data-field="status"]')).toContainText("1 to check");
  await expect(row.locator('td[data-field="with"]')).toHaveText("Northline Structural");
  await row.locator('td[data-field="title"]').click();
  await page.getByRole("button", { name: "Accept", exact: true }).click();
  await expect(row.locator('td[data-field="status"]')).toHaveText("answered");
  await expect(row.locator('td[data-field="with"]')).toHaveText("You");
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(row.locator('td[data-field="status"]')).toContainText("1 to check");
  await expect(row.locator('td[data-field="with"]')).toHaveText("Northline Structural");
});

test("keyboard selection, F2 to edit, and a 409 retry", async ({ page }) => {
  const project = await openRegister(page, "Synthetic register keys");
  await raise(page, "Window head flashing");
  await raise(page, "Stair balustrade height");
  const table = page.getByTestId("register-table");
  const first = page.locator("tr", { hasText: "Window head flashing" });
  await first.locator('td[data-field="title"]').click();
  await table.press("ArrowDown");
  await expect(page.locator("tr", { hasText: "Stair balustrade height" })).toHaveAttribute("aria-selected", "true");
  await table.press("Control+a");
  await expect(page.locator('tr[aria-selected="true"]')).toHaveCount(2);
  await table.press("Escape");
  await expect(page.locator('tr[aria-selected="true"]')).toHaveCount(0);

  // Another tab edits the row first. The screen keeps the old version (its
  // reads are held stale until the conflict), so its edit gets 409; the
  // register is read again and the second try saves.
  const before = await (await page.request.get(`/api/projects/${project}/delivery`)).json();
  let stale = true;
  await page.route("**/api/projects/*/delivery", route => stale ? route.fulfill({ json: before }) : route.continue());
  page.on("response", r => { if (r.url().endsWith("/ops") && r.status() === 409) stale = false; });
  const item = before.items.find((i: { title: string }) => i.title === "Window head flashing");
  await page.request.post(`/api/projects/${project}/ops`, { headers, data: { area: "delivery", ops: [{ op: "delivery.edit", fields: { item: item.id, version: item.version, owner_text: "Arc Architects" } }] } });
  await first.locator('td[data-field="title"]').click();
  await table.press("F2");
  await page.keyboard.press("Control+a");
  await page.keyboard.type("Window head flashing detail");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("alert")).toContainText("changed while you were editing");
  await expect(first.locator('td[data-field="with"]')).toHaveText("Arc Architects");
  await first.locator('td[data-field="title"]').dblclick();
  await page.keyboard.press("Control+a");
  await page.keyboard.type("Window head flashing detail");
  await page.keyboard.press("Enter");
  await expect(page.locator("tr", { hasText: "Window head flashing detail" }).locator('td[data-field="with"]')).toHaveText("Arc Architects");
});
