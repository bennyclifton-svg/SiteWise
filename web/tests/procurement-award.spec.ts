import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

test("a proposal attached to an existing firm keeps one column and appears in Documents", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Proposal attachment");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const pkg = await (await page.request.post(`/api/projects/${project}/procurement/rows`, { headers: origin, data: { discipline_id: "consultant.town_planner" } })).json();
  const base = `/api/projects/${project}/procurement/packages/${pkg.id}`;
  const initial = await (await page.request.post(`${base}/firms`, { headers: origin, data: { firm: { name: "Chosen Planning" } } })).json();
  const sub = initial.columns[0].submission.id;
  const lines = initial.items.filter((i: { base: boolean }) => i.base).map((i: { label: string }) => `${i.label}~100000`).join(",");
  const pdf = await (await page.request.get(`/__e2e/proposal.pdf?firm=Printed%20Planning&abn=1&lines=${encodeURIComponent(lines)}`)).body();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("row", { name: /^Town Planner/ }).getByRole("button", { name: /(?:Compare|Review) Town Planner/ }).click();
  await expect(page.locator(".prices-total")).toContainText("Unknown");
  await page.getByLabel("Upload proposal for Chosen Planning").setInputFiles({ name: "Fee proposal.pdf", mimeType: "application/pdf", buffer: pdf });
  await expect(page.getByLabel("Upload proposal for Chosen Planning")).toHaveCount(0);
  await expect(page.locator(".consultant-comparison")).toBeVisible({ timeout: 20_000 });
  await expect(page.locator(".consultant-comparison .prices-total")).toContainText("Not confirmed");
  const table = await (await page.request.get(`${base}/prices`)).json();
  expect(table.columns).toHaveLength(1);
  expect(table.columns[0].submission.id).toBe(sub);
  expect(table.columns[0].firm.name).toBe("Chosen Planning");
  const doc = await page.request.get(`/api/documents/${table.columns[0].submission.document_id}`);
  expect(doc.ok()).toBeTruthy();
  expect((await doc.json()).filename).toBe("Fee proposal.pdf");
  await page.getByRole("button", { name: "Show documents" }).click();
  await expect(page.getByRole("complementary", { name: "Document register" }).getByRole("checkbox", { name: "Select Fee proposal", exact: true })).toBeVisible();
});

// Drops synthetic proposal PDFs on a package row the way a PM drags files.
async function dropOnRow(page: Page, row: string, files: { name: string; url: string }[]) {
  const blobs = await Promise.all(files.map(async f => ({ name: f.name, b64: (await (await page.request.get(f.url)).body()).toString("base64") })));
  await page.getByRole("row", { name: new RegExp(`^${row}`) }).evaluate((el, files) => {
    const dt = new DataTransfer();
    for (const f of files) dt.items.add(new File([Uint8Array.from(atob(f.b64), c => c.charCodeAt(0))], f.name, { type: "application/pdf" }));
    for (const type of ["dragenter", "dragover", "drop"]) el.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
  }, blobs);
}

test("quoted fees stay separate from scope, a confirmed appointment awards and undo restores the plan", async ({ page }) => {
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
  await expect(row.getByRole("button", { name: "Compare Town Planner" })).toBeEnabled({ timeout: 20_000 });
  await row.getByRole("button", { name: "Compare Town Planner" }).click();

  // The source offers are visible before any RFP allocation or confirmation.
  const table = page.locator(".consultant-comparison .prices-table");
  await expect(table.locator("thead th")).toHaveCount(4);
  const total = table.locator("tr.prices-total");
  await expect(total).toContainText("Not confirmed");
  await expect(table.locator("td.lowest")).toHaveCount(0);
  await expect(table.getByRole("row", { name: /^Quoted packages and totals/ })).toContainText("$");
  await expect(table).toContainText("Heritage statement");

  // Confirm one appointment amount with its scope; no source amounts are added.
  const gammaTotal = (n - 1) * 105000 + 10000;
  await total.locator("td").nth(2).getByRole("button", { name: "Confirm amount and scope" }).click();
  await page.getByLabel("Amount excluding GST").fill(String(gammaTotal / 100));
  await page.getByLabel("Scope and source").fill("Planning appointment: advice and lodgement documents; excludes heritage statement.");
  await page.getByRole("button", { name: "Confirm fee", exact: true }).click();
  await expect(total.locator("td").nth(2)).toContainText(dollars(gammaTotal));
  await expect(total.locator("td").nth(0)).toContainText("Not confirmed");
  await page.screenshot({ path: "../.tools/prices-compare.png", fullPage: true });

  // Only the confirmed appointment can become a contract value.
  await total.locator("td").nth(2).getByRole("button", { name: "Award", exact: true }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByText("Step 1 of 2")).toBeVisible();
  await expect(dialog.getByText("To be written:")).toContainText(dollars(gammaTotal));
  await dialog.getByRole("button", { name: "Next: review" }).click();
  await expect(dialog.getByText("Confirmed comparison fee", { exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/award-review.png" });
  await dialog.getByRole("button", { name: "Award and write to cost plan" }).click();
  await expect(page.getByText(/Awarded to Gamma Planning/)).toBeVisible();

  // The one-page report: price, non-price with the PM's ratings, and a
  // recommendation drafted from the figures that the PM edits and signs off.
  await page.getByRole("button", { name: "Report", exact: true }).click();
  await expect(page.getByRole("heading", { name: /Town Planner: comparison of 3 proposals/ })).toBeVisible();
  await expect(page.getByLabel("Recommendation")).toHaveValue(/Gamma Planning Pty Ltd.*confirmed comparison fee/i);
  await expect(page.getByText("Authority fees are excluded.").first()).toBeVisible();
  await page.getByLabel("Rating for Team, Gamma Planning Pty Ltd").selectOption("strong");
  await expect(page.getByLabel("Rating for Team, Gamma Planning Pty Ltd")).toHaveValue("strong");
  await expect(page.getByRole("button", { name: "Save and sign off" })).toBeDisabled();
  await page.getByLabel("Recommendation").fill("Appoint Gamma Planning for the recorded scope and confirmed fee.");
  await page.getByRole("button", { name: "Save and sign off" }).click();
  await expect(page.getByText(/Signed off by/)).toBeVisible();
  await page.screenshot({ path: "../.tools/procurement-report.png", fullPage: true });
  await page.emulateMedia({ media: "print" });
  await page.screenshot({ path: "../.tools/procurement-report-print.png", fullPage: true });
  await page.emulateMedia({ media: "screen" });
  await page.getByRole("button", { name: "← Price table" }).click();

  // The cost plan shows only the confirmed appointment.
  await page.getByRole("button", { name: "Open cost plan" }).click();
  await expect(page.getByRole("button", { name: /^Town Planner: Gamma Planning/ })).toBeVisible();
  await expect(page.getByRole("button", { name: "Confirmed comparison fee", exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/award-cost-plan.png", fullPage: true });

  // Undo before any invoice restores the plan and the submissions.
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  // Procurement keeps its place: the price table is still open.
  await page.getByRole("button", { name: "Undo award" }).click();
  await expect(total.locator("td").nth(2).getByRole("button", { name: "Award", exact: true })).toBeVisible();
  const plan = await (await page.request.get(`/api/projects/${project}/cost-plan`)).json();
  expect(plan.items.some((i: { label: string }) => i.label === "Confirmed comparison fee")).toBeFalsy();
});
