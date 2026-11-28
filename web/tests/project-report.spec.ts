import { expect, test } from "@playwright/test";

// The project report in the owner's format
// (docs/plans/2026-11-26-project-report-golden-format.md): Report on the
// left nav, the owner's sections in order, every table editable row by row,
// the earlier sections as review lists that are never issued; Update, issue
// and download, and the PM's words carried into the next report.
test("project report: owner's sections, type over a cell, add and hide rows, issue and carry", async ({ page }, testInfo) => {
  page.on("pageerror", error => { throw error; });
  await page.setViewportSize({ width: 1600, height: 1000 });
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
  // A draft made before the owner's format refreshes into it on Update.
  await page.getByRole("button", { name: "Update", exact: true }).click();
  for (const section of ["01 Site and project context", "02 Executive summary", "03 Statutory approvals", "04 Programme review", "05 Cost planning",
    "06 Design management", "07 Site activity", "08 Variation and change control", "09 Insurances, security and bonds", "Attachment A Payment schedules"]) {
    await expect(page.getByRole("heading", { name: section, exact: true })).toBeVisible();
  }
  // The earlier sections are review lists, kept out of the issued report.
  await expect(page.getByText(/At a glance \(\d+\)/)).toBeVisible();
  await expect(page.getByLabel("Period to")).not.toHaveValue("");

  // Type over a cell: the words are the PM's, marked as theirs.
  const site = page.getByRole("table", { name: "Site activity" });
  await site.getByRole("button", { name: "Edit Summary, Safety and induction" }).click();
  await site.getByLabel("Summary, Safety and induction").fill("Inductions held for all new starters.");
  await site.getByLabel("Summary, Safety and induction").press("Enter");
  await expect(site.getByText("Inductions held for all new starters.")).toBeVisible();
  await expect(site.locator(".rt-dot")).toHaveCount(1);

  // Add a row of the PM's own, and hide a standard one.
  const approvals = page.getByRole("table", { name: "Authority approvals" });
  await approvals.getByLabel("New row in Authority approvals").fill("Sydney Water");
  await approvals.getByLabel("New row in Authority approvals").press("Enter");
  await expect(approvals.getByRole("button", { name: "Edit Authority, Sydney Water" })).toBeVisible();
  await approvals.getByRole("button", { name: "Row Energy distributor actions" }).click();
  await page.getByRole("menuitem", { name: "Hide row" }).click();
  await expect(page.getByText("1 row hidden")).toBeVisible();
  await expect(approvals.getByRole("button", { name: "Edit Authority, Energy distributor" })).toHaveCount(0);
  // An image dropped on a cell: saved by its bytes, shown under the words.
  const context = page.getByRole("table", { name: "Site and project context" });
  const cell = context.getByRole("button", { name: "Edit Detail, Development description" });
  await cell.evaluate(async el => {
    const canvas = document.createElement("canvas");
    canvas.width = 80; canvas.height = 40;
    const ctx = canvas.getContext("2d")!;
    ctx.fillStyle = "#1f4ea8"; ctx.fillRect(0, 0, 80, 40);
    const blob: Blob = await new Promise(r => canvas.toBlob(b => r(b!), "image/png"));
    const dt = new DataTransfer();
    dt.items.add(new File([blob], "site-sketch.png", { type: "image/png" }));
    el.closest("td")!.dispatchEvent(new DragEvent("drop", { dataTransfer: dt, bubbles: true, cancelable: true }));
  });
  await expect(context.getByRole("img", { name: "site-sketch" })).toBeVisible();
  await site.scrollIntoViewIfNeeded();
  await page.screenshot({ path: testInfo.outputPath("report-tables.png") });

  // Issue: the owner's format, as a PDF.
  await page.getByRole("button", { name: "Issue report", exact: true }).click();
  await expect(page.getByLabel("Include internal budget amounts in this external report")).toHaveCount(0);
  await page.getByLabel("Reporting date", { exact: true }).fill("2026-11-28");
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

  // The next draft is report 2 and carries the PM's words and rows.
  await page.getByRole("button", { name: "Update", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Project report No. 2", exact: true })).toBeVisible();
  await expect(site.getByText("Inductions held for all new starters.")).toBeVisible();
  await expect(approvals.getByRole("button", { name: "Edit Authority, Sydney Water" })).toBeVisible();
  await expect(page.getByText("1 row hidden")).toBeVisible();
  await page.getByRole("button", { name: "Show them again" }).click();
  await expect(approvals.getByRole("button", { name: "Edit Authority, Energy distributor" })).toBeVisible();
  await expect(link).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();

  // Report is a view of its own: no Back, and the nav marks it.
  await expect(page.getByRole("navigation", { name: "Breadcrumb" })).toHaveCount(0);
  await expect(report).toHaveAttribute("aria-current", "page");
});
