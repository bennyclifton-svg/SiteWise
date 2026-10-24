import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

// The construction budget is typed in the cost plan's totals, saved to the
// profile, and spread over the construction lines by the profile's split
// (docs/plans/2026-10-21-source-to-cost-plan-design.md, Part 2).
test("a typed construction budget is spread over the construction lines", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Budget spread");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Costs", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.building_class", "residential"], ["hdr.subclass", "house"], ["hdr.work_type", "new"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), await r.text()).toBeTruthy();
  }
  for (const system of ["envelope", "interiors", "substructure"]) {
    const r = await page.request.post(`/api/projects/${project}/works`, { headers: origin, data: { part_id: profile.parts[0].id, system_id: system, action: "new", title: system } });
    expect(r.ok(), await r.text()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Costs", exact: true }).click();
  const budget = page.getByRole("button", { name: "Construction budget", exact: true });
  await expect(page.locator(".cost-budget-note")).toContainText("Not set.");
  await budget.click();
  await page.getByLabel("Construction budget", { exact: true }).fill("1,000,000");
  await page.getByLabel("Construction budget", { exact: true }).press("Enter");
  await expect(page.locator(".cost-budget-note")).toContainText("Fully allocated");
  await expect(page.locator(".cost-budget-note")).toContainText("New house split");
  await expect(page.getByRole("button", { name: "Budget for Building envelope", exact: true })).toHaveClass(/cost-spread/);
  await expect(page.getByRole("row", { name: /^Construction subtotal/ })).toContainText("1,000,000.00");
  // The profile holds the same figure.
  const after = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  expect(JSON.stringify(after)).toContain("$1000000");
});
