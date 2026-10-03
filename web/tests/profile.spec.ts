import { expect, test, type Browser, type Page } from "@playwright/test";

async function signIn(browser: Browser): Promise<Page> {
  const page = await (await browser.newContext()).newPage();
  const res = await page.request.post("/__e2e/invite?org=a");
  expect(res.ok()).toBeTruthy();
  const { token } = (await res.json()) as { token: string };
  await page.goto(`/#token=${token}`);
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  return page;
}

test("thin brief: choose a building type, see typical systems, set one, switch projects", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("188 Balnarring Road");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "188 Balnarring Road" })).toBeVisible();

  // Layout: left nav with the project, profile sections, register beside them.
  await expect(page.getByRole("navigation", { name: "Projects" })).toContainText("188 Balnarring Road");
  for (const name of ["Project", "Systems", "Compliance"]) {
    await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
  }
  await expect(page.getByRole("complementary", { name: "Document register" })).toContainText("Documents");

  // Nothing read yet: compliance says where a value is usually stated.
  await expect(page.getByText("Usually in: Geotechnical report")).toBeVisible();

  await page.getByLabel("Building class", { exact: true }).selectOption("residential");
  await page.getByLabel("Building type", { exact: true }).selectOption("house");
  await page.getByLabel("Work type", { exact: true }).selectOption("new");

  // Typical systems appear, marked as suggestions rather than evidence.
  const water = page.locator(".pf-sys", { hasText: "Heated water" });
  await expect(water).toBeVisible();
  await expect(water.locator('.pf-mark[data-band="suggested"]')).toHaveText("Typical");

  // The user's word: mark it not included; it survives a reload.
  await water.getByText("Not included", { exact: true }).click();
  await expect(water.locator('.pf-mark[data-band="user"]')).toBeVisible();
  await page.reload();
  const again = page.locator(".pf-sys", { hasText: "Heated water" });
  await expect(again.locator('label[data-on]')).toHaveText("Not included");

  // Switch projects from the left nav.
  await page.getByRole("button", { name: "188 Balnarring Road" }).click();
  await page.getByRole("button", { name: "All projects and new project…" }).click();
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
});
