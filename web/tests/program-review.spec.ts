// Program review and readings (plan docs/plans/2026-10-18-program.md
// WP-P9 UI and WP-P12 to WP-P14 UI): Playwright steps 3 and 13 of section
// 11.5, a program file read by rule (no Jev) and accepted, and the hold
// point, R7 and evidence-without-stage lines.
//
// The e2e server's default program library (-program-library fixture) is a
// temp copy of knowledge/ with one test fragment: gas work on a new building
// brings "Gas rough-in" and "Gas fit-off", each with a reviewed 3–5–8 day
// duration. With the work type unknown the fragment is conditional, so the
// strip asks for the work type. The repo's knowledge/ is never edited.

import { expect, test, type Page } from "@playwright/test";

type Act = { id: string; parent_id: string | null; kind: string; label: string; duration_days: number | null; constraint_kind: string; constraint_date: string | null; catalogue_stage_id?: string | null };
type View = {
  program: { id: string; version: number };
  activities: Act[];
  computed: { finish: { low_index: number | null; high_index: number | null } };
  question: { key: string; label: string; options: string[]; values: string[] } | null;
  readings: { id: string }[];
  links: { id: string; pred_id: string; succ_id: string; type: string; lag_days: number; origin: string }[];
  changes_pending: number;
};

async function signIn(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
}

async function newProject(page: Page, name: string): Promise<string> {
  await signIn(page);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page).toHaveURL(/\/projects\/[0-9a-f-]{36}$/);
  return new URL(page.url()).pathname.split("/").pop()!;
}

const origin = (page: Page) => ({ Origin: new URL(page.url()).origin });
const read = async (page: Page, project: string): Promise<View> => (await page.request.get(`/api/projects/${project}/program`)).json();
const grid = (page: Page) => page.getByRole("treegrid", { name: "Program rows" });
const rowNamed = (page: Page, label: string) => grid(page).locator('[role="row"]', { has: page.locator(".pg-label", { hasText: new RegExp(`^${label}$`) }) });

async function profileSet(page: Page, project: string, key: string, value: string) {
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin(page), data: { part_id: profile.parts[0].id, value } });
  expect(r.ok(), await r.text()).toBeTruthy();
}

/** A gas work item: the fixture fragment's trigger. */
async function gasWork(page: Page, project: string) {
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const r = await page.request.post(`/api/projects/${project}/works`, {
    headers: origin(page),
    data: { part_id: profile.parts[0].id, system_id: "hydraulic.gas", action: "new", title: "Gas service" },
  });
  expect(r.ok(), await r.text()).toBeTruthy();
  return profile;
}

async function ops(page: Page, project: string, list: unknown[]): Promise<View> {
  const v = await read(page, project);
  const r = await page.request.post(`/api/projects/${project}/program/ops`, { headers: origin(page), data: { program_id: v.program.id, version: v.program.version, ops: list } });
  expect(r.ok(), await r.text()).toBeTruthy();
  return r.json();
}

async function openProgram(page: Page) {
  await page.getByRole("button", { name: "Program", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Program", exact: true })).toBeVisible();
}

async function build(page: Page) {
  await openProgram(page);
  await page.getByRole("button", { name: "Build program", exact: true }).click();
  await expect(grid(page)).toBeVisible();
}

const width = (v: View) => (v.computed.finish.high_index ?? 0) - (v.computed.finish.low_index ?? 0);

test("answer the strip question: the range narrows and the Profile shows the value (step 3)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const project = await newProject(page, "Program question");
  await gasWork(page, project);
  await build(page);

  // Every row but the conditional gas work gets a duration, so the finish
  // has a range that only the gas work makes.
  let v = await read(page, project);
  const parents = new Set(v.activities.map((a) => a.parent_id).filter(Boolean));
  const gas = v.activities.filter((a) => /Gas (rough-in|fit-off)$/.test(a.label));
  expect(gas.map((a) => a.label.startsWith("If ")).every(Boolean), JSON.stringify(gas)).toBe(true);
  expect(gas).toHaveLength(2);
  v = await ops(page, project, [
    ...v.activities.filter((a) => !parents.has(a.id) && !gas.includes(a) && a.kind !== "hold_point").map((a) => ({ op: "update", id: a.id, duration_days: 5 })),
    { op: "set_start", start_date: "2027-03-01" },
  ]);
  const before = width(v);
  expect(before).toBeGreaterThan(0);
  expect(v.question?.key).toBe("work_type");
  await page.reload();
  await openProgram(page);
  const range = page.getByTestId("program-range");
  await expect(range).toBeVisible();
  const wide = await range.textContent();

  // The strip asks the one fact that narrows the finish most; answer it.
  const strip = page.getByRole("region", { name: "Question" });
  await expect(strip.locator(".pg-q-label")).toHaveText(v.question!.label);
  await expect(strip.locator(".pg-q-why")).toContainText("Narrows the finish by up to");
  const option = v.question!.options[v.question!.values.indexOf("new")];
  await strip.getByRole("button", { name: option, exact: true }).click();
  await expect(range).not.toHaveText(wide!);
  await expect(page.locator(".pg-toast")).toContainText("Undo");
  const after = await read(page, project);
  expect(width(after)).toBeLessThan(before);
  expect(after.activities.filter((a) => /^Gas (rough-in|fit-off)$/.test(a.label))).toHaveLength(2);
  expect(after.question?.key ?? "").not.toBe("work_type");

  // The answer was written through the Profile.
  await page.getByRole("button", { name: "Profile", exact: true }).click();
  await expect(page.locator(".pf-work-line").getByRole("radio", { name: option })).toBeChecked();
});

test("a work item changed in the Profile raises the review: an edited row defaults to Keep, Adopt all, Undo (step 13)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const project = await newProject(page, "Program review");
  await profileSet(page, project, "hdr.work_type", "new");
  const profile = await gasWork(page, project);
  await build(page);
  let v = await read(page, project);
  const rough = v.activities.find((a) => a.label === "Gas rough-in")!;
  const fitOff = v.activities.find((a) => a.label === "Gas fit-off")!;
  expect(rough && fitOff).toBeTruthy();
  // The user edits one of the two rows the work item brought.
  v = await ops(page, project, [{ op: "update", id: rough.id, label: "Gas rough-in, level 1" }]);
  expect(v.changes_pending).toBe(0);
  await expect(page.locator(".pg-banner")).toHaveCount(0);

  // Exclude the gas work in the Profile (the system row's tick is its inclusion).
  await page.getByRole("button", { name: "Profile", exact: true }).click();
  const group = profile.systems.find((g: { rows: { leaf: string }[] }) => g.rows.some((r) => r.leaf === "hydraulic.gas"));
  const leaf = group.rows.find((r: { leaf: string }) => r.leaf === "hydraulic.gas");
  await page.locator(".sw-entry", { hasText: group.label }).click();
  const pane = page.getByRole("region", { name: group.label, exact: true });
  await pane.getByRole("checkbox", { name: leaf.label, exact: true }).uncheck();

  // The review runs in the background; the open Program hears of it.
  await openProgram(page);
  const banner = page.locator(".pg-banner");
  await expect(banner).toContainText(/^Scope and procurement changed: (.*, )?2 to remove(, .*)?\.\s*Review$/, { timeout: 20_000 });
  // Nothing changed on the program until a choice is made.
  expect((await read(page, project)).activities.some((a) => a.id === fitOff.id)).toBe(true);
  await expect(page.getByRole("button", { name: /^Review changes \(\d+\)$/ })).toBeVisible();
  await banner.getByRole("button", { name: "Review", exact: true }).click();

  const panel = page.getByRole("region", { name: "Review changes" });
  const item = (label: string) => panel.locator(".pg-review-item", { hasText: label });
  const choice = (label: string, c: "Adopt" | "Keep") => item(label).getByRole("radio", { name: c, exact: true });
  // The edited row defaults to Keep and says why; the untouched one to Adopt.
  await expect(item("Gas rough-in, level 1")).toContainText("you edited “Gas rough-in, level 1”");
  await expect(choice("Gas rough-in, level 1", "Keep")).toHaveAttribute("aria-checked", "true");
  await expect(choice("Gas fit-off", "Adopt")).toHaveAttribute("aria-checked", "true");
  await expect(item("Gas fit-off")).not.toContainText("you edited");

  // Adopt all adopts what defaults to Adopt; the edited row waits for an explicit choice.
  await panel.getByRole("button", { name: "Adopt all", exact: true }).click();
  await expect(rowNamed(page, "Gas fit-off")).toHaveCount(0);
  await expect(item("Gas fit-off")).toHaveCount(0);
  await expect(item("Gas rough-in, level 1")).toBeVisible();
  v = await read(page, project);
  expect(v.activities.some((a) => a.id === fitOff.id)).toBe(false);
  expect(v.activities.some((a) => a.id === rough.id)).toBe(true);
  expect(v.changes_pending).toBe(1);

  // One undo step restores the adopted removal.
  const toast = page.locator(".pg-toast");
  await expect(toast).toContainText("1 change adopted");
  await toast.getByRole("button", { name: "Undo", exact: true }).click();
  await expect.poll(async () => (await read(page, project)).activities.filter((a) => a.id === rough.id || a.id === fitOff.id).length).toBe(2);
  expect((await read(page, project)).activities.find((a) => a.id === rough.id)?.label).toBe("Gas rough-in, level 1");
});

test("a program file is read by rule, its dates proposed beside the row, accepted, and an unmatched row added (plan 10.2)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const project = await newProject(page, "Program import");
  await build(page);
  let v = await ops(page, project, [{ op: "set_start", start_date: "2027-03-01" }]);
  const parents = new Set(v.activities.map((a) => a.parent_id).filter(Boolean));
  const target = v.activities.find((a) => a.kind === "stage" && !parents.has(a.id))!;
  await page.reload();
  await openProgram(page);

  // A file whose first row is the stage's own label settles by rule; the
  // second matches nothing. The recorded Jev answers nothing here.
  const xlsx = await (await page.request.get(`/__e2e/program.xlsx?rows=${encodeURIComponent(`${target.label}~15/03/2027~26/03/2027,Site sign install~29/03/2027~30/03/2027`)}`)).body();
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await (await chooser).setFiles({ name: "Smith program.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: xlsx });

  // Filing and reading run in the background; the toolbar says what came back.
  await expect(page.getByRole("button", { name: "1 reading from Smith program.xlsx", exact: true })).toBeVisible({ timeout: 20_000 });
  const panel = page.getByRole("region", { name: "Readings" });
  const reading = panel.locator(`[data-target="${target.id}"]`);
  await expect(reading).toContainText(target.label);
  await expect(reading).toContainText("15 Mar 2027 – 26 Mar 2027");
  // Beside the row: a marker in the grid and an amber outline on the chart.
  await expect(grid(page).locator(`[data-id="${target.id}"] .pg-note-chip.is-reading`)).toBeVisible();
  await expect(page.locator(".pg-reading-mark")).toHaveCount(1);
  await expect(page.getByRole("button", { name: /^Accept all \(\d+\)$/ })).toBeVisible();

  await reading.getByRole("button", { name: `Accept reading for ${target.label}`, exact: true }).click();
  await expect(reading).toHaveCount(0);
  await expect(page.locator(".pg-reading-mark")).toHaveCount(0);
  v = await read(page, project);
  const accepted = v.activities.find((a) => a.id === target.id)!;
  expect(accepted.duration_days).toBe(10);
  expect([accepted.constraint_kind, accepted.constraint_date]).toEqual(["snet", "2027-03-15"]);
  await expect(grid(page).locator(`[data-id="${target.id}"] [data-col="duration"]`)).toHaveText("10d");

  // The row no activity matched is added in one click under a chosen stage.
  await panel.getByLabel("Add under").selectOption({ label: target.label });
  await panel.getByRole("button", { name: "Add Site sign install", exact: true }).click();
  await expect.poll(async () => (await read(page, project)).activities.find((a) => a.label === "Site sign install")?.parent_id).toBe(target.id);
  await expect(panel.getByRole("button", { name: "Add Site sign install", exact: true })).toHaveCount(0);

  // Dropping a file on the view imports it too; other file types are refused.
  const drop = async (name: string, bytes: number[]) =>
    page.locator(".pg-body").evaluate(
      (el, f) => {
        const dt = new DataTransfer();
        dt.items.add(new File([new Uint8Array(f.bytes)], f.name));
        el.dispatchEvent(new DragEvent("dragover", { bubbles: true, cancelable: true, dataTransfer: dt }));
        el.dispatchEvent(new DragEvent("drop", { bubbles: true, cancelable: true, dataTransfer: dt }));
      },
      { name, bytes },
    );
  await drop("notes.txt", [104, 105]);
  await expect(page.locator(".pg-toast")).toHaveText("Only PDF, DOCX and XLSX program files can be imported.");
  const other = await (await page.request.get(`/__e2e/program.xlsx?rows=${encodeURIComponent("Hoarding~01/04/2027~02/04/2027")}`)).body();
  await drop("Jones program.xlsx", [...other]);
  await expect(page.getByRole("button", { name: /from Jones program\.xlsx$/ })).toBeVisible({ timeout: 20_000 });
});

test("hold-point evidence, an awarded proposal's reading and evidence without a stage (plan 10.3, 10.4, 7.7)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const project = await newProject(page, "Program evidence");
  await build(page);
  const real = await read(page, project);
  const stages = real.activities.filter((a) => !a.parent_id);
  const design = stages.find((s) => s.catalogue_stage_id === "concept_design")!;
  // The server's lines, on a live program with one hold point added: these
  // readings come from background jobs (the passage pipeline, an award)
  // that the browser suite cannot drive, so the view is served with them.
  const hold = { ...design, id: "00000000-0000-4000-8000-0000000000aa", parent_id: design.id, kind: "hold_point", label: "Slab reinforcement inspection", duration_days: 0 };
  const doc = "00000000-0000-4000-8000-0000000000dd";
  const reading = (id: string, over: object) => ({ id, import_id: "", target_activity_id: null, catalogue_stage_id: null, field: "duration", value: "", parsed: {}, band: "green", decided_by: "rule", document_id: doc, document_title: "Architect proposal.pdf", applies: true, source: "proposal_r7", ...over });
  const served = {
    ...real,
    activities: [...real.activities, hold],
    readings: [
      reading("00000000-0000-4000-8000-000000000001", { target_activity_id: design.id, catalogue_stage_id: "concept_design", value: "6 weeks", text: "From the awarded proposal: Concept design 6 weeks" }),
      reading("00000000-0000-4000-8000-000000000002", { catalogue_stage_id: "approval_documentation", value: "4 weeks", text: "From the awarded proposal: Approval documentation 4 weeks" }),
    ],
    readings_pending: 3,
    checks: [...((real as unknown as { checks: unknown[] }).checks ?? []), { code: "evidence_without_stage", message: "Evidence for approval_documentation has no stage. Add the stage back or dismiss." }],
    hold_points: [
      {
        activity_id: hold.id,
        released: null,
        limit: "One hold point per interface per program: evidence for one pour cannot be told apart from another. You judge.",
        evidence: [{ reading_id: "00000000-0000-4000-8000-000000000003", document_id: doc, passage_id: "", document_title: "Inspection certificate", page: 2, band: "green", text: "Evidence found: Inspection certificate, page 2", document_date: "2027-05-04", date_label: "document dated" }],
      },
    ],
  };
  await page.route(`**/api/projects/${project}/program`, (route) =>
    route.request().method() === "GET" ? route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(served) }) : route.continue(),
  );
  const sent: { url: string; body: Record<string, unknown> }[] = [];
  await page.route(/\/program\/(hold-points\/release|readings\/dismiss)$/, (route) => {
    sent.push({ url: route.request().url(), body: route.request().postDataJSON() });
    return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(served) });
  });
  await page.reload();
  await openProgram(page);

  // R7: the awarded proposal's line, beside the stage it fills.
  const panel = page.getByRole("region", { name: "Readings" });
  await expect(page.getByRole("button", { name: "1 reading from awarded proposals", exact: true })).toBeVisible();
  await expect(panel.locator(`[data-target="${design.id}"]`)).toContainText("From the awarded proposal: Concept design 6 weeks");

  // 7.7: evidence whose stage is gone, named by the stage's label, dismissable.
  const notice = page.locator(".pg-banner", { hasText: "Evidence for Approval documentation has no stage. Add the stage back or dismiss." });
  await expect(notice).toBeVisible();
  await notice.getByRole("button", { name: "Dismiss", exact: true }).click();
  await expect.poll(() => sent.length).toBe(1);
  expect(sent[0].url).toMatch(/readings\/dismiss$/);
  expect(sent[0].body.ids).toEqual(["00000000-0000-4000-8000-000000000002"]);

  // 10.4: evidence links to the page, says "document dated", carries the
  // known limit, and only a click releases.
  const holds = page.locator(".pg-holds");
  await expect(holds).toHaveAttribute("open", "");
  const evidence = holds.locator(`[data-hold="${hold.id}"] .pg-hold-evidence`);
  const link = evidence.getByRole("link", { name: "Evidence found: Inspection certificate, page 2", exact: true });
  await expect(link).toHaveAttribute("href", `/api/documents/${doc}/file?view=1#page=2`);
  await expect(evidence).toContainText("document dated 4 May 2027");
  await expect(evidence).toHaveAttribute("title", /cannot be told apart/);
  await evidence.getByRole("button", { name: "Mark released", exact: true }).click();
  await expect.poll(() => sent.length).toBe(2);
  expect(sent[1].url).toMatch(/hold-points\/release$/);
  expect(sent[1].body).toMatchObject({ id: hold.id, reading_id: "00000000-0000-4000-8000-000000000003" });
});

test("a file's predecessor column proposes links between matched rows: added on accept, or by Add and Add all (plan 10.2)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const project = await newProject(page, "Program import links");
  await build(page);
  let v = await ops(page, project, [{ op: "set_start", start_date: "2027-03-01" }]);
  const parents = new Set(v.activities.map((a) => a.parent_id).filter(Boolean));
  const leaves = v.activities.filter((a) => a.kind === "stage" && !parents.has(a.id));
  // Three stages in program order with gaps, so no library link joins them
  // directly and no file link closes a loop.
  const [a, b, c] = [leaves[0], leaves[2], leaves[4]];
  const joined = (x: string, y: string) => v.links.some((l) => (l.pred_id === x && l.succ_id === y) || (l.pred_id === y && l.succ_id === x));
  expect([joined(a.id, b.id), joined(a.id, c.id), joined(b.id, c.id)]).toEqual([false, false, false]);
  await page.reload();
  await openProgram(page);

  // Rows number from 1: B follows A; C follows A and starts 2 days after B starts.
  const rows = [`${a.label}~15/03/2027~26/03/2027~`, `${b.label}~29/03/2027~09/04/2027~1`, `${c.label}~12/04/2027~16/04/2027~1;2SS+2d`].join(",");
  const xlsx = await (await page.request.get(`/__e2e/program.xlsx?rows=${encodeURIComponent(rows)}`)).body();
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Import", exact: true }).click();
  await (await chooser).setFiles({ name: "Links program.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: xlsx });
  await expect(page.getByRole("button", { name: "3 readings from Links program.xlsx", exact: true })).toBeVisible({ timeout: 20_000 });

  const panel = page.getByRole("region", { name: "Readings" });
  const links = panel.locator(".pg-file-links");
  const link = (p: Act, s: Act) => links.locator(`li[data-pred="${p.id}"][data-succ="${s.id}"]`);
  await expect(links).toContainText("3 links from the file's predecessor column");
  await expect(link(a, b)).toContainText(`${a.label} → ${b.label}`);
  await expect(link(a, b).locator(".pg-file-dates")).toHaveText("FS");
  await expect(link(b, c).locator(".pg-file-dates")).toHaveText("SS+2d");
  // Every end still waits for its reading: accepting adds the links.
  for (const [p, s] of [[a, b], [a, c], [b, c]]) await expect(link(p, s)).toContainText("Added when you accept the readings");
  await expect(links.getByRole("button")).toHaveCount(0);

  const accept = async (x: Act) => {
    const done = page.waitForResponse((r) => r.url().endsWith("/program/accept") && r.request().method() === "POST");
    await panel.getByRole("button", { name: `Accept reading for ${x.label}`, exact: true }).click();
    expect((await done).ok()).toBe(true);
  };
  await accept(a);
  await expect(link(a, b)).toBeVisible();
  await accept(b);
  await expect(link(a, b)).toHaveCount(0);
  await accept(c);
  await expect(links).toHaveCount(0);
  v = await read(page, project);
  const made = (p: Act, s: Act) => v.links.find((l) => l.pred_id === p.id && l.succ_id === s.id);
  expect(made(b, c)).toMatchObject({ type: "SS", lag_days: 2, origin: "document" });

  // Removed later, the file's links are offered back, each with Add, and Add all.
  v = await ops(page, project, [{ op: "link_delete", ids: [made(a, c)!.id, made(b, c)!.id] }]);
  await expect(link(a, c).getByRole("button", { name: `Add link ${a.label} to ${c.label}`, exact: true })).toBeVisible();
  await expect(link(b, c).getByRole("button", { name: `Add link ${b.label} to ${c.label}`, exact: true })).toBeVisible();
  let done = page.waitForResponse((r) => r.url().endsWith("/program/ops") && r.request().method() === "POST");
  await links.getByRole("button", { name: "Add all 2 links", exact: true }).click();
  const body = (await done).request().postDataJSON();
  expect(body.ops).toEqual([
    { op: "link_add", pred_id: a.id, succ_id: c.id, type: "FS", lag_days: 0, lag_unit: "working" },
    { op: "link_add", pred_id: b.id, succ_id: c.id, type: "SS", lag_days: 2, lag_unit: "working" },
  ]);
  await expect(links).toHaveCount(0);
  await expect(page.locator(".pg-toast")).toContainText("Added 2 links from the file.");

  // One link alone: its own Add.
  v = await read(page, project);
  v = await ops(page, project, [{ op: "link_delete", ids: [made(b, c)!.id] }]);
  await expect(links.getByRole("button", { name: /^Add all/ })).toHaveCount(0);
  done = page.waitForResponse((r) => r.url().endsWith("/program/ops") && r.request().method() === "POST");
  await link(b, c).getByRole("button", { name: `Add link ${b.label} to ${c.label}`, exact: true }).click();
  expect((await done).request().postDataJSON().ops).toHaveLength(1);
  await expect(links).toHaveCount(0);
  expect((await read(page, project)).links.some((l) => l.pred_id === b.id && l.succ_id === c.id)).toBe(true);
});

test("Not sure in the strip dismisses the question and changes nothing (plan 7.3)", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  const project = await newProject(page, "Program not sure");
  await build(page);
  await ops(page, project, [{ op: "set_start", start_date: "2027-03-01" }]);
  // Only a yes/no question offers Not sure, and the e2e library asks none,
  // so the view is served with one; the route answers the way the server's
  // not_sure path does: the question goes, the program is unchanged.
  const question = { key: "det.basement", kind: "determinant", label: "Is there a basement?", options: ["Yes", "No", "Not sure"], values: ["true", "false"], narrows_by_days: 15 };
  let dismissed = false;
  await page.route(`**/api/projects/${project}/program`, async (route) => {
    if (route.request().method() !== "GET") return route.continue();
    const res = await route.fetch();
    const json = await res.json();
    return route.fulfill({ response: res, json: { ...json, question: dismissed ? null : question } });
  });
  const sent: unknown[] = [];
  await page.route(`**/api/projects/${project}/program/question`, async (route) => {
    sent.push(route.request().postDataJSON());
    dismissed = true;
    const json = await read(page, project);
    return route.fulfill({ status: 200, contentType: "application/json", json: { ...json, question: null, toasts: ["We won't ask about is there a basement? again until it changes in the Profile."] } });
  });
  await page.reload();
  await openProgram(page);
  const before = await read(page, project);
  const strip = page.getByRole("region", { name: "Question" });
  await expect(strip.locator(".pg-q-label", { hasText: question.label })).toBeVisible();
  await strip.getByRole("button", { name: "Not sure", exact: true }).click();
  await expect.poll(() => sent.length).toBe(1);
  expect(sent[0]).toMatchObject({ key: "det.basement", value: "not_sure", version: before.program.version });
  await expect(page.locator(".pg-toast")).toContainText("We won't ask about");
  // Nothing to undo: the program did not change.
  await expect(page.locator(".pg-toast").getByRole("button", { name: "Undo" })).toHaveCount(0);
  await expect(page.getByRole("region", { name: "Question" })).toHaveCount(0);
  expect((await read(page, project)).program.version).toBe(before.program.version);
});
