import { expect, test, type Locator, type Page } from "@playwright/test";
import { dropTender, freshOwner, headers, modelSwitch, openPrices, tenderPackage, tenderTable, type TenderJSON, type TenderPackage, waitRead, whole } from "./tender-helpers";

// Editing the first table: every edit is a saved operation that calls no
// model, keeps the column equal to the tender sum, and can be undone.
// Every firm and figure here is invented.

function column(t: TenderJSON, sub: string) {
  return t.columns.find(c => c.submission_id === sub)!;
}
const cents = (v: string | null | undefined) => (v ? Math.round(Number(v) * 100) : 0);
function cellCents(t: TenderJSON, sub: string, row: string) {
  return cents(column(t, sub).cells[row]?.amount);
}
function systemRow(t: TenderJSON) {
  return t.rows.find(r => r.system)?.key ?? "tender:not_broken_down";
}
function total(t: TenderJSON, sub: string) {
  return Object.values(column(t, sub).cells).reduce((sum, c) => sum + cents(c.amount), 0);
}
const esc = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

async function setup(page: Page, tag: string): Promise<{ p: TenderPackage; a: string; b: string; table: Locator }> {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await freshOwner(page, tag);
  await modelSwitch(page, true);
  const p = await tenderPackage(page, "Tender edit");
  const [r1, r2, r3] = p.rows;
  const a = await dropTender(page, p, "Alder Planning", 21, [[r1.label, 1000], [r2.label, 2000], [r3.label, 3000]]);
  const b = await dropTender(page, p, "Birch Planning", 22, [[r1.label, 1500], [r2.label, 2500]]);
  await waitRead(page, p, [a, b]);
  await openPrices(page, p);
  const table = page.getByRole("table", { name: "As tendered, ex GST" });
  await expect(table).toBeVisible();
  return { p, a, b, table };
}

/** Opens the lines behind Alder Planning's cell on a row, if they are not open already. */
async function showLines(page: Page, table: Locator, row: string): Promise<Locator> {
  const region = page.getByRole("region", { name: `Alder Planning: lines in ${row}` });
  if (!(await region.isVisible())) await table.getByRole("button", { name: new RegExp(`^${esc(row)}, Alder Planning: .*Show lines$`) }).click();
  await expect(region).toBeVisible();
  return region;
}

test("move, correct, split, add and undo lines; the tender sum never moves by itself", async ({ page }) => {
  const { p, a, table } = await setup(page, "tender-edit");
  const [r1, r2, r3] = p.rows;
  const cell = (row: string, amount: number) => table.getByRole("button", { name: `${row}, Alder Planning: ${whole(amount)}. Show lines` });
  const sum = table.getByLabel("Tender sum, Alder Planning", { exact: true });
  const undo = page.getByRole("button", { name: "Undo", exact: true });
  await expect(undo).toBeDisabled();

  // 1. Right-click a line and move it to another row: two cells change, the sum does not.
  let lines = await showLines(page, table, r1.label);
  await lines.getByRole("row", { name: new RegExp(esc(r1.label)) }).locator(".tender-line-label").click({ button: "right" });
  await page.getByRole("menuitem", { name: `Move to ${r2.label}` }).click();
  await expect(cell(r2.label, 3000)).toBeVisible();
  await expect(sum).toHaveText(whole(6000));
  let t = await tenderTable(page, p);
  expect(cellCents(t, a, r1.key)).toBe(0);
  expect(cellCents(t, a, r2.key)).toBe(300000);
  expect(total(t, a)).toBe(600000);
  await expect(undo).toBeEnabled();

  // 2. Undo: the table is as it was one step before.
  await undo.click();
  await expect(cell(r1.label, 1000)).toBeVisible();
  await expect(cell(r2.label, 2000)).toBeVisible();

  // 3. Select a line, press M and choose "Not broken down" with the keyboard.
  lines = await showLines(page, table, r1.label);
  await lines.getByRole("row", { name: new RegExp(esc(r1.label)) }).locator(".tender-line-label").click();
  await page.keyboard.press("m");
  const menu = page.getByRole("menu", { name: `Actions for ${r1.label}` });
  await expect(menu).toBeVisible();
  await menu.getByRole("menuitem", { name: "Move to Not broken down" }).focus();
  await page.keyboard.press("Enter");
  await expect(table.getByRole("button", { name: `Not broken down, Alder Planning: ${whole(1000)}. Show lines` })).toBeVisible();
  t = await tenderTable(page, p);
  expect(cellCents(t, a, systemRow(t))).toBe(100000);
  expect(total(t, a)).toBe(600000);
  await undo.click();
  await expect(cell(r1.label, 1000)).toBeVisible();

  // 4. Correct a line's amount: "Not broken down" takes the difference, the sum stays.
  lines = await showLines(page, table, r3.label);
  await lines.getByRole("row", { name: new RegExp(esc(r3.label)) }).locator(".tender-line-label").dblclick();
  const editor = page.getByRole("group", { name: `Correct ${r3.label}` });
  await editor.getByLabel("Amount").fill("2500");
  await editor.getByRole("button", { name: "Save" }).click();
  await expect(cell(r3.label, 2500)).toBeVisible();
  await expect(sum).toHaveText(whole(6000));
  t = await tenderTable(page, p);
  expect(cellCents(t, a, systemRow(t))).toBe(50000);
  expect(total(t, a)).toBe(600000);

  // 5. Split a line across two rows: both parts show, the column total is unchanged.
  lines = await showLines(page, table, r2.label);
  await lines.getByRole("row", { name: new RegExp(esc(r2.label)) }).locator(".tender-line-label").click({ button: "right" });
  await page.getByRole("menuitem", { name: "Split…" }).click();
  const split = page.getByRole("dialog", { name: "Split line" });
  await split.getByLabel("Part 1 amount").fill("500");
  await split.getByLabel("Part 1 row").selectOption({ label: r1.label });
  await split.getByLabel("The rest goes to").selectOption({ label: r2.label });
  await split.getByRole("button", { name: "Split", exact: true }).click();
  await expect(cell(r1.label, 1500)).toBeVisible();
  await expect(cell(r2.label, 1500)).toBeVisible();
  t = await tenderTable(page, p);
  expect(total(t, a)).toBe(600000);

  // 6. Set the tender sum: a double-click opens it (no Correct button, owner
  // 15 Nov 2026), and it then says it is the manager's.
  await sum.dblclick();
  const sumEditor = page.getByRole("group", { name: "Correct the tender sum" });
  await sumEditor.getByLabel("Tender sum", { exact: true }).fill("6500");
  await sumEditor.getByRole("button", { name: "Save" }).click();
  await expect(sum).toHaveText(whole(6500));
  await expect(table.getByText("set by you")).toBeVisible();
  t = await tenderTable(page, p);
  expect(total(t, a)).toBe(650000);

  // 7. Add a line, then delete it.
  lines = await showLines(page, table, r1.label);
  await lines.getByRole("button", { name: "Add a line for Alder Planning" }).click();
  await lines.getByLabel("New line wording").fill("Site visit");
  await lines.getByLabel("New line amount").fill("250");
  await lines.getByLabel("New line amount").press("Enter");
  await expect(cell(r1.label, 1750)).toBeVisible();
  await lines.getByRole("button", { name: "Actions for Site visit" }).click();
  await page.getByRole("menuitem", { name: "Delete" }).click();
  await expect(cell(r1.label, 1500)).toBeVisible();

  // 8. Undo three times: the delete, the add, then the tender sum.
  await undo.click();
  await expect(cell(r1.label, 1750)).toBeVisible();
  await undo.click();
  await expect(cell(r1.label, 1500)).toBeVisible();
  await undo.click();
  await expect(sum).toHaveText(whole(6000));
  await expect(table.getByText("set by you")).toHaveCount(0);
});

test("an edit made elsewhere reloads the table and says so", async ({ page }) => {
  // Without the event stream this screen does not hear of the other edit,
  // as when the two land within the same moment.
  await page.route(/\/api\/events/, route => route.abort());
  const { p, a, table } = await setup(page, "tender-stale");
  const [r1, r2] = p.rows;
  const lines = await showLines(page, table, r1.label);

  // Someone else edits the same package first: the version moves on.
  const before = await tenderTable(page, p);
  const line = column(before, a).lines.find(l => l.counted && l.row === r2.key)!;
  const res = await page.request.post(`${p.base}/tender/ops`, { headers, data: { version: before.version, ops: [{ op: "move_line", line: line.id, row: "" }] } });
  expect(res.ok()).toBeTruthy();

  await lines.getByRole("row", { name: new RegExp(esc(r1.label)) }).locator(".tender-line-label").click({ button: "right" });
  await page.getByRole("menuitem", { name: `Move to ${r2.label}` }).click();
  await expect(page.getByText("This table changed. It has been reloaded.")).toBeVisible();
  // The other edit shows; this one was not applied.
  await expect(table.getByRole("button", { name: `Not broken down, Alder Planning: ${whole(2000)}. Show lines` })).toBeVisible();
  const after = await tenderTable(page, p);
  expect(cellCents(after, a, r1.key)).toBe(100000);
});
