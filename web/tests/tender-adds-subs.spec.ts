import { expect, test } from "@playwright/test";
import { dropTender, freshOwner, modelSwitch, openPrices, tenderPackage, tenderTable, type TenderJSON, waitRead, whole } from "./tender-helpers";

// The second table: the system suggests, the manager applies, and the tender
// sum plus what is applied is the like-for-like total. The browser never
// adds. Every firm and figure here is invented.

const cents = (v: string | null | undefined) => (v ? Math.round(Number(v) * 100) : 0);
const col = (t: TenderJSON, sub: string) => t.columns.find(c => c.submission_id === sub)!;
const lfl = (t: TenderJSON, sub: string) => cents(col(t, sub).like_for_like);

test("adds and subs: suggestions, apply, a gap, the manager's own row, a line taken out, removal and undo", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1100 });
  await freshOwner(page, "tender-adds");
  await modelSwitch(page, true);
  const p = await tenderPackage(page, "Tender adds and subs");
  const [r1, r2, r3] = p.rows;
  // A prices every row and offers an option; B excludes a row; C states an extra.
  const a = await dropTender(page, p, "Alder Planning", 31, [[r1.label, 1000], [r2.label, 2000], [r3.label, 3000]], ["Option: epoxy grout", 1800]);
  const b = await dropTender(page, p, "Birch Planning", 32, [[r1.label, 1500], [r2.label, 2500], [`${r3.label} (excluded)`, -1]]);
  const c = await dropTender(page, p, "Cedar Planning", 33, [[r1.label, 1200], [r2.label, 2200], [r3.label, 4000]], ["Extra: asbestos removal", 700]);
  let t = await waitRead(page, p, [a, b, c]);

  // 1. Three suggestions, none applied: like for like equals each tender sum.
  expect(t.adjustments.map(x => x.reason).sort()).toEqual(["allowance", "gap", "option"]);
  expect(t.suggested).toBe(3);
  for (const column of t.columns) expect(cents(column.like_for_like)).toBe(cents(column.tender_sum));
  await openPrices(page, p);
  const adds = page.getByRole("region", { name: "Adds and subs" });
  await expect(adds.getByText("3 suggestions not applied")).toBeVisible();
  await expect(adds.getByLabel("Like for like, Alder Planning")).toHaveText(whole(6000));
  await expect(adds.getByLabel("Like for like, Birch Planning")).toHaveText(whole(4000));
  await expect(adds.getByLabel("Like for like, Cedar Planning")).toHaveText(whole(7400));
  // B left a row out, so the lower sum is not marked lowest; A is.
  expect(col(t, a).lowest).toBe(true);

  // 2. Apply C's extra, the firm's own figure: its like-for-like rises by it, and it stays applied.
  await adds.getByRole("button", { name: /^Apply Extra to the price: .*asbestos removal for Cedar Planning$/ }).click();
  await expect(adds.getByLabel("Like for like, Cedar Planning")).toHaveText(whole(8100));
  await expect(adds.getByText("2 suggestions not applied")).toBeVisible();
  await openPrices(page, p);
  await expect(page.getByRole("region", { name: "Adds and subs" }).getByLabel("Like for like, Cedar Planning")).toHaveText(whole(8100));

  // 3. B's gap: the manager chooses the highest of the others.
  const gapRow = adds.getByRole("row", { name: new RegExp(`Allowance for ${r3.label}`) });
  await expect(gapRow).toContainText(`Not priced. Others ${whole(3000)} to ${whole(4000)}`);
  await gapRow.getByRole("button", { name: /^Apply Allowance for .* for Birch Planning$/ }).click();
  await gapRow.getByRole("button", { name: "Use the highest" }).click();
  await expect(gapRow).toContainText(`+${whole(4000)}`);
  await expect(gapRow).toContainText("highest of the others");
  await expect(adds.getByLabel("Like for like, Birch Planning")).toHaveText(whole(8000));

  // 4. The manager's own row, with a typed subtraction for A.
  await adds.getByLabel("New add or sub").fill("Site shed");
  await adds.getByLabel("New add or sub").press("Enter");
  const shed = adds.getByRole("row", { name: /Site shed/ });
  await shed.getByRole("button", { name: "Type an amount for Alder Planning on Site shed" }).click();
  await shed.getByLabel("Amount, minus for a subtraction").fill("-5000");
  await shed.getByRole("button", { name: "Save" }).click();
  await expect(shed).toContainText(`−${whole(5000)}`);
  await expect(adds.getByLabel("Like for like, Alder Planning")).toHaveText(whole(1000));

  // 5. A tendered line taken out: a subtraction for that firm; the first table is unchanged.
  const table = page.getByRole("table", { name: "As tendered, ex GST" });
  await table.getByRole("button", { name: `${r2.label}, Alder Planning: ${whole(2000)}. Show lines` }).click();
  await page.getByRole("region", { name: `Alder Planning: lines in ${r2.label}` }).getByRole("row", { name: new RegExp(r2.label.split(" ")[0]) }).locator(".tender-line-label").click({ button: "right" });
  await page.getByRole("menuitem", { name: "Take out as a subtraction" }).click();
  await expect(adds.getByRole("row", { name: new RegExp(`Less: ${r2.label.split(" ")[0]}`) })).toContainText(`−${whole(2000)}`);
  await expect(table.getByLabel("Tender sum, Alder Planning", { exact: true })).toHaveText(whole(6000));
  await expect(adds.getByLabel("Like for like, Alder Planning")).toHaveText(`-${whole(1000)}`);
  t = await tenderTable(page, p);
  expect(cents(col(t, a).tender_sum)).toBe(600000);
  expect(lfl(t, a)).toBe(-100000);

  // 6. Remove A's option: it is gone, and still gone after the tender is read again.
  await adds.getByRole("button", { name: "Remove Option: epoxy grout" }).click();
  await expect(adds.getByRole("row", { name: /Option: epoxy grout/ })).toHaveCount(0);
  const version = (await tenderTable(page, p)).version;
  await table.getByRole("columnheader", { name: /Alder Planning/ }).getByRole("button", { name: "Read prices again for Alder Planning" }).click();
  await expect.poll(async () => {
    const now = await tenderTable(page, p);
    return now.version > version && col(now, a).state === "read";
  }, { timeout: 30_000 }).toBeTruthy();
  t = await tenderTable(page, p);
  expect(t.adjustments.some(x => x.label.startsWith("Option:"))).toBe(false);
  // The applied rows outlived the re-read.
  expect(lfl(t, a)).toBe(-100000);

  // 7. Undo: the removed suggestion is back.
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(adds.getByRole("row", { name: /Option: epoxy grout/ })).toBeVisible();

  // 8. The lowest mark sits on the lowest like-for-like total with no open gap.
  t = await tenderTable(page, p);
  const lowest = t.columns.filter(x => x.lowest);
  expect(lowest).toHaveLength(1);
  expect(lowest[0].submission_id).toBe(a);
  await expect(adds.getByRole("row", { name: /^Like for like/ })).toContainText("Lowest");
  await page.screenshot({ path: "../.tools/tender-adds-subs.png", fullPage: true });
});
