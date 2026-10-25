import { expect, test } from "@playwright/test";

// Program plan 11.5 step 12 (WP-P10): procurement first. Creating an RFP on
// a fresh project creates the program; Program has the same stages; a stage
// renamed in Program makes the RFP draft offer Update or Keep, and Update
// rewrites its stage rows. Stage controls stay inside the RFP.
// Program is driven through its API so this does not depend on the chart.
test("procurement first shares one stage plan with Program", async ({ page, baseURL }) => {
  const origin = { Origin: new URL(baseURL!).origin };
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Stage plan");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop();
  expect((await page.request.get(`/api/projects/${project}/program`)).status()).toBe(404);

  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Add discipline", exact: true }).click();
  await page.getByLabel("Discipline", { exact: true }).selectOption("consultant.architect");
  await page.getByRole("button", { name: "Add row", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Architect", exact: true }).click();
  await expect(page.getByRole("heading", { name: "RFP — Architect", exact: true })).toBeVisible();

  // Opening Program finds the same stages, already linked.
  const read = await page.request.get(`/api/projects/${project}/program`);
  expect(read.status()).toBe(200);
  let program = await read.json();
  expect(program.program.created_by_surface).toBe("procurement");
  const stages = program.activities.filter((a: { kind: string }) => a.kind === "stage");
  const concept = stages.find((s: { catalogue_stage_id: string }) => s.catalogue_stage_id === "concept_design");
  expect(concept.label).toBe("Concept design");
  const rows = page.getByRole("region", { name: "Program stages" });
  await expect(rows).toContainText("Concept design");

  // Rename the stage in Program: the RFP draft offers Update or Keep.
  const ops = await page.request.post(`/api/projects/${project}/program/ops`, { headers: origin,
    data: { program_id: program.program.id, version: program.program.version, ops: [{ op: "update", id: concept.id, label: "Sketch design" }] } });
  expect(ops.ok()).toBeTruthy();
  program = await ops.json();
  await page.getByRole("button", { name: "← Procurement", exact: true }).click();
  await page.getByRole("button", { name: "View RFP for Architect", exact: true }).click();
  const notice = page.getByRole("status", { name: "Stage changes for RFP draft" });
  await expect(notice).toContainText("Concept design renamed Sketch design");
  await notice.getByRole("button", { name: "Update", exact: true }).click();
  await expect(notice).toHaveCount(0);
  await expect(rows).toContainText("Sketch design");
  await expect(rows).not.toContainText("Concept design");

  await page.getByRole("button", { name: "← Procurement", exact: true }).click();
  await expect(page.getByRole("button", { name: "Stages for Architect", exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "View RFP for Architect", exact: true }).click();
  await expect(rows).toContainText("Sketch design");
});
