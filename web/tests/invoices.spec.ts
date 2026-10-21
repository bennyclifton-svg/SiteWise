import { expect, test, type Page } from "@playwright/test";
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

const money = (s: string) => s.replace(/[,\s]/g, "");

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
  await expect(page.getByTestId("unallocated-row")).toBeVisible();

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

  // Cost plan tab: invoiced to date is the ledger total.
  await page.getByRole("tab", { name: /Cost plan/ }).click();
  const total = page.getByTestId("invoiced-to-date");
  await expect(total).not.toHaveText("—");
  const before = await total.textContent();
  const unallocated = page.getByTestId("unallocated-row");
  await expect(unallocated).toContainText("0.00");
  await expect(unallocated).not.toContainText("1,493.80");

  // Re-map one invoice to no line: the summary moves it to Unallocated at once.
  await page.getByRole("tab", { name: /Invoices/ }).click();
  await table.getByRole("button", { name: "INV-00004", exact: true }).click();
  const panel = page.getByRole("complementary", { name: "Invoice INV-00004" });
  await expect(panel.getByLabel("Ex GST (AUD)")).toHaveValue("1,493.80");
  await panel.getByLabel("Activity line").selectOption({ label: "Unallocated" });
  await expect(panel.getByText("Cleared by you").first()).toBeVisible();
  await panel.getByRole("button", { name: "Close", exact: true }).click();
  await page.getByRole("tab", { name: /Cost plan/ }).click();
  await expect(total).toHaveText(before!);
  await expect(unallocated).toContainText("1,493.80");
  await expect(unallocated.getByRole("button")).toBeVisible();
  const shown = money((await total.textContent())!);

  // Export: the workbook's Cost plan total is the screen's invoiced to date.
  const [download] = await Promise.all([page.waitForEvent("download"), page.getByRole("link", { name: "Export to Excel" }).click()]);
  const path = testInfo.outputPath("ledger.xlsx");
  await download.saveAs(path);
  const xlsx = readFileSync(path);
  const cost = unzip(xlsx, "xl/worksheets/sheet1.xml");
  const sum = cost.match(/<f>SUM\(D\d+:D\d+\)<\/f><v>([-0-9.]+)<\/v>/);
  expect(sum?.[1]).toBe(shown);
  expect(cost).toContain("Unallocated");
  const invoicesSheet = unzip(xlsx, "xl/worksheets/sheet2.xml");
  expect((invoicesSheet.match(/>INV-000\d\d</g) ?? []).length).toBe(10);

  // The Unallocated figure opens the Invoices tab filtered to that line.
  await page.locator(".cost-grid-scroll").evaluate(el => { el.scrollLeft = el.scrollWidth; });
  await unallocated.getByRole("button").click();
  await expect(page.getByRole("tab", { name: /Invoices/ })).toHaveAttribute("aria-selected", "true");
  await expect(page.getByLabel("Activity line filter")).toHaveValue("unallocated");
  await expect(table.getByRole("row")).toHaveCount(2);

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
