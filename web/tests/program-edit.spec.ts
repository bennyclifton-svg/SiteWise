// Program editing and bulk selection (plan docs/plans/2026-10-18-program.md
// WP-P8): Playwright steps 5–10 and 14 of section 11.5, the edge cases of
// 13.1 that a browser shows (held drags, bulk moves with a held row, cycles,
// two tiers, deleting a stage, a milestone given days, 409), the touch long
// press, and the drag-frame budget of 11.2 at 2,000 activities with 20 bars
// selected.

import { expect, test, type Page, type Route } from "@playwright/test";
import { compute } from "../src/program/engine";
import type { Input } from "../src/program/types";

type Act = { id: string; parent_id: string | null; kind: string; label: string; duration_days: number | null };
type Lnk = { id: string; pred_id: string; succ_id: string; type: string; lag_days: number; lag_unit: string; origin: string };
type View = { program: { id: string; version: number }; activities: Act[]; links: Lnk[]; computed: { activities: Record<string, { es: number | null; ef: number | null }> } };

async function signIn(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
}

async function newProject(page: Page, name: string): Promise<string> {
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page).toHaveURL(/\/projects\/[0-9a-f-]{36}$/);
  return new URL(page.url()).pathname.split("/").pop()!;
}

const origin = (page: Page) => ({ Origin: new URL(page.url()).origin });

/** A built program with Profile facts, three packages, durations and a start date. */
async function builtProgram(page: Page, name: string): Promise<string> {
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
  let r = await page.request.post(`/api/projects/${project}/program/prepare`, { headers: origin(page), data: { surface: "program" } });
  expect(r.ok(), await r.text()).toBeTruthy();
  const view = (await r.json()) as View;
  const parents = new Set(view.activities.map((a) => a.parent_id).filter(Boolean));
  const ops: unknown[] = view.activities.filter((a) => !parents.has(a.id)).map((a, i) => ({ op: "update", id: a.id, duration_days: 5 + ((i * 7) % 23) }));
  ops.push({ op: "set_start", start_date: "2027-03-01" });
  r = await page.request.post(`/api/projects/${project}/program/ops`, { headers: origin(page), data: { program_id: view.program.id, version: view.program.version, ops } });
  expect(r.ok(), await r.text()).toBeTruthy();
  await page.setViewportSize({ width: 1440, height: 1000 });
  await openProgram(page);
  return project;
}

async function openProgram(page: Page) {
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Program", exact: true })).toBeVisible();
  await expect(page.locator(".pg-layer-bars .pg-bar").first()).toBeVisible();
  await settled(page);
}

const read = async (page: Page, project: string): Promise<View> => (await page.request.get(`/api/projects/${project}/program`)).json();
const grid = (page: Page) => page.getByRole("treegrid", { name: "Program rows" });
const row = (page: Page, id: string) => grid(page).locator(`[role="row"][data-id="${id}"]`);
const bar = (page: Page, id: string) => page.locator(`.pg-layer-bars [data-id="${id}"] .pg-shape`);
const toast = (page: Page) => page.locator(".pg-toast");
const opsDone = (page: Page) => page.waitForResponse((r) => /\/program\/(ops|undo|redo)$/.test(r.url()) && r.request().method() === "POST");
const esc = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const kidsOf = (v: View, stage: string) => v.activities.filter((a) => a.parent_id === stage);

/** Two animation frames: a fit, zoom or scroll has rendered. */
const settled = (page: Page) => page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(() => r(null)))));

/** An element's centre once its box stops moving (fits and zooms apply a frame later). */
async function centre(page: Page, sel: ReturnType<Page["locator"]>) {
  // Inside the program pane, centre the target in the chart's visible part:
  // the sticky timescale, the sticky grid and the page's fixed ask bar would
  // otherwise sit over a row scrolled just into view.
  await sel.evaluate((el) => {
    const body = el.closest(".pg-body") as HTMLElement | null;
    if (!body) return el.scrollIntoView({ block: "center" });
    const r = el.getBoundingClientRect();
    const b = body.getBoundingClientRect();
    const gw = el.closest(".pg-chartcol") ? parseFloat(getComputedStyle(body).getPropertyValue("--pg-grid-w")) || 0 : 0;
    body.scrollTop += r.top + r.height / 2 - (b.top + 44 + (Math.min(b.height, innerHeight - b.top - 120) - 44) / 2);
    if (gw) {
      const left = b.left + gw;
      const width = b.width - gw;
      if (r.left < left + 20 || r.right > left + width - 20) body.scrollLeft += r.left + r.width / 2 - (left + width / 2);
    }
  });
  let b = (await sel.boundingBox())!;
  for (let i = 0; i < 20; i++) {
    await settled(page);
    const n = (await sel.boundingBox())!;
    if (n.x === b.x && n.y === b.y && n.width === b.width) break;
    b = n;
  }
  return { x: b.x + b.width / 2, y: b.y + b.height / 2, b };
}

async function dragMouse(page: Page, from: { x: number; y: number }, to: { x: number; y: number }, steps = 12) {
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps });
  await page.mouse.up();
}

test("rename by click, F2 and Tab through names (step 5)", async ({ page }) => {
  const project = await builtProgram(page, "Edit names");
  await page.getByRole("button", { name: "Expand all", exact: true }).click();
  let v = await read(page, project);
  const stage = v.activities.find((a) => !a.parent_id && kidsOf(v, a.id).length)!;
  const [a1, a2, a3] = kidsOf(v, stage.id);

  // A click on the name of the selected row renames it.
  await row(page, stage.id).locator(".pg-label").click();
  await row(page, stage.id).locator(".pg-label").click();
  const stageName = page.getByRole("textbox", { name: `Name of ${stage.label}` });
  await stageName.fill("Concept stage");
  let done = opsDone(page);
  await stageName.press("Enter");
  await done;
  await expect(row(page, stage.id).locator(".pg-label")).toHaveText("Concept stage");

  // F2 on a focused name cell.
  await row(page, a1.id).locator('[data-col="name"]').click();
  await page.keyboard.press("F2");
  await page.getByRole("textbox", { name: `Name of ${a1.label}` }).fill("First task");
  done = opsDone(page);
  await page.keyboard.press("Enter");
  await done;

  // Double-click, then Tab saves and opens the next name.
  await row(page, a2.id).locator(".pg-label").dblclick();
  await page.getByRole("textbox", { name: `Name of ${a2.label}` }).fill("Second task");
  done = opsDone(page);
  await page.keyboard.press("Tab");
  await done;
  const third = page.getByRole("textbox", { name: `Name of ${a3.label}` });
  await expect(third).toBeFocused();
  await third.fill("Third task");
  done = opsDone(page);
  await page.keyboard.press("Tab");
  await done;
  // Escape cancels the next one.
  await page.keyboard.press("Escape");
  await expect(page.locator(".pg-edit")).toHaveCount(0);

  // An empty name keeps the cell open with a message.
  await row(page, a1.id).locator(".pg-label").dblclick();
  await page.getByRole("textbox", { name: "Name of First task" }).fill("   ");
  await page.keyboard.press("Enter");
  await expect(row(page, a1.id).locator(".pg-cell-error")).toHaveText("Names are 1 to 200 characters.");
  await page.keyboard.press("Escape");

  v = await read(page, project);
  const label = (id: string) => v.activities.find((a) => a.id === id)!.label;
  expect([label(stage.id), label(a1.id), label(a2.id), label(a3.id)]).toEqual(["Concept stage", "First task", "Second task", "Third task"]);
});

test("select with Ctrl and Shift, drag together, Delete and Undo (step 6)", async ({ page }) => {
  const project = await builtProgram(page, "Edit bulk");
  await page.getByRole("button", { name: "Expand all", exact: true }).click();
  const v0 = await read(page, project);
  const acts = grid(page).locator('[role="row"][aria-level="2"]');
  const ids = await acts.evaluateAll((els) => els.map((e) => e.getAttribute("data-id")!));
  await acts.nth(0).click();
  await acts.nth(4).click({ modifiers: ["Control"] });
  await acts.nth(8).click({ modifiers: ["Control"] });
  await expect(page.getByText("3 selected")).toBeVisible();
  // Shift-click a range from the anchor (the last Ctrl-clicked row) to the
  // activity next to it, keeping the Ctrl-clicked rows.
  const all = await grid(page).locator('[role="row"][data-id]').evaluateAll((els) => els.map((e) => [e.getAttribute("data-id")!, e.getAttribute("aria-level")!]));
  const at = all.findIndex(([id]) => id === ids[8]);
  const next = all[at + 1]?.[1] === "2" ? all[at + 1][0] : all[at - 1][0];
  await row(page, next).click({ modifiers: ["Shift"] });
  await expect(page.getByText("4 selected")).toBeVisible();
  await expect(grid(page).locator('[role="row"][aria-selected="true"]')).toHaveCount(4);
  const chosen = [ids[0], ids[4], ids[8], next];
  for (const id of chosen) await expect(page.locator(`.pg-layer-bars [data-id="${id}"]`)).toHaveClass(/is-selected/);

  // Drag one selected bar: all four move by the same working days, later.
  const c = await centre(page, bar(page, ids[4]));
  const done = opsDone(page);
  await dragMouse(page, c, { x: c.x + 90, y: c.y });
  const body = (await done).request().postDataJSON();
  expect(body.ops).toHaveLength(1);
  expect(body.ops[0].op).toBe("move");
  expect([...body.ops[0].ids].sort()).toEqual([...chosen].sort());
  const days = body.ops[0].days as number;
  expect(days).toBeGreaterThan(0);
  const v1 = await read(page, project);
  for (const id of chosen) expect(v1.computed.activities[id].es! - v0.computed.activities[id].es!).toBeGreaterThanOrEqual(days);
  await expect(page.getByText("4 selected")).toBeVisible();

  // Delete the selection, then Undo restores every row.
  await grid(page).locator(`[data-id="${ids[0]}"] [data-col="name"]`).click({ modifiers: ["Control"] });
  await grid(page).locator(`[data-id="${ids[0]}"] [data-col="name"]`).click({ modifiers: ["Control"] });
  await expect(page.getByText("4 selected")).toBeVisible();
  let wrote = opsDone(page);
  await page.keyboard.press("Delete");
  await wrote;
  for (const id of chosen) await expect(row(page, id)).toHaveCount(0);
  expect((await read(page, project)).activities.length).toBe(v0.activities.length - 4);
  wrote = opsDone(page);
  await page.keyboard.press("Control+z");
  await wrote;
  for (const id of chosen) await expect(row(page, id)).toHaveCount(1);
  expect((await read(page, project)).activities.length).toBe(v0.activities.length);

  // Deleting a stage with activities asks first, naming rows and links.
  await page.getByRole("button", { name: "Collapse all", exact: true }).click();
  const stage = v0.activities.find((a) => !a.parent_id && kidsOf(v0, a.id).length)!;
  await row(page, stage.id).locator('[data-col="name"]').click();
  await page.keyboard.press("Delete");
  const dialog = page.getByRole("dialog", { name: "Delete stage" });
  await expect(dialog).toContainText(new RegExp(`Delete ${stage.label} with its activities\\? ${1 + kidsOf(v0, stage.id).length} rows and \\d+ links are removed\\.`));
  wrote = opsDone(page);
  await dialog.getByRole("button", { name: "Delete", exact: true }).click();
  await wrote;
  await expect(row(page, stage.id)).toHaveCount(0);
  await toast(page).getByRole("button", { name: "Undo" }).click();
  await expect(row(page, stage.id)).toHaveCount(1);
  expect((await read(page, project)).activities.length).toBe(v0.activities.length);
});

test("drag a stage row to reorder: activities follow, the toast, reload keeps the chain (step 7)", async ({ page }) => {
  const project = await builtProgram(page, "Edit reorder");
  const v0 = await read(page, project);
  const stages = v0.activities.filter((a) => !a.parent_id);
  const moving = stages.find((s, i) => i > 0 && i < 4 && kidsOf(v0, s.id).length) ?? stages[1];
  const target = stages[5]; // drop below this stage's row
  const handle = row(page, moving.id).locator(".pg-handle");
  await row(page, moving.id).hover();
  const from = await centre(page, handle);
  const to = await centre(page, row(page, target.id));
  const done = opsDone(page);
  await dragMouse(page, from, { x: from.x, y: to.b.y + to.b.height - 2 }, 15);
  const body = (await done).request().postDataJSON();
  expect(body.ops[0]).toMatchObject({ op: "reorder", ids: [moving.id], after_id: target.id });
  await expect(toast(page)).toContainText(`Moved ${moving.label} after ${target.label}; sequence updated.`);
  await expect(toast(page).getByRole("button", { name: "Undo" })).toBeVisible();

  await page.reload();
  await openProgram(page);
  const order = await grid(page).locator('[role="row"][aria-level="1"]').evaluateAll((els) => els.map((e) => e.getAttribute("data-id")));
  expect(order.indexOf(moving.id)).toBe(order.indexOf(target.id) + 1);
  const v1 = await read(page, project);
  // The stage kept its activities, and the chain now runs from the stage above it.
  expect(kidsOf(v1, moving.id).map((a) => a.id)).toEqual(kidsOf(v0, moving.id).map((a) => a.id));
  expect(v1.links.some((l) => l.pred_id === target.id && l.succ_id === moving.id)).toBe(true);
});

test("resize a stage in the collapsed view, then its first activity takes the days (step 8)", async ({ page }) => {
  const project = await builtProgram(page, "Edit first child");
  const v0 = await read(page, project);
  const leaf = v0.activities.find((a) => !a.parent_id && !kidsOf(v0, a.id).length && a.duration_days !== null)!;
  await page.getByRole("button", { name: "Zoom in", exact: true }).click();
  await settled(page);
  // Drag the right end of its bar: a resize.
  const shape = bar(page, leaf.id);
  const c = await centre(page, shape);
  // The end zone is inside a bar wide enough for it, otherwise just past its end.
  const end = { x: c.b.width >= 24 ? c.b.x + c.b.width - 3 : c.b.x + c.b.width + 3, y: c.y };
  let done = opsDone(page);
  await dragMouse(page, end, { x: end.x + 60, y: end.y });
  const body = (await done).request().postDataJSON();
  expect(body.ops).toEqual([{ op: "resize", id: leaf.id, duration_days: expect.any(Number) }]);
  const days = body.ops[0].duration_days as number;
  expect(days).toBeGreaterThan(leaf.duration_days!);
  expect((await read(page, project)).activities.find((a) => a.id === leaf.id)!.duration_days).toBe(days);

  // Draw a bar in the stage's empty space: its first activity.
  const c2 = await centre(page, bar(page, leaf.id));
  done = opsDone(page);
  await dragMouse(page, { x: c2.b.x + c2.b.width + 40, y: c2.y }, { x: c2.b.x + c2.b.width + 120, y: c2.y });
  const addRes = await done;
  const add = addRes.request().postDataJSON();
  expect(add.ops[0]).toMatchObject({ op: "add", parent_id: leaf.id, label: "New activity" });
  await expect(toast(page)).toContainText(`${leaf.label}'s ${days} days moved to New activity.`);
  // The new row opens for naming.
  await expect(page.getByRole("textbox", { name: "Name of New activity" })).toBeFocused();
  await page.keyboard.press("Escape");
});

test("link by dots, the popover, typed predecessors, Link in sequence (step 9) and refusals (step 10)", async ({ page }) => {
  const project = await builtProgram(page, "Edit links");
  await page.getByRole("button", { name: "Expand all", exact: true }).click();
  const v0 = await read(page, project);
  const withKids = v0.activities.filter((a) => !a.parent_id && kidsOf(v0, a.id).length >= 3);
  // Predecessors from the stage before the last, so none starts on day 0 at the chart's edge.
  const [p1, p2] = kidsOf(v0, withKids[withKids.length - 2].id);
  const [s1, s2, s3] = kidsOf(v0, withKids[withKids.length - 1].id);
  const dot = (id: string, end: "S" | "F") => page.locator(`.pg-layer-bars [data-id="${id}"] circle[data-dot="${end}"]`);

  const linkByDots = async (pred: string, pe: "S" | "F", succ: string, se: "S" | "F" | null) => {
    // Scroll once (to the source), then read both points: scrolling to the
    // target would move the source.
    await centre(page, dot(pred, pe));
    const box = async (l: ReturnType<Page["locator"]>) => {
      const r = (await l.boundingBox())!;
      return { x: r.x + r.width / 2, y: r.y + r.height / 2 };
    };
    const a = await box(dot(pred, pe));
    const b = await box(se ? dot(succ, se) : bar(page, succ));
    const done = opsDone(page);
    // Moving onto the bar's area shows its dots; then press on the dot.
    await page.mouse.move(a.x, a.y);
    await page.mouse.down();
    await page.mouse.move(b.x, b.y, { steps: 15 });
    await page.mouse.up();
    return (await done).request().postDataJSON().ops[0];
  };
  expect(await linkByDots(p1.id, "F", s1.id, "S")).toMatchObject({ op: "link_add", pred_id: p1.id, succ_id: s1.id, type: "FS" });
  expect(await linkByDots(p1.id, "S", s2.id, "S")).toMatchObject({ type: "SS" });
  expect(await linkByDots(p1.id, "F", s3.id, "F")).toMatchObject({ type: "FF" });
  expect(await linkByDots(p2.id, "S", s1.id, "F")).toMatchObject({ type: "SF" });
  // Dropped on a bar's middle: finish to start, no lag.
  expect(await linkByDots(p2.id, "F", s3.id, null)).toMatchObject({ type: "FS", lag_days: 0 });
  let v = await read(page, project);
  const made = (p: string, s: string) => v.links.find((l) => l.pred_id === p && l.succ_id === s)!;
  expect([made(p1.id, s1.id).type, made(p1.id, s2.id).type, made(p1.id, s3.id).type, made(p2.id, s1.id).type]).toEqual(["FS", "SS", "FF", "SF"]);

  // Click the FS link: the popover sets lag −2.
  const fs = made(p1.id, s1.id);
  // A point on this link's line where no other link or bar lies on top.
  const pt = await page.locator(`.pg-layer-links [data-id="${fs.id}"] .pg-link-line`).evaluate((el: SVGPathElement, id: string) => {
    const m = el.getScreenCTM()!;
    const len = el.getTotalLength();
    for (let k = 1; k < 40; k++) {
      const p = el.getPointAtLength((len * k) / 40);
      const at = { x: p.x * m.a + p.y * m.c + m.e, y: p.x * m.b + p.y * m.d + m.f };
      const top = document.elementsFromPoint(at.x, at.y).find((e) => e.closest("[data-link], .pg-bar"));
      if (top?.closest("[data-link]")?.getAttribute("data-id") === id && !document.elementsFromPoint(at.x, at.y).some((e) => e.closest(".pg-bar"))) return at;
    }
    throw new Error("no clear point on the link");
  }, fs.id);
  await page.mouse.click(pt.x, pt.y);
  const pop = page.getByRole("dialog", { name: `Link ${p1.label} → ${s1.label}` });
  await expect(pop).toBeVisible();
  await pop.getByLabel("Lag").fill("-2");
  let done = opsDone(page);
  await pop.getByRole("button", { name: "Save" }).click();
  await done;
  v = await read(page, project);
  expect(made(p1.id, s1.id).lag_days).toBe(-2);

  // Type 12SS+2d style predecessors; Enter on the last cell adds a row below.
  const n = v.activities.findIndex((a) => a.id === p2.id) + 1;
  await row(page, s2.id).locator('[data-col="predecessors"]').dblclick();
  const cellInput = page.getByRole("textbox", { name: `Predecessors of ${s2.label}` });
  const before = await cellInput.inputValue();
  await cellInput.fill(`${before ? before + ", " : ""}${n}SS+2d`);
  done = opsDone(page);
  await cellInput.press("Enter");
  await done;
  v = await read(page, project);
  expect(made(p2.id, s2.id)).toMatchObject({ type: "SS", lag_days: 2, lag_unit: "working" });
  await expect(page.getByRole("textbox", { name: "Name of New activity" })).toBeFocused();
  await page.keyboard.press("Escape");
  // Invalid input keeps the cell open with a message.
  await row(page, s3.id).locator('[data-col="predecessors"]').dblclick();
  await page.getByRole("textbox", { name: `Predecessors of ${s3.label}` }).fill("12XX");
  await page.keyboard.press("Enter");
  await expect(row(page, s3.id).locator(".pg-cell-error")).toContainText("is not a predecessor");
  await page.keyboard.press("Escape");

  // Link in sequence on a selection of three unlinked rows.
  const [q1, q2, q3] = kidsOf(v, withKids[1].id).length >= 3 ? kidsOf(v, withKids[1].id) : [p1, p2, s1];
  await row(page, q1.id).locator('[data-col="name"]').click();
  await row(page, q3.id).locator('[data-col="name"]').click({ modifiers: ["Control"] });
  await row(page, s3.id).locator('[data-col="name"]').click({ modifiers: ["Control"] });
  done = opsDone(page);
  await page.getByRole("button", { name: "Link in sequence", exact: true }).click();
  expect((await done).request().postDataJSON().ops[0]).toEqual({ op: "link_chain", ids: [q1.id, q3.id, s3.id] });
  v = await read(page, project);
  expect(v.links.some((l) => l.pred_id === q1.id && l.succ_id === q3.id)).toBe(true);
  expect(v.links.some((l) => l.pred_id === q3.id && l.succ_id === s3.id)).toBe(true);
  void q2;

  // Step 10: a cycle is refused and named; indenting an activity is refused.
  const s1n = v.activities.findIndex((a) => a.id === s1.id) + 1;
  await row(page, p1.id).locator('[data-col="predecessors"]').dblclick();
  await page.getByRole("textbox", { name: `Predecessors of ${p1.label}` }).fill(String(s1n));
  done = opsDone(page);
  await page.keyboard.press("Tab");
  expect((await done).status()).toBe(422);
  await expect(toast(page)).toHaveText(new RegExp(`loop: .*${p1.label}.*→.*${s1.label}|loop: .*${s1.label}.*→.*${p1.label}`));
  expect((await read(page, project)).links.some((l) => l.pred_id === s1.id && l.succ_id === p1.id)).toBe(false);

  // Focus the name cell by keyboard (a click on a selected name renames it).
  await row(page, p1.id).locator('[data-col="start"]').click();
  await page.keyboard.press("ArrowLeft");
  await page.keyboard.press("ArrowLeft");
  await expect(row(page, p1.id).locator('[data-col="name"]')).toBeFocused();
  done = opsDone(page);
  await page.keyboard.press("Tab");
  expect((await done).status()).toBe(422);
  await expect(toast(page)).toHaveText("Programs have two levels: stages and activities.");
});

test("held drags snap back, a bulk nudge with a held row moves nothing, a milestone given days asks once (13.1)", async ({ page }) => {
  const project = await builtProgram(page, "Edit held");
  await page.getByRole("button", { name: "Expand all", exact: true }).click();
  const v0 = await read(page, project);
  // An activity whose start a link sets: dragging it earlier is held.
  const driven = v0.links.find((l) => {
    const s = v0.activities.find((a) => a.id === l.succ_id)!;
    return s.parent_id && l.type === "FS" && v0.computed.activities[s.id].es! >= 5 && v0.computed.activities[s.id].es === v0.computed.activities[l.pred_id].ef;
  })!;
  const succ = v0.activities.find((a) => a.id === driven.succ_id)!;
  const pred = v0.activities.find((a) => a.id === driven.pred_id)!;
  await page.getByRole("button", { name: "Zoom in", exact: true }).click();
  await settled(page);
  let posts = 0;
  page.on("request", (r) => r.url().endsWith("/program/ops") && r.method() === "POST" && posts++);
  const c = await centre(page, bar(page, succ.id));
  await page.mouse.move(c.x, c.y);
  await page.mouse.down();
  // One snap unit earlier: still after the program start, so the link is what holds it.
  const ppd = c.b.width / succ.duration_days!;
  const step = (ppd >= 12 ? 1 : 5) * ppd * 1.2;
  await page.mouse.move(c.x - step, c.y, { steps: 6 });
  await expect(page.locator(".pg-tip text")).toHaveText(`Held by ${pred.label}`);
  await page.mouse.up();
  await expect(page.locator(".pg-tip text")).toHaveText(`Held by ${pred.label}`);
  await page.waitForTimeout(300);
  expect(posts).toBe(0);
  // The bar is back where it was.
  expect(Math.abs((await bar(page, succ.id).boundingBox())!.x - c.b.x)).toBeLessThan(1);
  // Its accessible name says what holds it.
  await expect(page.locator(`.pg-layer-bars [data-id="${succ.id}"]`)).toHaveAttribute("aria-label", new RegExp(`held by ${pred.label}$`));

  // Escape mid-drag restores and sends nothing.
  await page.mouse.move(c.x, c.y);
  await page.mouse.down();
  await page.mouse.move(c.x + 60, c.y, { steps: 6 });
  await page.keyboard.press("Escape");
  await page.mouse.up();
  await page.waitForTimeout(300);
  expect(posts).toBe(0);

  // Shift+Left on a selection holding a held row: nothing moves, the rows are named.
  const free = v0.activities.find((a) => a.parent_id && a.id !== succ.id && v0.computed.activities[a.id].es! > 0 && !v0.links.some((l) => l.succ_id === a.id))!;
  await row(page, succ.id).locator('[data-col="name"]').click();
  if (free) await row(page, free.id).locator('[data-col="name"]').click({ modifiers: ["Control"] });
  const done = opsDone(page);
  await page.keyboard.press("Shift+ArrowLeft");
  expect((await done).status()).toBe(422);
  await expect(toast(page)).toContainText(new RegExp(`held by ${esc(pred.label)}`, "i"));
  const v1 = await read(page, project);
  expect(v1.program.version).toBe(v0.program.version);

  // A milestone given days becomes an activity after one confirm.
  const stage = v0.activities.find((a) => !a.parent_id && kidsOf(v0, a.id).length)!;
  const add = await page.request.post(`/api/projects/${project}/program/ops`, {
    headers: origin(page),
    data: { program_id: v1.program.id, version: v1.program.version, ops: [{ op: "add", temp_id: "m", parent_id: stage.id, kind: "milestone", label: "Handover" }] },
  });
  expect(add.ok(), await add.text()).toBeTruthy();
  const ms = ((await add.json()) as { temp_ids: Record<string, string> }).temp_ids.m;
  await expect(row(page, ms)).toHaveCount(1); // arrives by SSE
  await row(page, ms).locator('[data-col="duration"]').dblclick();
  await page.getByRole("textbox", { name: "Duration of Handover" }).fill("5");
  await page.keyboard.press("Enter");
  const confirm = page.getByRole("dialog", { name: "Milestone to activity" });
  await expect(confirm).toContainText("Handover is a milestone. Giving it 5 days makes it an activity.");
  const wrote = opsDone(page);
  await confirm.getByRole("button", { name: "Make it an activity" }).click();
  await wrote;
  const after = (await read(page, project)).activities.find((a) => a.id === ms)!;
  expect([after.kind, after.duration_days]).toEqual(["activity", 5]);
});

test("two browsers: an edit arrives by SSE; a stale write shows the other version (step 14)", async ({ page, browser }) => {
  const project = await builtProgram(page, "Edit two users");
  const v0 = await read(page, project);
  const stage = v0.activities.find((a) => !a.parent_id && a.duration_days !== null)!;
  const other = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  const b = await other.newPage();
  await signIn(b);
  await b.goto(`/projects/${project}`);
  await openProgram(b);
  await expect(row(b, stage.id)).toBeVisible();

  // An edit in A arrives in B through program.changed.
  await row(page, stage.id).locator(".pg-label").dblclick();
  await page.getByRole("textbox", { name: `Name of ${stage.label}` }).fill("Renamed in A");
  await page.keyboard.press("Enter");
  await expect(row(b, stage.id).locator(".pg-label")).toHaveText("Renamed in A");

  // B starts dragging; A edits meanwhile; B refetches only on release, so
  // its move carries the old version: 409 and A's version shown.
  await b.getByRole("button", { name: "Zoom in", exact: true }).click();
  await settled(b);
  const c = await centre(b, bar(b, stage.id));
  await b.mouse.move(c.x, c.y);
  await b.mouse.down();
  await b.mouse.move(c.x + 50, c.y, { steps: 5 });
  await row(page, stage.id).locator(".pg-label").dblclick();
  await page.getByRole("textbox", { name: "Name of Renamed in A" }).fill("Renamed again");
  const aDone = opsDone(page);
  await page.keyboard.press("Enter");
  await aDone;
  await b.waitForTimeout(800);
  await expect(row(b, stage.id).locator(".pg-label")).toHaveText("Renamed in A"); // no refetch during the drag
  const bDone = b.waitForResponse((r) => r.url().endsWith("/program/ops") && r.request().method() === "POST");
  await b.mouse.move(c.x + 90, c.y, { steps: 4 });
  await b.mouse.up();
  expect((await bDone).status()).toBe(409);
  await expect(toast(b)).toHaveText("Someone else changed the program; showing their version.");
  await expect(row(b, stage.id).locator(".pg-label")).toHaveText("Renamed again");
  await other.close();
});

test("a user-made stage can be told which catalogue stage it is (owner answer Q5)", async ({ page }) => {
  const project = await builtProgram(page, "Edit catalogue");
  const v = await read(page, project);
  const last = v.activities.filter((a) => !a.parent_id).at(-1)!;
  // Enter on a stage's last cell adds a user-made stage below it.
  await row(page, last.id).locator('[data-col="predecessors"]').dblclick();
  const done = opsDone(page);
  await page.keyboard.press("Enter");
  await done;
  await page.getByRole("textbox", { name: "Name of New stage" }).fill("Site establishment");
  const named = opsDone(page);
  await page.keyboard.press("Enter");
  await named;
  const picker = page.getByRole("combobox", { name: "Catalogue stage for Site establishment" });
  await expect(picker).toHaveValue("other");
  const wrote = opsDone(page);
  await picker.selectOption({ label: "Construction" });
  expect((await wrote).request().postDataJSON().ops[0]).toMatchObject({ op: "update", catalogue_stage_id: "construction" });
  await expect(picker).toHaveCount(0);
});

// --- browser budget and touch (plan 11.2, 9.4) ---------------------------------

function syntheticView(project: string, stages: number, per: number) {
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
      activities.push(row(id, sid, k, "activity", `Stage ${s + 1} activity ${k + 1}`, 2 + ((s + k) % 9)));
      if (prev) links.push(lnk(uuid(100000 + n), prev, id));
      prev = id;
    }
    if (prevStage) links.push(lnk(uuid(200000 + n), prevStage, sid));
    prevStage = sid;
  }
  const input: Input = {
    calendar: { start: "2027-03-01", working_week: 31, days: [] },
    activities: activities.map((a) => ({ id: a.id as string, parent: (a.parent_id as string) ?? "", ordinal: a.ordinal as number, kind: a.kind as string, label: a.label as string, duration: a.duration_days as number | null, constraint: "asap" })),
    links: links.map((l) => ({ id: l.id as string, pred: l.pred_id as string, succ: l.succ_id as string, type: "FS", lag: 0, lag_unit: "working" })),
  };
  return {
    program: { id: uuid(999999), project_id: project, version: 1, start_date: "2027-03-01", start_origin: null, skeleton_id: "sk.generic", stage_revision: 1, working_week: 31 },
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
  function lnk(id: string, pred: string, succ: string) {
    return { id, pred_id: pred, succ_id: succ, type: "FS", lag_days: 0, lag_unit: "working", origin: "library", version: 1 };
  }
}

/** Serves a synthetic view for GET /program (all stages expanded); POST /ops is captured and answered with the same view. */
async function serveSynthetic(page: Page, project: string, stages: number, per: number) {
  const view = syntheticView(project, stages, per);
  const body = JSON.stringify(view);
  const posted: unknown[] = [];
  await page.route(`**/api/projects/${project}/program`, (route: Route) =>
    route.request().method() === "GET" ? route.fulfill({ status: 200, contentType: "application/json", body }) : route.continue(),
  );
  await page.route(`**/api/projects/${project}/program/ops`, (route: Route) => {
    posted.push(route.request().postDataJSON());
    return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ ...view, toasts: [] }) });
  });
  const stageIds = view.activities.filter((a) => a.parent_id === null).map((a) => a.id);
  await page.evaluate(([p, ids]) => localStorage.setItem(`sitewise.program.expanded.v1.${p}`, JSON.stringify(ids)), [project, stageIds] as const);
  return { view, posted };
}

test("drag frame ≤ 16 ms p95 at 2,000 activities with 20 bars selected (plan 11.2)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signIn(page);
  const project = await newProject(page, "Edit drag budget");
  const { posted } = await serveSynthetic(page, project, 100, 20); // 2,100 rows, 2,000 activities
  await page.reload();
  await openProgram(page);
  const acts = grid(page).locator('[role="row"][aria-level="2"]');
  await acts.first().click();
  await acts.nth(19).click({ modifiers: ["Shift"] });
  await expect(page.getByText("20 selected")).toBeVisible();
  await page.keyboard.press("Shift+F"); // fit to the selection so its bars are wide enough to grab
  await page.waitForTimeout(200);
  const id = await acts.nth(10).getAttribute("data-id");
  const c = await centre(page, bar(page, id!));
  await page.mouse.move(c.x, c.y);
  await page.mouse.down();
  // Each step crosses at least one snap unit, so every frame recomputes.
  for (let i = 1; i <= 60; i++) {
    await page.mouse.move(c.x + i * 9, c.y);
    await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => r(null))));
  }
  const frames = await page.evaluate(() => (window as unknown as { __pgDragFrames: number[] }).__pgDragFrames.slice());
  await page.keyboard.press("Escape");
  await page.mouse.up();
  expect(posted).toEqual([]); // cancelled: nothing sent
  frames.sort((a, b) => a - b);
  const p95 = frames[Math.min(frames.length - 1, Math.floor(frames.length * 0.95))];
  console.log(`program drag frame, 2,000 activities, 20 selected: ${frames.length} recomputed frames, p50 ${frames[Math.floor(frames.length / 2)].toFixed(2)} ms, p95 ${p95.toFixed(2)} ms, max ${frames.at(-1)!.toFixed(2)} ms`);
  expect(frames.length).toBeGreaterThan(20);
  expect(p95).toBeLessThanOrEqual(16);
});

test.describe("touch", () => {
  test.use({ hasTouch: true });
  test("a long press drags a bar; a quick swipe on a bar still scrolls (plan 9.4, step 11)", async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 700 });
    await signIn(page);
    const project = await newProject(page, "Edit touch");
    const { view, posted } = await serveSynthetic(page, project, 10, 10);
    await page.reload();
    await openProgram(page);
    const body = page.locator(".pg-body");
    await body.evaluate((el) => el.scrollIntoView({ block: "end" }));
    await page.getByRole("button", { name: "Zoom in", exact: true }).click();
  await settled(page);
    const target = (view.activities[3] as { id: string }).id;
    const cdp = await page.context().newCDPSession(page);
    const at = async () => {
      const c = await centre(page, bar(page, target));
      return { x: Math.round(c.x), y: Math.round(c.y) };
    };
    // A quick vertical swipe starting on a bar scrolls the chart.
    let p = await at();
    const top0 = await body.evaluate((el) => el.scrollTop);
    await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [p] });
    for (let k = 1; k <= 8; k++) await cdp.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: [{ x: p.x, y: p.y - k * 20 }] });
    await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    await expect.poll(() => body.evaluate((el) => el.scrollTop)).toBeGreaterThan(top0 + 40);
    expect(posted).toEqual([]);
    // A 350 ms long press, then a move: a drag that posts a move.
    p = await at();
    await cdp.send("Input.dispatchTouchEvent", { type: "touchStart", touchPoints: [p] });
    await page.waitForTimeout(500);
    for (let k = 1; k <= 8; k++) await cdp.send("Input.dispatchTouchEvent", { type: "touchMove", touchPoints: [{ x: p.x + k * 15, y: p.y }] });
    await cdp.send("Input.dispatchTouchEvent", { type: "touchEnd", touchPoints: [] });
    await expect.poll(() => posted.length).toBe(1);
    const op = (posted[0] as { ops: { op: string; ids: string[]; days: number }[] }).ops[0];
    expect(op.op).toBe("move");
    expect(op.ids).toEqual([target]);
    expect(op.days).toBeGreaterThan(0);
  });
});
