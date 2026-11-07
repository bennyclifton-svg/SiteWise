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
  // Profile (owner, 7 Nov 2026): Update sits in the Project heading with a
  // three-dot menu holding Download and Add part; no top-right Download.
  const more = page.getByRole("button", { name: "Profile actions", exact: true });
  const heading = (await page.getByRole("heading", { name: "Project", exact: true }).boundingBox())!;
  const up = (await update.boundingBox())!;
  const dots = (await more.boundingBox())!;
  expect(Math.abs(up.y + up.height / 2 - (heading.y + heading.height / 2))).toBeLessThan(6);
  expect(dots.x).toBeGreaterThan(up.x + up.width);
  await expect(download).toHaveCount(0);
  await more.click();
  await expect(page.getByRole("menuitem", { name: "Download project profile" })).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Add part" })).toBeVisible();
  await page.keyboard.press("Escape");
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
    if (name !== 'Profile') await expect(download).toBeVisible();
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
  await expect(more).toBeVisible();
  await page.screenshot({ path: '../tmp/panel-actions-mobile.png', fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
});
