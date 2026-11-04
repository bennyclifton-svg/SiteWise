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
  await expect(page.getByRole("combobox", { name: "Work type", exact: true }).locator("option:checked")).toHaveText("New build");
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
  await page.getByRole("button", { name: "Source requirements", exact: true }).click();
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
  for (const name of ["Project", "Scope and works"]) {
    await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
  }
  await expect(page.getByRole("complementary", { name: "Document register" })).toContainText("Documents");

  // Nothing read yet: a compliance input, under its system, says where it is usually stated.
  const scope = page.getByRole("region", { name: "Scope and works" });
  await scope.locator(".sw-entry", { hasText: "Substructure" }).click();
  await expect(page.getByText("Usually in: Geotechnical or salinity report")).toBeVisible();

  await page.getByRole("combobox", { name: "Building category", exact: true }).selectOption({ label: "Residential" });
  await page.getByRole("combobox", { name: "Building class", exact: true }).selectOption({ label: "House (Class 1a)" });
  await page.getByRole("combobox", { name: "Work type", exact: true }).selectOption({ label: "New build" });

  // Typical systems are ticked in scope, marked as suggestions rather than evidence.
  await scope.locator(".sw-entry", { hasText: "Hydraulic" }).click();
  const water = scope.locator(".sw-line", { hasText: "Heated water" });
  await expect(water.getByRole("checkbox", { name: "Heated water" })).toBeChecked();
  await expect(water.locator(".pf-scope-origin")).toHaveText("Typical");
  await expect(water.getByLabel("Who provides Heated water")).toHaveValue("");
  await water.getByRole("button", { name: "Details for Heated water" }).click();
  await expect(water.locator(".sw-strip")).toBeVisible();

  // The user's word: take it out of scope; it survives a reload.
  await water.getByRole("checkbox", { name: "Heated water" }).uncheck();
  await expect(water.locator(".pf-scope-origin")).toHaveText("Removed by you");
  await page.reload();
  const again = page.getByRole("region", { name: "Scope and works" });
  await again.locator(".sw-entry", { hasText: "Hydraulic" }).click();
  await expect(again.getByRole("checkbox", { name: "Heated water" })).not.toBeChecked();

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
  for (const n of names) await expect(page.locator(`tbody.reg-doc[data-filename="${n}"]`)).toHaveAttribute("id", /^doc-/);

  // Click the first row, Shift-click the last: a range of three. The row
  // menu (right-click or ⋮) acts on the whole selection.
  const lines = page.locator("tbody.reg-doc .reg-no");
  await lines.nth(0).click();
  await lines.nth(2).click({ modifiers: ["Shift"] });
  await expect(page.locator("tbody.reg-doc[data-selected]")).toHaveCount(3);
  const menu = page.getByRole("menu");
  const act = async (row: number, action: string, how: "right" | "more" = "right") => {
    if (how === "right") await lines.nth(row).click({ button: "right" });
    else await rows.nth(row).getByRole("button", { name: /^More actions for / }).click();
    await menu.getByRole("menuitem", { name: action, exact: true }).click();
  };
  await lines.nth(1).click({ button: "right" });
  await expect(menu).toContainText("3 documents selected");
  await menu.getByRole("menuitem", { name: "Exclude from profile", exact: true }).click();
  // Not-read rows are marked by a struck-through eye beside the title.
  for (let i = 0; i < 3; i++) {
    await expect(rows.nth(i)).toHaveAttribute("data-excluded", "true");
    await expect(rows.nth(i).locator(".reg-unread")).toHaveAttribute("title", /your choice/);
  }
  await expect(page.locator(".profile-reading")).toContainText("Included for profile: 0 of 3 documents");

  // The profile line filters the register to what it does not read.
  await page.locator(".profile-reading").getByRole("button", { name: "Show" }).click();
  await expect(page.locator(".reg-filter")).toBeVisible();
  await expect(rows).toHaveCount(3);
  await page.locator(".reg-filter").getByRole("button", { name: "Show all" }).click();

  // A menu on an unselected row acts on that row alone; Reset returns the
  // whole selection to automatic.
  await page.locator("tbody.reg-doc .reg-no").nth(0).click();
  await act(0, "Include in profile", "more");
  await expect(rows.nth(0)).not.toHaveAttribute("data-excluded", "true");
  await expect(page.locator(".profile-reading")).toContainText("Included for profile: 1 of 3 documents");
  await lines.nth(0).click();
  await lines.nth(2).click({ modifiers: ["Shift"] });
  await act(2, "Reset profile reading to automatic");
  for (let i = 0; i < 3; i++) await expect(rows.nth(i).locator('.reg-unread[title*="your choice"]')).toHaveCount(0);

  // Escape clears the selection; Ctrl-click selects without opening.
  await lines.nth(0).press("Escape");
  await expect(page.locator("tbody.reg-doc[data-selected]")).toHaveCount(0);
  await rows.nth(1).locator("tr.reg-line").click({ modifiers: ["Control"] });
  await expect(page.locator("tbody.reg-doc[data-selected]")).toHaveCount(1);
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
  for (const n of names) await expect(page.locator(`tbody.reg-doc[data-filename="${n}"]`)).toHaveAttribute("id", /^doc-/);

  const lines = page.locator("tbody.reg-doc .reg-no");
  const menu = page.getByRole("menu");
  await lines.nth(0).click();
  await lines.nth(1).click({ modifiers: ["Shift"] });
  await lines.nth(1).click({ button: "right" });
  await menu.getByRole("menuitem", { name: "Delete 2 documents…", exact: true }).click();

  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("Delete 2 documents, including");
  await expect(dialog).toContainText("can't be undone");
  // Cancel has the focus, so Enter never deletes by accident.
  await expect(dialog.getByRole("button", { name: "Cancel" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(rows).toHaveCount(3);

  await lines.nth(0).click({ button: "right" });
  await menu.getByRole("menuitem", { name: "Delete 2 documents…", exact: true }).click();
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(rows).toHaveCount(1);
  await expect(page.locator("tbody.reg-doc[data-selected]")).toHaveCount(0);

  // The ⋮ menu deletes just its row; the list stays empty after reload.
  await rows.nth(0).getByRole("button", { name: /^More actions for / }).click();
  await menu.getByRole("menuitem", { name: "Delete…", exact: true }).click();
  await expect(dialog).toContainText("Delete ");
  await dialog.getByRole("button", { name: "Delete" }).click();
  await expect(rows).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  await expect(page.locator("tbody.reg-doc")).toHaveCount(0);
});

test("scope of works: a sprinkler pump replacement sees only what it needs", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Sprinkler pump replacement");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("heading", { name: "Scope and works" })).toBeVisible();

  await page.getByRole("combobox", { name: "Building category", exact: true }).selectOption({ label: "Industrial" });
  { const cls = page.getByRole("combobox", { name: "Building class", exact: true }); await cls.selectOption({ label: (await cls.locator("option", { hasText: /^Warehouse/ }).first().textContent())!.trim() }); }
  await page.getByRole("combobox", { name: "Work type", exact: true }).selectOption({ label: "Refurbishment / fit-out / upgrade" });
  const scope = page.getByRole("region", { name: "Scope and works" });
  await expect(scope).toContainText("0 systems");
  // Nothing in scope yet: no basis to hide a compliance input.
  await scope.locator(".sw-entry", { hasText: "Substructure" }).click();
  await expect(page.getByText("Usually in: Geotechnical or salinity report")).toBeVisible();

  await scope.locator(".sw-entry", { hasText: "Active fire protection" }).click();
  await scope.getByRole("checkbox", { name: "Automatic fire sprinklers" }).check();
  await expect(scope).toContainText("1 system");
  const sprinklers = scope.locator(".sw-line", { hasText: "Automatic fire sprinklers" });
  await expect(sprinklers.locator(".pf-scope-origin")).toHaveText("You");

  // Who provides it defaults to the contractor, shown but not stored.
  await expect(sprinklers.getByLabel("Who provides Automatic fire sprinklers").locator("option:checked")).toHaveText("Contractor (default)");
  await sprinklers.getByRole("button", { name: "Details for Automatic fire sprinklers" }).click();
  await expect(sprinklers.locator(".sw-strip")).toBeVisible();
  // Compliance inputs follow the scope and sit under their system.
  await expect(page.getByLabel("Top of storage height")).toBeVisible();
  await scope.locator(".sw-entry", { hasText: "Substructure" }).click();
  await expect(scope).toContainText(/\d+ not needed for this scope/);
  await expect(page.getByText("Usually in: Geotechnical or salinity report")).toBeHidden();
  await page.getByLabel("Show inputs this scope does not need").check();
  await expect(page.getByText("Usually in: Geotechnical or salinity report")).toBeVisible();
  await page.getByLabel("Show inputs this scope does not need").uncheck();

  // A preset adds a set at once; Reset hands every choice back.
  await scope.getByRole("button", { name: "Typical fit-out" }).click();
  await expect(scope).not.toContainText("1 system");
  await scope.getByRole("button", { name: "Reset to defaults" }).click();
  await expect(scope).toContainText("0 systems");
});

test("site and project details are answered once in Profile", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Project details in Profile");
  await page.getByRole("button", { name: "Create project" }).click();
  const details = page.locator(".pf-tiers");
  await expect(details).toBeVisible();
  await details.getByLabel("Zone").fill("R2");
  await details.getByLabel("Zone").blur();
  await details.getByLabel("Site address").fill("1 Crystal Street, Petersham");
  await details.getByLabel("Site address").blur();
  // Zone and the typed address each carry a Reset.
  await expect(details.getByRole("button", { name: "Reset" })).toHaveCount(2);
  // A site risk sits under the system it affects.
  await page.locator(".sw-entry", { hasText: "Substructure" }).click();
  await page.getByLabel("Basement or deep excavation").selectOption("yes");
  await page.reload();
  await expect(page.locator(".pf-tiers").getByLabel("Zone")).toHaveValue("R2");
  await expect(page.getByLabel("Site address")).toHaveValue("1 Crystal Street, Petersham");
  await page.locator(".sw-entry", { hasText: "Substructure" }).click();
  await expect(page.getByLabel("Basement or deep excavation")).toHaveValue("yes");
});

test("a site address read from documents shows until the user types one", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Address read from documents");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  // The profile as the server returns it once a document states the address.
  await page.route("**/api/projects/*/profile", async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    const address = body.site_details.find((f: { key: string }) => f.key === "site.address");
    Object.assign(address, {
      value: "568-572 Parramatta Road Petersham",
      band: "amber",
      origin: "document",
      sources: [{ document_id: "d1", passage_id: "p1", filename: "PPR.pdf", excerpt: "568-572 Parramatta Road Petersham", page: 1 }],
    });
    await route.fulfill({ response, json: body });
  });
  await page.reload();
  const details = page.locator(".pf-tiers");
  const address = details.getByLabel("Site address");
  await expect(address).toHaveValue("568-572 Parramatta Road Petersham");
  await expect(details.getByRole("button", { name: "Reset" })).toHaveCount(0);
  await expect(details.locator(".pf-row").filter({ hasText: "Site address" })).toHaveAttribute("data-band", "amber");

  // The user's word wins and survives a reload; Reset returns to the reading.
  await address.fill("1 Crystal Street, Petersham");
  await address.blur();
  await expect(details.getByRole("button", { name: "Reset" })).toHaveCount(1);
  await page.reload();
  await expect(details.getByLabel("Site address")).toHaveValue("1 Crystal Street, Petersham");
  await details.getByRole("button", { name: "Reset" }).click();
  await expect(details.getByLabel("Site address")).toHaveValue("568-572 Parramatta Road Petersham");
});

test("profile distinguishes inclusion, queued reading, progress and completion", async ({ browser }) => {
  const page = await signIn(browser);
  await page.getByLabel("New project").fill("Reading progress regression");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeVisible();
  let counts = { read_documents: 3, skipped_documents: 1, completed_documents: 0, unread_documents: 3, pending_documents: 0, active_documents: 0, failed_documents: 0 };
  await page.route("**/api/projects/*/profile", async route => {
    const response = await route.fetch();
    await route.fulfill({ response, json: { ...await response.json(), ...counts } });
  });
  await page.reload();
  await expect(page.locator(".profile-reading")).toContainText("Included for profile: 3 of 4 documents");
  const updates = page.getByRole("region", { name: "Profile updates" });
  await expect(updates.locator(".profile-reading")).toContainText("Included for profile: 3 of 4 documents");
  await expect(page.locator(".cockpit-scroll .pf-header")).toHaveCount(0);
  await expect(page.getByPlaceholder("Ask about your project documents")).toHaveCount(0);
  expect(await updates.evaluate(element => getComputedStyle(element).overflowY)).toBe("auto");
  await expect(page.getByRole("status").filter({ hasText: "Ready to read" })).toHaveText("Ready to read · 0 of 3 documents read");
  await expect(page.getByText("3 documents ready to read. Select Update project profile to start.")).toBeVisible();
  const progress = page.getByRole("progressbar", { name: "Documents read for profile" });
  await expect(progress).toHaveAttribute("aria-valuenow", "0");
  counts = { ...counts, unread_documents: 0, pending_documents: 3 };
  await page.reload();
  await expect(page.getByRole("button", { name: "Profile update queued", exact: true })).toBeEnabled();
  await expect(page.locator(".profile-progress")).toContainText("Reading queued · 3 queued · 0 of 3 documents read");
  counts = { ...counts, completed_documents: 1, pending_documents: 2, active_documents: 1 };
  await expect(progress).toHaveAttribute("aria-valuenow", "1");
  await expect(page.locator(".profile-progress")).toContainText("Reading documents · 1 processing · 1 queued · 1 of 3 documents read");
  await page.screenshot({ path: resolve(import.meta.dirname, "../../tmp/profile-progress-desktop.png") });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  await page.screenshot({ path: resolve(import.meta.dirname, "../../tmp/profile-progress-mobile.png") });
  counts = { ...counts, failed_documents: 1, active_documents: 0 };
  await expect(page.locator(".profile-progress")).toContainText("Reading needs attention · 2 queued · 1 of 3 documents read");
  await expect(page.getByRole("button", { name: "Retry project profile", exact: true })).toBeEnabled();
  counts = { ...counts, completed_documents: 3, pending_documents: 0, failed_documents: 0 };
  await expect(progress).toHaveAttribute("aria-valuenow", "3");
  await expect(page.locator(".profile-progress")).toContainText("Reading complete · 3 of 3 documents read");
  await expect(page.getByText("not read yet", { exact: false })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Update project profile", exact: true })).toBeEnabled();
  await updates.scrollIntoViewIfNeeded();
  const bounds = await updates.boundingBox();
  expect(bounds!.height).toBeLessThanOrEqual(144);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390);
  await page.setViewportSize({ width: 1280, height: 720 });
  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await expect(updates).toBeHidden();
  await expect(page.getByPlaceholder("Ask about your project documents")).toHaveCount(0);
  await page.getByRole("button", { name: "Profile", exact: true }).click();
  await expect(updates).toContainText("Reading complete");
  await page.context().close();
});
