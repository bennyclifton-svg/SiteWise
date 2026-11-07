import { expect, test } from "@playwright/test";

// Combined engagements, slice 4: a group has one set of stage lines. Combine
// two consultant rows, Create RFP on the group (which creates the program),
// rename a stage in Program: the group's RFP offers the rename once and
// Update clears it, while the members offer nothing. Uncombine, and each
// member's own stage lines are live again. Program itself has no row per
// package (owner direction 25 October 2026), so it is read through its API.
const origin = { Origin: "http://127.0.0.1:4173" };

test("a combined engagement has one set of stage lines; uncombine restores the members'", async ({ page }, testInfo) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Combined stages");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Procurement", exact: true })).toBeVisible();
  for (const [discipline, label] of [["consultant.structural", "Structural"], ["consultant.civil", "Civil"]]) {
    await expect(page.getByRole("button", { name: "New discipline", exact: true })).toBeEnabled();
    await page.waitForLoadState("networkidle");
    if (await page.getByRole("rowheader", { name: label, exact: true }).count() > 0) continue;
    await page.getByRole("button", { name: "New discipline", exact: true }).click();
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

  type Stage = { label: string; program_stage_id?: string };
  type Pkg = { id: string; title: string; role?: string; group_id?: string; stages: Stage[] };
  const packages = async () => ((await (await page.request.get(`/api/projects/${project}/packages`)).json()).items ?? []) as Pkg[];
  let items = await packages();
  const group = items.find(p => p.role === "group")!;
  const members = items.filter(p => p.group_id === group.id);
  expect(members).toHaveLength(2);
  expect(group.stages.length, "the group holds one line per stage its members include").toBeGreaterThan(0);

  // Create RFP on the group: the program is created and the group's lines link to it.
  await page.getByRole("button", { name: "Create RFP for Civil and Structural", exact: true }).click();
  await expect(page.getByRole("button", { name: "Update", exact: true })).toBeVisible();
  const read = await page.request.get(`/api/projects/${project}/program`);
  expect(read.status()).toBe(200);
  let program = await read.json();
  const concept = program.activities.find((a: { kind: string; catalogue_stage_id: string }) => a.kind === "stage" && a.catalogue_stage_id === "brief_concept");
  items = await packages();
  const lines = items.find(p => p.id === group.id)!.stages;
  expect(lines.every(l => !!l.program_stage_id), "every group line points at a program stage").toBeTruthy();
  expect(lines.filter(l => l.program_stage_id === concept.id)).toHaveLength(1);
  const programOrder = program.activities.filter((a: { kind: string }) => a.kind === "stage").map((a: { id: string }) => a.id);
  const positions = lines.map(l => programOrder.indexOf(l.program_stage_id));
  expect(positions, "group lines are in program order").toEqual([...positions].sort((a, b) => a - b));
  for (const m of members) {
    const sync = await (await page.request.get(`/api/projects/${project}/packages/${m.id}/stage-sync`)).json();
    expect(sync.group_id).toBe(group.id);
    expect(sync.stages).toHaveLength(0);
  }

  // Rename the stage in Program: the group's RFP offers it once.
  const ops = await page.request.post(`/api/projects/${project}/program/ops`, { headers: origin,
    data: { program_id: program.program.id, version: program.program.version, ops: [{ op: "update", id: concept.id, label: "Sketch design" }] } });
  expect(ops.ok()).toBeTruthy();
  program = await ops.json();
  await page.getByRole("button", { name: "← Procurement", exact: true }).click();
  await page.getByRole("button", { name: "View RFP for Civil and Structural", exact: true }).click();
  const notice = page.getByRole("status", { name: "Stage changes for RFP draft" });
  await expect(notice).toContainText("Brief and concept renamed Sketch design");
  await expect(notice).toHaveCount(1);
  await page.screenshot({ path: testInfo.outputPath("group-stage-notice.png") });
  const start = Date.now();
  await notice.getByRole("button", { name: "Update", exact: true }).click();
  await expect(notice).toHaveCount(0);
  testInfo.annotations.push({ type: "timing", description: `group RFP stage update: ${Date.now() - start}ms` });
  for (const m of members) {
    const sync = await (await page.request.get(`/api/projects/${project}/packages/${m.id}/stage-sync`)).json();
    expect(sync.offer, "a member offers nothing while grouped").toBe(false);
  }
  const groupSync = await (await page.request.get(`/api/projects/${project}/packages/${group.id}/stage-sync`)).json();
  expect(groupSync.changes.some((c: { kind: string }) => c.kind === "renamed")).toBeTruthy();
  const updated = await page.request.post(`/api/projects/${project}/packages/${group.id}/stage-sync`, { headers: origin, data: { choice: "update", revision: program.program.stage_revision } });
  expect(updated.ok()).toBeTruthy();
  expect((await packages()).find(p => p.id === group.id)!.stages.some(l => l.label === "Sketch design")).toBeTruthy();

  // Program shows its own stages, with no row per package.
  await page.getByRole("button", { name: "← Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Program", exact: true })).toBeVisible();
  const rows = page.getByRole("treegrid", { name: "Program rows" });
  await expect(rows).toContainText("Sketch design");
  await expect(rows).not.toContainText("Civil and Structural");

  // Uncombine: each member's own lines are live, and offer the rename.
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Uncombine Civil and Structural", exact: true }).click();
  await expect(page.getByRole("button", { name: "Uncombine Civil and Structural", exact: true })).toHaveCount(0);
  for (const m of members) {
    const sync = await (await page.request.get(`/api/projects/${project}/packages/${m.id}/stage-sync`)).json();
    expect(sync.group_id).toBeUndefined();
    expect(sync.stages.length).toBeGreaterThan(0);
    expect(sync.offer).toBe(true);
    expect(sync.changes.some((c: { kind: string; new_label?: string }) => c.kind === "renamed" && c.new_label === "Sketch design")).toBeTruthy();
  }
});
