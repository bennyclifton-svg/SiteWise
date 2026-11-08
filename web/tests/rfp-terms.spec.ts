import { expect, test, type Page } from "@playwright/test";

const origin = { Origin: "http://127.0.0.1:4173" };

async function signIn(page: Page, name: string) {
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill(name);
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await expect(page.getByRole("button", { name: "Procurement", exact: true })).toBeVisible();
  return new URL(page.url()).pathname.split("/").pop()!;
}

// Terms (owner, 8 Nov 2026): no banner, no contact or questions date; one
// table: the numbered terms first, then Item | Response led by the client
// agreement, named by the titles of documents chosen in the repository (no
// revision or date sub-table). Fields in the table are plain cells.
test("terms table: numbered terms then answers, agreement chosen from the repository", async ({ page }) => {
  const project = await signIn(page, "RFP terms table");
  await page.setViewportSize({ width: 1600, height: 1000 });
  await page.getByTestId("file-input").setInputFiles([0, 1].map(i => ({ name: `agreement-${i}.md`, mimeType: "text/markdown", buffer: Buffer.from(`# Agreement ${i}
`) })));
  await expect.poll(async () => (await (await page.request.get(`/api/projects/${project}/documents`)).json()).documents.every((d: { status: string }) => d.status === "filed")).toBe(true);
  const docs = (await (await page.request.get(`/api/projects/${project}/documents`)).json()).documents;
  for (let i = 0; i < docs.length; i++) {
    for (const [field, value] of Object.entries({ title: `Consultant Agreement ${i}`, revision: "C", date: "2026-11-01" })) {
      const r = await page.request.put(`/api/documents/${docs[i].id}/fields/${field}`, { headers: origin, data: { value } });
      expect(r.ok(), await r.text()).toBeTruthy();
    }
  }
  await page.request.post(`/api/projects/${project}/packages`, { headers: origin, data: { kind: "services", title: "Architect", discipline_id: "consultant.architect" } });
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  await page.getByRole("button", { name: "Create RFP for Architect", exact: true }).click();

  const terms = page.locator(".rfp-terms-table");
  await expect(terms).toBeVisible();
  await expect(page.locator(".rfp-terms-table")).toHaveCount(1);
  await expect(page.locator(".rfp-numbered-terms, .rfp-terms-docs")).toHaveCount(0);
  await expect(page.getByText("Terms not set", { exact: false })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^(Edit|Done editing) [0-9]+[.] Terms$/ })).toHaveCount(0);
  for (const label of ["Contact for questions", "Questions in writing by"]) await expect(page.getByText(label, { exact: true })).toHaveCount(0);
  // Numbered terms come first, then the head row, then the answers.
  const rows = terms.locator("tbody > tr");
  await expect(rows.first()).toHaveClass(/rfp-terms-line/);
  await expect(rows.first().locator("td").first()).toHaveText("1");
  const sub = terms.locator("tbody > tr.table-subhead");
  await expect(sub).toHaveText("ItemResponse");
  const lineCount = await terms.locator("tbody > tr.rfp-terms-line").count();
  expect(await sub.evaluate(r => (r as HTMLTableRowElement).rowIndex)).toBe(lineCount + 1);
  await expect(terms.locator("tbody > tr:not(.rfp-terms-line):not(.table-subhead) > th")).toHaveText(["Client agreement", "Professional indemnity cover", "Public liability cover", "Proposal validity (days)", "Rates hold until", "Payment terms (days)"]);
  await expect(terms.getByRole("textbox", { name: "Professional indemnity cover", exact: true })).toHaveValue("$20 million");
  await expect(terms.getByRole("textbox", { name: "Public liability cover", exact: true })).toHaveValue("$20 million");
  await expect(terms.getByRole("textbox", { name: "Rates hold until", exact: true })).toHaveValue("12 months after appointment");
  await expect(terms.getByRole("textbox", { name: "Proposal validity (days)", exact: true })).toHaveValue("90");
  // A field in a table cell is the cell: no border, no rounded box.
  const pi = terms.getByRole("textbox", { name: "Professional indemnity cover", exact: true });
  expect(await pi.evaluate(el => { const s = getComputedStyle(el); return [s.borderTopWidth, s.borderTopLeftRadius]; })).toEqual(["0px", "0px"]);
  // Without an agreement the row is marked and issue stays blocked.
  const agreement = terms.locator("tr.rfp-terms-agreement");
  await expect(agreement).toHaveAttribute("data-blocked", "true");
  await expect(agreement.locator("input")).toHaveCount(0);

  // Select one document in the repository, then Update agreement: its title, no revision.
  const repoRows = page.locator("tbody.reg-doc");
  await agreement.getByRole("button", { name: "Select agreement documents", exact: true }).click();
  await repoRows.nth(0).locator(".reg-title").click({ modifiers: ["Control"] });
  await agreement.getByRole("button", { name: "Update agreement", exact: true }).click();
  await expect(agreement.locator(".rfp-terms-agreement-name")).toHaveText(/^Consultant Agreement \d$/);
  await expect(agreement).not.toHaveAttribute("data-blocked", "true");

  // Dragging the other repository document onto the row adds it.
  await repoRows.nth(1).locator("tr.reg-line").dragTo(agreement);
  await expect(agreement.locator(".rfp-terms-agreement-name")).toHaveText(/^Consultant Agreement \d and Consultant Agreement \d$/);

  // A stepper nudges the payment days; the numbered term follows.
  await terms.getByRole("button", { name: "Increase Payment terms (days)" }).click();
  await expect(terms.locator("tr.rfp-terms-line", { hasText: "payment within 31 days" })).toHaveCount(1);
  await expect(rows.first()).toContainText(/^1Engagement is under Consultant Agreement \d and Consultant Agreement \d;/);
  await expect(terms).toContainText("Rates hold until 12 months after appointment");

  // Wording still edits, as one list.
  await page.getByRole("button", { name: "Edit wording: Terms", exact: true }).click();
  await expect(page.getByRole("textbox", { name: "Report wording" })).toBeVisible();
  await page.getByRole("button", { name: "Cancel edit", exact: true }).click();
  await page.locator("section.report-section", { has: terms }).screenshot({ path: "../.tools/rfp-terms-desktop.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
});
