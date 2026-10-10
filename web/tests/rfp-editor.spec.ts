import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

async function signIn(page: Page, name: string) {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  return new URL(page.url()).pathname.split("/").pop()!;
}

async function refreshDraft(page: Page) {
  const toolbar = page.locator(".report-toolbar");
  const version = (await toolbar.innerText()).match(/Version (\d+)/);
  expect(version).not.toBeNull();
  await toolbar.getByRole("button", { name: "Refresh draft", exact: true }).click();
  // Existing controls can still show the previous value while the new draft
  // loads. Wait for its version before testing persistence or another edit.
  await expect(toolbar).toContainText(`Version ${Number(version![1]) + 1} ·`);
}

test("one RFP report: page one edited where it prints, checks beside their values, issue at the end", async ({ page }) => {
  const project = await signIn(page, "RFP editor regression");
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const part = profile.parts[0].id;
  for (const [key, value] of [["hdr.work_type", "extend"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: part, value } });
    expect(r.ok(), `${key}: ${await r.text()}`).toBeTruthy();
  }
  for (const data of [{ kind: "services", title: "Architect", discipline_id: "consultant.architect" }, { kind: "services", title: "Civil" }]) {
    const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data });
    expect(r.ok()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Procurement", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Create RFP for Architect", exact: true }).click();
  await page.getByRole("button", { name: "Assemble draft", exact: true }).click();
  const useCompleted = page.getByRole("button", { name: "Use last completed state", exact: true });
  if (await useCompleted.isVisible().catch(() => false)) await useCompleted.click();

  // One report: page one is the project, always open in the Profile card's
  // columns; no decisions banner, no stale block.
  await expect(page.getByRole("complementary", { name: "Finish this RFP" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "3. Scope by stage", exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Decide before issue" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Edit 1. Project", exact: true })).toHaveCount(0);
  const pageOne = page.locator(".rfp-page-one");
  for (const column of ["The building", "Site and planning", "Delivery"]) await expect(pageOne.getByRole("group", { name: column, exact: true })).toBeVisible();
  await expect(pageOne.getByRole("group", { name: "Project conditions", exact: true })).toBeVisible();

  // A blocking check sits beside what it concerns; the Issue section counts them.
  const issue = page.getByRole("region", { name: "Issue", exact: true });
  await expect(issue.getByRole("button", { name: "Issue report", exact: true })).toBeDisabled();
  await expect(issue.getByRole("group", { name: "Issue details" }).getByLabel("RFP reference", { exact: true })).toBeVisible();
  const role = page.locator(".report-section").filter({ has: page.getByRole("heading", { name: "2. Your role", exact: true }) });
  await expect(role.getByText("Appointment not confirmed.", { exact: true })).toBeVisible();
  await expect(pageOne.locator('[data-control="fact:client"][data-blocked]')).toBeVisible();
  await page.screenshot({ path: "../.tools/rfp-report-desktop.png", fullPage: true });
  await role.locator(".rfp-markers li").filter({ hasText: "Appointment not confirmed" }).getByRole("button", { name: "Show", exact: true }).click();
  await role.getByRole("button", { name: "Confirm", exact: true }).click();
  await expect(role.getByText("Appointment not confirmed.", { exact: true })).toHaveCount(0);

  // A blank client quantity is marked under section 4 and opens its control.
  const allowances = page.locator(".report-section").filter({ has: page.getByRole("heading", { name: "4. Allowances", exact: true }) });
  await allowances.locator(".rfp-markers li").filter({ hasText: "Client presentations" }).getByRole("button", { name: "Show", exact: true }).click();
  await allowances.getByLabel("Client presentations", { exact: true }).fill("3");
  await allowances.getByLabel("Client presentations", { exact: true }).press("Enter");
  await expect(page.locator(".rfp-draft").getByRole("cell", { name: "Client presentations", exact: true })).toBeVisible();

  // Project facts are edited where they print on page one and saved to Profile.
  await pageOne.getByRole("button", { name: /^Client: .*Change$/ }).click();
  await pageOne.getByLabel("Client", { exact: true }).fill("Inner West Council");
  await pageOne.getByLabel("Client", { exact: true }).press("Enter");
  await expect(pageOne.locator('[data-control="fact:client"] .rfp-source')).toHaveText("Profile");
  await expect(pageOne.locator('[data-control="fact:client"]')).toContainText("Inner West Council");
  await expect(pageOne.locator('[data-control="fact:client"][data-blocked]')).toHaveCount(0);
  // An unset row waits behind "more", as on the Profile card.
  const site = pageOne.getByRole("group", { name: "Site and planning", exact: true });
  await site.getByRole("button", { name: /^\+ \d+ more$/ }).click();
  await site.getByRole("button", { name: "Zone: —. Change", exact: true }).click();
  await site.getByLabel("Zone", { exact: true }).fill("R2");
  await site.getByLabel("Zone", { exact: true }).press("Enter");
  await expect(pageOne.locator('[data-control="fact:zone"]')).toContainText("R2");
  // A condition answers in place.
  const country = pageOne.locator('[data-control="fact:flag.country"]');
  await country.getByRole("radio", { name: "Yes", exact: true }).check();
  await expect(country.locator(".rfp-source")).toHaveText("Profile");
  await page.screenshot({ path: "../.tools/rfp-report-editing.png", fullPage: true });
  const saved = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  expect(saved.details.find((d: { key: string }) => d.key === "proj.zone").value).toBe("R2");
  expect(saved.details.find((d: { key: string }) => d.key === "proj.client").value).toBe("Inner West Council");
  expect(saved.details.find((d: { key: string }) => d.key === "proj.flag.country").value).toBe("yes");

  // Scope position is this RFP's own choice, edited under section 5.
  await page.getByRole("button", { name: "Edit 5. Scope position", exact: true }).click();
  const positions = page.getByRole("group", { name: "Edit positions" });
  await positions.getByRole("radio", { name: "Help selecting other consultants: excluded" }).check();
  const scope = page.locator('.report-section').filter({ has: page.getByRole('heading', { name: '5. Scope position', exact: true }) });
  const excluded = scope.getByRole('table', { name: 'Excluded from this appointment', exact: true });
  await expect(excluded.getByRole('row').filter({ hasText: 'Help selecting other consultants' })).toContainText('party not named');
  await expect(scope.locator('.rfp-scope-position .report-wording').filter({ hasText: 'Included in the fee:' })).toHaveCount(0);
  await expect(positions.locator('[data-control="position:consultant_selection"] .rfp-source')).toHaveText("You");

  await refreshDraft(page);
  await expect(page.getByRole("group", { name: "Edit positions" }).getByRole("radio", { name: "Help selecting other consultants: excluded" })).toBeChecked();
  await page.getByRole("group", { name: "Edit positions" }).locator('[data-control="position:consultant_selection"]').getByRole("button", { name: "Reset", exact: true }).click();
  await expect(page.getByRole("group", { name: "Edit positions" }).getByRole("radio", { name: "Help selecting other consultants: optional" })).toBeChecked();

  await page.getByRole('button', { name: 'Done editing 5. Scope position', exact: true }).click();
  await page.setViewportSize({ width: 1280, height: 2600 });
  await scope.screenshot({ path: '../.tools/rfp-scope-desktop.png' });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: "../.tools/rfp-report-mobile.png", fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await scope.screenshot({ path: '../.tools/rfp-scope-mobile.png' });
  await page.setViewportSize({ width: 1280, height: 720 });

  // Protected wording must never be replaced by tables of old generated scope.
  await scope.getByRole('button', { name: 'Edit wording:', exact: true }).click();
  await page.getByLabel('Report wording', { exact: true }).fill('Client-approved scope wording; retain this exactly.');
  await page.getByRole('button', { name: 'Save wording', exact: true }).click();
  await expect(scope.getByText('Client-approved scope wording; retain this exactly.', { exact: true })).toBeVisible();
  await expect(scope.getByRole('table')).toHaveCount(0);
  await refreshDraft(page);
  await expect(scope.getByText('Client-approved scope wording; retain this exactly.', { exact: true })).toBeVisible();
  await expect(scope.getByRole('table')).toHaveCount(0);
});

test("a package titled Civil is offered the civil RFP and switches only when confirmed", async ({ page }) => {
  const project = await signIn(page, "RFP discipline suggestion");
  const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Civil" } });
  expect(r.ok()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Civil", exact: true }).click();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Issue", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Use the civil engineering services RFP", exact: true }).click();
  await expect(page.getByRole("region", { name: "Issue", exact: true })).toBeVisible();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toHaveCount(0);
});

test("electrical keeps its allocation schedule beside grouped positions, with separate fire disciplines", async ({ page }) => {
  const project = await signIn(page, "RFP services allocation");
  const catalog = await (await page.request.get("/api/rfp/disciplines")).json();
  const ids = catalog.items.map((item: { id: string }) => item.id);
  expect(ids).toContain("fire-services");
  expect(ids).toContain("fire-engineering");
  expect(ids).toContain("electrical-services");
  for (const data of [
    { kind: "services", title: "Electrical", discipline_id: "consultant.electrical" },
    { kind: "services", title: "Fire services", discipline_id: "consultant.fire_services" },
    { kind: "services", title: "Fire engineering", discipline_id: "consultant.fire_engineer" },
  ]) {
    const response = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data });
    expect(response.ok(), await response.text()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Electrical", exact: true }).click();
  await page.getByRole("button", { name: "Assemble draft", exact: true }).click();
  const useCompleted = page.getByRole("button", { name: "Use last completed state", exact: true });
  if (await useCompleted.isVisible().catch(() => false)) await useCompleted.click();
  const scope = page.locator(".report-section").filter({ has: page.getByRole("heading", { name: "5. Scope position", exact: true }) });
  const allocation = scope.getByRole("table").filter({ has: page.getByRole("columnheader", { name: "System / interface", exact: true }) });
  await expect(allocation.getByRole("row")).toHaveCount(8);
  await expect(allocation).toContainText(/smoke alarms/i);
  await expect(allocation).toContainText(/power.*recovery/i);
  await expect(scope.locator(".rfp-scope-position")).toBeVisible();
  await refreshDraft(page);
  await expect(allocation.getByRole("row")).toHaveCount(8);
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await scope.screenshot({ path: "../.tools/rfp-electrical-allocation-mobile.png" });
});
