import { expect, type Page } from "@playwright/test";

export const headers = { Origin: "http://127.0.0.1:4173" };

// A brand-new account owns its own organisation, so switching the model on
// here cannot change what any other test sees.
export async function freshOwner(page: Page, tag: string): Promise<void> {
  const email = `e2e-${tag}-${Date.now()}@example.com`;
  const res = await page.request.post("/api/signup", { headers, data: { email, password: "e2e password 1", accept_terms: true } });
  expect(res.ok()).toBeTruthy();
}

export async function modelSwitch(page: Page, enabled: boolean): Promise<void> {
  const res = await page.request.put("/api/account/model", { headers, data: { enabled } });
  expect(res.ok()).toBeTruthy();
}

export interface TenderPackage {
  project: string;
  pkg: string;
  base: string;
  /** Base rows whose labels start with different words: the stand-in model
   * places a line on the row that starts with the line's first word. */
  rows: { key: string; label: string }[];
}

/** A project with one package of the given discipline, and its rows. */
export async function tenderPackage(page: Page, name: string, discipline = "consultant.town_planner"): Promise<TenderPackage> {
  const created = await page.request.post("/api/projects", { headers, data: { name } });
  expect(created.ok()).toBeTruthy();
  const project = (await created.json()).id as string;
  const row = await page.request.post(`/api/projects/${project}/procurement/rows`, { headers, data: { discipline_id: discipline } });
  expect(row.ok()).toBeTruthy();
  const pkg = (await row.json()).id as string;
  const base = `/api/projects/${project}/procurement/packages/${pkg}`;
  const table = await (await page.request.get(`${base}/prices`)).json();
  const seen = new Set<string>();
  const rows: { key: string; label: string }[] = [];
  for (const it of table.items as { key: string; label: string; base: boolean; kind: string }[]) {
    const first = it.label.split(/\s+/)[0].toLowerCase();
    if (!it.base || it.kind === "rate" || it.label.includes(",") || it.label.includes("~") || seen.has(first)) continue;
    seen.add(first);
    rows.push({ key: it.key, label: it.label });
  }
  expect(rows.length).toBeGreaterThanOrEqual(3);
  return { project, pkg, base, rows: rows.slice(0, 6) };
}

/** Files a synthetic tender on the package: each line is a label and an
 * amount in dollars; the PDF prints a total of them. Returns the submission. */
export async function dropTender(page: Page, p: TenderPackage, firm: string, abn: number, lines: [string, number][], extra?: [string, number]): Promise<string> {
  // A negative amount prints the line with no price ("Not priced").
  const spec = lines.map(([label, dollars]) => `${label}~${dollars < 0 ? -1 : dollars * 100}`).join(",");
  // The extra is printed with its price but left out of the printed total.
  const more = extra ? `&extra=${encodeURIComponent(`${extra[0]}~${extra[1] * 100}`)}` : "";
  const pdf = await (await page.request.get(`/__e2e/proposal.pdf?firm=${encodeURIComponent(firm)}&abn=${abn}&lines=${encodeURIComponent(spec)}${more}`)).body();
  const up = await page.request.post(`${p.base}/submissions?name=${encodeURIComponent(`Tender (${firm}).pdf`)}`, { headers: { ...headers, "Content-Type": "application/pdf" }, data: pdf });
  expect(up.status()).toBe(201);
  return (await up.json()).submission_id as string;
}

export interface TenderJSON {
  version: number; on: boolean; suggested: number;
  rows: { key: string; label: string; system: boolean }[];
  columns: { submission_id: string; state: string; tender_sum: string | null; like_for_like: string | null; lowest: boolean; checked: boolean;
    cells: Record<string, { state: string; amount: string | null; mark: string }>;
    lines: { id: string; label: string; row: string; counted: boolean; amount: string | null }[] }[];
  adjustments: { id: string; label: string; origin: string; reason: string; amounts: Record<string, { amount: string | null; state: string; source: string }> }[];
}

export async function tenderTable(page: Page, p: TenderPackage): Promise<TenderJSON> {
  return (await (await page.request.get(`${p.base}/prices`)).json()).tender as TenderJSON;
}

/** Waits until every listed submission has been read by the tender pass. */
export async function waitRead(page: Page, p: TenderPackage, subs: string[]): Promise<TenderJSON> {
  let last: TenderJSON | undefined;
  await expect.poll(async () => {
    last = await tenderTable(page, p);
    return subs.every(s => last!.columns.find(c => c.submission_id === s)?.state === "read");
  }, { timeout: 30_000 }).toBeTruthy();
  return last!;
}

/** Opens the package's price table in the browser. */
export async function openPrices(page: Page, p: TenderPackage, rowName = "Town Planner", tab = ""): Promise<void> {
  await page.goto(`/projects/${p.project}`);
  await page.getByRole("button", { name: "Procurement", exact: true }).click();
  if (tab) await page.getByRole("tab", { name: new RegExp(`^${tab}`) }).click();
  await page.getByRole("row", { name: new RegExp(`^${rowName}`) }).getByRole("button", { name: new RegExp(`(?:Compare|Review) ${rowName}`) }).click();
}

export const money = (dollars: number) => `$${dollars.toLocaleString("en-AU", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
