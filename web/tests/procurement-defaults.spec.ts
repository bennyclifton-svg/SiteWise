import { expect, test } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

test("a commercial fit-out saves each service as a row; a removed row stays removed", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Fit-out defaults");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.work_type", "refurb"], ["hdr.building_class", "commercial"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), await r.text()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  for (const name of ["Electrical", "Hydraulic", "Mechanical", "Fire Services", "BCA", "Access", "Certifier"]) {
    await expect(page.getByRole("button", { name: `Remove ${name}`, exact: true })).toBeVisible();
  }
  // Proposed consultants are saved as packages, not shown as suggestions.
  const saved = await (await page.request.get(`/api/projects/${project}/packages`)).json();
  expect(saved.suggestions.filter((s: { kind: string }) => s.kind === "services")).toHaveLength(0);
  await expect(page.getByText("suggested", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("rowheader", { name: /Services Engineer/ })).toHaveCount(0);
  await expect(page.locator(".novation-mark")).toHaveCount(0);

  await page.getByRole("button", { name: "Remove Certifier", exact: true }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Remove", exact: true }).click();
  await expect(page.getByRole("rowheader", { name: /^Certifier/ })).toHaveCount(0);
  await page.reload();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("rowheader", { name: /^Electrical/ })).toBeVisible();
  await expect(page.getByRole("rowheader", { name: /^Certifier/ })).toHaveCount(0);
  // The PM can still add it back by hand.
  await page.getByRole("button", { name: "Add discipline", exact: true }).click();
  await expect(page.getByLabel("Discipline", { exact: true }).locator('option[value="consultant.certifier"]')).toHaveCount(1);
});
