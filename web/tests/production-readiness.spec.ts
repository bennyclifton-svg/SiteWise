import { expect, test } from "@playwright/test";

test("project loading recovers, search is local and sign-out revokes access", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  let fail = true;
  await page.route("**/api/projects", route => route.request().method() === "GET" && fail ? route.fulfill({ status: 503, body: "unavailable" }) : route.continue());
  await page.goto(`/#token=${token}`);
  await expect(page.getByRole("alert")).toContainText("Could not load projects");
  fail = false;
  await page.getByRole("button", { name: "Retry projects" }).click();
  await expect(page.getByText("Could not load projects")).toHaveCount(0);
  await page.getByLabel("New project").fill("Readiness recovery");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\//);
  await page.getByRole("link", { name: "SiteWise, all projects" }).click();
  await page.getByRole("searchbox", { name: "Find a project" }).fill("no-matching-project-xyz");
  await expect(page.getByRole("status")).toContainText("No projects match");
  await page.getByRole("searchbox", { name: "Find a project" }).fill("Readiness recovery");
  await expect(page.getByRole("link", { name: /Readiness recovery/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Delete Readiness recovery", exact: true })).toBeHidden();
  await page.getByText("Actions", { exact: true }).click();
  await page.getByRole("button", { name: "Delete Readiness recovery", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Sign in with your invite" })).toBeVisible();
  expect((await page.request.get("/api/session")).status()).toBe(401);
});

test("narrow shell keeps navigation, controls and tables reachable", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Narrow project workspace");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\//);
  await expect(page.locator(".ask-bar:not(.profile-feedback), .nav-chats, .devstatus")).toHaveCount(0);
  for (const name of ["Specifications", "Costs", "Procurement", "Program", "Profile"]) {
    await page.getByRole("button", { name: "Project navigation", exact: true }).click();
    await page.getByRole("button", { name, exact: true }).click();
    await expect(page.getByRole("button", { name: "Project navigation", exact: true })).toHaveAttribute("aria-expanded", "false");
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
  }
  await page.screenshot({ path: "../tmp/readiness-mobile.png", fullPage: true });
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(page.getByRole("button", { name: "Profile", exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollHeight)).toBeLessThanOrEqual(900);
  await page.screenshot({ path: "../tmp/readiness-desktop.png", fullPage: true });
  // Trace opens in light; the account menu at the foot of the nav picks dark and back.
  await page.getByRole("button", { name: "Account menu", exact: true }).click();
  await page.getByRole("menuitemradio", { name: "Dark", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.getByRole("menuitemradio", { name: "Light", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.keyboard.press("Escape");
  await page.getByRole("link", { name: "Skip to content" }).focus();
  const skip = await page.getByRole("link", { name: "Skip to content" }).boundingBox();
  expect(skip!.width).toBeGreaterThan(100);
  expect(skip!.y).toBeGreaterThanOrEqual(0);
  await page.screenshot({ path: "../tmp/readiness-light.png", fullPage: true });
});

test("a live profile refresh preserves the correction being typed", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Profile draft protection");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\//);
  let reading = "First source client";
  let reads = 0;
  await page.route("**/api/projects/*/profile", async route => {
    const response = await route.fetch();
    const body = await response.json();
    body.details.find((field: { key: string }) => field.key === "proj.client").value = reading;
    body.pending_documents = 1;
    reads++;
    await route.fulfill({ response, json: body });
  });
  await page.reload();
  const client = page.getByRole("textbox", { name: "Client", exact: true });
  await expect(client).toHaveValue("First source client");
  await client.fill("My unfinished correction");
  const before = reads;
  reading = "Later source client";
  await expect.poll(() => reads, { timeout: 15000 }).toBeGreaterThan(before);
  await expect(client).toHaveValue("My unfinished correction");
});
