import { test, expect } from "@playwright/test";

test("a proposal can move, detach and reattach without losing its firm or fee", async ({ page }) => {
  const headers = { Origin: "http://127.0.0.1:4173" };
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Proposal reassignment");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  const base = `/api/projects/${project}`;
  const addPackage = async (discipline_id: string) => {
    const r = await page.request.post(`${base}/procurement/rows`, { headers, data: { discipline_id } });
    expect(r.ok()).toBeTruthy(); return r.json();
  };
  const hydraulic = await addPackage("consultant.hydraulic");
  const civil = await addPackage("consultant.civil");
  const added = await page.request.post(`${base}/procurement/packages/${hydraulic.id}/firms`, { headers, data: { firm: { name: "Synthetic Flow Consultant" } } });
  expect(added.ok()).toBeTruthy();
  const initial = await added.json();
  const sub = initial.columns[0].submission;
  const fee = initial.items.find((i: { kind: string; base: boolean }) => i.kind === "stage" && i.base);
  expect(fee).toBeTruthy();
  expect((await page.request.put(`${base}/submissions/${sub.id}/cells/${encodeURIComponent(fee.key)}`, { headers, data: { state: "priced", amount: "4850", note: "Synthetic fee" } })).ok()).toBeTruthy();
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Review Hydraulic", exact: true }).click();
  await page.getByRole("button", { name: "Synthetic Flow Consultant", exact: true }).click();
  // Firm details are not a document: they take the right-hand column in place of the register.
  const panel = page.locator('#project-documents section[aria-label="Firm details"]');
  await expect(panel).toBeVisible();
  await expect(page.locator("#project-documents .reg-table")).toBeHidden();
  await panel.getByLabel("Package", { exact: true }).selectOption(civil.id);
  await page.screenshot({ path: "../.tools/proposal-move-desktop.png", fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: "../.tools/proposal-move-mobile.png", fullPage: true });
  await page.setViewportSize({ width: 1280, height: 900 });
  await panel.getByRole("button", { name: "Move proposal", exact: true }).click();
  await page.getByRole("button", { name: "Review Civil", exact: true }).click();
  await page.getByRole("button", { name: "Synthetic Flow Consultant", exact: true }).click();
  await panel.getByLabel("Package", { exact: true }).selectOption("");
  await panel.getByRole("button", { name: "Detach proposal", exact: true }).click();
  const waiting = page.locator("tr.row-unplaced");
  await expect(waiting).toHaveCount(1);
  await expect(waiting.locator(".firm-chip-name")).toHaveText("Synthetic Flow Consultant");
  await expect(waiting.getByRole("combobox")).toContainText("Choose package");
  await page.screenshot({ path: "../.tools/procurement-unplaced.png" });
  await waiting.getByRole("combobox").selectOption(civil.id);
  await expect(waiting).toHaveCount(0);
  const final = await (await page.request.get(`${base}/procurement/packages/${civil.id}/prices`)).json();
  const col = final.columns.find((c: { submission: { id: string } }) => c.submission.id === sub.id);
  expect(col.firm.name).toBe("Synthetic Flow Consultant");
  expect(col.submission.firm_id).toBe(sub.firm_id);
  expect([...Object.values(col.cells), ...col.others].some((c: any) => c.amount === "4850.00")).toBeTruthy();
  const timings: number[] = [];
  for (let i = 0; i < 20; i++) {
    const start = performance.now();
    const response = await page.request.patch(`${base}/submissions/${sub.id}`, { headers, data: { package_id: i % 2 ? civil.id : hydraulic.id } });
    timings.push(performance.now() - start);
    expect(response.ok()).toBeTruthy();
  }
  timings.sort((a,b) => a-b);
  console.log(`proposal move n=20 p50=${timings[9].toFixed(1)}ms p90=${timings[17].toFixed(1)}ms`);
  expect(timings[9]).toBeLessThanOrEqual(100);
  expect(timings[17]).toBeLessThanOrEqual(250);
});
