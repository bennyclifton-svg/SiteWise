import { expect, test } from "@playwright/test";
import { dropTender, freshOwner, modelSwitch, money, openPrices, tenderPackage, tenderTable, waitRead, whole } from "./tender-helpers";

// The first table: each firm's column equals its printed tender sum, the
// lines behind a cell can be seen, and Award sits beside the tender sum with
// no separate "checked" step. Every firm and figure here is invented.
test("as tendered: columns equal the tender sum, lines sit behind cells, and Award sits beside the sum", async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 });
  await freshOwner(page, "tender-table");
  await modelSwitch(page, true);
  const p = await tenderPackage(page, "Tender table");
  const [r1, r2, r3] = p.rows;
  const a = await dropTender(page, p, "Alder Planning", 11, [[r1.label, 1000], [r2.label, 2000], [r3.label, 3000]]);
  const b = await dropTender(page, p, "Birch Planning", 12, [[r1.label, 1500], [r2.label, 2500], ["Zebra crossing study", 500]]);
  const read = await waitRead(page, p, [a, b]);

  // In the data: each column's priced cells add up to its tender sum.
  for (const col of read.columns) {
    const cents = Object.values(col.cells).reduce((sum, c) => sum + (c.amount ? Math.round(Number(c.amount) * 100) : 0), 0);
    expect(cents).toBe(Math.round(Number(col.tender_sum) * 100));
  }

  await openPrices(page, p);
  const table = page.getByRole("table", { name: "As tendered, ex GST" });
  await expect(table).toBeVisible();
  await expect(table.getByLabel("Tender sum, Alder Planning", { exact: true })).toHaveText(whole(6000));
  await expect(table.getByLabel("Tender sum, Birch Planning", { exact: true })).toHaveText(whole(4500));
  // Whole dollars by default; Decimals in the ⋮ menu shows cents, and is remembered.
  await page.getByRole("button", { name: "More actions", exact: true }).click();
  await page.getByRole("menuitemcheckbox", { name: "Decimals" }).click();
  await expect(table.getByLabel("Tender sum, Alder Planning", { exact: true })).toHaveText(money(6000));
  await openPrices(page, p);
  await expect(table.getByLabel("Tender sum, Alder Planning", { exact: true })).toHaveText(money(6000));
  await page.getByRole("button", { name: "More actions", exact: true }).click();
  await page.getByRole("menuitemcheckbox", { name: "Decimals" }).click();
  await expect(table.getByLabel("Tender sum, Alder Planning", { exact: true })).toHaveText(whole(6000));
  // A filled column needs no "Received"; Read again waits beside the cross.
  await expect(table.getByRole("columnheader", { name: /Alder Planning/ })).not.toContainText("Received");
  await expect(table.getByRole("button", { name: "Read prices again for Alder Planning" })).toHaveCount(1);
  // The line that matches no row is "Not broken down" for the second firm.
  await expect(table.getByRole("button", { name: `Not broken down, Birch Planning: ${whole(500)}. Show lines` })).toBeVisible();

  // The firm's own line sits behind the cell, with its page.
  await table.getByRole("button", { name: `${r1.label}, Alder Planning: ${whole(1000)}. Show lines` }).click();
  const lines = page.getByRole("region", { name: `Alder Planning: lines in ${r1.label}` });
  await expect(lines).toContainText(r1.label);
  await expect(lines.getByRole("button", { name: "p. 1" })).toBeVisible();
  // An open row shows every firm's lines at once, with no column heads.
  await expect(page.getByRole("region", { name: `Birch Planning: lines in ${r1.label}` })).toContainText(r1.label);
  await expect(lines.getByRole("columnheader")).toHaveCount(0);
  // Opening another row by its chevron leaves the first open; the chevron closes it again.
  await table.getByRole("button", { name: `Show lines for ${r2.label}` }).click();
  await expect(page.getByRole("region", { name: `Alder Planning: lines in ${r2.label}` })).toBeVisible();
  await expect(lines).toBeVisible();
  await table.getByRole("button", { name: `Hide lines for ${r2.label}` }).click();
  await expect(page.getByRole("region", { name: `Alder Planning: lines in ${r2.label}` })).toHaveCount(0);
  await expect(lines).toBeVisible();

  // Award sits beside the tender sum and is ready at once: pressing it is the
  // manager's acceptance (owner, 15 Nov 2026). The head has no date and no check.
  await expect(table.getByRole("button", { name: /checked/i })).toHaveCount(0);
  await expect(table.getByRole("columnheader", { name: /Alder Planning/ }).getByRole("button", { name: "Award", exact: true })).toHaveCount(0);
  await expect(table.getByRole("cell").filter({ has: page.getByLabel("Tender sum, Alder Planning", { exact: true }) }).getByRole("button", { name: "Award", exact: true })).toBeEnabled();
  await expect(table.getByRole("columnheader", { name: /Alder Planning/ })).not.toContainText(/\d{4}/);

  // Like lines sit level across the firms: each firm's r1 line is in the same slot.
  const birchLines = page.getByRole("region", { name: `Birch Planning: lines in ${r1.label}` });
  const yA = (await lines.getByRole("row").first().boundingBox())!.y;
  const yB = (await birchLines.getByRole("row").first().boundingBox())!.y;
  expect(Math.abs(yA - yB)).toBeLessThan(1);
  await page.screenshot({ path: "../.tools/tender-table.png", fullPage: true });

  // A line moved down a place (Alt+Down, the grip's keyboard move) is saved
  // for everyone; the other firm's line stays where it was.
  const slotOf = async (sub: string, label: string) => (await tenderTable(page, p)).columns.find(c => c.submission_id === sub)!.lines.find(l => l.label === label)!.slot;
  await lines.getByRole("row").first().focus();
  await page.keyboard.press("Alt+ArrowDown");
  await expect.poll(() => slotOf(a, r1.label)).toBe(1);
  expect(await slotOf(b, r1.label)).toBe(0);
  await expect(lines.locator("tr.tender-slot-empty")).toHaveCount(1);
  // By mouse: drag the line back up into the empty place.
  const from = (await lines.getByRole("row", { name: new RegExp(r1.label) }).boundingBox())!;
  const into = (await lines.locator("tr.tender-slot-empty").boundingBox())!;
  await page.mouse.move(from.x + 30, from.y + from.height / 2);
  await page.mouse.down();
  await page.mouse.move(from.x + 34, from.y + from.height / 2 - 6, { steps: 5 });
  await page.mouse.move(into.x + 30, into.y + into.height / 2, { steps: 10 });
  await page.mouse.up();
  await expect.poll(() => slotOf(a, r1.label)).toBe(0);

  // A firm's head moves left or right by its grip, for everyone.
  const heads = async () => (await table.locator("thead th.prices-firm .prices-firm-name").allTextContents()).map(s => s.replace(/new$/, "").trim());
  expect(await heads()).toEqual(["Alder Planning", "Birch Planning"]);
  await table.getByRole("button", { name: /^Move Alder Planning/ }).focus();
  await page.keyboard.press("Alt+ArrowRight");
  await expect.poll(heads).toEqual(["Birch Planning", "Alder Planning"]);
  await openPrices(page, p);
  await expect.poll(heads).toEqual(["Birch Planning", "Alder Planning"]);
  // By mouse: the grip shows on hover and is dragged onto the other firm's
  // head. (The table is wider than its panel here, so start from its left.)
  await table.evaluate(t => { t.closest(".prices-scroll")!.scrollLeft = 0; });
  await table.getByRole("columnheader", { name: /Birch Planning/ }).hover();
  const grip = (await table.getByRole("button", { name: /^Move Birch Planning/ }).boundingBox())!;
  const onto = (await table.getByRole("columnheader", { name: /Alder Planning/ }).locator(".prices-firm-name").boundingBox())!;
  await page.mouse.move(grip.x + 6, grip.y + 12);
  await page.mouse.down();
  await page.mouse.move(grip.x + 20, grip.y + 14, { steps: 5 });
  await page.mouse.move(onto.x + 8, onto.y + onto.height / 2, { steps: 10 });
  await page.mouse.up();
  await expect.poll(heads).toEqual(["Alder Planning", "Birch Planning"]);

  // The cross in a firm's head detaches its proposal, as on the Procurement
  // board: the column leaves the comparison and the filed document stays.
  await table.getByRole("columnheader", { name: /Birch Planning/ }).hover();
  await table.getByRole("button", { name: "Detach proposal for Birch Planning" }).click();
  await expect(table.getByRole("columnheader", { name: /Birch Planning/ })).toHaveCount(0);
  await expect(table.getByRole("columnheader", { name: /Alder Planning/ })).toBeVisible();
  expect((await tenderTable(page, p)).columns.map(c => c.submission_id)).not.toContain(b);
  // With one firm left the table still has room for three: two spare columns.
  await expect(table.locator("thead th.tender-spare")).toHaveText(["Firm 2", "Firm 3"]);
  await expect(table.locator("td[data-band]")).toHaveCount(0);
});

test("with the switch off a dropped tender shows the old screen and nothing is read by the model", async ({ page }) => {
  await freshOwner(page, "tender-off");
  const p = await tenderPackage(page, "Tender switch off");
  await dropTender(page, p, "Cedar Planning", 13, [[p.rows[0].label, 1000]]);
  await expect.poll(async () => (await (await page.request.get(`${p.base}/prices`)).json()).columns[0]?.submission.status, { timeout: 30_000 }).not.toBe("reading");
  await openPrices(page, p);
  await expect(page.getByRole("region", { name: "Price table" })).toContainText("Cedar Planning");
  await expect(page.getByRole("table", { name: "As tendered, ex GST" })).toHaveCount(0);
  const t = await tenderTable(page, p);
  expect(t.on).toBe(false);
  expect(t.columns).toHaveLength(0);
});
