import { expect, test } from "@playwright/test";

test("cost selection, shared drag handles, inline names and row actions", async ({page}, info) => {
  const {token}=await(await page.request.post('/__e2e/invite?org=a')).json(); await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Cost selection'); await page.getByRole('button',{name:'Create project',exact:true}).click();
  await page.getByRole('button',{name:'Costs',exact:true}).click();
  for (const name of ['First custom','Second custom','Third custom']) {
    await page.getByRole('button',{name:'+ Add line to Contingency',exact:true}).click();
    await page.getByLabel('New Contingency item',{exact:true}).fill(name); await page.getByLabel('New Contingency item',{exact:true}).press('Enter');
    await expect(page.locator('.cost-row [data-col="item"]').getByText(name,{exact:true})).toBeVisible();
  }
  const row=(name:string)=>page.locator('.cost-row').filter({has:page.locator('[data-col="item"]').getByText(name,{exact:true})});
  const first=row('First custom'), second=row('Second custom'), third=row('Third custom');
  await first.locator('[data-col="item"]').click();
  await third.locator('[data-col="item"]').click({modifiers:['Control']});
  await expect(first).toHaveAttribute('aria-selected','true'); await expect(second).toHaveAttribute('aria-selected','false'); await expect(third).toHaveAttribute('aria-selected','true');
  await second.locator('[data-col="item"]').click({modifiers:['Shift']});
  await expect(first).toHaveAttribute('aria-selected','true'); await expect(second).toHaveAttribute('aria-selected','true');
  await first.locator('[data-col="code"]').click();
  await third.locator('[data-col="code"]').click({modifiers:['Control']});
  const handle=first.locator('.table-row-handle'); await handle.scrollIntoViewIfNeeded();
  const from=(await handle.boundingBox())!, to=(await second.boundingBox())!;
  await page.mouse.move(from.x+5,from.y+5); await page.mouse.down(); await page.mouse.move(from.x+5,to.y+to.height-2,{steps:8}); await page.mouse.up();
  await expect.poll(async()=>page.locator('.cost-row [data-col="item"]').allTextContents()).toContain('Third custom');
  await expect.poll(async()=>{
    const names=await page.locator('.cost-row [data-col="item"]').allTextContents(); return names.indexOf('Second custom')<names.indexOf('First custom')&&names.indexOf('First custom')<names.indexOf('Third custom');
  }).toBe(true);
  await page.reload(); await page.getByRole('button',{name:'Costs',exact:true}).click();
  await expect.poll(async()=>{
    const names=await page.locator('.cost-row [data-col="item"]').allTextContents(); return names.indexOf('Second custom')<names.indexOf('First custom');
  }).toBe(true);
  await first.getByText('First custom',{exact:true}).dblclick();
  await page.getByRole('textbox',{name:'First custom',exact:true}).fill('Renamed custom'); await page.getByRole('textbox',{name:'First custom',exact:true}).press('Enter');
  await expect(row('Renamed custom')).toBeVisible();
  await row('Renamed custom').getByRole('button',{name:'Claimed to Date for Renamed custom'}).click();
  await page.getByRole('textbox',{name:'Claimed to Date for Renamed custom amount'}).fill('12.34');
  await page.getByLabel('Claims date for Renamed custom').fill('2026-10-24');
  await row('Renamed custom').getByRole('button',{name:'Save',exact:true}).click();
  await expect(row('Renamed custom').getByRole('button',{name:'Claimed to Date for Renamed custom'})).toHaveText('12.34');
  await page.setViewportSize({width:1440,height:1000});
  await row('Renamed custom').getByRole('button',{name:'Actions for Renamed custom',exact:true}).click();
  await expect(page.locator('.cost-row-menu:popover-open')).toBeVisible();
  await expect(page.locator('.cost-row-menu:popover-open')).not.toContainText(/Edit|benchmark|scope reference/i);
  await page.screenshot({path:info.outputPath('cost-actions-desktop.png'),fullPage:true});
  await page.keyboard.press('Escape'); await expect(page.locator('.cost-row-menu:popover-open')).toHaveCount(0);
  await page.setViewportSize({width:390,height:844});
  await row('Renamed custom').getByRole('button',{name:'Actions for Renamed custom',exact:true}).click();
  await page.screenshot({path:info.outputPath('cost-actions-mobile.png'),fullPage:true});
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
