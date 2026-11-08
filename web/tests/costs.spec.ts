import { expect, test } from "@playwright/test";
// Grid columns: code, category, item, budget, approved contract, forecast
// variations, approved variations, forecast final cost, claimed to date (from
// invoices), this month, remaining (approved contract less claimed).
const FORECAST = 7, REMAINING = 10;
// These checks read exact cents, so they turn the Cost Plan's Decimals switch
// on (it defaults off) unless a test has already set it.
test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => { try { if (localStorage.getItem("sitewise.costs.decimals") === null) localStorage.setItem("sitewise.costs.decimals", "true"); } catch { /* Storage is optional. */ } });
});

test("cost grid keeps unknowns, edits budget in place and keeps the ledger mechanics", async ({ page }, testInfo) => {
  const {token}=await(await page.request.post('/__e2e/invite?org=a')).json(); await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Cost controls');await page.getByRole('button',{name:'Create project',exact:true}).click();
  await page.getByRole('button',{name:'Costs',exact:true}).click();
  await expect(page.getByRole("button",{name:"Actions for Escalation",exact:true})).toBeVisible();
  await page.getByLabel('New Contingency item',{exact:true}).fill('Unallocated allowance');await page.getByLabel('New Contingency item',{exact:true}).press('Enter');
  const grid=page.getByRole('table');
  for (const h of ['Forecast Variations','Approved Variations','This Month']) await expect(grid.getByRole('columnheader',{name:h})).toHaveAttribute('aria-disabled','true');
  const rowOf=(label:string)=>grid.getByRole('row').filter({has:page.locator('[data-col="item"]').getByText(label,{exact:true})});
  const row=rowOf('Unallocated allowance');
  const budget=row.getByRole('button',{name:'Budget for Unallocated allowance'});
  await expect(budget).toHaveText('—');await expect(row.getByRole('cell').nth(FORECAST)).toHaveText('—');await expect(row.getByRole('cell').nth(REMAINING)).toHaveText('—');

  // A failed inline save keeps the typed figure; retrying saves it.
  let fail=true;await page.route('**/cost-plan/items/*/values/budget',route=>fail?route.fulfill({status:500,body:'failure'}):route.continue());
  await budget.click();const input=page.getByRole('textbox',{name:'Budget for Unallocated allowance'});await input.fill('1,000');await input.press('Enter');
  await expect(page.getByRole('alert')).toContainText('Could not save');await expect(input).toHaveValue('1,000');fail=false;
  await input.press('Enter');await expect(budget).toHaveText('1,000.00');
  await expect(row.getByRole('cell').nth(FORECAST)).toHaveText('1,000.00');await expect(row.getByRole('cell').nth(REMAINING)).toHaveText('—');
  await budget.click();await input.fill('ten');await input.press('Enter');await expect(input).toHaveAttribute('aria-invalid','true');await input.press('Escape');await expect(budget).toHaveText('1,000.00');

  await row.getByRole('button',{name:'Actions for Unallocated allowance',exact:true}).click();
  const menu=page.locator('.cost-row-menu:popover-open');
  await expect(menu.getByText(/Edit|benchmark|scope reference/i)).toHaveCount(0);
  await menu.getByRole('button',{name:'Subdivide allowance',exact:true}).click();
  const drawer=page.getByRole('complementary',{name:'Cost line'});
  await drawer.getByLabel('Allocation 1 label').fill('First allocation');await drawer.getByLabel('Allocation 1 amount').fill('300');await drawer.getByLabel('Allocation 2 label').fill('Second allocation');await drawer.getByLabel('Allocation 2 amount').fill('400');await drawer.getByRole('button',{name:'Save subdivision',exact:true}).click();
  await expect(grid.getByText('First allocation',{exact:true})).toBeVisible();await expect(grid.getByText('Unallocated — Unallocated allowance',{exact:true})).toBeVisible();
  await expect(row.getByRole('button', { name: 'Collapse Unallocated allowance', exact: true })).toHaveAttribute('aria-expanded', 'true');
  await expect(row.getByRole('cell').nth(FORECAST)).toBeEmpty();
  const project=new URL(page.url()).pathname.split('/').pop();const totals=await(await page.request.get(`/api/projects/${project}/cost-plan/totals`)).json();expect(totals.overall.budget.known_subtotal).toBe('1000.00');expect(totals.overall.budget.amount).toBeNull();expect(totals.overall.estimate).toBeUndefined();
  await drawer.getByRole('button',{name:'Close',exact:true}).click();await expect(drawer).toBeHidden();

  // An approved contract becomes the forecast; remaining is the contract less invoices to date.
  await page.getByLabel('New Contingency item',{exact:true}).fill('Head contract');await page.getByLabel('New Contingency item',{exact:true}).press('Enter');
  const head=rowOf('Head contract');await head.getByRole('button',{name:'Budget for Head contract'}).click();await page.getByRole('textbox',{name:'Budget for Head contract'}).fill('500');await page.getByRole('textbox',{name:'Budget for Head contract'}).press('Enter');await expect(head.getByRole('button',{name:'Budget for Head contract'})).toHaveText('500.00');
  await head.getByRole('button',{name:'Approved Contract for Head contract',exact:true}).click();
  const contract=page.getByRole('textbox',{name:'Approved Contract for Head contract',exact:true});
  let contractFails=true, writes=0;
  await page.route('**/cost-plan/items/*/values/commitment',route=>{writes++;return contractFails?route.fulfill({status:500,body:'failure'}):route.continue();});
  await contract.fill('450.00');await contract.press('Enter');
  await expect(contract).toHaveValue('450.00');await expect(page.getByRole('alert')).toContainText('Could not save');
  contractFails=false;await contract.press('Enter');
  await expect(head.getByRole('button',{name:'Approved Contract for Head contract',exact:true})).toHaveText('450.00');
  await expect(drawer).toBeHidden();expect(writes).toBe(2);
  await head.getByRole('button',{name:'Approved Contract for Head contract',exact:true}).click();await contract.fill('wrong');await contract.press('Enter');await expect(contract).toHaveAttribute('aria-invalid','true');await contract.press('Escape');
  await expect(head.getByRole('cell').nth(FORECAST)).toHaveText('450.00');await expect(head.getByRole('cell').nth(REMAINING)).toHaveText('450.00');
  await head.getByRole('button',{name:'Approved Contract for Head contract',exact:true}).click();await contract.fill('');await contract.press('Enter');
  await expect(head.getByRole('button',{name:'Approved Contract for Head contract',exact:true})).toHaveText('\u2014');await expect(head.getByRole('cell').nth(FORECAST)).toHaveText('500.00');
  await head.getByRole('button',{name:'Approved Contract for Head contract',exact:true}).click();await contract.fill('450');await contract.press('Tab');
  await expect(head.getByRole('button',{name:'Approved Contract for Head contract',exact:true})).toHaveText('450.00');
  const sub=grid.getByRole('row',{name:/Contingency subtotal/});await expect(sub).toContainText('1,500.00');await expect(sub.getByRole('cell').nth(7)).toHaveText('450.00');

  await page.getByRole('button',{name:'Cost plan actions',exact:true}).click();await page.getByRole('menuitem',{name:'Freeze baseline',exact:true}).click();await page.getByRole('button',{name:'Confirm baseline',exact:true}).click();await page.getByRole('button',{name:'Cost plan actions',exact:true}).click();await expect(page.getByRole('menuitem',{name:'View frozen baseline',exact:true})).toBeVisible();await page.getByRole('menuitem',{name:'View frozen baseline',exact:true}).click();await expect(page.getByText('Viewing r1 · baseline',{exact:true})).toBeVisible();await expect(page.getByLabel(/^New .* item$/)).toHaveCount(0);await expect(head.getByRole('button',{name:'Budget for Head contract'})).toBeDisabled();await expect(head.getByRole('button',{name:'Approved Contract for Head contract'})).toBeDisabled();
  await page.setViewportSize({width:1360,height:1000});await page.screenshot({path:testInfo.outputPath('next-wave-costs-desktop.png'),fullPage:true});await page.setViewportSize({width:390,height:844});await page.screenshot({path:testInfo.outputPath('next-wave-costs-mobile.png'),fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBeTruthy();
});

test("arrow keys move between cost cells like a spreadsheet, saving as they go", async ({ page }) => {
  const {token}=await(await page.request.post('/__e2e/invite?org=a')).json(); await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Arrow keys');await page.getByRole('button',{name:'Create project',exact:true}).click();
  await page.getByRole('button',{name:'Costs',exact:true}).click();
  for (const name of ['Allowance A','Allowance B']) {
    await page.getByLabel('New Contingency item',{exact:true}).fill(name);await page.getByLabel('New Contingency item',{exact:true}).press('Enter');
    await expect(page.getByRole('button',{name:`Budget for ${name}`,exact:true})).toBeVisible();
  }
  const a=page.getByRole('textbox',{name:'Budget for Allowance A',exact:true}), b=page.getByRole('textbox',{name:'Budget for Allowance B',exact:true});
  await page.getByRole('button',{name:'Budget for Allowance A',exact:true}).click();
  await a.fill('100');await a.press('ArrowDown');
  // Down saves the cell and opens the same column on the next line.
  await expect(b).toBeFocused();
  await expect(page.getByRole('button',{name:'Budget for Allowance A',exact:true})).toHaveText('100.00');
  await b.fill('200');await b.press('ArrowUp');
  await expect(a).toBeFocused();await expect(a).toHaveValue('100.00');
  await expect(page.getByRole('button',{name:'Budget for Allowance B',exact:true})).toHaveText('200.00');
  // A value the cell refuses keeps the cell open instead of moving on.
  await a.fill('ten');await a.press('ArrowDown');
  await expect(a).toHaveAttribute('aria-invalid','true');await expect(a).toBeFocused();
  await a.press('Escape');await expect(page.getByRole('button',{name:'Budget for Allowance A',exact:true})).toHaveText('100.00');

  // Tab and Shift+Tab go along the row; the figure arrives selected, as in Excel.
  const ac=page.getByRole('textbox',{name:'Approved Contract for Allowance A',exact:true});
  await page.getByRole('button',{name:'Budget for Allowance A',exact:true}).click();
  await a.press('Tab');await expect(ac).toBeFocused();
  await ac.fill('90');await ac.press('Shift+Tab');
  await expect(a).toBeFocused();
  await expect(page.getByRole('button',{name:'Approved Contract for Allowance A',exact:true})).toHaveText('90.00');
  expect(await a.evaluate((el: HTMLInputElement) => [el.selectionStart, el.selectionEnd, el.value.length])).toEqual([0, 6, 6]);
  // Right moves on only from the end of the text; inside the figure it moves the caret.
  await a.press('End');await a.press('ArrowLeft');await a.press('ArrowRight');
  await expect(a).toBeFocused();
  await a.press('ArrowRight');await expect(ac).toBeFocused();
  // Left from the start of the text moves back.
  await ac.press('Home');await ac.press('ArrowLeft');await expect(a).toBeFocused();
  // Tab off the last editable cell of a line wraps to the next line.
  await a.press('Tab');await expect(ac).toBeFocused();
  await ac.press('Tab');await expect(page.getByRole('textbox',{name:'Budget for Allowance B',exact:true})).toBeFocused();
});

test("the ghost line saves item, budget and approved contract in one write", async ({ page }, testInfo) => {
  const {token}=await(await page.request.post('/__e2e/invite?org=a')).json(); await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Ghost line');await page.getByRole('button',{name:'Create project',exact:true}).click();
  await page.getByRole('button',{name:'Costs',exact:true}).click();
  const item=page.getByLabel('New Fees and charges item',{exact:true});
  await expect(item).toHaveAttribute('placeholder','Add item…');
  await expect(page.getByLabel('New Fees and charges category',{exact:true})).toHaveCount(0);
  await item.focus();await expect(page.getByLabel('New Fees and charges category',{exact:true})).toHaveValue('authority_fees');
  // Clicking in and out of a blank ghost line creates nothing.
  let creates=0, values=0;
  page.on('request',r=>{if(r.method()==='POST'&&/\/cost-plan\/items$/.test(r.url()))creates++;if(/\/values\//.test(r.url()))values++;});
  await item.focus();await item.blur();
  expect(creates).toBe(0);
  // A bad figure holds the line and marks the cell.
  await item.fill('Section 7.11 contributions');await page.getByLabel('New Fees and charges budget').fill('12,5x');await page.getByLabel('New Fees and charges budget').press('Enter');
  await expect(page.getByLabel('New Fees and charges budget')).toHaveAttribute('aria-invalid','true');expect(creates).toBe(0);
  await page.getByLabel('New Fees and charges budget').fill('12,500');await page.getByLabel('New Fees and charges approved contract').fill('12000');await page.getByLabel('New Fees and charges approved contract').press('Enter');
  const row=page.getByRole('row').filter({has:page.locator('[data-col="item"]').getByText('Section 7.11 contributions',{exact:true})});
  await expect(row.getByRole('button',{name:'Budget for Section 7.11 contributions'})).toHaveText('12,500.00');
  await expect(row.getByRole('button',{name:'Approved Contract for Section 7.11 contributions'})).toHaveText('12,000.00');
  expect(creates).toBe(1);expect(values).toBe(0);
  // A fresh ghost line waits underneath.
  await expect(item).toHaveValue('');await expect(page.getByLabel('New Fees and charges budget')).toHaveValue('');
  await expect(page.getByRole('row',{name:/Fees and charges subtotal/})).toContainText('12,500.00');
  await page.setViewportSize({width:1600,height:900});await page.screenshot({path:testInfo.outputPath('ghost-line-desktop.png')});await page.evaluate(()=>document.documentElement.dataset.theme='dark');await page.screenshot({path:testInfo.outputPath('ghost-line-dark.png')});await page.evaluate(()=>document.documentElement.dataset.theme='light');
  await page.evaluate(()=>document.documentElement.dataset.theme='dark');await page.getByRole('button',{name:'Actions for Section 7.11 contributions',exact:true}).click();await expect(page.getByRole('button',{name:'Delete Section 7.11 contributions'})).toBeVisible();await page.screenshot({path:testInfo.outputPath('row-menu-dark.png')});await page.keyboard.press('Escape');await page.evaluate(()=>document.documentElement.dataset.theme='light');
  // Speed: a ghost-line save is one cost write (budget p50 100 ms, p90 250 ms).
  const times:number[]=[];
  for(let n=0;n<20;n++){
    await item.fill(`Fee ${n}`);await page.getByLabel('New Fees and charges budget').fill(String(100+n));
    const done=page.waitForResponse(r=>r.request().method()==='POST'&&/\/cost-plan\/items$/.test(r.url()));
    const t=Date.now();await item.press('Enter');await done;times.push(Date.now()-t);
    await expect(page.getByRole('button',{name:`Budget for Fee ${n}`,exact:true})).toBeVisible();
  }
  times.sort((a,b)=>a-b);console.log(`ghost line save p50=${times[9]}ms p90=${times[17]}ms; budgets 100/250ms`);
  expect(times[9]).toBeLessThanOrEqual(100);expect(times[17]).toBeLessThanOrEqual(250);
});
