import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

async function signIn(page: Page, name: string) {
  const invite = await page.request.post("/__e2e/invite?org=a");
  const { token } = await invite.json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Reports", exact: true })).toBeVisible();
  return new URL(page.url()).pathname.split("/").pop()!;
}

test("finish an architect RFP: decisions first, then positions and counts that survive refresh", async ({ page }) => {
  const project = await signIn(page, "RFP editor regression");
  const profile = await (await page.request.get(`/api/projects/${project}/profile`)).json();
  const part = profile.parts[0].id;
  for (const [key, value] of [["hdr.work_type", "extend"], ["hdr.cond.planning", "da"], ["hdr.cond.procurement_route", "traditional"]]) {
    const r = await page.request.put(`/api/projects/${project}/profile/${key}`, { headers: origin, data: { part_id: part, value } });
    expect(r.ok(), `${key}: ${await r.text()}`).toBeTruthy();
  }
  for (const data of [{ kind: "services", title: "Architect", discipline_id: "consultant.architect" }, { kind: "services", title: "Civil" }]) {
    const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data });
    expect(r.ok()).toBeTruthy();
  }
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await page.getByLabel("Services package").selectOption({ label: "Architect · discipline RFP" });
  await expect(page.getByLabel("Services package").locator("option", { hasText: "Civil · confirm discipline" })).toHaveCount(1);
  await page.getByRole("button", { name: "Create RFP draft", exact: true }).click();
  await page.getByRole("button", { name: "Assemble draft", exact: true }).click();
  const useCompleted = page.getByRole("button", { name: "Use last completed state", exact: true });
  if (await useCompleted.isVisible().catch(() => false)) await useCompleted.click();

  const panel = page.getByRole("complementary", { name: "Finish this RFP" });
  await expect(panel).toBeVisible();
  await expect(page.getByRole("heading", { name: "3. Scope by stage", exact: true })).toBeVisible();
  const decisions = panel.getByRole("region", { name: "Decisions before issue" });
  await expect(decisions.getByText("Appointment not confirmed", { exact: true })).toBeVisible();
  await page.screenshot({ path: "../.tools/rfp-editor-desktop.png", fullPage: true });
  await page.setViewportSize({ width: 1920, height: 1080 });
  await page.screenshot({ path: "../.tools/rfp-editor-wide.png" });
  await page.setViewportSize({ width: 1280, height: 720 });
  await decisions.getByRole("button", { name: /^Confirm: / }).click();
  await expect(decisions.getByText("Appointment not confirmed", { exact: true })).toHaveCount(0);

  // A blank client quantity is a decision with its control inline.
  await decisions.getByLabel("Client presentations", { exact: true }).fill("3");
  await decisions.getByLabel("Client presentations", { exact: true }).press("Enter");
  await expect(page.locator(".rfp-draft").getByRole("cell", { name: "Client presentations", exact: true })).toBeVisible();
  await expect(decisions.getByText("Client presentations: count not set", { exact: true })).toHaveCount(0);

  await panel.getByText("Scope position", { exact: true }).click();
  await panel.getByRole("radio", { name: "Help selecting other consultants: excluded" }).check();
  await expect(page.locator(".rfp-draft .report-wording").filter({ hasText: "Help selecting other consultants (party not named)" })).toBeVisible();
  await expect(panel.locator('[data-control="position:consultant_selection"] .rfp-source')).toHaveText("You");

  await page.getByRole("button", { name: "Refresh draft", exact: true }).click();
  const excluded = panel.getByRole("radio", { name: "Help selecting other consultants: excluded" });
  if (!(await excluded.isVisible())) await panel.getByText("Scope position", { exact: true }).click();
  await expect(panel.getByRole("radio", { name: "Help selecting other consultants: excluded" })).toBeChecked();

  await panel.locator('[data-control="position:consultant_selection"]').getByRole("button", { name: "Reset", exact: true }).click();
  await expect(panel.locator('[data-control="position:consultant_selection"] .rfp-source')).not.toHaveText("You");
  await expect(panel.getByRole("radio", { name: "Help selecting other consultants: optional" })).toBeChecked();

  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: "../.tools/rfp-editor-mobile.png", fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await page.setViewportSize({ width: 1280, height: 720 });
});

test("a package titled Civil is offered the civil RFP and switches only when confirmed", async ({ page }) => {
  const project = await signIn(page, "RFP discipline suggestion");
  const r = await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Civil" } });
  expect(r.ok()).toBeTruthy();
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP draft", exact: true }).click();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toBeVisible();
  await expect(page.getByRole("complementary", { name: "Finish this RFP" })).toHaveCount(0);
  await page.getByRole("button", { name: "Use the civil engineering services RFP", exact: true }).click();
  await expect(page.getByRole("complementary", { name: "Finish this RFP" })).toBeVisible();
  await expect(page.getByText("This package looks like civil engineering services", { exact: true })).toHaveCount(0);
});
