import { expect, test, type Locator } from "@playwright/test";

// Combined engagements, slice 2: combine two consultant rows, Create RFP on
// the group, change a choice in one discipline's part, and reach the one
// Issue for the whole document.
const origin = { Origin: "http://127.0.0.1:4173" };

async function expandStages(part: Locator) {
  const toggles = part.locator(".rfp-stage-toggle");
  for (let i = 0; i < await toggles.count(); i++) {
    const t = toggles.nth(i);
    if (await t.getAttribute("aria-expanded") === "false") await t.click();
  }
}

test("a combined engagement has one RFP: each discipline's part edits its own RFP", async ({ page }, testInfo) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Combined RFP");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.work_type", "extend"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), `${key}: ${await r.text()}`).toBeTruthy();
  }
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Procurement", exact: true })).toBeVisible();
  for (const [discipline, label] of [["consultant.structural", "Structural"], ["consultant.civil", "Civil"]]) {
    // The project's facts may already have made the row.
    await expect(page.getByRole("button", { name: "Add discipline", exact: true })).toBeEnabled();
    await page.waitForLoadState("networkidle");
    if (await page.getByRole("rowheader", { name: label, exact: true }).count() > 0) continue;
    await page.getByRole("button", { name: "Add discipline", exact: true }).click();
    await page.getByLabel("Discipline", { exact: true }).selectOption(discipline);
    await page.getByRole("button", { name: "Add row", exact: true }).click();
    await expect(page.getByRole("rowheader", { name: label, exact: true })).toBeVisible();
  }
  // Rows are selected as in Program: click, then Ctrl-click.
  await page.getByRole("rowheader", { name: "Structural", exact: true }).click();
  await page.getByRole("rowheader", { name: "Civil", exact: true }).click({ modifiers: ["Control"] });
  await page.getByRole("button", { name: "Combine 2", exact: true }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Combine", exact: true }).click();
  await expect(page.getByRole("button", { name: "Uncombine Civil and Structural", exact: true })).toBeVisible();

  // The group row offers Create RFP.
  const start = Date.now();
  await page.getByRole("button", { name: "Create RFP for Civil and Structural", exact: true }).click();
  await expect(page.getByRole("button", { name: "Refresh draft", exact: true })).toBeVisible();
  const civil = page.getByRole("region", { name: "2.1 Civil engineering services", exact: true });
  const structural = page.getByRole("region", { name: "2.2 Structural engineering services", exact: true });
  await expect(structural.locator(".rfp-stage-scope")).toBeVisible();
  testInfo.annotations.push({ type: "timing", description: `group RFP create and first build: ${Date.now() - start}ms` });
  await expect(civil.locator(".rfp-stage-scope")).toBeVisible();
  await expect(page.getByText(/This RFP is for one engagement, Civil and Structural, covering Civil engineering services and Structural engineering services/)).toBeVisible();
  await expect(page.getByRole("heading", { name: "3. Fee schedule", exact: true })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Sum for Structural engineering services", exact: true })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Engagement total (the sums above)", exact: true })).toBeVisible();
  // One page one and one set of proposal dates for the whole document; a
  // part has no fee column of its own.
  await expect(page.locator(".rfp-page-one")).toHaveCount(1);
  await expect(page.getByRole("group", { name: "Proposal dates" })).toHaveCount(1);
  await expect(structural.getByRole("columnheader", { name: "Fee $" })).toHaveCount(0);

  // Exclude a line in the structural part: it is written to the structural
  // engineer's own report and the document shows it at once.
  const toolbar = page.locator(".report-toolbar");
  const before = Number((await toolbar.innerText()).match(/Version (\d+)/)![1]);
  await expandStages(structural);
  // An open stage's lines fill the table's four columns, no stray fee cell.
  const widths = await structural.locator(".rfp-stage-scope tr").evaluateAll(rows =>
    rows.map(r => [...r.children].reduce((n, c) => n + ((c as HTMLTableCellElement).colSpan || 1), 0)));
  expect(new Set(widths), "every stage table row is as wide as its header").toEqual(new Set([4]));
  const boxes = structural.locator('tr.rfp-stage-line[data-line] input[type="checkbox"][aria-label$=": excluded"]:not(:checked):not(:disabled)');
  const name = (await boxes.first().getAttribute("aria-label"))!;
  const edit = Date.now();
  await boxes.first().check();
  await expect(toolbar).toContainText(`Version ${before + 1}`);
  testInfo.annotations.push({ type: "timing", description: `member setting through the group: ${Date.now() - edit}ms` });
  await expandStages(structural);
  await expect(structural.getByRole("checkbox", { name, exact: true })).toBeChecked();
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json() as { id: string; title: string }[];
  const groupReport = reports.find(r => r.title.includes("Civil and Structural"))!;
  const view = await (await page.request.get(`/api/reports/${groupReport.id}`)).json();
  const member = view.rfp_group.members.find((m: { title: string }) => m.title.startsWith("Structural"));
  const own = await (await page.request.get(`/api/reports/${member.report_id}`)).json();
  type Line = { source: string; value: string };
  const stages = own.draft.sections.flatMap((s: { blocks: { id: string; basis?: { stages?: { lines?: Line[] }[] } }[] }) => s.blocks).find((b: { id: string }) => b.id === "rfp:stages").basis.stages as { lines?: Line[] }[];
  const chosen = stages.flatMap(s => s.lines ?? []).filter(l => l.source === "user" && l.value === "X");
  expect(chosen.length, "the choice is stored on the structural engineer's own report").toBe(1);
  await page.screenshot({ path: testInfo.outputPath("group-rfp.png"), fullPage: true });

  // One Issue for the whole document: every member's checks are listed with
  // their discipline, and the draft rules keep it closed.
  const issue = page.getByRole("region", { name: "Issue", exact: true });
  await expect(issue).toContainText("The civil and structural rules are a draft");
  await expect(issue).toContainText("Structural engineering services:");
  await expect(issue).toContainText("Civil engineering services:");
  await expect(issue.getByRole("button", { name: "Issue report", exact: true })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Issue report", exact: true })).toHaveCount(1);

  // Back on Procurement the group row opens the same report.
  await page.getByRole("button", { name: "← Procurement", exact: true }).click();
  await expect(page.getByRole("button", { name: "View RFP for Civil and Structural", exact: true })).toBeVisible();
});
