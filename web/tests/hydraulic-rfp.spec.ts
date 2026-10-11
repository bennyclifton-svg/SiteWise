import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

test("Hydraulic pilot: dates first, scope selections, combined stage pricing and persistence", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Hydraulic pilot review");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.work_type", "new"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), await r.text()).toBeTruthy();
  }
  const pkg = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Hydraulic", discipline_id: "consultant.hydraulic" } });
  expect(pkg.ok(), await pkg.text()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Hydraulic", exact: true }).click();
  await expect(page.getByRole("heading", { name: "2. Scope of services", exact: true })).toBeVisible();
  const dates = page.getByRole("group", { name: "Proposal dates", exact: true });
  await expect(dates.getByLabel("Release date", { exact: true })).toBeVisible();
  await expect(dates.getByLabel("Proposals close", { exact: true })).toBeVisible();
  for (const label of ["RFP reference", "Contact for questions", "Questions in writing by", "Start by", "Reporting date"]) {
    await expect(page.getByLabel(label, { exact: true })).toHaveCount(0);
  }
  await expect(page.getByRole("heading", { name: /Evaluation|Your role|Scope position/ })).toHaveCount(0);
  await expect(page.getByText("This appointment buys", { exact: true })).toHaveCount(0);
  const design = page.getByRole("checkbox", { name: "Include S5 Design in base fee", exact: true });
  const tender = page.getByRole("checkbox", { name: "Include S6 Tender in base fee", exact: true });
  await expect(design).toBeChecked();
  await expect(tender).not.toBeChecked();
  await tender.check();
  await expect(tender).toBeEnabled();
  await expect(tender).toBeChecked();
  await tender.uncheck();
  await expect(tender).toBeEnabled();
  await expect(page.getByLabel("S6 Tender: unchecked stage", { exact: true })).toHaveValue("option");
  const scope = page.locator('.hydraulic-scope');
  await expect(scope.locator('table')).toHaveCount(1);
  await expect(page.getByRole('group', { name: 'Edit positions', exact: true })).toHaveCount(0);
  await expect(scope.getByText('Rule', { exact: true })).toHaveCount(0);
  await expect(scope).toContainText('Water Services Coordinator');
  await expect(scope.getByRole('radio', { name: 'Prepare hydraulic authority submissions: Unknown', exact: true })).toBeChecked();
  await scope.getByRole('radio', { name: 'Prepare hydraulic authority submissions: Included', exact: true }).check();
  await expect(page.getByRole('checkbox', { name: 'Include S4 Authority in base fee', exact: true })).toBeChecked();
  const unknown = scope.getByRole('radio', { name: 'Design rainwater collection and reuse: Unknown', exact: true });
  await unknown.check();
  await expect(unknown).toBeEnabled();
  await page.reload();
  await page.getByRole('button', { name: 'Procurement', exact: true }).click();
  await page.getByRole('button', { name: 'View RFP for Hydraulic', exact: true }).click();
  await expect(unknown).toBeChecked();
  await expect(scope.locator('[data-control="position:rainwater"]')).toContainText('Scope not decided');
  await scope.getByRole('radio', { name: 'Design potable and hot water services: Optional price', exact: true }).check();
  await expect(scope.getByRole('radio', { name: 'Design sanitary plumbing and sewer connections: Included', exact: true })).toBeChecked();
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json();
  const reportId = reports[0].id;
  const saved = await (await page.request.get(`/api/reports/${reportId}`)).json();
  const wording = await page.request.put(`/api/reports/${reportId}/edits/rfp%3Apositions`, { headers: origin, data: { text: 'Owner scope note retained.', version: saved.draft.version } });
  expect(wording.ok(), await wording.text()).toBeTruthy();
  await page.reload();
  await page.getByRole('button', { name: 'Procurement', exact: true }).click();
  await page.getByRole('button', { name: 'View RFP for Hydraulic', exact: true }).click();
  await expect(scope).toContainText('Owner scope note retained.');
  await expect(scope.locator('table')).toHaveCount(1);
  const included = page.getByRole("radio", { name: "Design rainwater collection and reuse: Included", exact: true });
  const optional = page.getByRole("radio", { name: "Design rainwater collection and reuse: Optional price", exact: true });
  await included.check();
  await expect(included).toBeEnabled();
  await expect(included).toBeChecked();
  await optional.check();
  await expect(optional).toBeEnabled();
  await expect(included).not.toBeChecked();
  await expect(page.locator('.hydraulic-stage-table')).toContainText("Design rainwater collection and reuse");
  await dates.getByLabel("Release date", { exact: true }).fill("2026-10-12");
  await Promise.all([page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes("/settings/field:issued")), dates.getByLabel("Release date", { exact: true }).blur()]);
  await expect(dates.getByLabel("Release date", { exact: true })).toBeEnabled();
  await dates.getByLabel("Proposals close", { exact: true }).fill("2026-10-30");
  await Promise.all([page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes("/settings/field:closes")), dates.getByLabel("Proposals close", { exact: true }).blur()]);
  await expect(dates.getByLabel("Proposals close", { exact: true })).toBeEnabled();
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "View RFP for Hydraulic", exact: true }).click();
  await expect(optional).toBeChecked();
  await expect(tender).not.toBeChecked();
  await expect(dates.getByLabel("Release date", { exact: true })).toHaveValue("2026-10-12");
  await expect(dates.getByLabel("Proposals close", { exact: true })).toHaveValue("2026-10-30");
  await expect(page.getByRole("columnheader", { name: "Fee $", exact: true })).toHaveCount(1);
  const datesBox = await dates.boundingBox();
  const projectBox = await page.getByRole("heading", { name: "1. Project", exact: true }).boundingBox();
  expect(datesBox!.y).toBeLessThan(projectBox!.y);
  await page.setViewportSize({ width: 1280, height: 2600 });
  await page.locator('.hydraulic-scope').screenshot({ path: "../.tools/hydraulic-pilot-desktop.png" });
  await page.locator('.hydraulic-stage-table').screenshot({ path: "../.tools/hydraulic-pilot-stages.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  const widths = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, viewport: window.innerWidth }));
  expect(widths.scroll).toBeLessThanOrEqual(widths.viewport);
  await page.locator('.hydraulic-scope').screenshot({ path: "../.tools/hydraulic-pilot-mobile.png" });
});




