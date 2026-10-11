import {expect,test} from "@playwright/test";

test("cost defaults, four sections, deletion and inline quantities",async({page,baseURL},testInfo)=>{
 const {token}=await(await page.request.post('/__e2e/invite?org=a')).json();await page.goto(`/#token=${token}`);
 await page.getByLabel('New project').fill('Cost autofill');await page.getByRole('button',{name:'Create project',exact:true}).click();
 await expect(page.getByRole('button',{name:'Costs',exact:true})).toBeVisible();
 const project=new URL(page.url()).pathname.split('/').pop(),base=`/api/projects/${project}`;
 const profile=await(await page.request.get(base+'/profile')).json();
 for(const discipline_id of ['consultant.architect','consultant.structural','consultant.mechanical','trade.main_works']){
  const r=await page.request.post(base+'/procurement/rows',{headers:{Origin:baseURL!},data:{discipline_id}});expect(r.ok(),await r.text()).toBeTruthy();
 }
 for(const system_id of ['structure','envelope','mechanical','access-egress.barriers','access-egress.stairs-and-ramps']){
  const r=await page.request.post(base+'/works',{headers:{Origin:baseURL!},data:{part_id:profile.parts[0].id,system_id,action:'new',title:system_id}});expect(r.ok(),await r.text()).toBeTruthy();
 }
 await page.getByRole('button',{name:'Costs',exact:true}).click();
 await expect(page.getByRole('button',{name:'Delete Escalation',exact:true})).toBeVisible();
 await expect(page.locator('.cost-group th')).toHaveText(['Fees and charges','Consultants','Construction','Contingency']);
 await expect(page.getByRole('button',{name:/^(Keep|Dismiss|Add all suggested)/})).toHaveCount(0);
 const architect=page.getByRole('row').filter({has:page.getByRole('button',{name:'Architect',exact:true})});
 await architect.getByRole('button',{name:'Budget for Architect',exact:true}).click();await page.getByRole('textbox',{name:'Budget for Architect',exact:true}).fill('50000');await page.getByRole('textbox',{name:'Budget for Architect',exact:true}).press('Enter');
 await expect(architect.getByRole('button',{name:'Budget for Architect',exact:true})).toHaveText('50,000.00');await expect(architect.locator('.cost-tag')).toHaveCount(0);
 let deleteFails=true;
 await page.route('**/cost-plan/items/*',route=>route.request().method()==='DELETE' && deleteFails ? route.fulfill({status:500,body:'failure'}) : route.continue());
 await page.getByRole('button',{name:'Delete Escalation',exact:true}).click();
 await expect(page.getByRole('alert')).toContainText('Could not save');
 await expect(page.getByRole('button',{name:'Escalation',exact:true})).toBeVisible();
 deleteFails=false;
 await page.getByRole('button',{name:'Delete Escalation',exact:true}).click();await expect(page.getByRole('button',{name:'Escalation',exact:true})).toHaveCount(0);
 await page.reload();await page.getByRole('button',{name:'Costs',exact:true}).click();await expect(page.getByRole('button',{name:'Delete Architect',exact:true})).toBeVisible();
 await expect(page.getByRole('button',{name:'Escalation',exact:true})).toHaveCount(0);
 const saved=await(await page.request.get(base+'/cost-plan')).json();expect(saved.suggestions).toHaveLength(0);expect(saved.items.length).toBeGreaterThan(12);
 await page.getByRole('button',{name:'+ Add line to Construction',exact:true}).click();await page.getByLabel('New Construction physical scope').selectOption('envelope');await page.getByLabel('New Construction item').fill('Facade access allowance');await page.getByLabel('New Construction item').press('Enter');
 const row=page.getByRole('row').filter({has:page.getByRole('button',{name:'Facade access allowance',exact:true})});await expect(row).toContainText('Main Works');
 await page.getByLabel('Quantities',{exact:true}).check();await row.getByLabel('quantity for Facade access allowance').fill('10');await row.getByLabel('unit for Facade access allowance').fill('m2');await row.getByLabel('rate for Facade access allowance').fill('120');await row.getByLabel('rate for Facade access allowance').press('Enter');
 await expect(row.getByRole('button',{name:'Budget for Facade access allowance'})).toHaveText('1,200.00');

 const physicalBefore=await(await page.request.get(base+'/cost-plan')).json();
 const electricalRow=page.getByRole('row').filter({has:page.getByRole('button',{name:'Mechanical services',exact:true})});
 const allPackages=await(await page.request.get(base+'/packages')).json();
 const main=allPackages.items.find((p:any)=>p.kind==='works');
 await electricalRow.getByRole('button',{name:'Category for Mechanical services',exact:true}).click();
 await electricalRow.getByLabel('Category for Mechanical services',{exact:true}).selectOption(main.id);
 await expect(electricalRow.getByRole('button',{name:'Category for Mechanical services',exact:true})).toHaveText('Main Works');
 const after=await(await page.request.get(base+'/cost-plan')).json();
 const beforeLine=physicalBefore.items.find((i:any)=>i.label==='Mechanical services');
 const afterLine=after.items.find((i:any)=>i.cost_item_id===beforeLine.cost_item_id);
 expect(afterLine.system_id).toBe(beforeLine.system_id);expect(afterLine.package_id).toBe(main.id);
 const access=page.locator('.cost-parent').filter({has:page.getByRole('button',{name:'Access and egress',exact:true})});
 page.once('dialog',async dialog=>{expect(dialog.message()).toMatch(/child lines/);await dialog.dismiss();});
 await access.getByRole('button',{name:'Delete Access and egress',exact:true}).click();await expect(access).toBeVisible();
 page.once('dialog',async dialog=>{await dialog.accept();});
 await access.getByRole('button',{name:'Delete Access and egress',exact:true}).click();await expect(access).toHaveCount(0);
 const deleted=await(await page.request.get(base+'/cost-plan')).json();
 expect(deleted.items.some((i:any)=>i.source_key?.startsWith('works:access-egress'))).toBeFalsy();
 const refreshed=await(await page.request.post(base+'/cost-plan/prepare',{headers:{Origin:baseURL!},data:{}})).json();
 expect(refreshed.plan.items.some((i:any)=>i.source_key?.startsWith('works:access-egress'))).toBeFalsy();
 await page.locator('.cost-grid-scroll').evaluate(e=>{e.scrollTop=0;e.scrollLeft=0;});
 await page.screenshot({path:testInfo.outputPath('cost-autofill-desktop.png'),fullPage:true});
 await page.locator('.cost-grid-scroll').evaluate(e=>{e.scrollLeft=e.scrollWidth;});
 const heading=await page.locator('.cost-group th span').first().boundingBox(),scroll=await page.locator('.cost-grid-scroll').boundingBox();
 expect(heading!.x).toBeGreaterThanOrEqual(scroll!.x);
 await page.screenshot({path:testInfo.outputPath('cost-autofill-actions.png'),fullPage:true});
 await page.locator('.cost-grid-scroll').evaluate(e=>{e.scrollLeft=0;});
 await page.setViewportSize({width:390,height:844});await page.screenshot({path:testInfo.outputPath('cost-autofill-mobile.png'),fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBeTruthy();
 await page.getByRole('button',{name:'Freeze baseline',exact:true}).click();await page.getByRole('button',{name:'Confirm baseline',exact:true}).click();await page.getByRole('button',{name:'View frozen baseline',exact:true}).click();await expect(page.getByRole('button',{name:/Add line to/})).toHaveCount(0);await expect(page.getByRole('button',{name:/^Delete /})).toHaveCount(0);
});
