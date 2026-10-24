import { expect, test } from "@playwright/test";
// Grid columns: code, category, item, budget, approved contract, forecast
// variations, approved variations, forecast final cost, budget variance,
// claimed to date, this month, remaining.
const FORECAST = 7, VARIANCE = 8, REMAINING = 11;
test("cost grid keeps unknowns, edits budget in place and keeps the ledger mechanics", async ({ page }, testInfo) => {
  const {token}=await(await page.request.post('/__e2e/invite?org=a')).json(); await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Cost controls');await page.getByRole('button',{name:'Create project',exact:true}).click();
  await page.getByRole('button',{name:'Costs',exact:true}).click();
  await expect(page.getByRole("button",{name:"Delete Escalation",exact:true})).toBeVisible();
  await page.getByRole('button',{name:'+ Add line to Contingency',exact:true}).click();await page.getByLabel('New Contingency item',{exact:true}).fill('Unallocated allowance');await page.getByLabel('New Contingency item',{exact:true}).press('Enter');
  const grid=page.getByRole('table');
  for (const h of ['Forecast Variations','Approved Variations','This Month']) await expect(grid.getByRole('columnheader',{name:h})).toHaveAttribute('aria-disabled','true');
  const rowOf=(label:string)=>grid.getByRole('row').filter({has:page.locator('[data-col="item"]').getByText(label,{exact:true})});
  const row=rowOf('Unallocated allowance');
  const budget=row.getByRole('button',{name:'Budget for Unallocated allowance'});
  await expect(budget).toHaveText('—');await expect(row.getByRole('cell').nth(FORECAST)).toHaveText('—');await expect(row.getByRole('cell').nth(VARIANCE)).toHaveText('—');

  // A failed inline save keeps the typed figure; retrying saves it.
  let fail=true;await page.route('**/cost-plan/items/*/values/budget',route=>fail?route.fulfill({status:500,body:'failure'}):route.continue());
  await budget.click();const input=page.getByRole('textbox',{name:'Budget for Unallocated allowance'});await input.fill('1,000');await input.press('Enter');
  await expect(page.getByRole('alert')).toContainText('Could not save');await expect(input).toHaveValue('1,000');fail=false;
  await input.press('Enter');await expect(budget).toHaveText('1,000.00');
  await expect(row.getByRole('cell').nth(FORECAST)).toHaveText('1,000.00');await expect(row.getByRole('cell').nth(VARIANCE)).toHaveText('0.00');await expect(row.getByRole('cell').nth(REMAINING)).toHaveText('—');
  await budget.click();await input.fill('ten');await input.press('Enter');await expect(input).toHaveAttribute('aria-invalid','true');await input.press('Escape');await expect(budget).toHaveText('1,000.00');

  await row.getByRole('button',{name:'Actions for Unallocated allowance',exact:true}).click();
  const menu=page.locator('.cost-row-menu:popover-open');
  await expect(menu.getByText(/Edit|benchmark|scope reference/i)).toHaveCount(0);
  await menu.getByRole('button',{name:'Subdivide allowance',exact:true}).click();
  const drawer=page.getByRole('complementary',{name:'Cost line'});
  await drawer.getByLabel('Allocation 1 label').fill('First allocation');await drawer.getByLabel('Allocation 1 amount').fill('300');await drawer.getByLabel('Allocation 2 label').fill('Second allocation');await drawer.getByLabel('Allocation 2 amount').fill('400');await drawer.getByRole('button',{name:'Save subdivision',exact:true}).click();
  await expect(grid.getByText('First allocation',{exact:true})).toBeVisible();await expect(grid.getByText('Unallocated — Unallocated allowance',{exact:true})).toBeVisible();
  await expect(row).toContainText('Group · amounts are on the lines below');
  const project=new URL(page.url()).pathname.split('/').pop();const totals=await(await page.request.get(`/api/projects/${project}/cost-plan/totals`)).json();expect(totals.overall.budget.known_subtotal).toBe('1000.00');expect(totals.overall.budget.amount).toBeNull();expect(totals.overall.estimate).toBeUndefined();
  await drawer.getByRole('button',{name:'Close',exact:true}).click();await expect(drawer).toBeHidden();

  // An approved contract set in the drawer becomes the forecast; variance is budget less forecast.
  await page.getByRole('button',{name:'+ Add line to Contingency',exact:true}).click();await page.getByLabel('New Contingency item',{exact:true}).fill('Head contract');await page.getByLabel('New Contingency item',{exact:true}).press('Enter');
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
  await expect(head.getByRole('cell').nth(FORECAST)).toHaveText('450.00');await expect(head.getByRole('cell').nth(VARIANCE)).toHaveText('50.00');
  await head.getByRole('button',{name:'Approved Contract for Head contract',exact:true}).click();await contract.fill('');await contract.press('Enter');
  await expect(head.getByRole('button',{name:'Approved Contract for Head contract',exact:true})).toHaveText('\u2014');await expect(head.getByRole('cell').nth(FORECAST)).toHaveText('500.00');
  await head.getByRole('button',{name:'Approved Contract for Head contract',exact:true}).click();await contract.fill('450');await contract.press('Tab');
  await expect(head.getByRole('button',{name:'Approved Contract for Head contract',exact:true})).toHaveText('450.00');
  const sub=grid.getByRole('row',{name:/Contingency subtotal/});await expect(sub).toContainText('1,500.00');await expect(sub.getByRole('cell').nth(6)).toHaveText('—');

  await page.getByRole('button',{name:'Freeze baseline',exact:true}).click();await page.getByRole('button',{name:'Confirm baseline',exact:true}).click();await expect(page.getByText('Cost Plan r2 · draft',{exact:true})).toBeVisible();await page.getByRole('button',{name:'View frozen baseline',exact:true}).click();await expect(page.getByText('Cost Plan r1 · baseline',{exact:true})).toBeVisible();await expect(page.getByRole('button',{name:/Add line to/})).toHaveCount(0);await expect(head.getByRole('button',{name:'Budget for Head contract'})).toBeDisabled();await expect(head.getByRole('button',{name:'Approved Contract for Head contract'})).toBeDisabled();
  await page.setViewportSize({width:1360,height:1000});await page.screenshot({path:testInfo.outputPath('next-wave-costs-desktop.png'),fullPage:true});await page.setViewportSize({width:390,height:844});await page.screenshot({path:testInfo.outputPath('next-wave-costs-mobile.png'),fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBeTruthy();
});
