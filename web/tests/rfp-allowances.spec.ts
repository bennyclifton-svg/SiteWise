import { expect, test, type Page } from "@playwright/test";

// Allowances and transmittal (owner direction, 8 November 2026): allowances
// in two pairs of columns with plain quantity cells and no Edit wording
// button; the transmittal's buttons in its blue band with the table under it.
const origin = { Origin: "http://127.0.0.1:4173" };

async function architectRFP(page: Page) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Allowances");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  for (const [key, value] of [["hdr.work_type", "extend"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: profile.parts[0].id, value } });
    expect(r.ok(), `${key}: ${await r.text()}`).toBeTruthy();
  }
  expect((await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Architect", discipline_id: "consultant.architect" } })).ok()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Architect", exact: true }).click();
  await expect(page.locator(".rfp-stage-scope")).toHaveCount(1);
}

test("allowances in two pairs, plain quantities; transmittal buttons in its band", async ({ page }) => {
  await page.setViewportSize({ width: 1500, height: 1000 });
  await architectRFP(page);
  const table = page.locator(".rfp-allowance-pairs");
  await expect(table).toHaveCount(1);
  await expect(table.locator("thead th")).toHaveText(["Item", "Qty", "Item", "Qty"]);
  const block = page.locator(".report-block", { has: table });
  // The paragraph stays; its Edit wording button goes.
  await expect(block.getByText(/^Price exactly these/)).toBeVisible();
  await expect(block.getByRole("button", { name: /^Edit wording/ })).toHaveCount(0);

  // Quantities: plain, wide enough for three digits, nudged in place.
  const bim = table.getByLabel("BIM level of detail quantity", { exact: true });
  await expect(bim).toHaveValue(/^[0-9]{3}$/);
  const fit = await bim.evaluate(el => { const i = el as HTMLInputElement; const c = getComputedStyle(i); return { s: i.scrollWidth, c: i.clientWidth, w: c.width, f: c.fontSize, p: c.paddingLeft }; });
  expect(fit.s, JSON.stringify(fit)).toBeLessThanOrEqual(fit.c);
  expect(await bim.evaluate(el => getComputedStyle(el).borderTopWidth)).toBe("0px");
  const presentations = table.getByLabel("Client presentations quantity", { exact: true });
  const before = Number(await presentations.inputValue());
  await Promise.all([
    page.waitForResponse(r => r.request().method() === "PUT" && r.url().endsWith("/settings/allowance%3Aclient_presentations")),
    table.getByRole("button", { name: "Increase Client presentations", exact: true }).click(),
  ]);
  await expect(presentations).toHaveValue(String(before + 1));

  // Rows as compact as the stage table's lines.
  const stage = page.locator("#rfp-stage-brief_concept");
  await stage.getByRole("button", { name: /^Brief and concept/ }).click();
  const lineHeight = (await stage.locator('tr[data-line="site_inspection"]').boundingBox())!.height;
  await page.locator(".rfp-allowance-pairs").scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../.tools/rfp-allowances.png" });
  const rowHeight = Math.min(...await table.locator("tbody tr").evaluateAll(rows => rows.map(r => r.getBoundingClientRect().height)));
  expect(Math.abs(rowHeight - lineHeight), `allowance ${rowHeight}, line ${lineHeight}`).toBeLessThanOrEqual(3);

  // Transmittal: no instruction; buttons inside the band; table right under.
  const section = page.locator(".report-section", { has: page.locator(".rfp-transmittal") });
  await expect(section.getByText(/Ctrl\/Cmd-click/)).toHaveCount(0);
  const band = (await section.locator(".section-band, .report-section-head, h3").first().boundingBox())!;
  const update = (await section.getByRole("button", { name: /Update Transmittal/ }).boundingBox())!;
  expect(update.y).toBeGreaterThanOrEqual(band.y - 1);
  expect(update.y + update.height).toBeLessThanOrEqual(band.y + Math.max(band.height, 36) + 1);
  const head = (await section.locator(".rfp-transmittal-table thead").boundingBox())!;
  expect(head.y - (band.y + band.height)).toBeLessThanOrEqual(2);
  await page.locator(".rfp-allowance-pairs").scrollIntoViewIfNeeded();
  await page.screenshot({ path: "../.tools/rfp-allowances.png" });
});
