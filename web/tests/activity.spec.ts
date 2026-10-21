import { expect, test, type Page } from "@playwright/test";

async function project(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Activity states");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "Activity states" })).toBeVisible();
  return page.url().split("/projects/")[1];
}

test("queued uploads wait for a slot and failed uploads stop animating", async ({ page }) => {
  const id = await project(page);
  const release: (() => void)[] = [];
  await page.route(`**/api/projects/${id}/files*`, async route => {
    await new Promise<void>(resolve => release.push(resolve));
    await route.fulfill({ status: 500, json: { error: "Upload failed" } });
  });
  const picker = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Add files" }).click();
  await (await picker).setFiles(Array.from({ length: 6 }, (_, i) => ({ name: `batch-${i}.pdf`, mimeType: "application/pdf", buffer: Buffer.from("%PDF-1.4") })));
  await expect(page.locator(".reg-line .activity-icon[data-state=queued]")).toHaveCount(2);
  await expect(page.locator(".reg-line .activity-icon[data-state=active]")).toHaveCount(4);
  await expect(page.locator(".reg-activity")).toContainText("4 processing · 2 waiting");
  release.splice(0).forEach(resolve => resolve());
  await expect(page.locator(".reg-line .activity-icon[data-state=queued]")).toHaveCount(0);
  await expect(page.locator(".reg-line .activity-icon[data-state=active]")).toHaveCount(2);
  release.splice(0).forEach(resolve => resolve());
  await expect(page.locator(".reg-line .activity-icon")).toHaveCount(0);
  await expect(page.locator(".reg-activity")).toHaveCount(0);
});

test("reading states stay honest, stop on completion, and respect reduced motion", async ({ page }, testInfo) => {
  const id = await project(page);
  const docs = [
    { id: "33333333-3333-4333-8333-000000000981", project_id: id, filename: "Queued drawing.pdf", status: "pending", reason: "ocr_queued", created_at: "2026-10-19T00:00:00Z", fields: [] },
    { id: "33333333-3333-4333-8333-000000000982", project_id: id, filename: "Reading drawing.pdf", status: "pending", reason: "ocr_reading", created_at: "2026-10-19T00:00:00Z", fields: [] },
    { id: "33333333-3333-4333-8333-000000000983", project_id: id, filename: "Image review.pdf", status: "filed", reason: "", visual_status: "queued", created_at: "2026-10-19T00:00:00Z", fields: [] },
  ];
  await page.route(`**/api/projects/${id}/documents`, r => r.fulfill({ json: { cursor: 0, project: { id, name: "Activity states" }, documents: docs } }));
  for (const doc of docs) await page.route(`**/api/documents/${doc.id}`, r => r.fulfill({ json: doc }));
  await page.reload();
  const queued = page.locator('[data-filename="Queued drawing.pdf"] .reg-line');
  const reading = page.locator('[data-filename="Reading drawing.pdf"] .reg-line');
  await expect(queued).toContainText("OCR queued");
  await expect(queued.locator(".activity-icon svg")).toHaveCSS("animation-name", "none");
  await expect(reading).toContainText("Reading lettering");
  await expect(reading.locator(".activity-icon svg")).toHaveCSS("animation-name", "activity-turn");
  await expect(page.getByText("Waiting for image reading…", { exact: true })).toBeVisible();
  await expect(reading.locator(".activity")).not.toHaveCSS("color", await queued.locator(".activity").evaluate(el => getComputedStyle(el).color));
  await expect(reading.locator("td").first()).not.toHaveCSS("background-color", await queued.locator("td").first().evaluate(el => getComputedStyle(el).backgroundColor));
  await page.setViewportSize({ width: 1700, height: 1000 });
  await page.locator(".register").screenshot({ path: testInfo.outputPath("activity-desktop.png") });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.locator(".register").screenshot({ path: testInfo.outputPath("activity-mobile.png") });
  await page.emulateMedia({ reducedMotion: "reduce" });
  await expect(reading.locator(".activity-icon svg")).toHaveCSS("animation-name", "none");
  docs[0].reason = "ocr_reading";
  docs[1].reason = "ocr_review"; docs[1].status = "filed";
  await page.reload();
  await expect(queued).toContainText("Reading lettering");
  await expect(reading.locator(".activity-icon")).toHaveCount(0);
  await expect(reading).toContainText("OCR complete · check fields");
});
