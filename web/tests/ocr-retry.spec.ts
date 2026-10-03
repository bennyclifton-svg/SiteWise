import { expect, test } from "@playwright/test";

test("Retry OCR shows completion without an SSE notification", async ({ page }) => {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("OCR retry delivery");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "OCR retry delivery" })).toBeVisible();
  const projectId = page.url().split("/projects/")[1];
  const doc = {
    id: "33333333-3333-4333-8333-000000000997", project_id: projectId,
    filename: "CC-11 LEVEL 08 E.pdf", created_at: "2026-10-03T06:00:00Z",
    status: "not_filed", reason: "no_text_layer", fields: [],
  };
  await page.route(`**/api/projects/${projectId}/documents`, route => route.fulfill({ json: {
    cursor: 0, project: { id: projectId, name: "OCR retry delivery" }, documents: [doc],
  } }));
  let reads = 0;
  await page.route(`**/api/documents/${doc.id}`, route => { reads++; return route.fulfill({ json: doc }); });
  await page.route(`**/api/documents/${doc.id}/filing`, route => {
    doc.status = "pending"; doc.reason = "ocr_queued";
    return route.fulfill({ status: 202, json: doc });
  });
  await page.reload();
  const row = page.locator("tbody.reg-doc");
  await row.locator(".reg-line").click();
  await row.getByRole("button", { name: "Retry OCR", exact: true }).click();
  await expect(row.locator(".ocr-summary")).toHaveText("OCR queued");
  // The worker finishes, but no event is delivered to this browser.
  doc.status = "filed"; doc.reason = "ocr_review";
  await expect(row.locator(".ocr-summary")).toHaveText("OCR complete · check fields", { timeout: 5000 });
  expect(reads).toBeGreaterThan(0);
  const completedReads = reads;
  await page.waitForTimeout(2500);
  expect(reads).toBe(completedReads);
});
