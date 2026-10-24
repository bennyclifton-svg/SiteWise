import { expect, test, type Locator, type Page } from "@playwright/test";
import { inflateRawSync } from "node:zlib";
import { readFileSync } from "node:fs";

// Reads one member of a zip (an .xlsx) without a dependency.
function unzip(buf: Buffer, name: string): string {
  let eocd = buf.length - 22;
  while (eocd >= 0 && buf.readUInt32LE(eocd) !== 0x06054b50) eocd--;
  const count = buf.readUInt16LE(eocd + 10);
  let p = buf.readUInt32LE(eocd + 16);
  for (let i = 0; i < count; i++) {
    const method = buf.readUInt16LE(p + 10), size = buf.readUInt32LE(p + 20), nameLen = buf.readUInt16LE(p + 28), extra = buf.readUInt16LE(p + 30), comment = buf.readUInt16LE(p + 32), local = buf.readUInt32LE(p + 42);
    const entry = buf.subarray(p + 46, p + 46 + nameLen).toString("utf8");
    if (entry === name) {
      const start = local + 30 + buf.readUInt16LE(local + 26) + buf.readUInt16LE(local + 28);
      const data = buf.subarray(start, start + size);
      return (method === 8 ? inflateRawSync(data) : data).toString("utf8");
    }
    p += 46 + nameLen + extra + comment;
  }
  throw new Error(`${name} not in workbook`);
}

// The cost grid groups thousands; the ledger API does not.
const grouped = (m: string) => {
  const negative = m.startsWith("-");
  const [whole, cents = "00"] = m.replace("-", "").split(".");
  return `${negative ? "−" : ""}${whole.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}.${cents.padEnd(2, "0")}`;
};

type HeldLine = { label: string; invoiced: string; invoices: number };

async function ledgerSummary(page: Page, project: string) {
  const body = await (await page.request.get(`/api/projects/${project}/invoices`)).json() as {
    summary: { invoiced: string; unallocated: { invoiced: string }; lines: HeldLine[] };
  };
  return body.summary;
}

function lineInvoices(plan: Locator, line: HeldLine) {
  const word = line.invoices === 1 ? "invoice" : "invoices";
  return plan.getByTitle(`Show the ${line.invoices} ${word} on ${line.label}`, { exact: true });
}

async function expectNoInvoiceFooter(plan: Locator) {
  await expect(plan.locator(".cost-ledger-foot")).toHaveCount(0);
  await expect(plan.getByText("Invoices not on a current line")).toHaveCount(0);
  await expect(plan.getByText("Invoices with no activity line")).toHaveCount(0);
  await expect(plan.getByText("Invoiced to date (all lines)")).toHaveCount(0);
  await expect(plan.getByTestId("unallocated-row")).toHaveCount(0);
}

async function dropInvoices(page: Page, numbers: number[]) {
  const files = await Promise.all(numbers.map(async n => ({ n, b64: (await (await page.request.get(`/__e2e/invoice.pdf?n=${n}`)).body()).toString("base64") })));
  await page.getByText("Drop invoices here").evaluate((el, files) => {
    const dt = new DataTransfer();
    for (const f of files) {
      const bytes = Uint8Array.from(atob(f.b64), c => c.charCodeAt(0));
      dt.items.add(new File([bytes], `INV-${String(f.n).padStart(5, "0")}.pdf`, { type: "application/pdf" }));
    }
    const zone = el.closest("section")!;
    for (const type of ["dragenter", "dragover", "drop"]) zone.dispatchEvent(new DragEvent(type, { bubbles: true, cancelable: true, dataTransfer: dt }));
  }, files);
}

test("ten invoices dropped at once land in the ledger, re-map moves the summary, and the workbook matches the screen", async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1700, height: 1000 });
  const { token } = await (await page.request.post("/__e2e/invite?org=a")).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel("New project").fill("Invoice ledger");
  await page.getByRole("button", { name: "Create project", exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split("/").pop()!;
  for (const title of ["Architect", "Structural engineer"]) {
    const r = await page.request.post(`/api/projects/${project}/packages`, { data: { kind: "services", title }, headers: { Origin: new URL(page.url()).origin } });
    expect(r.status()).toBe(201);
  }
  await page.getByRole("button", { name: "Costs", exact: true }).click();
  await expect(page.getByRole("tab", { name: /Cost plan/ })).toHaveAttribute("aria-selected", "true");
  const plan = page.locator("#cost-panel-plan");
  // The tab count arrives with the ledger. The invoice footer is not part of the plan.
  await expect(page.getByRole("tab", { name: /Invoices/ })).toContainText("0");
  await expect(plan.getByRole("row", { name: /Construction budget/ })).toBeVisible();
  await expectNoInvoiceFooter(plan);

  // Drop ten invoices at once; every row appears as it is read.
  await page.getByRole("tab", { name: /Invoices/ }).click();
  await expect(page.getByText("No invoices yet.")).toBeVisible();
  const started = Date.now();
  await dropInvoices(page, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]);
  const table = page.getByRole("table");
  await expect(table.getByRole("row")).toHaveCount(11, { timeout: 15_000 });
  await expect(table.getByText("Reading invoice…")).toHaveCount(0, { timeout: 15_000 });
  testInfo.annotations.push({ type: "batch of 10, last row read", description: `${Date.now() - started} ms in the browser` });
  for (const n of [1, 5, 10]) await expect(table.getByRole("button", { name: `INV-${String(n).padStart(5, "0")}`, exact: true })).toBeVisible();
  // Rules read the synthetic layout: amounts are from the document.
  await expect(table.getByRole("row").nth(1).getByText("From document").first()).toBeVisible();

  // A re-drop of the same bytes makes no new row.
  await dropInvoices(page, [3]);
  await expect(page.getByText("Already filed")).toBeVisible();
  await expect(table.getByRole("row")).toHaveCount(11);

  // Cost plan: invoices that landed on a line stay in that line's invoiced column.
  await page.getByRole("tab", { name: /Cost plan/ }).click();
  await expectNoInvoiceFooter(plan);
  const before = await ledgerSummary(page, project);
  expect(before.unallocated.invoiced).toBe("0.00");
  const held = before.lines.filter(l => l.invoices > 0);
  expect(held.length).toBeGreaterThan(0);
  for (const line of held) await expect(lineInvoices(plan, line)).toHaveText(grouped(line.invoiced));

  // Re-map one invoice to no line. The Invoices tab shows Unallocated.
  // The cost plan does not grow a row for it, and the lines that still hold invoices keep their amounts.
  await page.getByRole("tab", { name: /Invoices/ }).click();
  await table.getByRole("button", { name: "INV-00004", exact: true }).click();
  const panel = page.getByRole("complementary", { name: "Invoice INV-00004" });
  await expect(panel.getByLabel("Ex GST (AUD)")).toHaveValue("1,493.80");
  await panel.getByLabel("Activity line").selectOption({ label: "Unallocated" });
  await expect(panel.getByText("Cleared by you").first()).toBeVisible();
  await panel.getByRole("button", { name: "Close", exact: true }).click();
  await expect(table.getByRole("row", { name: /INV-00004/ })).toContainText("Unallocated");
  await page.getByRole("tab", { name: /Cost plan/ }).click();
  await expect(plan.getByRole("row", { name: /Construction budget/ })).toBeVisible();
  await expectNoInvoiceFooter(plan);
  const after = await ledgerSummary(page, project);
  expect(after.invoiced).toBe(before.invoiced);
  expect(after.unallocated.invoiced).toBe("1493.80");
  const still = after.lines.filter(l => l.invoices > 0);
  expect(still.length).toBeGreaterThan(0);
  for (const line of still) await expect(lineInvoices(plan, line)).toHaveText(grouped(line.invoiced));
  const shown = after.invoiced;

  // Export: the workbook's Cost plan total is the ledger total, including the invoice now off a line.
  await page.getByRole("button", { name: "Download", exact: true }).click();
  const [download] = await Promise.all([page.waitForEvent("download"), page.getByRole("menuitem", { name: "Export to Excel" }).click()]);
  const path = testInfo.outputPath("ledger.xlsx");
  await download.saveAs(path);
  const xlsx = readFileSync(path);
  const cost = unzip(xlsx, "xl/worksheets/sheet1.xml");
  const sum = cost.match(/<f>SUM\(D\d+:D\d+\)<\/f><v>([-0-9.]+)<\/v>/);
  expect(sum?.[1]).toBe(shown);
  expect(cost).toContain("Unallocated");
  const invoicesSheet = unzip(xlsx, "xl/worksheets/sheet2.xml");
  expect((invoicesSheet.match(/>INV-000\d\d</g) ?? []).length).toBe(10);

  // The Invoices tab still lists the invoice with no activity line.
  await page.getByRole("tab", { name: /Invoices/ }).click();
  await page.getByLabel("Activity line filter").selectOption("unallocated");
  await expect(page.getByRole("tab", { name: /Invoices/ })).toHaveAttribute("aria-selected", "true");
  await expect(table.getByRole("row")).toHaveCount(2);
  await expect(table.getByRole("button", { name: "INV-00004", exact: true })).toBeVisible();

  await page.getByRole("tab", { name: /Variations/ }).click();
  await expect(page.getByText("Variations are coming")).toBeVisible();
  await page.getByRole("tab", { name: /Invoices/ }).click();
  await page.getByLabel("Activity line filter").selectOption("");
  await page.screenshot({ path: testInfo.outputPath("invoices-desktop.png"), fullPage: true });
  await page.emulateMedia({ media: "print" });
  await page.screenshot({ path: testInfo.outputPath("invoices-print.png"), fullPage: true });
  await page.emulateMedia({ media: "screen" });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: testInfo.outputPath("invoices-mobile.png"), fullPage: true });
});
