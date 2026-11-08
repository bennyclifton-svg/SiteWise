import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

test("Hydraulic pilot: dates last, scope by stage, one fee per stage and persistence", async ({ page }) => {
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
  await expect(page.getByRole("heading", { name: "Scope of services", exact: true })).toBeVisible();
  const dates = page.getByRole("group", { name: "Proposal dates", exact: true });
  await expect(dates.getByLabel("Release date", { exact: true })).toBeVisible();
  await expect(dates.getByLabel("Proposals close", { exact: true })).toBeVisible();
  for (const label of ["RFP reference", "Contact for questions", "Questions in writing by", "Start by", "Reporting date"]) {
    await expect(page.getByLabel(label, { exact: true })).toHaveCount(0);
  }
  await expect(page.getByRole("heading", { name: /Evaluation|Your role|Scope position/ })).toHaveCount(0);
  await expect(page.getByText("This appointment buys", { exact: true })).toHaveCount(0);
  // Scope by stage (3 November 2026): one table, each stage collapsed, every
  // stage, line and service either included or excluded, one fee per stage.
  const scope = page.locator(".rfp-stage-scope");
  await expect(scope).toHaveCount(1);
  await expect(page.locator(".hydraulic-scope, .hydraulic-stage-table")).toHaveCount(0);
  await expect(scope.getByRole("radio")).toHaveCount(0);
  const tender = scope.getByRole("checkbox", { name: "S6 Tender: excluded", exact: true });
  // The whole-stage tick sits on the closed stage only (owner, 8 Nov 2026).
  await expect(scope.getByRole("checkbox", { name: "S5 Design: included", exact: true })).toBeChecked();
  await scope.getByRole("button", { name: "S5 Design", exact: true }).click();
  await expect(scope.getByRole("checkbox", { name: "S5 Design: included", exact: true })).toHaveCount(0);
  const rainwaterIn = scope.getByRole("checkbox", { name: "Design rainwater collection and reuse: included", exact: true });
  const rainwaterOut = scope.getByRole("checkbox", { name: "Design rainwater collection and reuse: excluded", exact: true });
  await expect(rainwaterIn).toBeChecked();
  await rainwaterOut.check();
  await expect(rainwaterOut).toBeChecked();
  await expect(rainwaterIn).not.toBeChecked();
  await tender.check();
  await expect(tender).toBeChecked();
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "View RFP for Hydraulic", exact: true }).click();
  await expect(tender).toBeChecked();
  await scope.getByRole("button", { name: "S5 Design", exact: true }).click();
  await expect(rainwaterOut).toBeChecked();
  await rainwaterIn.check();
  await expect(rainwaterIn).toBeChecked();
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json();
  const reportId = reports[0].id;
  const saved = await (await page.request.get(`/api/reports/${reportId}`)).json();
  const wording = await page.request.put(`/api/reports/${reportId}/edits/rfp%3Astages`, { headers: origin, data: { text: 'Owner scope note retained.', version: saved.draft.version } });
  expect(wording.ok(), await wording.text()).toBeTruthy();
  await page.reload();
  await page.getByRole('button', { name: 'Procurement', exact: true }).click();
  await page.getByRole('button', { name: 'View RFP for Hydraulic', exact: true }).click();
  await expect(page.getByText('Owner scope note retained.', { exact: true })).toBeVisible();
  await expect(scope).toHaveCount(1);
  await dates.getByLabel("Release date", { exact: true }).fill("2026-10-12");
  await Promise.all([page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes("/settings/field:issued")), dates.getByLabel("Release date", { exact: true }).blur()]);
  await expect(dates.getByLabel("Release date", { exact: true })).toBeEnabled();
  await dates.getByLabel("Proposals close", { exact: true }).fill("2026-10-30");
  await Promise.all([page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes("/settings/field:closes")), dates.getByLabel("Proposals close", { exact: true }).blur()]);
  await expect(dates.getByLabel("Proposals close", { exact: true })).toBeEnabled();
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "View RFP for Hydraulic", exact: true }).click();
  await expect(tender).toBeChecked();
  await expect(dates.getByLabel("Release date", { exact: true })).toHaveValue("2026-10-12");
  await expect(dates.getByLabel("Proposals close", { exact: true })).toHaveValue("2026-10-30");
  await expect(page.getByRole("columnheader", { name: "Fee $", exact: true })).toHaveCount(1);
  const datesBox = await dates.boundingBox();
  const projectBox = await page.getByRole("region", { name: "Project", exact: true }).boundingBox();
  // Dates close the RFP, in the Issue section, below the project and scope.
  expect(datesBox!.y).toBeGreaterThan(projectBox!.y);
  const order = await page.locator(".report-section h3").allTextContents();
  // On screen the sections drop their print numbers and page one its title (owner, 8 Nov 2026).
  expect(order.slice(0, 3)).toEqual(["Scope of services", "Allowances", "Transmittal"]);
  await page.setViewportSize({ width: 1280, height: 2600 });
  await scope.screenshot({ path: "../.tools/hydraulic-pilot-desktop.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  const widths = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, viewport: window.innerWidth }));
  expect(widths.scroll).toBeLessThanOrEqual(widths.viewport);
  await scope.screenshot({ path: "../.tools/hydraulic-pilot-mobile.png" });
});




