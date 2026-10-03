import { resolve } from "node:path";
import { expect, test, type Browser, type Page } from "@playwright/test";

test("stopped profile work offers retry despite a queued dependent job", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Interrupted profile regression");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  let active = 0;
  let failed = 1;
  await page.route("**/api/projects/*/profile", async route => {
    const response = await route.fetch();
    const body = await response.json();
    await route.fulfill({ response, json: { ...body, pending_documents: 1, active_documents: active, failed_documents: failed } });
  });
  await page.reload();
  await expect(page.getByRole("button", { name: "Retry project profile", exact: true })).toBeEnabled();
  await expect(page.getByText("Document processing stopped.", { exact: false })).toBeVisible();
  failed = 0;
  await expect(page.getByRole("button", { name: "Profile update queued", exact: true })).toBeVisible();
  active = 1;
  await expect(page.getByRole("button", { name: "Updating project profile…", exact: true })).toBeVisible();
  active = 0;
  failed = 1;
  await expect(page.getByRole("button", { name: "Retry project profile", exact: true })).toBeEnabled();
});

test("profile update clears its pending state when completion SSE is missed", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Profile completion regression");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  let requested = false;
  let completionReads = 0;
  await page.route("**/api/projects/*/profile", async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    if (requested) {
      completionReads++;
      body.pending_documents = 0;
      body.active_documents = 0;
      body.header.find((f: { key: string }) => f.key === "hdr.work_type").value = "new";
    }
    await route.fulfill({ response, json: body });
  });
  await page.route("**/api/projects/*/profile/read", async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    requested = true;
    body.pending_documents = 1;
    body.active_documents = 1;
    body.queued = 1;
    await route.fulfill({ response, json: body });
  });
  await page.getByRole("button", { name: "Update project profile", exact: true }).click();
  await expect(page.getByRole("button", { name: "Updating project profile…", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible({ timeout: 10000 });
  await expect(page.getByLabel("Work type", { exact: true })).toHaveValue("new");
  expect(completionReads).toBeGreaterThan(0);
});

async function signIn(browser: Browser): Promise<Page> {
  const page = await (await browser.newContext()).newPage();
  const res = await page.request.post("/__e2e/invite?org=a");
  expect(res.ok()).toBeTruthy();
  const { token } = (await res.json()) as { token: string };
  await page.goto(`/#token=${token}`);
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  return page;
}

test("source requirements preserve full text, page links, filters and mobile layout", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Source requirements regression");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  let requestedFilter = "";
  await page.route("**/api/projects/*/profile/sources?*", async route => {
    requestedFilter = new URL(route.request().url()).searchParams.get("outcome") ?? "";
    await route.fulfill({ json: { more: false, records: [{
      id: "source-83", document_id: "ea55b0eb-44b7-41e6-957b-3be7fc5058f3", filename: "Bankstown PPR October 2015.pdf",
      ordinal: 1747, text: "The Contractor must provide spares for tiles, pavers and luminaires. All spares must be labelled and delivered to the Principal’s depot.",
      page: 83, location: "Page 83", section: "SPARES", context: "", category: "requirement", provider: "contractor", scope: "whole_project",
      outcome: "needs_mapping", keys: [], unresolved: [], systems: [],
    }] } });
  });
  await page.getByRole("button", { name: "Source requirements · View", exact: true }).click();
  const source = page.locator("#profile-sources");
  await expect(source.getByText("The Contractor must provide spares", { exact: false })).toBeVisible();
  await expect(source.getByRole("link", { name: /Page 83/ })).toHaveAttribute("href", /view=1#page=83$/);
  await source.getByLabel(/^Show/).selectOption("needs_mapping");
  await expect.poll(() => requestedFilter).toBe("needs_mapping");
  await expect(source.getByText("No specific system assigned")).toBeVisible();
  await page.screenshot({ path: resolve(import.meta.dirname, "../../.tools/profile-source-desktop.png"), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(source.getByRole("link", { name: /Page 83/ })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  await page.screenshot({ path: resolve(import.meta.dirname, "../../.tools/profile-source-mobile.png"), fullPage: true });
  await page.context().close();
});

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
