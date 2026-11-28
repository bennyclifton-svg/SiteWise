import { expect, test } from "@playwright/test";

// The Cost Plan's Variations tab
// (docs/plans/2026-11-17-variations-progress-claims.md, Task 10): add and
// approve a variation; every figure is the server's. Progress claims have
// their own tab and test (progress-claims.spec.ts).
test("a variation moves the contract", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Variations");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  await page.request.post(`/__e2e/contract?project=${project}`);

  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: "Variations", exact: true }).click();
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
});
