import { expect, test, type Page } from "@playwright/test";

// Scope lines edited in place (owner direction, 8 November 2026): a closed
// stage shows a count and a preview, an open stage has one fee cell, every
// line is reworded where it reads, "Add item…" adds a line, and lines are
// deleted alone or together with Ctrl selections in one write.
const origin = { Origin: "http://127.0.0.1:4173" };
type Line = { key: string; text: string; value: string; custom?: boolean; edited?: boolean };

async function architectRFP(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Scope lines");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.work_type", "extend"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), `${key}: ${await r.text()}`).toBeTruthy();
  }
  expect((await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Architect", discipline_id: "consultant.architect" } })).ok()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Architect", exact: true }).click();
  await expect(page.locator(".rfp-stage-scope")).toHaveCount(1);
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json() as { id: string }[];
  return reports[0].id;
}

async function stageLines(page: Page, report: string, stage: string): Promise<Line[]> {
  const view = await (await page.request.get(`/api/reports/${report}`)).json();
  const block = view.draft.sections.flatMap((s: { blocks: { id: string }[] }) => s.blocks).find((b: { id: string }) => b.id === "rfp:stages");
  return block.basis.stages.find((s: { key: string }) => s.key === stage).lines;
}

test("scope lines: count, one fee cell, reword, add, delete together", async ({ page }) => {
  const report = await architectRFP(page);
  const stage = page.locator("#rfp-stage-brief_concept");
  const lines = await stageLines(page, report, "brief_concept");
  expect(lines.length).toBeGreaterThan(3);

  // Closed: the count in a circle, a preview of the lines, one $ fee cell.
  const count = Number(await stage.locator(".rfp-stage-badge").textContent());
  expect(count).toBeGreaterThanOrEqual(lines.length);
  await expect(stage.locator(".rfp-stage-preview")).toContainText(lines[0].text.slice(1, 12));
  await expect(stage.locator(".rfp-fee-stage")).toHaveText("$");

  await page.setViewportSize({ width: 1500, height: 1000 });
  await stage.scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../.tools/rfp-scope-lines-closed.png" });
  // Open: no summary row, the stage name and one fee cell run down the lines.
  await stage.getByRole("button", { name: /^Brief and concept/ }).click();
  await expect(stage.locator(".rfp-stage-row")).toHaveCount(0);
  await expect(stage.locator(".rfp-fee-stage")).toHaveCount(1);
  const rows = await stage.locator("tr").count();
  await expect(stage.locator(".rfp-fee-stage")).toHaveAttribute("rowspan", String(rows));
  await expect(stage.locator(".rfp-stage-name")).toHaveAttribute("rowspan", String(rows));
  await expect(stage.getByRole("checkbox", { name: "Brief and concept: included", exact: true })).toHaveCount(0);

  // Reword a standard line where it reads; clicking away saves it.
  const first = stage.locator(`tr[data-line="${lines[0].key}"]`);
  await first.locator("p.inline-text").click();
  const box = first.locator("textarea");
  await expect(box).toBeFocused();
  await box.press("End");
  await box.pressSequentially(" with the client");
  await Promise.all([
    page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes(`/settings/wording:brief_concept.${lines[0].key}`)),
    page.locator(".rfp-stage-scope thead th").first().click(),
  ]);
  await expect(first).toHaveAttribute("data-edited", "true");
  await expect(first.locator("p.inline-text")).toHaveText(/with the client$/);

  // "Add item…" adds the user's own line.
  const ghost = stage.getByRole("textbox", { name: "New scope line for Brief and concept", exact: true });
  await ghost.click();
  await ghost.fill("Coordinate the survey");
  await Promise.all([page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes("/settings/custom:brief_concept")), ghost.press("Enter")]);
  const added = stage.locator("tr.rfp-line", { hasText: "Coordinate the survey" });
  await expect(added).toHaveCount(1);

  await page.locator(".rfp-stage-scope thead th").first().click();
  await stage.scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../.tools/rfp-scope-lines-open.png" });
  // Ctrl+click two lines, then Delete: one write removes both.
  const second = stage.locator(`tr[data-line="${lines[1].key}"]`);
  await second.click({ modifiers: ["Control"] });
  await added.click({ modifiers: ["Control"] });
  await expect(stage.locator("tr[data-selected]")).toHaveCount(2);
  const toolbar = page.getByRole("toolbar", { name: "Selected scope lines" });
  await expect(toolbar).toContainText("2 lines selected");
  let writes = 0;
  page.on("request", r => { if (r.url().includes("/settings") && r.method() !== "GET") writes++; });
  await Promise.all([page.waitForResponse(r => r.request().method() === "POST" && r.url().endsWith("/settings")), toolbar.getByRole("button", { name: "Delete" }).click()]);
  expect(writes).toBe(1);
  await expect(second).toHaveCount(0);
  await expect(added).toHaveCount(0);

  // Right-click: delete one line from the menu.
  const third = stage.locator(`tr[data-line="${lines[2].key}"]`);
  await third.click({ button: "right" });
  await Promise.all([page.waitForResponse(r => r.request().method() === "POST" && r.url().endsWith("/settings")), page.getByRole("menuitem", { name: "Delete line" }).click()]);
  await expect(third).toHaveCount(0);

  // Saved: the reworded line kept, the deleted lines gone, the count down.
  const after = await stageLines(page, report, "brief_concept");
  expect(after.map(l => l.key)).not.toContain(lines[1].key);
  expect(after.map(l => l.key)).not.toContain(lines[2].key);
  expect(after.some(l => l.custom)).toBeFalsy();
  expect(after.find(l => l.key === lines[0].key)).toMatchObject({ edited: true });
  await expect(stage.locator(".rfp-stage-badge")).toHaveText(String(count - 2));

  // Reset to standard wording from the menu.
  await first.click({ button: "right" });
  await Promise.all([page.waitForResponse(r => r.request().method() === "DELETE" && decodeURIComponent(r.url()).includes(`/settings/wording:brief_concept.${lines[0].key}`)), page.getByRole("menuitem", { name: "Reset to standard wording" }).click()]);
  await expect(first).not.toHaveAttribute("data-edited", "true");
});
