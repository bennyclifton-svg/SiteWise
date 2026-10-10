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

test("one RFP report: decide first, edit each section in place, project facts save to Profile", async ({ page }) => {
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
  await page.getByLabel("Services package").selectOption({ label: "Architect · discipline RFP" });
  await expect(page.getByLabel("Services package").locator("option", { hasText: "Civil · confirm discipline" })).toHaveCount(1);
  await page.getByRole("button", { name: "Create RFP draft", exact: true }).click();
  await page.getByRole("button", { name: "Assemble draft", exact: true }).click();
  const useCompleted = page.getByRole("button", { name: "Use last completed state", exact: true });
  if (await useCompleted.isVisible().catch(() => false)) await useCompleted.click();

  // One report: no side panel, decisions in a banner above the draft.
  await expect(page.getByRole("complementary", { name: "Finish this RFP" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "3. Scope by stage", exact: true })).toBeVisible();
  const decisions = page.getByRole("region", { name: "Decide before issue" });
  await expect(decisions.getByText("Appointment not confirmed", { exact: true })).toBeVisible();
  await expect(decisions.getByText("Project details not set", { exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/rfp-report-desktop.png", fullPage: true });
  await decisions.getByRole("button", { name: /^Confirm: / }).click();
  await expect(decisions.getByText("Appointment not confirmed", { exact: true })).toHaveCount(0);

  // A blank client quantity is a decision with its control inline.
  await decisions.getByLabel("Client presentations", { exact: true }).fill("3");
  await decisions.getByLabel("Client presentations", { exact: true }).press("Enter");
  await expect(page.locator(".rfp-draft").getByRole("cell", { name: "Client presentations", exact: true })).toBeVisible();

  // Project facts are edited under "1. Project" and saved to Profile.
  await page.getByRole("button", { name: "Edit 1. Project", exact: true }).click();
  const projectEdit = page.getByRole("group", { name: "Edit project" });
  await expect(projectEdit.getByText("Saved to Profile", { exact: true })).toBeVisible();
  await projectEdit.getByLabel("Client", { exact: true }).fill("Inner West Council");
  await projectEdit.getByLabel("Client", { exact: true }).press("Enter");
  await expect(projectEdit.locator('[data-control="fact:client"] .rfp-source')).toHaveText("Profile");
  await expect(decisions.getByText("Project details not set", { exact: true })).toHaveCount(0);
  await projectEdit.getByLabel("Zone", { exact: true }).fill("R2");
  await projectEdit.getByLabel("Zone", { exact: true }).press("Enter");
  await expect(projectEdit.locator('[data-control="fact:zone"] .rfp-source')).toHaveText("Profile");
  await page.screenshot({ path: "../.tools/rfp-report-editing.png", fullPage: true });
  const saved = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  expect(saved.details.find((d: { key: string }) => d.key === "proj.zone").value).toBe("R2");
  expect(saved.details.find((d: { key: string }) => d.key === "proj.client").value).toBe("Inner West Council");

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
  await page.getByRole("button", { name: "Create RFP draft", exact: true }).click();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toBeVisible();
  await expect(page.getByRole("region", { name: "Decide before issue" })).toHaveCount(0);
  await page.getByRole("button", { name: "Use the civil engineering services RFP", exact: true }).click();
  await expect(page.getByRole("region", { name: "Decide before issue" })).toBeVisible();
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
  await page.getByLabel("Services package").selectOption({ label: "Electrical · discipline RFP" });
  await page.getByRole("button", { name: "Create RFP draft", exact: true }).click();
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
