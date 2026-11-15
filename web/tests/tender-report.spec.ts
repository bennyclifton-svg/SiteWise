import { expect, test } from "@playwright/test";
import { dropTender, freshOwner, headers, modelSwitch, money, openPrices, tenderPackage, waitRead } from "./tender-helpers";

// The comparison report shows the tender sum, the applied adds and subs and
// the like-for-like total; it is signed off without a column check; and the
// two tables print. Every firm and figure is invented.
test("the report shows like-for-like totals, signs off, and the tables print", async ({ page }) => {
  await page.setViewportSize({ width: 1500, height: 1100 });
  await freshOwner(page, "tender-report");
  await modelSwitch(page, true);
  const p = await tenderPackage(page, "Tender report");
  const [r1, r2, r3] = p.rows;
  const a = await dropTender(page, p, "Alder Planning", 51, [[r1.label, 1000], [r2.label, 2000], [r3.label, 3000]], ["Extra: site meetings", 400]);
  const b = await dropTender(page, p, "Birch Planning", 52, [[r1.label, 1500], [r2.label, 2500], [r3.label, 2600]]);
  const t = await waitRead(page, p, [a, b]);
  // Apply A's extra through the same route the screen uses.
  const extra = t.adjustments.find(x => x.reason === "allowance")!;
  const res = await page.request.post(`${p.base}/tender/ops`, { headers, data: { version: t.version, ops: [{ op: "apply_adjustment", adjustment: extra.id, submission: a, applied: true }] } });
  expect(res.ok()).toBeTruthy();

  await openPrices(page, p);
  await expect(page.getByRole("table", { name: "As tendered, ex GST" })).toBeVisible();

  // Print: both tables, firms in columns, no lines panel, no buttons to press.
  await page.emulateMedia({ media: "print" });
  await expect(page.getByRole("table", { name: "As tendered, ex GST" })).toBeVisible();
  await expect(page.getByRole("table", { name: "Adds and subs" })).toBeVisible();
  // Award and the detach cross exist on screen and stay off the paper.
  await expect(page.locator(".prices-award")).toHaveCount(2);
  for (const btn of await page.locator(".prices-award, .prices-detach").all()) await expect(btn).toBeHidden();
  const pdf = await page.pdf({ format: "A4", landscape: true });
  const pages = (pdf.toString("latin1").match(/\/Type\s*\/Page[^s]/g) ?? []).length;
  expect(pages).toBeGreaterThanOrEqual(1);
  expect(pages).toBeLessThanOrEqual(3);
  await page.emulateMedia({ media: "screen" });

  // The report: tender sum, applied adds and subs, like for like.
  await page.getByRole("button", { name: "More actions", exact: true }).click();
  await page.getByRole("menuitem", { name: "Report" }).click();
  const report = page.getByRole("region", { name: "Comparison report" });
  const like = report.locator(".report-like-for-like");
  await expect(like.getByRole("row", { name: /^Tender sum/ })).toContainText(money(6000));
  await expect(like.getByRole("row", { name: /^Applied adds and subs/ })).toContainText(`+${money(400)} (1 row)`);
  await expect(like.getByRole("row", { name: /^Like for like/ })).toContainText(money(6400));
  await expect(like.getByRole("row", { name: /^Like for like/ })).toContainText(money(6600));
  await expect(like.getByRole("row", { name: /^Checked by the manager/ })).toHaveCount(0);

  // Sign-off needs no column check: signing is the acceptance.
  await report.getByLabel("Recommendation").fill("Appoint Alder Planning.");
  await report.getByRole("button", { name: "Save and sign off" }).click();
  await expect(report.getByRole("button", { name: "Signed off" })).toBeVisible();
});
