// Program view, read-only chart (plan docs/plans/2026-10-18-program.md
// WP-P7; Playwright steps 1–4, 11 and 15 of section 11.5, and the browser
// budgets of 11.2 for first paint and scroll). The committed library is
// empty-but-valid (WP-P1), so a built program has stages and package
// activities with every duration unknown; tests that need bars give rows
// durations through POST /program/ops, the same path WP-P8's edits use.

import { expect, test, type Page, type Route } from "@playwright/test";
import { compute } from "../src/program/engine";
import { LEFT_ROOM } from "../src/program/fit";
import type { Input } from "../src/program/types";

async function signIn(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
}

/** Creates a project and returns its id once the project page is open. */
async function newProject(page: Page, name: string): Promise<string> {
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page).toHaveURL(/\/projects\/[0-9a-f-]{36}$/);
  await expect(page.getByRole("button", { name: "Program", exact: true })).toBeVisible();
  return new URL(page.url()).pathname.split("/").pop()!;
}

const origin = (page: Page) => ({ Origin: new URL(page.url()).origin });

/** A project with Profile facts and three packages (plan 11.5). */
async function seededProject(page: Page, name: string): Promise<string> {
  await signIn(page);
  const project = await newProject(page, name);
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.work_type", "refurb"], ["hdr.building_class", "commercial"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin(page), data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), await r.text()).toBeTruthy();
  }
  for (const title of ["Architect", "Structural", "Head contract"]) {
    const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin(page), data: { kind: "services", title } });
    expect(r.ok(), await r.text()).toBeTruthy();
  }
  return project;
}

async function openProgram(page: Page) {
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Program", exact: true })).toBeVisible();
}

/** Gives every leaf row a duration and the program a start date, through the ops route. */
async function giveDurations(page: Page, project: string, start = "2027-03-01") {
  const view = await (await page.request.get(`/api/projects/${project}/program`)).json();
  const parents = new Set(view.activities.map((a: { parent_id: string | null }) => a.parent_id).filter(Boolean));
  const ops: unknown[] = view.activities
    .filter((a: { id: string }) => !parents.has(a.id))
    .map((a: { id: string }, i: number) => ({ op: "update", id: a.id, duration_days: 5 + ((i * 7) % 23) }));
  ops.push({ op: "set_start", start_date: start });
  const r = await page.request.post(`/api/projects/${project}/program/ops`, {
    headers: origin(page),
    data: { program_id: view.program.id, version: view.program.version, ops },
  });
  expect(r.ok(), await r.text()).toBeTruthy();
}

const grid = (page: Page) => page.getByRole("treegrid", { name: "Program rows" });
const stageRows = (page: Page) => grid(page).locator('[role="row"][aria-level="1"]');
const activityRows = (page: Page) => grid(page).locator('[role="row"][aria-level="2"]');

test("build, collapse and expand, the question strip and Accept all (steps 1–4)", async ({ page }) => {
  const project = await seededProject(page, "Program steps");
  await openProgram(page);

  // 1. Empty state → Build program → collapsed view with the stages.
  await expect(page.getByText("No program yet.")).toBeVisible();
  await page.getByRole("button", { name: "Build program", exact: true }).click();
  await expect(stageRows(page)).toHaveCount(9);
  await expect(activityRows(page)).toHaveCount(0);
  await expect(page.getByTestId("program-status")).toHaveText(/^Indicative: 0 of \d+ durations known$/);
  await expect(page.getByTestId("program-skeleton")).toContainText("no other program template matches the Profile");
  // Every duration is unknown: no bars, a visible unknown cue on every row, and no finish.
  await expect(page.locator(".pg-layer-bars .pg-bar")).toHaveCount(0);
  await expect(page.locator(".pg-layer-bars .pg-unknown")).toHaveCount(9);
  await expect(page.getByTestId("program-finish")).toHaveText("Not known yet");

  // With durations the header shows a finish date (a range needs reviewed
  // library rows with low and high, WP-P15).
  await giveDurations(page, project);
  await page.reload();
  await openProgram(page);
  await expect(page.getByTestId("program-finish")).toHaveText(/^\d{1,2} [A-Z][a-z]{2} 20\d\d$/);
  await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();

  // 2. Expand one stage, Expand all, Collapse all; reload keeps the view.
  await page.getByRole("button", { name: "Expand Concept design", exact: true }).click();
  await expect(activityRows(page)).toHaveCount(3);
  await expect(grid(page).locator('[role="row"][aria-expanded="true"]')).toHaveCount(1);
  await page.getByRole("button", { name: "Expand all", exact: true }).click();
  await expect(activityRows(page)).toHaveCount(12);
  await page.getByRole("button", { name: "Collapse all", exact: true }).click();
  await expect(activityRows(page)).toHaveCount(0);
  // Keyboard: Ctrl+] expands all, Ctrl+[ collapses all.
  await grid(page).locator('[role="row"][data-id]').first().click();
  await page.keyboard.press("Control+]");
  await expect(activityRows(page)).toHaveCount(12);
  await page.keyboard.press("Control+[");
  await expect(activityRows(page)).toHaveCount(0);
  await page.getByRole("button", { name: "Expand Construction", exact: true }).click();
  await page.reload();
  await openProgram(page);
  await expect(activityRows(page)).toHaveCount(3);
  await expect(page.getByRole("button", { name: "Collapse Construction", exact: true })).toBeVisible();

  // 3. The question strip. This project's work type is known and no other
  // library predicate reads an unknown fact, so no question is asked and
  // nothing blocks the chart. Answering a question is step 3 in
  // program-review.spec.ts, on the e2e server's test library.
  const view = await (await page.request.get(`/api/projects/${project}/program`)).json();
  expect(view.question).toBeNull();
  await expect(page.locator(".pg-q-why")).toHaveCount(0);

  // 4. Accept all → the amber proposal marks clear.
  const proposed = view.activities.filter((a: { review_status: string }) => a.review_status === "proposed").length;
  expect(proposed).toBeGreaterThan(0);
  await expect(page.getByRole("button", { name: `Accept all (${proposed})`, exact: true })).toBeVisible();
  await expect(page.locator(".pg-proposed").first()).toBeVisible();
  await expect(page.locator(".pg-bar.is-proposed").first()).toBeVisible();
  await page.getByRole("button", { name: `Accept all (${proposed})`, exact: true }).click();
  await expect(page.locator(".pg-proposed")).toHaveCount(0);
  await expect(page.locator(".pg-bar.is-proposed")).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Accept all/ })).toHaveCount(0);
});

test("a start date offered in the strip sets the program start", async ({ page }) => {
  const project = await seededProject(page, "Program start");
  await openProgram(page);
  await page.getByRole("button", { name: "Build program", exact: true }).click();
  await expect(page.locator(".pg-relative")).toBeVisible();
  await page.getByLabel("Start date").fill("2027-03-01");
  await page.getByRole("button", { name: "Set start", exact: true }).click();
  await expect(page.locator(".pg-relative")).toHaveCount(0);
  const view = await (await page.request.get(`/api/projects/${project}/program`)).json();
  expect(view.program.start_date).toBe("2027-03-01");
});

test("fit to screen, selection and height (step 11)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  const project = await seededProject(page, "Program fit");
  await openProgram(page);
  await page.getByRole("button", { name: "Build program", exact: true }).click();
  await expect(stageRows(page)).toHaveCount(9);
  await giveDurations(page, project);
  await page.reload();
  await openProgram(page);
  await page.getByRole("button", { name: "Expand all", exact: true }).click();
  const body = page.locator(".pg-body");
  const chartBox = async () => {
    const b = (await body.boundingBox())!;
    const gw = await page.evaluate(() => parseFloat(getComputedStyle(document.querySelector(".pg-body")!).getPropertyValue("--pg-grid-w")));
    return { left: b.x + gw, right: b.x + b.width };
  };

  // Fit to screen: every bar lies inside the visible chart.
  await page.getByRole("button", { name: "Fit to screen", exact: true }).click();
  await page.waitForTimeout(100);
  const c = await chartBox();
  const shapes = page.locator(".pg-layer-bars .pg-bar .pg-shape");
  const n = await shapes.count();
  expect(n).toBeGreaterThan(5);
  for (let i = 0; i < n; i++) {
    const b = (await shapes.nth(i).boundingBox())!;
    expect(b.x).toBeGreaterThanOrEqual(c.left - 1);
    expect(b.x + b.width).toBeLessThanOrEqual(c.right + 1);
  }

  // Fit to selection: the selected activity's bar fills most of the chart width.
  const target = activityRows(page).first();
  await target.click();
  await expect(page.getByText("1 selected")).toBeVisible();
  const id = await target.getAttribute("data-id");
  await page.getByRole("button", { name: "Fit to selection", exact: true }).click();
  await page.waitForTimeout(100);
  const bar = (await page.locator(`.pg-layer-bars [data-id="${id}"] .pg-shape`).boundingBox())!;
  // Most of the width the fit leaves for bars: 48 px free on the right, the start dot's room on the left.
  expect(bar.width).toBeGreaterThan((c.right - c.left - 48 - LEFT_ROOM) * 0.7);
  expect(bar.x).toBeGreaterThanOrEqual(c.left - 1);
  expect(bar.x + bar.width).toBeLessThanOrEqual(c.right + 1);

  // Fit height: rows shrink to fit (20–28 px), or the view says to collapse.
  await page.getByRole("button", { name: "Fit height", exact: true }).click();
  const rows = await grid(page).locator('[role="row"][data-id]').count();
  const viewH = await body.evaluate((el) => el.clientHeight - 44);
  const rowH = await grid(page).locator('[role="row"][data-id]').first().evaluate((el) => el.getBoundingClientRect().height);
  const expected = Math.min(28, Math.max(20, Math.floor(viewH / rows)));
  expect(rowH).toBe(expected);
  if (Math.floor(viewH / rows) < 20) await expect(page.getByText("Too many rows to fit; try collapsing stages")).toBeVisible();
  else await expect(page.getByText("Too many rows to fit; try collapsing stages")).toHaveCount(0);

});

test.describe("touch", () => {
  test.use({ hasTouch: true });
  test("one-finger scroll still scrolls the program (step 11; long-press drag is WP-P8)", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 700 });
    await signIn(page);
    const project = await newProject(page, "Program touch");
    await serveSynthetic(page, project, 10, 10);
    await page.reload();
    await page.getByRole("button", { name: "Program", exact: true }).click();
    await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();
    const body = page.locator(".pg-body");
    await body.evaluate((el) => el.scrollIntoView({ block: "end" }));
    const bb = (await body.boundingBox())!;
    const cdp = await page.context().newCDPSession(page);
    const at = { x: Math.round(bb.x + bb.width * 0.6), y: Math.round(bb.y + bb.height * 0.6) };
    // One finger, real touch events: a vertical swipe over the chart, then a horizontal one.
    const swipe = async (dx: number, dy: number) => {
      await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [at] });
      for (let k = 1; k <= 10; k++) await cdp.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: [{ x: at.x + (dx * k) / 10, y: at.y + (dy * k) / 10 }] });
      await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    };
    await swipe(0, -200);
    await expect.poll(() => body.evaluate((el) => el.scrollTop)).toBeGreaterThan(50);
    // Fitted on open, the chart is as wide as its pane; zoom in so it can move sideways.
    await page.getByRole("button", { name: "Zoom in", exact: true }).click();
    await page.getByRole("button", { name: "Zoom in", exact: true }).click();
    await body.evaluate((el) => (el.scrollLeft = 0));
    await swipe(-200, 0);
    await expect.poll(() => body.evaluate((el) => el.scrollLeft)).toBeGreaterThan(50);
    // The page itself never moved sideways.
    expect(await page.evaluate(() => window.scrollX)).toBe(0);
  });
});

// --- synthetic programs for the browser budgets (plan 11.2) ---------------

/** A program of `stages` stages × `per` activities, chained, with a computed view from the TS engine. */
function syntheticView(project: string, stages: number, per: number, start: string | null = "2027-03-01") {
  const activities: Record<string, unknown>[] = [];
  const links: Record<string, unknown>[] = [];
  const uuid = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, "0")}`;
  let n = 1;
  let prevStage: string | null = null;
  for (let s = 0; s < stages; s++) {
    const sid = uuid(n++);
    activities.push(row(sid, null, s, "stage", `Stage ${s + 1}`, null));
    let prev: string | null = null;
    for (let k = 0; k < per; k++) {
      const id = uuid(n++);
      activities.push(row(id, sid, k, k === per - 1 ? "milestone" : "activity", `Stage ${s + 1} activity ${k + 1}`, k === per - 1 ? 0 : 2 + ((s + k) % 9)));
      if (prev) links.push(lnk(uuid(100000 + n), prev, id, "FS"));
      prev = id;
    }
    if (prevStage) links.push(lnk(uuid(200000 + n), prevStage, sid, s % 5 === 0 ? "SS" : "FS"));
    prevStage = sid;
  }
  const input: Input = {
    calendar: { start: start ?? "", working_week: 31, days: [] },
    activities: activities.map((a) => ({ id: a.id as string, parent: (a.parent_id as string) ?? "", ordinal: a.ordinal as number, kind: a.kind as string, label: a.label as string, duration: a.duration_days as number | null, constraint: "asap" })),
    links: links.map((l) => ({ id: l.id as string, pred: l.pred_id as string, succ: l.succ_id as string, type: l.type as string, lag: 0, lag_unit: "working" })),
  };
  return {
    program: { id: uuid(999999), project_id: project, version: 1, start_date: start, start_origin: null, skeleton_id: "sk.generic", stage_revision: 1, working_week: 31 },
    activities,
    links,
    calendar: { days: [] },
    computed: compute(input),
    checks: [],
    question: null,
    start_suggestion: null,
    skeleton: { id: "sk.generic", label: "Generic stage sequence", reason: "Synthetic program for the browser budget." },
    readings_pending: 0,
    changes_pending: 0,
  };
  function row(id: string, parent: string | null, ordinal: number, kind: string, label: string, d: number | null) {
    return { id, parent_id: parent, ordinal, kind, label, duration_days: d, duration_low: null, duration_high: null, duration_origin: d === null ? null : "user", constraint_kind: "asap", constraint_date: null, review_status: "accepted", origin: "user", version: 1 };
  }
  function lnk(id: string, pred: string, succ: string, type: string) {
    return { id, pred_id: pred, succ_id: succ, type, lag_days: 0, lag_unit: "working", origin: "library", version: 1 };
  }
}

/** Serves a synthetic view for GET /program, all stages expanded. */
async function serveSynthetic(page: Page, project: string, stages: number, per: number) {
  const view = syntheticView(project, stages, per);
  const body = JSON.stringify(view);
  await page.route(`**/api/projects/${project}/program`, (route: Route) =>
    route.request().method() === "GET" ? route.fulfill({ status: 200, contentType: "application/json", body }) : route.continue(),
  );
  const stageIds = view.activities.filter((a) => a.parent_id === null).map((a) => a.id);
  await page.evaluate(([p, ids]) => localStorage.setItem(`sitewise.program.expanded.v1.${p}`, JSON.stringify(ids)), [project, stageIds] as const);
  return view;
}

test("first chart paint ≤ 150 ms after GET returns with 1,000 activities (plan 11.2)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signIn(page);
  const project = await newProject(page, "Program paint");
  await serveSynthetic(page, project, 50, 20); // 1,050 rows, 1,000 activities
  await page.reload();
  await expect(page.getByRole("button", { name: "Program", exact: true })).toBeVisible();
  // Stamp the first frame that shows a bar, after the GET's response ends.
  await page.evaluate(() => {
    const w = window as unknown as { __paint?: number };
    const mo = new MutationObserver(() => {
      if (w.__paint === undefined && document.querySelector(".pg-layer-bars .pg-bar")) {
        mo.disconnect();
        requestAnimationFrame(() => setTimeout(() => (w.__paint = performance.now()), 0));
      }
    });
    mo.observe(document.body, { childList: true, subtree: true });
  });
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();
  await expect.poll(() => page.evaluate(() => (window as unknown as { __paint?: number }).__paint ?? null)).not.toBeNull();
  const ms = await page.evaluate((p) => {
    const get = performance.getEntriesByType("resource").filter((e) => e.name.endsWith(`/api/projects/${p}/program`)).at(-1) as PerformanceResourceTiming;
    return (window as unknown as { __paint: number }).__paint - get.responseEnd;
  }, project);
  console.log(`program first paint after GET, 1,000 activities: ${ms.toFixed(1)} ms`);
  expect(await grid(page).locator('[role="row"][data-id]').count()).toBeLessThan(80); // virtualised
  expect(ms).toBeLessThanOrEqual(150);
});

test("scrolling 2,000 activities has no long task over 50 ms (plan 11.2)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signIn(page);
  const project = await newProject(page, "Program scroll");
  await serveSynthetic(page, project, 100, 20); // 2,100 rows, 2,000 activities
  await page.reload();
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();
  await page.waitForTimeout(300);
  await page.evaluate(() => {
    const w = window as unknown as { __long: number[] };
    w.__long = [];
    new PerformanceObserver((list) => list.getEntries().forEach((e) => w.__long.push(e.duration))).observe({ type: "longtask" });
  });
  const body = page.locator(".pg-body");
  const bb = (await body.boundingBox())!;
  await page.mouse.move(bb.x + bb.width * 0.7, bb.y + bb.height / 2);
  for (let i = 0; i < 60; i++) {
    await page.mouse.wheel(0, 600);
    await page.waitForTimeout(16);
  }
  for (let i = 0; i < 20; i++) {
    await page.mouse.wheel(400, -300);
    await page.waitForTimeout(16);
  }
  await page.waitForTimeout(200);
  const top = await body.evaluate((el) => el.scrollTop);
  expect(top).toBeGreaterThan(10000);
  // The window followed the scroll: rows near the bottom are rendered.
  const first = await grid(page).locator('[role="row"][data-id]').first().getAttribute("aria-rowindex");
  expect(Number(first)).toBeGreaterThan(300);
  const long = await page.evaluate(() => (window as unknown as { __long: number[] }).__long);
  console.log(`program scroll, 2,000 activities: ${long.length} long tasks, max ${Math.max(0, ...long).toFixed(1)} ms`);
  expect(long.filter((d) => d > 50)).toEqual([]);
});

test("print: A3 landscape, every row, the timescale on each page (plan 9.7)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signIn(page);
  const project = await newProject(page, "Program print");
  await serveSynthetic(page, project, 5, 15); // 80 rows
  await page.reload();
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();
  await page.emulateMedia({ media: "print" });
  await page.evaluate(() => window.dispatchEvent(new Event("beforeprint")));
  // No virtualisation while printing: all 80 rows across three pages, each with its header.
  await expect(page.locator(".pg-print-page")).toHaveCount(3);
  await expect(page.locator(".pg-print-page .pg-print-head")).toHaveCount(3);
  await expect(page.locator(".pg-print [role='row'][data-id]")).toHaveCount(80);
  await expect(page.locator(".pg-toolbar")).toBeHidden();
  await expect(page.locator(".pg-summary")).toBeVisible();
  const pdf = await page.pdf({ preferCSSPageSize: true, printBackground: true });
  const boxes = [...pdf.toString("latin1").matchAll(/\/MediaBox\s*\[\s*0 0 ([\d.]+) ([\d.]+)\s*\]/g)].map((m) => [+m[1], +m[2]]);
  expect(boxes.length).toBeGreaterThanOrEqual(3);
  for (const [w, h] of boxes) {
    // A3 landscape is 420 × 297 mm = 1190.55 × 841.89 pt.
    expect(Math.abs(w - 1190.55)).toBeLessThan(2);
    expect(Math.abs(h - 841.89)).toBeLessThan(2);
  }
  await page.evaluate(() => window.dispatchEvent(new Event("afterprint")));
  await page.emulateMedia({ media: "screen" });
  await expect(page.locator(".pg-print")).toHaveCount(0);
});

// --- step 15: screenshots, light and dark, desktop and 390 px ---------------

for (const theme of ["light", "dark"] as const) {
  for (const size of [{ name: "desktop", width: 1440, height: 900 }, { name: "phone", width: 390, height: 844 }]) {
    test(`screenshots ${theme} ${size.name}, collapsed and expanded (step 15)`, async ({ page }) => {
      await page.setViewportSize({ width: size.width, height: size.height });
      await page.addInitScript((t) => {
        try {
          localStorage.setItem("sitewise.colorTheme.v1", t);
        } catch {
          // The theme falls back to dark.
        }
      }, theme);
      await signIn(page);
      const project = await newProject(page, `Program shots ${theme} ${size.name}`);
      const view = syntheticView(project, 6, 4);
      // Fixed ids and dates: the picture depends on the program only.
      await page.route(`**/api/projects/${project}/program`, (route) =>
        route.request().method() === "GET" ? route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(view) }) : route.continue(),
      );
      await page.getByRole("button", { name: "Program", exact: true }).click();
      await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();
      await page.evaluate(() => document.fonts.ready);
      const section = page.locator("section.program");
      if (size.name === "phone") {
        // Name column only; the chart scrolls inside its pane; the page never scrolls sideways.
        await expect(grid(page).locator('[data-col="duration"]')).toHaveCount(0);
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
        expect(overflow).toBeLessThanOrEqual(0);
        // Zoomed in, the chart is wider than its pane and scrolls inside it.
        await page.getByRole("button", { name: "Zoom in", exact: true }).click();
        await page.getByRole("button", { name: "Zoom in", exact: true }).click();
        expect(await page.locator(".pg-body").evaluate((el) => el.scrollWidth > el.clientWidth)).toBe(true);
        expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0);
        await page.getByRole("button", { name: "Fit to screen", exact: true }).click();
      }
      await expect(section).toHaveScreenshot(`program-${theme}-${size.name}-collapsed.png`, { animations: "disabled", maxDiffPixelRatio: 0.01, mask: [page.locator(".program-head .pg-chip")] });
      await page.getByRole("button", { name: "Expand all", exact: true }).click();
      await expect(activityRows(page).first()).toBeVisible();
      await expect(section).toHaveScreenshot(`program-${theme}-${size.name}-expanded.png`, { animations: "disabled", maxDiffPixelRatio: 0.01 });
    });
  }
}
