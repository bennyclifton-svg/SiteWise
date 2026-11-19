import { expect, test } from "@playwright/test";

// The Cost Plan's Variations and claims tab
// (docs/plans/2026-11-17-variations-progress-claims.md, Task 10): add and
// approve a variation, drop a claim, accept it, issue the schedule; every
// figure is the server's.
test("a variation moves the contract and a dropped claim is assessed and issued", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Variations and claims");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: /Variations and claims/ }).click();
  const head = page.getByLabel("Main Works figures");
  await expect(head).toContainText("150,000");

  // Add V01 in the grey row, on Substructure, claimed 12,400.
  await page.getByLabel("New variation description").fill("Rock excavation");
  await page.getByLabel("New variation line").selectOption({ label: "Substructure" });
  await page.getByLabel("New variation claimed amount").fill("12400");
  await page.getByLabel("New variation claimed amount").press("Enter");
  await expect(page.getByLabel("Status of V01")).toHaveValue("submitted");
  await expect(page.getByRole("button", { name: "Undo" })).toBeVisible();

  // Approve at 11,000: the contract sum to date moves by 11,000.
  page.once("dialog", d => void d.accept("11000"));
  await page.getByLabel("Status of V01").selectOption("approved");
  await expect(head).toContainText("161,000");

  // Drop the builder's claim on the contract's claims table.
  const claim = await (await page.request.get("/__e2e/claim.md")).body();
  await page.getByTestId("claim-input").setInputFiles({ name: "PC-04.md", mimeType: "text/markdown", buffer: claim });
  await expect(page.getByRole("button", { name: "PC-04" })).toBeVisible({ timeout: 20_000 });
  await page.getByRole("button", { name: "PC-04" }).click();
  const panel = page.getByRole("complementary", { name: "Claim PC-04" });
  await expect(panel.getByLabel("Line for Substructure")).toHaveValue(/[0-9a-f-]{36}/);
  await panel.getByRole("button", { name: "Accept claimed" }).click();
  await expect(panel.getByLabel("Schedule totals")).toContainText("83,700");
  // Issue: the schedule is frozen and the claim counts in Claimed to Date.
  await panel.getByRole("button", { name: "Issue payment schedule" }).click();
  await expect(panel.getByRole("link", { name: "Download PDF" })).toBeVisible();
  const ledger = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as { summary: { invoiced: string }; claims: { alert: string }[] };
  expect(ledger.summary.invoiced).toBe("86000.00");
  // Served: the schedule's alert clears, and with it the banner.
  await panel.getByLabel("Schedule served on").fill("2026-10-20");
  await panel.getByLabel("Schedule served on").blur();
  await expect(page.locator(".claim-banner")).toHaveCount(0);
});
