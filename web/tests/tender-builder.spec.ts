import { expect, test } from "@playwright/test";
import { dropTender, freshOwner, headers, modelSwitch, money, openPrices, tenderPackage, tenderTable, waitRead } from "./tender-helpers";

// A head contract with the model switched on is compared on the Main Works
// rows, in the same two tables, and is awarded from its checked column: no
// separate confirmation of a quoted total. Every firm and figure is invented.
test("a head contract is compared on the Main Works rows and awarded from its checked column", async ({ page }) => {
  await page.setViewportSize({ width: 1500, height: 1100 });
  await freshOwner(page, "tender-builder");
  await modelSwitch(page, true);
  const p = await tenderPackage(page, "Tender head contract", "trade.main_works");
  expect(p.rows.map(r => r.label)).toContain("Preliminaries and site establishment");
  expect(p.rows.some(r => r.key === "stage:lump_sum")).toBe(false);
  const six = p.rows.slice(0, 6);
  const a = await dropTender(page, p, "Alder Builders", 41, six.map((r, i) => [r.label, 10000 * (i + 1)] as [string, number]));
  const b = await dropTender(page, p, "Birch Builders", 42, six.map((r, i) => [r.label, 12000 * (i + 1)] as [string, number]));
  const read = await waitRead(page, p, [a, b]);
  const colA = read.columns.find(c => c.submission_id === a)!;
  expect(Math.round(Number(colA.tender_sum))).toBe(210000);

  await openPrices(page, p, "Main Works", "Trades");
  const table = page.getByRole("table", { name: "As tendered, ex GST" });
  await expect(table).toBeVisible();
  await expect(page.getByRole("table", { name: "Adds and subs" })).toBeVisible();
  await expect(table.getByLabel("Tender sum, Alder Builders", { exact: true })).toHaveText(money(210000));
  await expect(table.getByLabel("Tender sum, Birch Builders", { exact: true })).toHaveText(money(252000));
  // The old builder panels are not shown for a tender-read column.
  await expect(page.locator(".builder-quoted")).toHaveCount(0);

  // Award follows the checked state, not a confirmed lump sum.
  const award = table.getByRole("columnheader", { name: /Alder Builders/ }).getByRole("button", { name: "Award", exact: true });
  await expect(award).toBeDisabled();
  await table.getByRole("button", { name: "Mark Alder Builders checked" }).click();
  await expect(award).toBeEnabled();

  // Awarding every priced row writes cost plan lines that add to the tender sum.
  const t = await tenderTable(page, p);
  const col = t.columns.find(c => c.submission_id === a)!;
  const lines = t.rows.filter(r => col.cells[r.key]?.amount != null).map(r => ({ label: r.label, item_keys: [r.key] }));
  const prices = await (await page.request.get(`${p.base}/prices`)).json();
  const version = prices.columns.find((c: { submission: { id: string } }) => c.submission.id === a).submission.version;
  const preview = await page.request.post(`${p.base}/award/preview`, { headers, data: { submission_id: a, lines, total: "", version } });
  expect(preview.ok()).toBeTruthy();
  expect(Math.round(Number((await preview.json()).total))).toBe(210000);
  await award.click();
  await expect(page.getByRole("dialog")).toContainText(money(210000));
});
