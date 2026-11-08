import { expect, test, type Page } from "@playwright/test";

// Word and PDF downloads from the three-dot menu (owner direction,
// 8 November 2026): the draft laid out as the screen shows it.
const origin = { Origin: "http://127.0.0.1:4173" };

async function architectRFP(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Export sample");
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

test("the three-dot menu downloads the draft as PDF and Word", async ({ page }, testInfo) => {
  const report = await architectRFP(page);
  // Delete one standard line so the download shows the user's edits.
  const deleted = await page.request.post(`/api/reports/${report}/settings`, { headers: origin, data: { changes: [{ key: "line:brief_concept.area_schedule", value: "D" }], version: (await (await page.request.get(`/api/reports/${report}`)).json()).draft.version } });
  expect(deleted.ok(), await deleted.text()).toBeTruthy();
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: /^View RFP for / }).click();
  await expect(page.locator(".rfp-stage-scope")).toHaveCount(1);

  const probe = await page.request.get(`/api/reports/${report}/export?format=pdf`);
  expect(probe.ok(), `${probe.status()} ${(await probe.body()).toString().slice(0, 300)}`).toBeTruthy();
  const menu = page.getByRole("button", { name: "Report actions" });
  for (const [label, ext, type] of [["Download PDF", "pdf", "application/pdf"], ["Download Word", "docx", "wordprocessingml"]] as const) {
    await menu.click();
    const started = Date.now();
    const [download] = await Promise.all([page.waitForEvent("download"), page.getByRole("menuitem", { name: label, exact: true }).click()]);
    expect(download.suggestedFilename()).toMatch(new RegExp(`Export sample RFP.*\\.${ext}$`));
    await download.saveAs(`../.tools/export-sample.${ext}`);
    testInfo.annotations.push({ type: "timing", description: `${label}: ${Date.now() - started}ms` });
    // The server's own timing for the same file, against report_export.
    const times: number[] = [];
    for (let i = 0; i < 5; i++) {
      const t = Date.now();
      const r = await page.request.get(`/api/reports/${report}/export?format=${ext}`);
      expect(r.ok()).toBeTruthy();
      expect(r.headers()["content-type"]).toContain(type);
      times.push(Date.now() - t);
    }
    times.sort((a, b) => a - b);
    testInfo.annotations.push({ type: "timing", description: `${ext} export p50 ${times[2]}ms, max ${times[4]}ms` });
    console.log(`TIMING ${ext} export p50 ${times[2]}ms, max ${times[4]}ms`);
  }
  // Print stays, as the browser's own print.
  await menu.click();
  await expect(page.getByRole("menuitem", { name: "Print", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  expect((await page.request.get(`/api/reports/${report}/export?format=html`)).status()).toBe(400);
});
