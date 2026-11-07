import { expect, test } from "@playwright/test";
import type { CostPlan } from "../src/costApi";

// These checks read exact cents, so they turn the Cost Plan's Decimals switch
// on (it defaults off) unless a test has already set it.
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => { try { if (localStorage.getItem("sitewise.costs.decimals") === null) localStorage.setItem("sitewise.costs.decimals", "true"); } catch { /* Storage is optional. */ } });
});

test("cost groups switch detail locally and invoice links reveal their activity", async ({ page }, testInfo) => {
  const { token } = await (await page.request.post('/__e2e/invite?org=a')).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Fee disclosure');
  await page.getByRole('button', { name: 'Create project', exact: true }).click();
  await page.waitForURL(/\/projects\/[0-9a-f-]{36}$/);
  const project = new URL(page.url()).pathname.split('/').pop()!;
  const headers = { Origin: new URL(page.url()).origin };
  const base = `/api/projects/${project}/cost-plan`;
  const pkg = await (await page.request.post(`/api/projects/${project}/packages`, { headers, data: { kind: 'services', title: 'Architect' } })).json();
  let plan = await (await page.request.get(base)).json() as CostPlan;
  const value = (amount: string) => ({ amount, value_state: 'known', origin: 'user', meaning: 'stated' });
  async function add(label: string, parent: string, posting: boolean, values = {}) {
    const response = await page.request.post(`${base}/items`, { headers, data: {
      plan_version_id: plan.id, version: plan.version, label, parent_item_id: parent,
      line_kind: 'fee', package_id: pkg.id, posting, origin: 'user', meaning: 'allowance', values,
    } });
    expect(response.ok(), await response.text()).toBeTruthy();
    plan = await response.json();
    return plan.items.find(i => i.label === label)!.cost_item_id;
  }
  const group = await add('Architect: Atelier North Pty Ltd', '', false);
  const concept = await add('Concept', group, true, { commitment: value('14500.00') });
  await add('Planning approval', group, true, { commitment: value('24000.00') });
  await add('Design development', group, true, { commitment: value('22000.00') });
  await add('Tender documents', group, true, { commitment: value('26000.00') });
  await add('Procurement & contract', group, true, { commitment: value('6500.00') });
  await add('Design support during construction', group, true, { commitment: value('3500.00') });

  // Upload an unread file and set its fields explicitly: this tests navigation,
  // not AI reading, and never needs a live provider.
  const uploaded = await page.request.post(`/api/projects/${project}/invoices?name=concept.txt`, { headers: { ...headers, 'Content-Type': 'text/plain' }, data: 'Invoice for concept' });
  expect(uploaded.ok()).toBeTruthy();
  const invoice = await uploaded.json();
  await expect.poll(async () => (await (await page.request.get(`/api/projects/${project}/invoices`)).json()).invoices[0]?.status).not.toBe('reading');
  const edited = await page.request.patch(`/api/projects/${project}/invoices/${invoice.id}`, { headers, data: { fields: { kind: 'invoice', number: 'CONCEPT-1', supplier: pkg.id, line: concept, ex_gst: '1450.00', gst: '145.00', total: '1595.00', currency: 'AUD', date: '2026-10-01' } } });
  expect(edited.ok(), await edited.text()).toBeTruthy();

  await page.getByRole('button', { name: 'Costs', exact: true }).click();
  const groupRow = page.locator(`.cost-row[data-id="${group}"]`);
  const childRow = page.locator(`.cost-row[data-id="${concept}"]`);
  await expect(childRow).toBeVisible();
  await expect(groupRow.locator('td').nth(4)).toBeEmpty();
  await expect(groupRow.locator('td').nth(3)).toBeEmpty();
  await groupRow.getByRole('button', { name: 'Collapse Architect: Atelier North Pty Ltd', exact: true }).press('Enter');
  await expect(childRow).toHaveCount(0);
  await groupRow.getByRole('button', { name: 'Expand Architect: Atelier North Pty Ltd', exact: true }).press('Space');
  await expect(childRow).toBeVisible();
  const totalsBefore = await (await page.request.get(`${base}/totals`)).json();
  const calls: string[] = [];
  const record = (request: { url(): string }) => { if (request.url().includes('/api/')) calls.push(request.url()); };
  page.on('request', record);
  const times: number[] = [];
  for (let n = 0; n < 20; n++) {
    const ms = await page.evaluate(async () => {
      const start = performance.now();
      const button = document.querySelector<HTMLButtonElement>('.cost-master-toggle[aria-label="Collapse all"]')!;
      button.click();
      await new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve())));
      return performance.now() - start;
    });
    times.push(ms);
    await expect(childRow).toHaveCount(0);
    await expect(groupRow.locator('td').nth(4)).toHaveText('96,500.00');
    await page.getByRole('button', { name: 'Expand all', exact: true }).click();
    await expect(childRow).toBeVisible();
  }
  page.off('request', record);
  expect(calls).toEqual([]);
  times.sort((a, b) => a - b);
  console.log(`cost disclosure p50=${times[9].toFixed(1)}ms p90=${times[17].toFixed(1)}ms; budgets 100/250ms`);
  expect(times[9]).toBeLessThanOrEqual(100); expect(times[17]).toBeLessThanOrEqual(250);
  await page.getByRole('button', { name: 'Collapse all', exact: true }).click();
  await page.reload();
  await expect(childRow).toHaveCount(0);
  await expect(groupRow.locator('td').nth(4)).toHaveText('96,500.00');
  // Decimals off shows whole dollars and survives a reload; on brings cents back.
  await page.getByRole('button', { name: 'Cost plan actions', exact: true }).click();
  await page.getByRole('menuitemcheckbox', { name: 'Decimals', exact: true }).click();
  await expect(page.getByRole('menuitemcheckbox', { name: 'Decimals', exact: true })).toHaveAttribute('aria-checked', 'false');
  await expect(groupRow.locator('td').nth(4)).toHaveText('96,500');
  await page.reload();
  await expect(groupRow.locator('td').nth(4)).toHaveText('96,500');
  await page.getByRole('button', { name: 'Cost plan actions', exact: true }).click();
  await page.getByRole('menuitemcheckbox', { name: 'Decimals', exact: true }).click();
  await page.keyboard.press('Escape');
  await expect(groupRow.locator('td').nth(4)).toHaveText('96,500.00');
  await page.setViewportSize({ width: 1600, height: 1000 });
  await page.screenshot({ path: testInfo.outputPath('collapsed-desktop.png'), fullPage: true });
  await groupRow.getByTitle('Show the 1 invoice on Architect: Atelier North Pty Ltd', { exact: true }).click();
  await expect(page.locator('.inv-table')).toContainText('CONCEPT-1');
  const csv = await page.request.get(`/api/projects/${project}/invoices/export?format=csv&sheet=invoices&line=${group}`);
  expect(await csv.text()).toContain('CONCEPT-1');
  await page.getByRole('button', { name: 'Concept', exact: true }).click();
  await expect(childRow).toBeVisible();
  await expect(childRow).toHaveAttribute('aria-selected', 'true');
  await expect(childRow).toBeFocused();
  await expect(groupRow.locator('td').nth(4)).toBeEmpty();
  await page.screenshot({ path: testInfo.outputPath('expanded-desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.screenshot({ path: testInfo.outputPath('expanded-mobile.png'), fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
  expect(await (await page.request.get(`${base}/totals`)).json()).toEqual(totalsBefore);
});
