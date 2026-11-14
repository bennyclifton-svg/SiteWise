import { expect, test } from "@playwright/test";
import { dropTender, freshOwner, modelSwitch, money, openPrices, tenderPackage, tenderTable, waitRead } from "./tender-helpers";

// The first table: each firm's column equals its printed tender sum, the
// lines behind a cell can be seen, and a column is "to check" until the
// manager marks it checked. Every firm and figure here is invented.
test("as tendered: columns equal the tender sum, lines sit behind cells, and a column is checked before award", async ({ page }) => {
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
  await expect(table.getByLabel("Tender sum, Alder Planning", { exact: true })).toHaveText(money(6000));
  await expect(table.getByLabel("Tender sum, Birch Planning", { exact: true })).toHaveText(money(4500));
  // The line that matches no row is "Not broken down" for the second firm.
  await expect(table.getByRole("button", { name: `Not broken down, Birch Planning: ${money(500)}. Show lines` })).toBeVisible();

  // The firm's own line sits behind the cell, with its page.
  await table.getByRole("button", { name: `${r1.label}, Alder Planning: ${money(1000)}. Show lines` }).click();
  const lines = page.getByRole("region", { name: `Alder Planning: lines in ${r1.label}` });
  await expect(lines).toContainText(r1.label);
  await expect(lines.getByRole("button", { name: "p. 1" })).toBeVisible();

  // To check until the manager says so; award waits for it.
  await expect(table.locator('.tender-state[data-state="to-check"]')).toHaveCount(2);
  const award = table.getByRole("columnheader", { name: /Alder Planning/ }).getByRole("button", { name: "Award", exact: true });
  await expect(award).toBeDisabled();
  await table.getByRole("button", { name: "Mark Alder Planning checked" }).click();
  await expect(table.getByRole("button", { name: "Checked: Alder Planning. Un-check" })).toBeVisible();
  await expect(award).toBeEnabled();
  await openPrices(page, p);
  await expect(page.getByRole("button", { name: "Checked: Alder Planning. Un-check" })).toBeVisible();
  expect((await tenderTable(page, p)).columns.find(c => c.submission_id === a)?.checked).toBe(true);

  await page.screenshot({ path: "../.tools/tender-table.png", fullPage: true });
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
