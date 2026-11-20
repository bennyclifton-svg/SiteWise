import { expect, test, type Page } from "@playwright/test";

// The register (docs/plans/2026-11-17-rfis-review-comments.md, Task 5),
// in the right-hand panel since 20 November 2026: a narrow list beside
// every view, the grey new row, Go's numbers and due dates, a row opened as
// a sheet in the middle panel, a possible answer accepted with Undo,
// keyboard selection, a 409 retry, and a Kind filter that lists only kinds
// with rows. Synthetic project; no Jev, no model.

const headers = { Origin: "http://127.0.0.1:4173" };

/** Five working days after today in Sydney, Monday to Friday (no Program). */
function dueIn5(): string {
  const today = new Intl.DateTimeFormat("en-CA", { timeZone: "Australia/Sydney", year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
  const d = new Date(`${today}T00:00:00Z`);
  const working = (x: Date) => x.getUTCDay() !== 0 && x.getUTCDay() !== 6;
  for (let n = 0; n < 5;) { d.setUTCDate(d.getUTCDate() + 1); if (working(d)) n++; }
  return d.toISOString().slice(0, 10);
}

const panel = (page: Page) => page.getByRole("tablist", { name: "Right panel" });
const showRegister = (page: Page) => panel(page).getByRole("tab", { name: "Register", exact: true }).click();
const sheet = (page: Page) => page.locator(".register-sheet");

async function openRegister(page: Page, name: string) {
  await page.setViewportSize({ width: 1650, height: 920 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  await showRegister(page);
  return new URL(page.url()).pathname.split("/").pop() as string;
}

async function raise(page: Page, title: string) {
  await page.getByLabel("Raise an RFI").fill(title);
  await page.getByLabel("Raise an RFI").press("Enter");
  await expect(page.locator(".register-panel tr", { hasText: title })).toBeVisible();
}

async function itemOf(page: Page, project: string, title: string) {
  const view = await (await page.request.get(`/api/projects/${project}/delivery`)).json();
  return view.items.find((i: { title: string }) => i.title === title);
}

test("an RFI typed in the grey row is numbered and due in five working days; a possible answer is accepted with Undo", async ({ page }) => {
  const project = await openRegister(page, "Synthetic register");
  // The Register is beside the view, not a view of its own.
  await expect(page.getByRole("group", { name: "Project views" }).getByRole("button", { name: "Register" })).toHaveCount(0);
  await expect(page.getByText("Nothing is recorded as owed yet.")).toBeVisible();
  await expect(page.getByLabel("Kind")).toHaveCount(0);

  await raise(page, "Basement wall waterproofing");
  const row = page.locator(".register-panel tr", { hasText: "Basement wall waterproofing" });
  await expect(row.locator("th")).toHaveText("RFI-001");
  await expect(row.locator('td[data-field="due"]')).toContainText(dueIn5());
  await expect(row.locator(".register-status")).toHaveText("Draft");
  // Only the kind with rows is offered.
  await expect(page.getByLabel("Kind").locator("option")).toHaveText(["All kinds", "RFIs"]);

  // Undo puts the register back as it was: empty.
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(page.getByText("Nothing is recorded as owed yet.")).toBeVisible();
  await raise(page, "Basement wall waterproofing");
  await expect(row.locator("th")).toHaveText("RFI-001");

  // Sent by the PM outside SiteWise: Open. A reply citing it files.
  const rfi = await itemOf(page, project, "Basement wall waterproofing");
  const opened = await page.request.post(`/api/projects/${project}/ops`, { headers, data: { area: "delivery", ops: [{ op: "delivery.edit", fields: { item: rfi.id, version: rfi.version, status: "open", owner_text: "Northline Structural" } }] } });
  expect(opened.ok()).toBeTruthy();
  expect((await page.request.post(`/__e2e/register-reply?project=${project}&text=${encodeURIComponent("Issued in response to RFI-001: sheet membrane, laps 150 mm.")}`)).ok()).toBeTruthy();
  await page.reload();
  // The panel's side is remembered for this viewer.
  await expect(panel(page).getByRole("tab", { name: "Register" })).toHaveAttribute("aria-selected", "true");
  await expect(row.locator(".register-status")).toContainText("1 to check");
  await expect(row.locator('td[data-field="with"]')).toHaveText("Northline Structural");

  // The row opens as a sheet in the middle; its history carries the
  // possible answer with Accept.
  await row.locator('td[data-field="title"]').click();
  await expect(sheet(page).getByRole("heading", { name: "RFI-001 · Basement wall waterproofing" })).toBeVisible();
  await expect(sheet(page).locator(".register-sheet-head .register-status")).toHaveText("Open · with Northline Structural");
  const history = sheet(page).getByRole("list", { name: "History" });
  await expect(history.locator("li")).toHaveText([/Raised by .+@.+ · draft/, /Draft → Open/, /Possible answer: Engineer's reply/]);
  await history.getByRole("button", { name: "Accept", exact: true }).click();
  await expect(row.locator(".register-status")).toHaveText("Answered");
  await expect(row.locator('td[data-field="with"]')).toHaveText("You");
  await expect(history.locator("li")).toHaveText([/Raised by .+@.+ · draft/, /Draft → Open/, /Open → Answered/, /Answered by Engineer's reply/]);
  await expect(history.locator('li[data-tone="answer"]')).toHaveCount(1);
  await sheet(page).getByRole("button", { name: "Undo", exact: true }).click();
  await expect(row.locator(".register-status")).toContainText("1 to check");
  await expect(row.locator('td[data-field="with"]')).toHaveText("Northline Structural");
  // Back returns to the view that was there.
  await page.getByRole("button", { name: "Back to Project Profile" }).click();
  await expect(sheet(page)).toHaveCount(0);
  await expect(page.getByRole("region", { name: "Project profile" })).toBeVisible();
});

test("keyboard selection, F2 to edit, and a 409 retry", async ({ page }) => {
  const project = await openRegister(page, "Synthetic register keys");
  await raise(page, "Window head flashing");
  await raise(page, "Stair balustrade height");
  const table = page.getByTestId("register-table");
  const first = page.locator(".register-panel tr", { hasText: "Window head flashing" });
  await first.locator('td[data-field="title"]').click();
  await table.press("ArrowDown");
  await expect(page.locator(".register-panel tr", { hasText: "Stair balustrade height" })).toHaveAttribute("aria-selected", "true");
  await table.press("Control+a");
  await expect(page.locator('.register-panel tr[aria-selected="true"]')).toHaveCount(2);
  await table.press("Escape");
  await expect(page.locator('.register-panel tr[aria-selected="true"]')).toHaveCount(0);

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
  await table.focus();
  await table.press("F2");
  await page.keyboard.press("Control+a");
  await page.keyboard.type("Window head flashing detail");
  await page.keyboard.press("Enter");
  await expect(page.locator(".register-panel").getByRole("alert")).toContainText("changed while you were editing");
  await expect(first.locator('td[data-field="with"]')).toHaveText("Arc Architects");
  await first.locator('td[data-field="title"]').dblclick();
  await page.keyboard.press("Control+a");
  await page.keyboard.type("Window head flashing detail");
  await page.keyboard.press("Enter");
  await expect(page.locator(".register-panel tr", { hasText: "Window head flashing detail" }).locator('td[data-field="with"]')).toHaveText("Arc Architects");
});

test("an RFI's sheet: To, Cc and Subject kept over a reload, Ref only as their ref, the transmittal from the Documents side, Send with its reason", async ({ page }) => {
  const project = await openRegister(page, "Synthetic RFI sheet");
  expect((await page.request.post(`/__e2e/file-document?project=${project}&title=${encodeURIComponent("Slab edge detail")}&kind=drawing`, { data: "Slab edge detail, sheet S-101." })).ok()).toBeTruthy();
  await raise(page, "Slab edge set-down");
  // One raised by someone else, with their number.
  expect((await page.request.post(`/api/projects/${project}/ops`, { headers, data: { area: "delivery", ops: [{ op: "delivery.add", fields: { kind: "rfi", title: "Builder query on stair", ref: "BLD-RFI-12", raised_by: "Main builder", raised_on: "2026-11-18" } }] } })).ok()).toBeTruthy();
  await page.reload();

  const row = page.locator(".register-panel tr", { hasText: "Slab edge set-down" });
  await row.locator('td[data-field="title"]').click();
  const s = sheet(page);
  await expect(s.getByRole("heading", { name: "RFI-001 · Slab edge set-down" })).toBeVisible();
  // Ref is not on an RFI the PM raised.
  await expect(s.getByLabel("Their ref", { exact: true })).toHaveCount(0);
  await expect(s.getByLabel("Subject", { exact: true })).toHaveAttribute("placeholder", "RFI-001 Slab edge set-down");
  const send = s.getByRole("button", { name: "Send", exact: true });
  await expect(send).toBeDisabled();
  await expect(s.locator(".register-send-what")).toHaveText("Add who it goes to under To.");

  await s.getByLabel("To", { exact: true }).fill("eng@consult.test");
  await s.getByLabel("Cc", { exact: true }).fill("site@builder.test");
  await s.getByLabel("Subject", { exact: true }).fill("RFI-001 Slab edge set-down at the balcony");
  await s.getByLabel("Question", { exact: true }).click();
  await expect(s.getByRole("button", { name: "Undo", exact: true })).toBeVisible();
  await expect(send).toHaveAttribute("title", "Send to 2 addresses with 0 documents");

  // The transmittal from the Documents side, with the RFI still open.
  await panel(page).getByRole("tab", { name: "Documents" }).click();
  await page.locator(".reg-doc").filter({ hasText: "Slab edge detail" }).locator("tr.reg-line").click();
  await s.getByRole("button", { name: "Update Transmittal" }).click();
  await expect(s.locator(".rfp-transmittal-table tbody tr")).toContainText("Slab edge detail");
  await expect(send).toHaveAttribute("title", "Send to 2 addresses with 1 document");

  // A reload keeps what the PM typed.
  await page.reload();
  await expect(panel(page).getByRole("tab", { name: "Documents" })).toHaveAttribute("aria-selected", "true");
  await showRegister(page);
  await row.locator('td[data-field="title"]').click();
  await expect(s.getByLabel("To", { exact: true })).toHaveValue("eng@consult.test");
  await expect(s.getByLabel("Cc", { exact: true })).toHaveValue("site@builder.test");
  await expect(s.getByLabel("Subject", { exact: true })).toHaveValue("RFI-001 Slab edge set-down at the balcony");

  // A refusal is shown in its own words, in place.
  await page.route("**/api/projects/*/transmittals", route => route.fulfill({ status: 403, contentType: "text/plain", body: "Confirm your email address to send mail." }));
  await send.click();
  await expect(s.locator(".register-send-what")).toHaveText("Confirm your email address to send mail.");
  await page.unroute("**/api/projects/*/transmittals");

  // With sending off on the server, Send is off and says why.
  await page.route("**/api/projects/*/delivery", async route => {
    const res = await route.fetch();
    await route.fulfill({ response: res, json: { ...(await res.json()), send_off: "Sending mail is not set up on this server." } });
  });
  await page.reload();
  await row.locator('td[data-field="title"]').click();
  await expect(send).toBeDisabled();
  await expect(send).toHaveAttribute("title", "Sending mail is not set up on this server.");
  await expect(s.locator(".register-send-what")).toHaveText("Sending mail is not set up on this server.");
  await page.unroute("**/api/projects/*/delivery");

  // An RFI someone else raised shows their number as Their ref.
  await page.locator(".register-panel tr", { hasText: "Builder query on stair" }).locator('td[data-field="title"]').click();
  await expect(s.getByLabel("Their ref", { exact: true })).toHaveValue("BLD-RFI-12");
  await expect(s.getByRole("list", { name: "History" }).locator("li").first()).toContainText("Raised by Main builder");
});
