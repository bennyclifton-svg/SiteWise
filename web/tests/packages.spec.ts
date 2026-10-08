import { expect, test } from "@playwright/test";

test("packages open on a proposed head contract that follows the procurement route", async ({ page }) => {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Proposed packages");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Packages", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop();
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const work = await page.request.post(`/api/projects/${project}/works`, { headers: { Origin: "http://127.0.0.1:4173" }, data: { part_id: profile.parts[0].id, system_id: "mechanical.air-conditioning", action: "replace", title: "Rooftop chiller" } });
  expect(work.ok()).toBeTruthy();

  await page.getByRole("button", { name: "Packages", exact: true }).click();
  await expect(page.getByText("Procurement route not answered.")).toBeVisible();
  const proposed = page.getByRole("region", { name: "Proposed packages", exact: true });
  await expect(proposed.getByText("Carry out the works: Rooftop chiller", { exact: true })).toBeVisible();
  await expect(proposed.getByText("Design: Rooftop chiller", { exact: true })).toBeVisible();
  const gaps = page.getByRole("region", { name: "Gaps", exact: true });
  await expect(gaps.getByText("Rooftop chiller: nobody designs it")).toBeVisible();

  await proposed.getByRole("button", { name: "Accept Head contract", exact: true }).click();
  await expect(gaps.getByText("Every accepted work item is covered.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Head contract", exact: true })).toBeVisible();
  await expect(page.locator(".report-block .report-wording")).toHaveCount(2);
  const packages = await (await page.request.get(`/api/projects/${project}/packages`)).json();
  expect(packages.items).toHaveLength(1);
  expect(packages.items[0].works_scope).toBe("head_contract");
  expect(packages.proposed_works).toBeNull();

  await page.getByRole("button", { name: "Answer on the profile", exact: true }).click();
  await expect(page.getByLabel("Procurement route", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Packages", exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
});
