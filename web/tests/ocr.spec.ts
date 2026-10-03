import { expect, test } from "@playwright/test";

test("OCR stages stay clear in the register and on mobile", async ({ page }) => {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("OCR progress check");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "OCR progress check" })).toBeVisible();
  const projectId = page.url().split("/projects/")[1];
  let reason = "ocr_queued";
  let status = "pending";
  await page.route(`**/api/projects/${projectId}/documents`, async (route) => route.fulfill({ json: {
    cursor: 0, project: { id: projectId, name: "OCR progress check" }, documents: [{
      id: "33333333-3333-4333-8333-000000000999", project_id: projectId,
      filename: "CC-06 LEVEL 02 F.pdf", created_at: "2026-10-03T06:00:00Z", status, reason, fields: [],
    }],
  } }));
  for (const [next, title] of [["ocr_queued", "OCR queued"], ["ocr_reading", "Reading lettering"], ["ocr_classifying", "Filing recovered text"]]) {
    reason = next;
    await page.reload();
    const row = page.locator("tbody.reg-doc");
    await expect(row.locator(".ocr-summary")).toHaveText(title);
    await row.locator(".reg-line").click();
    await expect(row.getByRole("status", { name: "Text recovery" })).toBeVisible();
    await expect(row.locator('[aria-current="step"]')).toHaveCount(1);
    await expect(row.getByRole("button", { name: "Retry filing" })).toHaveCount(0);
  }
  await page.setViewportSize({ width: 1440, height: 960 });
  await page.screenshot({ path: "test-results/ocr-desktop.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.locator(".ocr-status")).toBeVisible();
  const panel = await page.locator(".ocr-status").boundingBox();
  expect(panel!.x + panel!.width).toBeLessThanOrEqual(391);
  await page.locator(".ocr-status").scrollIntoViewIfNeeded();
  await page.screenshot({ path: "test-results/ocr-mobile.png" });
  status = "filed"; reason = "ocr_review";
  await page.reload();
  await expect(page.locator(".ocr-summary")).toHaveText("OCR complete · check fields");
  status = "not_filed"; reason = "ocr_failed";
  await page.reload();
  await page.locator(".reg-line").click();
  await expect(page.getByRole("button", { name: "Retry OCR" })).toBeVisible();
  await expect(page.getByText(/OCR could not finish/)).toBeVisible();
});
