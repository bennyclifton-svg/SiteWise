import { expect, test, type Page } from "@playwright/test";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

// Consent conditions in the register (docs/plans/2026-11-17-consent-conditions.md,
// Task 9) on the public Seven Hills fixture: Read conditions, the Kind
// filter, rows grouped by gate and "to check", an edit that accepts a row,
// evidence linked, and Add to RFP. Jev is the e2e server's stand-in; no
// live call, no model.

const headers = { Origin: "http://127.0.0.1:4173" };
const notice = readFileSync(resolve(import.meta.dirname, "../../internal/consent/testdata/seven-hills-notice.md"), "utf8");

// The register is Correspondence's first tab (25 November 2026).
const registerTab = (page: Page) => page.getByRole("group", { name: "Project views" }).getByRole("button", { name: "Correspondence" });
async function openRegister(page: Page) {
  await page.reload();
  await registerTab(page).click();
}

test("a notice is read into conditions grouped by gate; one is edited, evidenced and put on its RFP", async ({ page }) => {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Seven Hills conditions");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(registerTab(page)).toBeVisible();
  const project = new URL(page.url()).pathname.split("/").pop() as string;

  // The arborist's package and its RFP draft, so Add to RFP has somewhere to go.
  const pkg = await page.request.post(`/api/projects/${project}/packages`, { headers, data: { kind: "services", title: "Arborist", discipline_id: "consultant.arborist" } });
  expect(pkg.ok(), await pkg.text()).toBeTruthy();
  const pkgId = (await pkg.json()).id;
  const report = await page.request.post(`/api/projects/${project}/reports`, { headers, data: { kind: "rfp", package_id: pkgId } });
  expect(report.ok(), await report.text()).toBeTruthy();
  const draft = await page.request.post(`/api/reports/${(await report.json()).id}/draft`, { headers, data: { only_if_missing: true } });
  expect(draft.ok(), await draft.text()).toBeTruthy();

  for (const [title, kind, text] of [["Notice of determination DA-2025-0733", "statutory_instrument", notice], ["Tree protection certificate", "certificate", "Tree protection fencing installed and inspected by the project arborist."]]) {
    const filed = await page.request.post(`/__e2e/file-document?project=${project}&title=${encodeURIComponent(title)}&kind=${kind}`, { headers, data: text });
    expect(filed.ok(), await filed.text()).toBeTruthy();
  }
  expect((await page.request.post("/__e2e/run-consent")).ok()).toBeTruthy();

  // Read conditions from the register's menu; a document with none says so.
  await openRegister(page);
  await page.getByRole("button", { name: "Register actions", exact: true }).click();
  await page.getByRole("menuitem", { name: "Read conditions…" }).click();
  await page.getByLabel("Notice to read").selectOption({ label: "Tree protection certificate" });
  await page.getByRole("button", { name: "Read", exact: true }).click();
  await expect(page.locator(".register-list").getByRole("alert")).toContainText("No numbered conditions found");
  await page.getByRole("button", { name: "Register actions", exact: true }).click();
  await page.getByRole("menuitem", { name: "Read conditions…" }).click();
  await page.getByLabel("Notice to read").selectOption({ label: "Notice of determination DA-2025-0733" });
  await page.getByRole("button", { name: "Read", exact: true }).click();
  await expect(page.getByText("Notice of determination DA-2025-0733: Queued")).toBeVisible();
  expect((await page.request.post("/__e2e/run-consent")).ok()).toBeTruthy();

  await openRegister(page);
  await expect(page.getByLabel("Kind").locator("option")).toContainText(["All kinds", "Conditions · 7 to check"]);
  await page.getByLabel("Kind").selectOption({ label: "Conditions · 7 to check" });
  await expect(page.getByText("Read 7 of 7")).toBeVisible();
  await expect(page.locator("tr.register-group th")).toHaveText(["Before CC", "Before work starts", "During works", "Before OC", "Before SC"]);
  const tree = page.locator(".register-list tr", { hasText: "Condition 12" });
  await expect(tree.locator("th")).toHaveText("12");
  await expect(tree.locator('td[data-field="with"]')).toHaveText("Arborist");
  await expect(tree.locator('td[data-field="due"]')).toHaveText("Before work starts");
  await expect(tree.locator('td[data-field="status"]')).toHaveText("Open · evidence needed · to check");
  await expect(tree.locator('td[data-field="status"]')).toHaveAttribute("data-band", "amber");

  // An edit accepts the row: who acts on the BASIX condition.
  const basix = page.locator(".register-list tr", { hasText: "Condition 6" });
  await basix.locator('td[data-field="with"]').dblclick();
  await page.keyboard.type("BASIX assessor");
  await page.keyboard.press("Enter");
  await expect(basix.locator('td[data-field="with"]')).toHaveText("BASIX assessor");
  await expect(basix.locator('td[data-field="status"]')).toHaveText("Open · evidence needed");

  // Evidence linked from the condition's sheet in the middle panel:
  // evidence filed, never satisfied.
  await tree.locator('td[data-field="title"]').click();
  await expect(page.locator(".register-sheet").getByRole("heading", { name: /^Condition 12 · / })).toBeVisible();
  await expect(page.locator(".register-council")).toHaveText("Tree protection installed and certified by the project arborist");
  await page.getByLabel("Evidence document").selectOption({ label: "Tree protection certificate" });
  await page.getByRole("button", { name: "Link", exact: true }).click();
  await expect(page.locator(".register-sheet-head .register-status")).toHaveText("Evidence filed");
  await expect(page.locator(".register-sheet").getByRole("list", { name: "History" }).locator("li").last()).toContainText("Open → Evidence filed");

  // Add to RFP puts one line on the arborist's RFP draft.
  await page.getByRole("button", { name: "Add to RFP", exact: true }).click();
  await expect(page.getByRole("button", { name: "On the RFP", exact: true })).toBeVisible();
  const reports = await (await page.request.get(`/api/projects/${project}/reports`)).json();
  const view = await (await page.request.get(`/api/reports/${reports[0].id}`)).json();
  expect(JSON.stringify(view)).toContain("Meet consent condition 12 (DA/2025/0733): Tree protection installed and certified by the project arborist");
});
