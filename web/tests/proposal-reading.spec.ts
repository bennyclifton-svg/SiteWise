import { test, expect } from "@playwright/test";

test("Markdown proposal shows its package, stage breakdown and source qualifications", async ({ page }) => {
  await page.setViewportSize({ width: 1872, height: 1000 });
  const headers = { Origin: "http://127.0.0.1:4173" };
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Proposal source accuracy");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const base = `/api/projects/${new URL(page.url()).pathname.split("/").pop()}`;
  const pkg = await (await page.request.post(`${base}/procurement/rows`, { headers, data: { discipline_id: "consultant.architect" } })).json();
  const source = "# FEE PROPOSAL\n**Example Studio Pty Ltd**\ninfo@examplestudio.test\n## Scope of services\nConcept design and construction documentation.\n**Exclusions** unless separately agreed: legal advice and landscape design.\n## Fee basis\nFixed professional fee: **$30,000 excluding GST** ($33,000 incl. GST), staged as follows:\n| Stage | Trigger | Fee (ex GST) |\n| --- | --- | --- |\n| Concept | Signed | $10,000 |\n| Documentation | Issue | $20,000 |\n";
  const upload = await page.request.post(`${base}/procurement/packages/${pkg.id}/submissions?name=fee-example-studio.md`, { headers, data: Buffer.from(source) });
  expect(upload.status()).toBe(201);
  await expect.poll(async () => {
    const board = await (await page.request.get(`${base}/procurement/board`)).json();
    return board.packages[pkg.id]?.[0]?.status;
  }).toBe("received");
  const table = await (await page.request.get(`${base}/procurement/packages/${pkg.id}/prices`)).json();
  expect(table.columns[0].quote.fees).toHaveLength(3);
  expect(table.columns[0].quote.fees.filter((f: any) => f.role.startsWith("Breakdown"))).toHaveLength(2);
  expect(table.columns[0].cells["comparison:appointment"]?.amount ?? null).toBeNull();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Review Architect", exact: true }).click();
  const comparison = page.locator(".consultant-comparison");
  await expect(comparison.getByRole("rowheader", { name: /^Exclusions and qualifications/ })).toBeVisible();
  await expect(comparison.getByText("Exclusions unless separately agreed: legal advice and landscape design.", { exact: true })).toBeVisible();
  const packages = comparison.getByRole("row").filter({ hasText: "Quoted packages and totals" });
  await expect(packages.getByText("$30,000.00", { exact: true })).toBeVisible();
  await expect(packages.getByText("$10,000.00", { exact: true })).toHaveCount(0);
  await expect(comparison.locator(".quote-source-row")).toHaveCount(3);
  await expect(comparison.getByText("$10,000.00", { exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/proposal-reading-desktop.png", fullPage: true });
});
