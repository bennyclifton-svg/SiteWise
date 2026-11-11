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

// Terms (owner, 10 Nov 2026): one table, No. | Term, ranked by importance
// with the client agreement first. Each answer (agreement, insurance, days)
// is edited inside the term that prints it. Rows select, edit and delete as
// the stage table's do, without In/Excluded ticks.
test("terms table: one numbered table, fields in their terms, rows edited in place", async ({ page }) => {
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
  await expect(page.locator(".rfp-terms-table")).toHaveCount(1);
  const report = ((await (await page.request.get(`/api/projects/${project}/reports`)).json()) as { id: string }[])[0].id;
  const printed = async () => {
    const view = await (await page.request.get(`/api/reports/${report}`)).json();
    return view.draft.sections.flatMap((s: { blocks: { id: string; text: string }[] }) => s.blocks).find((b: { id: string }) => b.id === "rfp:terms").text as string;
  };

  const terms = page.locator(".rfp-terms-table");
  await expect(terms).toHaveCount(1);
  await expect(terms.locator("thead th")).toHaveText(["No.", "Term"]);
  await expect(terms.locator("tr.table-subhead")).toHaveCount(0);
  await expect(page.getByText("Needed before issue", { exact: false })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Edit wording: Terms", exact: true })).toHaveCount(0);
  const lines = terms.locator("tbody > tr.rfp-term");
  await expect(lines.locator("td.rfp-terms-no")).toHaveText(["1", "2", "3", "4", "5", "6", "7", "8", "9"]);

  // 1 is the client agreement: a blank field, marked, with Choose agreement.
  const agreement = lines.nth(0);
  await expect(agreement).toHaveClass(/rfp-terms-agreement/);
  await expect(agreement).toContainText(/^1Engagement is under ___; your own terms do not apply/);
  await expect(agreement).toHaveAttribute("data-blocked", "true");
  await expect(agreement.getByRole("button", { name: "Show in repository" })).toHaveCount(0);
  // 5 is insurance, its two covers edited where they read; fields are plain.
  const insurance = lines.nth(4);
  await expect(insurance).toContainText("Hold professional indemnity cover of");
  const pi = insurance.getByRole("textbox", { name: "Professional indemnity cover", exact: true });
  await expect(pi).toHaveValue("$20 million");
  await expect(insurance.getByRole("textbox", { name: "Public liability cover", exact: true })).toHaveValue("$20 million");
  expect(await pi.evaluate(el => { const s = getComputedStyle(el); return [s.borderTopWidth, s.borderTopLeftRadius]; })).toEqual(["0px", "0px"]);
  await expect(lines.nth(6).getByRole("textbox", { name: "Rates hold until", exact: true })).toHaveValue("12 months after appointment");
  await expect(lines.nth(7).getByRole("textbox", { name: "Proposal validity (days)", exact: true })).toHaveValue("90");

  // Choose agreement takes the documents selected in the repository.
  const repoRows = page.locator("tbody.reg-doc");
  await repoRows.nth(0).locator(".reg-title").click({ modifiers: ["Control"] });
  await agreement.getByRole("button", { name: "Choose agreement", exact: true }).click();
  await expect(agreement.locator(".rfp-term-doc")).toHaveText(/^Consultant Agreement \d$/);
  await expect(agreement).not.toHaveAttribute("data-blocked", "true");
  await expect(agreement.getByRole("button", { name: "Show in repository" })).toBeVisible();
  // Dragging the other document onto the row adds it.
  const chosen = (await agreement.locator(".rfp-term-doc").textContent())!;
  await repoRows.filter({ hasNotText: chosen }).first().locator("tr.reg-line").dragTo(agreement);
  await expect(agreement.locator(".rfp-term-doc")).toHaveText(/^Consultant Agreement \d and Consultant Agreement \d$/);

  // A stepper nudges the payment days inside its term; the print follows.
  await lines.nth(5).getByRole("button", { name: "Increase Payment terms (days)" }).click();
  await expect.poll(printed).toContain("payment within 31 days");

  // Click selects a row; Enter edits its wording; the field stays in place.
  const validity = lines.nth(7);
  await validity.locator("td.rfp-term-text").click();
  await expect(validity).toHaveAttribute("aria-selected", "true");
  await validity.press("Enter");
  const box = validity.getByRole("textbox", { name: /^Term: / });
  await expect(box).toHaveValue("Your proposal stays valid for 90 days.");
  await box.fill("Proposals stay open for 90 days from closing.");
  await Promise.all([page.waitForResponse(r => r.request().method() === "PUT" && decodeURIComponent(r.url()).includes("/settings/wording:terms.validity")), box.press("Enter")]);
  await expect(validity).toHaveAttribute("data-edited", "true");
  await expect(validity.getByRole("textbox", { name: "Proposal validity (days)", exact: true })).toHaveValue("90");
  await expect.poll(printed).toContain("Proposals stay open for 90 days from closing.");
  // Right-click offers the standard wording back.
  await validity.locator("td.rfp-term-text").click({ button: "right" });
  await Promise.all([page.waitForResponse(r => r.request().method() === "DELETE" && decodeURIComponent(r.url()).includes("/settings/wording:terms.validity")), page.getByRole("menuitem", { name: "Reset to standard wording" }).click()]);
  await expect(validity).toContainText("Your proposal stays valid for");

  // Ctrl-click two rows, then the bin in the head deletes both; the rest renumber.
  await lines.nth(2).locator("td.rfp-term-text").click();
  await lines.nth(3).locator("td.rfp-term-text").click({ modifiers: ["Control"] });
  await terms.getByRole("button", { name: "Delete 2 selected terms" }).click();
  await expect(lines.locator("td.rfp-terms-no")).toHaveText(["1", "2", "3", "4", "5", "6", "7"]);
  await expect.poll(printed).not.toContain("is not a variation");

  // "Add term…" adds the user's own term at the end.
  const ghost = terms.getByRole("textbox", { name: "New term" });
  await ghost.fill("No mobilisation payment.");
  await ghost.press("Enter");
  await expect(lines.last()).toContainText(/^8No mobilisation payment\.$/);

  await page.locator("section.report-section", { has: terms }).screenshot({ path: "../.tools/rfp-terms-desktop.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
});
