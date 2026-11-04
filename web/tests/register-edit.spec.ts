import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { expect, test } from '@playwright/test';

const budgets = JSON.parse(readFileSync(resolve(import.meta.dirname, '../../bench/budgets.json'), 'utf8'));

test('document rows: click selects, double-click opens, the row menu renames and edits in bulk', async ({ page }) => {
  const { token } = await (await page.request.post('/__e2e/invite?org=a')).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Inline document editing');
  await page.getByRole('button', { name: 'Create project', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Inline document editing', exact: true })).toBeVisible();
  const project = new URL(page.url()).pathname.split('/').pop();
  await page.setViewportSize({ width: 1900, height: 1000 });
  await page.getByRole('separator', { name: 'Resize documents panel' }).press('End');
  await page.getByTestId('file-input').setInputFiles(Array.from({length: 5}, (_, i) => ({
    name: `document-${i}.md`, mimeType: 'text/markdown',
    buffer: Buffer.from(`# Document ${i}\nDocument number: A-10${i}\nRevision: A\n`),
  })));
  const rows = page.locator('tbody.reg-doc');
  await expect(rows).toHaveCount(5);
  await expect.poll(async () => (await (await page.request.get(`/api/projects/${project}/documents`)).json()).documents.every((d: {status: string}) => d.status === 'filed')).toBe(true);
  const pick = (i: number) => rows.nth(i).locator('.reg-title');
  const checked = () => rows.evaluateAll(els => els.flatMap((el,i) => el.hasAttribute('data-selected') ? [i] : []));
  const menu = page.getByRole('menu');
  // Right-click on a selected row acts on the whole selection.
  const menuAction = async (row: number, action: string) => { await rows.nth(row).locator('.reg-no').click({ button: 'right' }); await menu.getByRole('menuitem', { name: action, exact: true }).click(); };
  const setField = async (row: number, field: string, value: string) => {
    await menuAction(row, `Set ${field}…`);
    await page.getByRole('dialog').getByLabel(`New ${field}`, {exact:true}).selectOption(value);
    await page.getByRole('dialog').getByRole('button', {name:'Apply', exact:true}).click();
  };
  await pick(0).click();
  await expect.poll(checked).toEqual([0]);
  await pick(2).click({ modifiers: ['Control'] });
  await expect.poll(checked).toEqual([0,2]);
  await pick(4).click({ modifiers: ['Shift'] });
  await expect.poll(checked).toEqual([0,2,3,4]);
  await pick(3).click({ modifiers: ['Shift'] });
  await expect.poll(checked).toEqual([0,2,3]);
  await pick(4).click({ modifiers: ['Control','Shift'] });
  await expect.poll(checked).toEqual([0,2,3,4]);
  await pick(2).click({ modifiers: ['Control'] });
  await expect.poll(checked).toEqual([0,3,4]);
  await expect(page.getByRole('region', { name: 'Document viewer' })).toHaveCount(0);
  await rows.nth(1).locator('td.reg-no').dblclick();
  await expect(page.getByRole('region', { name: 'Document viewer' })).toBeVisible();
  // As in a file browser, the double-click's first click selects the row it opens.
  await expect.poll(checked).toEqual([1]);
  await page.getByRole('button', { name: 'Close document', exact: true }).click();
  await pick(0).click();
  await pick(3).click({ modifiers: ['Control'] });
  await pick(4).click({ modifiers: ['Control'] });
  await expect.poll(checked).toEqual([0,3,4]);
  // No toolbar appears for a selection; actions live in the row menu.
  await expect(page.getByRole('toolbar', { name: 'Selected documents' })).toHaveCount(0);

  // One action corrects the curated selection, preserving user provenance.
  const bulkIds = await rows.evaluateAll(els => els.filter(el => el.hasAttribute('data-selected')).map(el => el.id.slice(4)));
  const bulkCatalog = await (await page.request.get('/api/catalog')).json();
  // Kind and lifecycle are no longer register columns; discipline is set in bulk here.
  for (const [field, vocab] of [['discipline', 'disciplines']]) {
    await setField(3, field, bulkCatalog[vocab][0].id);
    await expect(page.getByText(`Updated ${field} for 3 documents.`, {exact:true})).toBeVisible();
    for (const id of bulkIds) {
      const saved = await (await page.request.get(`/api/documents/${id}`)).json();
      expect(saved.fields.find((f: {field:string}) => f.field === field)).toMatchObject({value:bulkCatalog[vocab][0].id,decided_by:'user'});
    }
  }
  const failurePath = `**/api/documents/${bulkIds[0]}/fields/discipline`;
  await page.route(failurePath, route => route.fulfill({status:503,body:'Unavailable'}));
  await setField(3, 'discipline', '');
  await expect(page.getByText('2 saved; 1 could not be saved.', {exact:false})).toBeVisible();
  await expect(page.locator('tbody.reg-doc[data-selected]')).toHaveCount(1);
  await page.unroute(failurePath);
  const failedRow = await rows.evaluateAll(els => els.findIndex(el => el.hasAttribute('data-selected')));
  await setField(failedRow, 'discipline', '');
  await expect(page.getByText('Updated discipline for 1 document.', {exact:true})).toBeVisible();
  for (const id of bulkIds) {
    const saved = await (await page.request.get(`/api/documents/${id}`)).json();
    expect(saved.fields.find((f: {field:string}) => f.field === 'discipline').value ?? '').toBe('');
  }

  // At the crowded breakpoint the title still gets useful room.
  await page.locator('.project-side-panel').evaluate(el => (el as HTMLElement).style.width = '650px');
  expect((await rows.first().locator('.reg-title').boundingBox())!.width).toBeGreaterThan(180);
  await page.locator('.project-side-panel').evaluate(el => (el as HTMLElement).style.removeProperty('width'));
  await page.getByRole('separator', { name: 'Resize documents panel' }).press('End');

  // Editing and normal clicks never trigger viewing. Escape never writes.
  // Rename comes from the row menu (right-click or ⋮); F2 edits the focused cell.
  const target = rows.nth(1);
  const id = (await target.getAttribute('id'))!.slice(4);
  const stable = page.locator(`#doc-${id}`);
  const title = stable.locator('.reg-title');
  const original = await title.textContent();
  let writes = 0;
  page.on('request', request => { if (request.method() === 'PUT' && request.url().includes(`/documents/${id}/fields/`)) writes++; });
  const rowHeight = (await stable.boundingBox())!.height;
  await stable.locator('.reg-no').click({ button: 'right' });
  await expect(menu.getByRole('menuitem', { name: 'Open', exact: true })).toBeFocused();
  await menu.getByRole('menuitem', { name: 'Rename', exact: true }).click();
  const editorStyle = await page.getByRole('textbox', {name:'Edit Title',exact:true}).evaluate(el => { const s = getComputedStyle(el); return {border:s.borderTopWidth,outline:s.outlineStyle,shadow:s.boxShadow}; });
  // The editor covers the cell with a square 1px line and the row keeps its height.
  expect(editorStyle).toEqual({border:'0px',outline:'none',shadow:expect.stringContaining('0px 0px 0px 1px inset')});
  expect((await stable.boundingBox())!.height).toBeCloseTo(rowHeight, 0);
  await page.getByRole('textbox', { name: 'Edit Title', exact: true }).fill('Discard this');
  await page.getByRole('textbox', { name: 'Edit Title', exact: true }).press('Escape');
  await expect(title).toHaveText(original!);
  expect(writes).toBe(0);
  await title.press('F2');
  const input = page.getByRole('textbox', { name: 'Edit Title', exact: true });
  await input.fill('Facade elevations');
  await input.press('Enter');
  await expect(title).toContainText('Facade elevations');
  await expect(title).toHaveAttribute('data-user', 'true');
  expect(writes).toBe(1);
  await expect(page.getByRole('region', { name: 'Document viewer' })).toHaveCount(0);

  // Refused values stay editable, with one write per commit and a usable retry.
  const path = `**/api/documents/${id}/fields/title`;
  await page.route(path, route => route.fulfill({status: 503, body: 'Unavailable'}));
  await stable.getByRole('button', { name: /^More actions for / }).click();
  await menu.getByRole('menuitem', { name: 'Rename', exact: true }).click();
  await input.fill('Retry title');
  await input.press('Enter');
  await expect(stable.getByRole('alert')).toContainText("Couldn't save");
  await expect(input).toHaveValue('Retry title');
  expect(writes).toBe(2);
  await page.unroute(path);
  await input.press('Enter');
  await expect(title).toContainText('Retry title');
  expect(writes).toBe(3);
  await title.press('F2');
  await input.fill('Blur saved title');
  await stable.locator('.reg-rev').click();
  await expect(title).toContainText('Blur saved title');
  expect(writes).toBe(4);

  const catalog = await (await page.request.get('/api/catalog')).json();
  await expect(page.locator('.reg-table thead')).not.toContainText('Kind');
  await expect(page.locator('.reg-table thead')).not.toContainText('Lifecycle');
  for (const [cls, label, vocab, field] of [['reg-disc','Discipline','disciplines','discipline']]) {
    await stable.locator(`.${cls}`).press('F2');
    const select = page.getByRole('combobox', { name: `Edit ${label}`, exact: true });
    await expect(select).toBeFocused();
    const current = await select.inputValue();
    const option = catalog[vocab].find((o: {id: string}) => o.id !== current)!;
    await select.selectOption(option.id);
    await expect(stable.locator(`.${cls}`)).toHaveText(option.label);
    const saved = await (await page.request.get(`/api/documents/${id}`)).json();
    expect(saved.fields.find((f: {field:string}) => f.field === field)).toMatchObject({value:option.id,decided_by:'user'});
  }
  await stable.locator('.reg-rev').press('F2');
  const rev = page.getByRole('textbox', { name: 'Edit Rev', exact: true });
  // Explicitly confirming an unchanged rule value protects it as a user value.
  await rev.press('Enter');
  await expect(stable.locator('.reg-rev')).toHaveAttribute('data-user', 'true');
  await stable.locator('.reg-rev').press('F2');
  await rev.fill('B');
  await rev.press('Tab');
  await expect(stable.locator('.reg-rev')).toHaveText('B');
  await expect(stable.locator('.reg-date')).toBeFocused();
  await stable.locator('.reg-no').press('F2');
  await page.getByRole('textbox', { name: 'Edit Doc no.', exact: true }).fill('Z-999');
  await page.getByRole('textbox', { name: 'Edit Doc no.', exact: true }).press('Enter');
  await expect(stable.locator('.reg-no')).toHaveText('Z-999');
  // The row moved under the current sort; selection follows current order.
  await rows.nth(0).locator('.reg-title').click();
  await rows.nth(2).locator('.reg-title').click({ modifiers: ['Shift'] });
  await expect.poll(checked).toEqual([0,1,2]);
  await page.locator('.register').screenshot({path:'test-results/register-inline-wide.png'});
  await page.setViewportSize({width:390,height:844});
  await title.scrollIntoViewIfNeeded();
  await title.press('F2');
  await expect(input).toBeVisible();
  const bounds = await input.boundingBox();
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(391);
  await page.locator('.register').screenshot({path:'test-results/register-inline-narrow.png'});
  await input.press('Escape');
  await page.reload();
  await expect(stable.locator('.reg-title')).toContainText('Blur saved title');
  await expect(stable.locator('.reg-rev')).toHaveText('B');

  const timings = await page.evaluate(async id => {
    const times:number[]=[];
    for(let i=0;i<20;i++) {
      const start=performance.now();
      const response=await fetch(`/api/documents/${id}/fields/revision`, {method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({value:i%2?'B':'C'})});
      if(!response.ok) throw new Error('Correction failed');
      await response.json(); times.push(performance.now()-start);
    }
    return times.sort((a,b)=>a-b);
  },id);
  const budget=budgets.paths.find((p:{name:string})=>p.name==='field_correction');
  console.log(`field_correction p50=${timings[9].toFixed(1)}ms p90=${timings[17].toFixed(1)}ms (budget ${budget.p50_us/1000}/${budget.p90_us/1000}ms)`);
  expect(timings[9]).toBeLessThanOrEqual(budget.p50_us/1000);
  expect(timings[17]).toBeLessThanOrEqual(budget.p90_us/1000);
});
