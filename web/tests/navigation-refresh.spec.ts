import { expect, test } from "@playwright/test";

test("refresh keeps the workbench page and cost tab", async ({ page, context }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Refresh navigation");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const projectURL = page.url();
  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await page.getByRole("tab", { name: /Invoices/ }).click();
  const cdp = await context.newCDPSession(page);
  await cdp.send("Network.setCacheDisabled", { cacheDisabled: true });
  await page.reload();
  await expect(page.getByRole("tab", { name: /Invoices/ })).toHaveAttribute("aria-selected", "true");
  for (const name of [/Variations/, /Cost plan/]) {
    await page.getByRole("tab", { name }).click();
    await page.reload();
    await expect(page.getByRole("tab", { name })).toHaveAttribute("aria-selected", "true");
  }
  for (const name of ["Procurement", "Specifications", "Program", "Profile"]) {
    await page.getByRole("button", { name, exact: true }).click();
    await page.reload();
    await expect(page.getByRole("button", { name, exact: true })).toHaveAttribute("aria-current", "page");
  }
  await page.goto(`${projectURL}?view=unknown&costTab=unknown`);
  await expect(page.getByRole("button", { name: "Profile", exact: true })).toHaveAttribute("aria-current", "page");
  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await expect(page.getByRole("tab", { name: /Cost plan/ })).toHaveAttribute("aria-selected", "true");
});

test("panel actions align with the workspace and documents stay visible across tabs", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Panel alignment");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const update = page.getByRole("button", { name: "Update project profile", exact: true });
  const download = page.getByRole("button", { name: "Download", exact: true });
  const docs = page.getByRole("complementary", { name: "Document register" });
  await expect(update).toBeVisible();
  const main = (await page.locator('.project-main-panel').boundingBox())!;
  const up = (await update.boundingBox())!;
  const down = (await download.boundingBox())!;
  expect(Math.abs(up.x - main.x - 8)).toBeLessThan(2);
  expect(Math.abs(down.x + down.width - (main.x + main.width - 8))).toBeLessThan(2);
  expect(Math.abs(up.y - down.y)).toBeLessThan(4);
  const hide = page.getByRole('button', { name: 'Hide documents', exact: true });
  const hideBox = (await hide.boundingBox())!;
  const docsBox = (await docs.boundingBox())!;
  expect(Math.abs(hideBox.x - docsBox.x - 12)).toBeLessThan(2);
  expect(hideBox.y + hideBox.height).toBeLessThanOrEqual(docsBox.y);
  expect(await update.getAttribute('class')).toBe(await hide.getAttribute('class'));
  await page.screenshot({ path: '../tmp/panel-actions-profile.png', fullPage: true });
  for (const name of ['Costs', 'Procurement', 'Specifications', 'Profile']) {
    await page.getByRole('button', { name, exact: true }).click();
    await expect(docs).toBeVisible();
    await expect(download).toBeVisible();
    if (name === 'Costs') await page.screenshot({ path: '../tmp/panel-actions-costs.png', fullPage: true });
  }
  await hide.click();
  await expect(docs).toBeHidden();
  await page.getByRole('button', { name: 'Costs', exact: true }).click();
  await expect(docs).toBeHidden();
  await page.getByRole('button', { name: 'Show documents', exact: true }).click();
  await expect(docs).toBeVisible();
  await page.getByRole('button', { name: 'Program', exact: true }).click();
  await expect(docs).toBeVisible();
  await page.getByRole('button', { name: 'Profile', exact: true }).click();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(update).toBeVisible();
  await expect(download).toBeVisible();
  await page.screenshot({ path: '../tmp/panel-actions-mobile.png', fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
});
