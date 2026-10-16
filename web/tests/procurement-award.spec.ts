import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

// Drops synthetic proposal PDFs on a package row the way a PM drags files.
async function dropOnRow(page: Page, row: string, files: { name: string; url: string }[]) {
  const blobs = await Promise.all(files.map(async f => ({ name: f.name, b64: (await (await page.request.get(f.url)).body()).toString("base64") })));
  await page.getByRole("row", { name: new RegExp(`^${row}`) }).evaluate((el, files) => {
    const dt = new DataTransfer();
    for (const f of files) dt.items.add(new File([Uint8Array.from(atob(f.b64), c => c.charCodeAt(0))], f.name, { type: "application/pdf" }));
    for (const type of ["dragenter", "dragover", "drop"]) el.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
  }, blobs);
}

test("three proposals dropped on a package compare, a corrected cell moves the totals, and award writes the cost plan", async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Award flow");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const pkg = await (await page.request.post(`/api/projects/${project}/procurement/rows`, { headers: origin, data: { discipline_id: "consultant.town_planner" } })).json();
  // The package's schedule names the rows a proposal is read into.
  const schedule = await (await page.request.get(`/api/projects/${project}/procurement/packages/${pkg.id}/prices`)).json();
  const stages: string[] = schedule.items.filter((i: { kind: string; base: boolean }) => i.base).map((i: { label: string }) => i.label);
  expect(stages.length).toBeGreaterThan(3);
  const n = stages.length;
  const dollars = (c: number) => `$${(c / 100).toLocaleString("en-AU", { minimumFractionDigits: 2 })}`;
  const proposal = (firm: string, abn: number, fee: (i: number) => number, extra = "") =>
    `/__e2e/proposal.pdf?firm=${encodeURIComponent(firm)}&abn=${abn}&lines=${encodeURIComponent(stages.map((s, i) => `${s}~${fee(i)}`).join(","))}&rates=${encodeURIComponent("Director~45000")}${extra}`;

  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await expect(page.getByRole("row", { name: /^Town Planner/ }).getByText("Drop a proposal here")).toBeVisible();
  await dropOnRow(page, "Town Planner", [
    { name: "Fee proposal (Alpha Planning).pdf", url: proposal("Alpha Planning Pty Ltd", 1, () => 100000) },
    { name: "Fee proposal (Beta Planning).pdf", url: proposal("Beta Planning Pty Ltd", 2, i => i === 1 ? -1 : 90000) },
    { name: "Fee proposal (Gamma Planning).pdf", url: proposal("Gamma Planning Pty Ltd", 3, () => 105000, `&extra=${encodeURIComponent("Heritage statement~120000")}`) },
  ]);
  const row = page.getByRole("row", { name: /^Town Planner/ });
  await expect(row.locator(".firm-chip")).toHaveCount(3, { timeout: 20_000 });
  await expect(row.getByText(dollars(n * 100000).replace(".00", ""))).toBeVisible({ timeout: 20_000 });
  await row.getByRole("button", { name: "Compare Town Planner" }).click();

  // One table: firms across, schedule down; the lowest complete total is marked,
  // and the cheaper-looking total with an item not priced is not.
  const table = page.locator(".prices-table");
  await expect(table.locator("thead th")).toHaveCount(4);
  const total = table.locator("tr.prices-total");
  await expect(total).toContainText(dollars(n * 100000));
  await expect(total).toContainText("1 item not priced");
  await expect(total.locator("td.lowest")).toContainText(dollars(n * 100000));
  await expect(table.locator("tr.prices-others")).toContainText("Heritage statement");

  // Correct one of Gamma's cells: the user's word is final and the totals follow.
  await table.getByRole("button", { name: new RegExp(`^${stages[0]}, Gamma Planning Pty Ltd`) }).click();
  await table.getByLabel("Amount ex GST").fill("100");
  await table.getByRole("button", { name: "Save", exact: true }).click();
  const gammaTotal = (n - 1) * 105000 + 10000;
  await expect(total.locator("td.lowest")).toContainText(dollars(gammaTotal));
  await page.screenshot({ path: "../.tools/prices-compare.png", fullPage: true });

  // Award Gamma: combine two stages into one line, review, confirm.
  const gamma = table.locator("thead th").nth(3);
  await gamma.getByRole("button", { name: "Award" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("Step 1 of 2")).toBeVisible();
  await dialog.getByLabel(`Select ${stages[1]} to combine`).check();
  await dialog.getByLabel(`Select ${stages[2]} to combine`).check();
  await dialog.getByLabel("Name for combined line").fill("Lodgement documents");
  await dialog.getByRole("button", { name: /Combine 2 selected/ }).click();
  await expect(dialog.getByText("To be written:")).toContainText(dollars(gammaTotal));
  await dialog.getByRole("button", { name: "Next: review" }).click();
  await expect(dialog.getByText("Lodgement documents")).toBeVisible();
  await page.screenshot({ path: "../.tools/award-review.png" });
  await dialog.getByRole("button", { name: "Award and write to cost plan" }).click();
  await expect(page.getByText(/Awarded to Gamma Planning/)).toBeVisible();
  await expect(gamma).toContainText("Awarded");
  await expect(table.locator("thead th").nth(1)).toContainText("Unsuccessful");

  // The cost plan shows the group and its two lines without a reload.
  await page.getByRole("button", { name: "Open cost plan" }).click();
  await expect(page.getByRole("button", { name: /^Town Planner: Gamma Planning/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Lodgement documents", exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/award-cost-plan.png", fullPage: true });

  // Undo before any invoice restores the plan and the submissions.
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  // Procurement keeps its place: the price table is still open.
  await page.getByRole("button", { name: "Undo award" }).click();
  await expect(page.locator(".prices-table thead th").nth(3).getByRole("button", { name: "Award" })).toBeVisible();
  const plan = await (await page.request.get(`/api/projects/${project}/cost-plan`)).json();
  expect(plan.items.some((i: { label: string }) => i.label === "Lodgement documents")).toBeFalsy();
});
