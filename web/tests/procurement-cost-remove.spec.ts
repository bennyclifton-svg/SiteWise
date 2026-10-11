import { expect, test } from "@playwright/test";

test("removing a procurement discipline offers its draft cost lines", async ({page, baseURL}, testInfo) => {
  const {token} = await (await page.request.post('/__e2e/invite?org=a')).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Linked cost removal');
  await page.getByRole('button', {name:'Create project', exact:true}).click();
  await page.getByRole('button', {name:'Procurement', exact:true}).click();
  await page.getByRole('button', {name:'Add discipline', exact:true}).click();
  await page.getByLabel('Discipline', {exact:true}).selectOption('consultant.structural');
  await page.getByRole('button', {name:'Add row', exact:true}).click();
  await expect(page.getByLabel('More for Structural', {exact:true})).toBeVisible();
  await page.getByRole('button', {name:'Costs', exact:true}).click();
  await expect(page.getByRole('button', {name:'Delete Structural', exact:true})).toBeVisible();
  await page.getByRole('button', {name:'Procurement', exact:true}).click();
  const review = async () => {
    await page.getByLabel('More for Structural', {exact:true}).click();
    await page.getByRole('button', {name:'Remove', exact:true}).click();
    await expect(page.getByRole('dialog')).toContainText('Structural');
    await expect(page.getByRole('dialog')).toContainText('1 linked cost-plan line');
  };
  await review();
  await page.screenshot({path:testInfo.outputPath('remove-desktop.png')});
  await page.setViewportSize({width:390,height:844});
  await page.screenshot({path:testInfo.outputPath('remove-mobile.png')});
  await page.getByRole('button', {name:'Cancel', exact:true}).click();
  await expect(page.getByLabel('More for Structural', {exact:true})).toBeVisible();
  await review();
  const start = Date.now();
  await page.getByRole('button', {name:'Remove and delete cost lines', exact:true}).click();
  await expect(page.getByLabel('More for Structural', {exact:true})).toHaveCount(0);
  testInfo.annotations.push({type:'timing',description:`confirmed removal UI: ${Date.now()-start}ms`});
  await page.getByRole('button', {name:'Costs', exact:true}).click();
  await expect(page.getByRole('button', {name:'Delete Structural', exact:true})).toHaveCount(0);
  const project = new URL(page.url()).pathname.split('/').pop();
  const response = await page.request.post(`/api/projects/${project}/cost-plan/prepare`, {headers:{Origin:baseURL!},data:{}});
  expect(response.ok()).toBeTruthy();
  expect((await response.json()).plan.items.some((i:any) => i.label === 'Structural')).toBeFalsy();
});
