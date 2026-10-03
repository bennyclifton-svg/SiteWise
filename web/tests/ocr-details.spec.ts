import { expect, test } from "@playwright/test";

test("missing details recover in place, with progress and a missed-event fallback", async ({ page }) => {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Bankstown recovery check");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "Bankstown recovery check" })).toBeVisible();
  const projectId = page.url().split("/projects/")[1];
  const doc = {
    id: "33333333-3333-4333-8333-000000000998", project_id: projectId,
    filename: "1115 CC-06 LEVEL 02 F.pdf", created_at: "2026-10-03T06:00:00Z", status: "filed", reason: "ocr_review",
    fields: [{ field: "kind", value: "drawing", band: "amber", decided_by: "jev" }, { field: "revision", value: "F", band: "amber", decided_by: "rule" }],
  };
  await page.route(`**/api/projects/${projectId}/documents`, route => route.fulfill({ json: { cursor: 0, project: { id: projectId, name: "Bankstown recovery check" }, documents: [doc] } }));
  await page.route(`**/api/documents/${doc.id}`, route => route.fulfill({ json: doc }));
  let requests = 0;
  let reject = true;
  await page.route(`**/api/documents/${doc.id}/details/reprocess`, async route => {
    requests++;
    if (reject) { await route.fulfill({ status: 503, body: "Unavailable" }); return; }
    doc.reason = "ocr_details_queued";
    await route.fulfill({ status: 202, json: doc });
  });
  await page.reload();
  const row = page.locator("tbody.reg-doc");
  await row.locator(".reg-line").click();
  const panel = row.getByRole("status", { name: "Text recovery" });
  await expect(panel.locator("p").filter({ hasText: "Still missing:" })).toContainText("drawing title, drawing number, discipline");
  await panel.getByRole("button", { name: "Reprocess missing details" }).click();
  await expect(panel.getByRole("alert")).toHaveText(/Recovery did not start/);
  reject = false;
  await panel.getByRole("button", { name: "Reprocess missing details" }).click();
  await expect(panel.getByText("Detail recovery queued", { exact: true })).toBeVisible();
  await expect(panel.getByRole("button")).toHaveCount(0);
  expect(requests).toBe(2);
  doc.reason = "ocr_details_reading";
  await expect(panel.getByText("Reading drawing details", { exact: true })).toBeVisible();
  await expect(page.locator("tbody.reg-doc")).toHaveCount(1);
  await page.setViewportSize({ width: 1440, height: 960 });
  await page.screenshot({ path: "test-results/ocr-details-desktop.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await panel.scrollIntoViewIfNeeded();
  const bounds = await panel.boundingBox();
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(391);
  await page.screenshot({ path: "test-results/ocr-details-mobile.png" });
  doc.reason = "ocr_details_classifying";
  await expect(panel.getByText("Checking missing details", { exact: true })).toBeVisible();
  doc.reason = "ocr_details_review";
  doc.fields.push({ field: "title", value: "LEVEL 2", band: "amber", decided_by: "rule" }, { field: "number", value: "CC-06", band: "amber", decided_by: "rule" });
  await expect(panel.getByText("Details recovered · check fields", { exact: true })).toBeVisible();
  await expect(panel.locator("p").filter({ hasText: "Still missing:" })).toContainText("discipline");
  await expect(panel.locator("p").filter({ hasText: "Still missing:" })).not.toContainText("drawing title");
  await expect(row.locator(".reg-title")).toContainText("LEVEL 2");
  await panel.getByRole("button", { name: "Reprocess missing details" }).click();
  await expect(panel.getByText("Detail recovery queued", { exact: true })).toBeVisible();
  doc.reason = "ocr_details_unchanged";
  await expect(panel.getByText("No new details recovered", { exact: true })).toBeVisible();
  await panel.getByRole("button", { name: "Reprocess missing details" }).click();
  await expect(panel.getByText("Detail recovery queued", { exact: true })).toBeVisible();
  doc.reason = "ocr_details_failed";
  await expect(panel.getByText("Detail recovery stopped", { exact: true })).toBeVisible();
  await expect(panel.getByRole("button", { name: "Reprocess missing details" })).toBeEnabled();
  await expect(row.locator(".reg-title")).toContainText("LEVEL 2");
});
