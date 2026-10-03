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

test("choose which documents the profile reads, in bulk", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Reading selection");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  const fixtures = resolve(import.meta.dirname, "../../testdata/identity");
  const names = ["identity-page.pdf", "docx-table.docx", "merged-cells.xlsx"];
  await page.getByTestId("file-input").setInputFiles(names.map((n) => resolve(fixtures, n)));
  const rows = page.locator("tbody.reg-doc");
  await expect(rows).toHaveCount(3);
  for (const n of names) await expect(page.locator(`tbody.reg-doc[data-filename="${n}"] .reg-sel input`)).toBeVisible();

  // Click the first checkbox, Shift-click the last: a range of three.
  const boxes = page.locator("tbody.reg-doc .reg-sel input");
  await boxes.nth(0).click();
  await boxes.nth(2).click({ modifiers: ["Shift"] });
  const bulk = page.getByRole("toolbar", { name: "Selected documents" });
  await expect(bulk).toContainText("3 selected");

  await bulk.getByRole("button", { name: "Don't read" }).click();
  for (let i = 0; i < 3; i++) {
    const read = rows.nth(i).locator(".reg-read");
    await expect(read).toHaveAttribute("aria-pressed", "false");
    await expect(read).toHaveAttribute("data-override", "true");
  }
  await expect(page.locator(".profile-reading")).toContainText("Reading 0 of 3 documents");

  // The profile line filters the register to what it does not read.
  await page.locator(".profile-reading").getByRole("button", { name: "Show" }).click();
  await expect(page.locator(".reg-filter")).toBeVisible();
  await expect(rows).toHaveCount(3);
  await page.locator(".reg-filter").getByRole("button", { name: "Show all" }).click();

  // One row's own toggle reads it; Reset returns everything to automatic.
  await rows.nth(0).locator(".reg-read").click();
  await expect(rows.nth(0).locator(".reg-read")).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator(".profile-reading")).toContainText("Reading 1 of 3 documents");
  await bulk.getByRole("button", { name: "Reset to automatic" }).click();
  for (let i = 0; i < 3; i++) await expect(rows.nth(i).locator(".reg-read")).not.toHaveAttribute("data-override", "true");

  // Ctrl-click selects without opening; Clear empties the selection.
  await bulk.getByRole("button", { name: "Clear" }).click();
  await expect(bulk).toBeHidden();
  await rows.nth(1).locator("tr.reg-line").click({ modifiers: ["Control"] });
  await expect(page.getByRole("toolbar", { name: "Selected documents" })).toContainText("1 selected");
  await expect(rows.nth(1)).not.toHaveAttribute("data-open", "true");
});

test("delete documents from the register after confirming", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Deletion");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  const fixtures = resolve(import.meta.dirname, "../../testdata/identity");
  const names = ["identity-page.pdf", "docx-table.docx", "merged-cells.xlsx"];
  await page.getByTestId("file-input").setInputFiles(names.map((n) => resolve(fixtures, n)));
  const rows = page.locator("tbody.reg-doc");
  await expect(rows).toHaveCount(3);
  for (const n of names) await expect(page.locator(`tbody.reg-doc[data-filename="${n}"] .reg-sel input`)).toBeVisible();

  const headerBin = page.getByRole("button", { name: "Delete selected documents" });
  await expect(headerBin).toBeDisabled();
  const boxes = page.locator("tbody.reg-doc .reg-sel input");
  await boxes.nth(0).click();
  await boxes.nth(1).click({ modifiers: ["Shift"] });
  await headerBin.click();

  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("Delete 2 documents, including");
  await expect(dialog).toContainText("can't be undone");
  // Cancel has the focus, so Enter never deletes by accident.
  await expect(dialog.getByRole("button", { name: "Cancel" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(rows).toHaveCount(3);

  await page.getByRole("toolbar", { name: "Selected documents" }).getByRole("button", { name: "Delete" }).click();
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(rows).toHaveCount(1);
  await expect(page.getByRole("toolbar", { name: "Selected documents" })).toBeHidden();

  // A row's own bin deletes just that row; the list stays empty after reload.
  await rows.nth(0).getByRole("button", { name: /^Delete / }).click();
  await expect(dialog).toContainText("Delete ");
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(rows).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  await expect(page.locator("tbody.reg-doc")).toHaveCount(0);
});
