import { expect, test } from "@playwright/test";

// The project report (docs/plans/2026-11-17-monthly-report.md, Task 7):
// Report on the left nav (25 November 2026), with an RFP's section bands;
// Update, comment, issue and download.
test("project report: open, update, comment, issue and download", async ({ page }) => {
  page.on("pageerror", error => { throw error; });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Project report check");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop();

  // Opening Report twice makes one report.
  const report = page.getByRole("group", { name: "Project views" }).getByRole("button", { name: "Report", exact: true });
  await report.dblclick();
  await expect(page.getByRole("heading", { name: "Project report No. 1", exact: true })).toBeVisible();
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json();
  expect(reports.filter((r: { kind: string }) => r.kind === "monthly")).toHaveLength(1);
  for (const section of ["At a glance", "Needs attention", "Changes since the last report", "Cost", "Program", "Procurement", "Registers", "Next period"]) {
    await expect(page.getByRole("heading", { name: section, exact: true })).toBeVisible();
  }
  await expect(page.getByText("No cost plan yet.", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("This is the first issued report; changes are measured from the next one.", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Update", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Project report No. 1", exact: true })).toBeVisible();

  // The PM's comment under Cost.
  const cost = page.locator(".report-section").filter({ has: page.getByRole("heading", { name: "Cost", exact: true }) });
  await cost.locator(".report-block").filter({ has: page.getByRole("heading", { name: "Comment", exact: true }) }).locator("p.inline-text").click();
  await page.getByLabel("Report wording").fill("The cost plan is prepared next month.");
  await page.getByRole("button", { name: "Save wording", exact: true }).click();
  await expect(cost.getByText("The cost plan is prepared next month.", { exact: true })).toBeVisible();

  // Issue: the budget checkbox is not offered for this kind.
  await page.getByRole("button", { name: "Issue report", exact: true }).click();
  await expect(page.getByLabel("Include internal budget amounts in this external report")).toHaveCount(0);
  await page.getByLabel("Reporting date", { exact: true }).fill("2026-11-17");
  const stale = page.getByLabel("Issue the saved state despite changed or incomplete sources");
  if (await stale.count()) {
    await stale.check();
    await page.getByLabel("Reason for using saved state").fill("No profile yet on this test project.");
  }
  await page.getByRole("button", { name: "Freeze and issue PDF", exact: true }).click();
  const link = page.getByRole("link", { name: /Download issue 1/ });
  await expect(link).toBeVisible();
  const download = await page.request.get((await link.getAttribute("href"))!);
  expect(download.headers()["content-type"]).toContain("application/pdf");
  expect((await download.body()).subarray(0, 5).toString()).toBe("%PDF-");

  // The next draft is report 2 and carries the comment for review.
  await page.getByRole("button", { name: "Update", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Project report No. 2", exact: true })).toBeVisible();
  await expect(cost.getByText("The cost plan is prepared next month.", { exact: true })).toBeVisible();
  await expect(cost.getByText("The source changed.", { exact: false })).toBeVisible();
  await expect(link).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();

  // Report is a view of its own: no Back, and the nav marks it.
  await expect(page.getByRole("navigation", { name: "Breadcrumb" })).toHaveCount(0);
  await expect(report).toHaveAttribute("aria-current", "page");
});
