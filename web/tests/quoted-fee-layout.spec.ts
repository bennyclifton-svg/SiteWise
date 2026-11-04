import { test, expect } from "@playwright/test";

test("quoted review leads with source fees and adds agreed fees per firm", async ({ page }) => {
  const headers = { Origin: "http://127.0.0.1:4173" };
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Quoted fee layout");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const base = `/api/projects/${new URL(page.url()).pathname.split("/").pop()}`;
  const pkg = await (await page.request.post(`${base}/procurement/rows`, { headers, data: { discipline_id: "consultant.architect" } })).json();
  const upload = async (firm: string, _abn: number) => {
    const source = `# FEE PROPOSAL
**${firm}**
## Fee basis
Fixed professional fee: **$3,000 excluding GST**, staged as follows:
| Stage | Fee (ex GST) |
| --- | --- |
| Schematic design | $1,000 |
| Detailed design | $2,000 |
`;
    expect((await page.request.post(`${base}/procurement/packages/${pkg.id}/submissions?name=${encodeURIComponent(firm + '.md')}`, { headers, data: Buffer.from(source) })).ok()).toBeTruthy();
  };
  await upload("First Architecture Pty Ltd", 1);
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Review Architect", exact: true }).click();
  const comparison = page.locator(".consultant-comparison");
  const total = comparison.locator("tr.prices-total");
  await expect(total).toContainText("$3,000.00", { timeout: 20000 });
  await expect(comparison.locator("thead")).toContainText("First Architecture Pty Ltd");
  await expect(comparison.locator("thead")).not.toContainText(".pdf");
  await expect(comparison.getByText("Use a different amount", { exact: true })).toHaveCount(0);
  await expect(comparison.locator(".appointment-row").first()).toBeVisible();
  await page.getByRole("button", { name: "+ Add row", exact: true }).click();
  await page.getByLabel("Service description").fill("Agreed contingency");
  await page.getByRole("button", { name: "Save row", exact: true }).click();
  await expect(total).toContainText("Not confirmed");
  const addition = comparison.getByRole("row").filter({ has: page.getByRole("rowheader", { name: "Agreed contingency", exact: true }) });
  await addition.locator("summary").click();
  await addition.getByLabel("Amount ex GST").fill("500");
  await addition.getByLabel("Agreement note").fill("Agreed with architect by telephone");
  await addition.getByRole("button", { name: "Save fee", exact: true }).click();
  await expect(total).toContainText("$3,500.00");
  await addition.locator("summary").click();
  await page.screenshot({ path: "../.tools/quoted-fees-single.png", fullPage: true });
  await total.getByRole("button", { name: "Award to First Architecture Pty Ltd", exact: true }).click();
  const award = page.getByRole("dialog");
  await expect(award).toContainText("Agreed contingency");
  await expect(award).toContainText("$3,500.00");
  await expect(award.getByRole("radio", { name: "By Stage", exact: true })).toBeChecked();
  await expect(award.locator(".award-total")).toContainText("3 lines");
  await award.getByRole("button", { name: "Next: review", exact: true }).click();
  await expect(award.locator(".award-preview li")).toHaveCount(3);
  await expect(award.locator(".award-preview")).toContainText("$3,500.00");
  await award.getByRole("button", { name: "Back", exact: true }).click();
  await award.getByRole("radio", { name: "One total", exact: true }).check();
  await expect(award.locator(".award-total")).toContainText("$3,500.00");
  await expect(award.locator(".award-total")).toContainText("1 line");
  await award.getByRole("button", { name: "Next: review", exact: true }).click();
  await expect(award.locator(".award-preview li")).toHaveCount(1);
  await expect(award.locator(".award-preview")).toContainText("$3,500.00");
  await award.getByRole("button", { name: "Back", exact: true }).click();
  await award.getByRole("radio", { name: "By Stage", exact: true }).check();
  await expect(award.locator(".award-total")).toContainText("3 lines");
  await page.screenshot({ path: "../.tools/award-stages-added-fee.png", fullPage: true });
  await award.getByRole("button", { name: "Cancel", exact: true }).click();
  await upload("Second Architecture Pty Ltd", 2);
  await page.getByRole("button", { name: "← Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Compare Architect", exact: true }).click();
  await expect(comparison.locator("thead")).toContainText("Second Architecture Pty Ltd", { timeout: 20000 });
  await expect(total.locator("td").first()).toContainText("$3,500.00");
  await expect(total.locator("td:not(.comparison-placeholder)").last()).toContainText("Not confirmed");
  await page.screenshot({ path: "../.tools/quoted-fees-multiple.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  await page.screenshot({ path: "../.tools/quoted-fees-mobile.png", fullPage: true });
});

test("scope arrows persist and award only the included fees", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const headers = { Origin: "http://127.0.0.1:4173" };
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Structural appointment scope");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const base = `/api/projects/${new URL(page.url()).pathname.split("/").pop()}`;
  const pkg = await (await page.request.post(`${base}/procurement/rows`, { headers, data: { discipline_id: "consultant.structural" } })).json();
  const source = `# FEE PROPOSAL
**Scope Engineers Pty Ltd**
## Fee basis
Fixed professional fee: **$41,800 excluding GST**, staged as follows:
| Stage | Fee (ex GST) |
| --- | --- |
| Schematic design | $12,540 |
| Detailed design | $20,900 |
| Construction services | $8,360 |
## Scope
Structural design for the residence.
## Exclusions
Geotechnical investigations are excluded.
## Conditions
Payment within 30 days.
`;
  expect((await page.request.post(`${base}/procurement/packages/${pkg.id}/submissions?name=Structural-fee.md`, { headers, data: Buffer.from(source) })).ok()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: /Review Structural/ }).click();
  const comparison = page.locator(".consultant-comparison");
  const total = comparison.locator(".prices-total");
  await expect(total).toContainText("$41,800.00", { timeout: 20000 });
  await expect(comparison.locator(".appointment-terms > details[open]")).toHaveCount(0);
  await expect(page.getByText("Schedule details and scope corrections", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Quoted packages and totals", { exact: true })).toHaveCount(0);
  // Map the source line to one RFP stage and retain its original wording in details.
  const table = await (await page.request.get(`${base}/procurement/packages/${pkg.id}/prices`)).json();
  const schematic = table.columns[0].appointment_rows.find((r: {source_label: string}) => r.source_label === "Schematic design");
  const stage = table.items.find((i: {kind: string; key: string}) => i.kind === "stage" && i.key !== "comparison:appointment");
  const sourceRow = comparison.getByRole("row").filter({ has: page.getByRole("rowheader", { name: schematic.label, exact: true }) });
  await sourceRow.locator("summary").click();
  for (const checkbox of await sourceRow.getByRole("checkbox").all()) await checkbox.uncheck();
  await sourceRow.getByRole("checkbox", { name: stage.label, exact: true }).check();
  await sourceRow.getByRole("button", { name: "Save mapping", exact: true }).click();
  await expect(comparison.getByRole("rowheader", { name: stage.label, exact: true })).toHaveCount(1);
  const mapped = comparison.getByRole("row").filter({ has: page.getByRole("rowheader", { name: stage.label, exact: true }) });
  // A remapped row is new in the table and starts collapsed.
  await expect(mapped.locator("details[open]")).toHaveCount(0);
  const excludedLabel = table.columns[0].appointment_rows.find((r: {source_label: string}) => r.source_label === "Construction services").label;
  await comparison.getByRole("button", { name: `Exclude ${excludedLabel}, Scope Engineers Pty Ltd`, exact: true }).click();
  await expect(total).toContainText("$33,440.00");
  await comparison.getByRole("button", { name: `Include ${excludedLabel}, Scope Engineers Pty Ltd`, exact: true }).click();
  await expect(total).toContainText("$41,800.00");
  await comparison.getByRole("button", { name: `Exclude ${excludedLabel}, Scope Engineers Pty Ltd`, exact: true }).click();
  await expect(total).toContainText("$33,440.00");
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: /Review Structural/ }).click();
  await expect(total).toContainText("$33,440.00");
  await expect(comparison.getByRole("button", { name: `Include ${excludedLabel}, Scope Engineers Pty Ltd`, exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/appointment-scope-desktop.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  await page.screenshot({ path: "../.tools/appointment-scope-mobile.png" });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await total.getByRole("button", { name: "Award to Scope Engineers Pty Ltd", exact: true }).click();
  const award = page.getByRole("dialog");
  await expect(award.locator(".award-table tbody tr")).toHaveCount(2);
  for (const checkbox of await award.getByRole("checkbox").all()) await checkbox.check();
  await award.getByLabel("Name for combined line").fill("Design services");
  await award.getByRole("button", { name: "Combine 2 selected", exact: true }).click();
  await expect(award.locator(".award-total")).toContainText("$33,440.00");
  await award.getByRole("button", { name: "Next: review", exact: true }).click();
  await expect(award.locator(".award-preview li")).toHaveCount(1);
  await award.getByRole("button", { name: "Award and write to cost plan", exact: true }).click();
  await expect(award).toHaveCount(0);
  await page.getByRole("button", { name: "Open cost plan", exact: true }).click();
  await expect(page.getByRole("cell", { name: "Design services", exact: true })).toBeVisible();
  const plan = await (await page.request.get(`${base}/cost-plan`)).json();
  const line = plan.items.find((i: {label: string}) => i.label === "Design services");
  expect(line.values.commitment.amount).toBe("33440.00");
  expect(line.values.budget).toBeUndefined();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Undo award", exact: true }).click();
  await expect(total).toContainText("$33,440.00");
  const undone = await (await page.request.get(`${base}/cost-plan`)).json();
  expect(undone.items.some((i: {label: string}) => i.label === "Design services")).toBeFalsy();
});
