import { expect, test, type Browser, type Page, type Route } from "@playwright/test";

// Run-time feedback on the Profile (docs/design/2026-11-10-run-time-feedback.md).
// The update is played through the real page with the server's replies
// shaped per phase: progress, finds and revisions are what the strip and the
// marks follow, so nothing here depends on a live reading.

async function signIn(browser: Browser): Promise<Page> {
  const page = await (await browser.newContext()).newPage();
  const res = await page.request.post("/__e2e/invite?org=a");
  expect(res.ok()).toBeTruthy();
  const { token } = (await res.json()) as { token: string };
  await page.goto(`/#token=${token}`);
  await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();
  return page;
}

type Field = { key: string; value: string; band: string; sources: unknown[] };

/** Shapes the server's replies for one simulated update: 0 rest, 1 reading, 2 label pass committed, 3 done. */
async function simulateUpdate(page: Page) {
  const run = { phase: 0, base: 0 };
  const doc = "00000000-0000-4000-8000-0000000000d1";
  const fulfil = async (route: Route, shape: (body: Record<string, unknown>) => void) => {
    const response = await route.fetch();
    const body = await response.json();
    shape(body);
    await route.fulfill({ response, json: body });
  };
  await page.route("**/api/projects/*/profile", (route) => fulfil(route, (body) => {
    run.base ||= (body.revision as number) ?? 0;
    if (run.phase === 0) return;
    Object.assign(body, { read_documents: 1, pending_documents: run.phase < 3 ? 1 : 0, active_documents: run.phase < 3 ? 1 : 0, completed_documents: run.phase === 3 ? 1 : 0, revision: run.base + (run.phase >= 2 ? 1 : 0) });
    if (run.phase >= 2) {
      const set = (list: Field[], key: string, value: string) => Object.assign(list.find((f) => f.key === key)!, { value, band: "amber", sources: [{ document_id: doc, excerpt: value }] });
      set(body.header as Field[], "hdr.work_type", "new");
      set(body.details as Field[], "proj.zone", "MU1 Mixed Use");
    }
  }));
  await page.route("**/api/projects/*/profile/read", (route) => fulfil(route, (body) => {
    run.phase = 1;
    Object.assign(body, { read_documents: 1, pending_documents: 1, active_documents: 1, queued: 1 });
  }));
  await page.route("**/api/projects/*/profile/progress", (route) => route.fulfill({ json: {
    revision: run.base + (run.phase >= 2 ? 1 : 0), preparing: 0,
    documents: run.phase === 1 ? [{ document_id: doc, filename: "design-statement.pdf", state: "reading", pass: "label", units: 96, labelled: 41, evidence: 0 }]
      : run.phase === 2 ? [{ document_id: doc, filename: "design-statement.pdf", state: "reading", pass: "evidence", units: 96, labelled: 96, evidence: 12 }] : [],
  } }));
  await page.route("**/api/projects/*/profile/finds*", (route) => route.fulfill({ json: { finds: run.phase === 1 ? [
    { passage_id: "p1", document_id: doc, stage: "label", page: 4, keys: ["proj.zone"], systems: [], excerpt: "The site is zoned MU1 Mixed Use under the LEP", at: new Date().toISOString() },
  ] : [] } }));
  await page.route("**/api/projects/*/profile/next", (route) => route.fulfill({ json: { procurement: 2, costs: 3, program: 1, drafts: 0 } }));
  return run;
}

test("an update shows its reading, marks what it changed and says what it did", async ({ browser }) => {
  const page = await signIn(browser);
  await page.setViewportSize({ width: 1400, height: 900 });
  await page.getByLabel("New project").fill("Run feedback");
  await page.getByRole("button", { name: "Create project" }).click();
  const update = page.getByRole("button", { name: "Update", exact: true });
  await expect(update).toBeVisible();
  // Update sits in the title band, not on the sheet.
  await expect(page.locator(".cockpit-shell-header .header-page-actions").getByRole("button", { name: "Update", exact: true })).toBeVisible();
  const run = await simulateUpdate(page);

  await update.click();
  const strip = page.getByRole("region", { name: "Profile updates" });
  await expect(strip.locator(".pf-strip")).toHaveAttribute("data-mode", "run");
  await expect(strip.locator(".run-doc")).toHaveCount(1);
  await expect(strip.locator(".run-now")).toContainText("label pass · 41 of 96 passages");
  // The passage that spoke to Zone: named in the strip and counted on its row, not yet applied.
  await expect(strip.locator(".run-find")).toContainText("Zone");
  await expect(page.locator('.pf-row[data-key="proj.zone"] .pend-count')).toHaveText("+1");

  // The label pass commits: the values change, each with a revision mark saying what it was.
  run.phase = 2;
  const zone = page.locator('.pf-titleblock .pf-row[data-key="proj.zone"]');
  await expect(zone).toHaveAttribute("data-changed", "true");
  await expect(zone.locator(".rev-mark")).toHaveAttribute("title", "Read in an update: MU1 Mixed Use");
  await expect(page.locator('.pf-row[data-key="hdr.work_type"] .rev-mark')).toHaveAttribute("title", /Read in an update: New build/);
  await expect(page.locator('.pf-row[data-key="proj.zone"] .pend-count')).toHaveCount(0);
  await expect(page.locator(".pf-t.pencil-jev", { hasText: "MU1 Mixed Use" })).toBeVisible();

  // Done: the receipt counts the changes, and the views it affects carry counts.
  run.phase = 3;
  await expect(strip.locator(".pf-strip")).toHaveAttribute("data-mode", "done");
  await expect(strip).toContainText("Update finished.");
  await expect(strip.getByRole("button", { name: /2 filled/ })).toBeVisible();
  await expect(strip.getByRole("button", { name: /Cost Plan 3 lines proposed/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Costs", exact: true }).locator(".nav-count")).toHaveText("3");
  await expect(page.getByRole("button", { name: "Procurement", exact: true }).locator(".nav-count")).toHaveText("2");
  await strip.getByRole("button", { name: /2 filled/ }).click();
  await expect(page.locator(".ink-flash").first()).toBeAttached();

  // Mark all seen clears the marks, and this browser remembers it.
  await strip.getByRole("button", { name: "Mark all seen" }).click();
  await expect(page.locator(".rev-mark")).toHaveCount(0);
  await page.reload();
  await expect(page.locator('.pf-row[data-key="proj.zone"]')).toBeVisible();
  await expect(page.locator(".rev-mark")).toHaveCount(0);
  await page.context().close();
});

test("with reduced motion an update marks its changes without moving them", async ({ browser }) => {
  const page = await signIn(browser);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.getByLabel("New project").fill("Run feedback still");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.getByRole("button", { name: "Update", exact: true })).toBeVisible();
  const run = await simulateUpdate(page);
  await page.getByRole("button", { name: "Update", exact: true }).click();
  await expect(page.getByRole("region", { name: "Profile updates" }).locator(".pf-strip")).toHaveAttribute("data-mode", "run");
  run.phase = 2;
  await expect(page.locator('.pf-titleblock .pf-row[data-key="proj.zone"] .rev-mark')).toBeVisible();
  await expect(page.locator(".ink-write, .ink-glint, .ink-on, .ink-off")).toHaveCount(0);
  await page.context().close();
});

test("the scope's first rows are on the first screen of a laptop", async ({ browser }) => {
  const page = await signIn(browser);
  await page.setViewportSize({ width: 1536, height: 864 });
  await page.getByLabel("New project").fill("First screen");
  await page.getByRole("button", { name: "Create project" }).click();
  await expect(page.locator(".sw-entry").first()).toBeVisible();
  await page.getByRole("combobox", { name: "Building category", exact: true }).selectOption({ label: "Residential" });
  const visible = await page.evaluate(() => {
    const scroller = document.querySelector(".cockpit-scroll")!.getBoundingClientRect();
    const bottom = Math.min(scroller.bottom, innerHeight);
    return [...document.querySelectorAll(".sw-entry")].filter((e) => {
      const r = e.getBoundingClientRect();
      return r.top >= scroller.top && r.bottom <= bottom;
    }).length;
  });
  expect(visible).toBeGreaterThanOrEqual(5);
  // Each card value is one line: label, value, then how it was decided.
  const heights = await page.locator(".pf-titleblock .pf-row").evaluateAll((rows) => rows.map((r) => Math.round(r.getBoundingClientRect().height)));
  expect(Math.max(...heights)).toBeLessThanOrEqual(31);
  await page.context().close();
});
