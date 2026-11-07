import { expect, test } from "@playwright/test";

// Combined engagements, slice 1: tick two consultant rows, combine them,
// rename the group, see one parent fee line, then uncombine.
test("two consultants combine into one engagement and uncombine", async ({page}, testInfo) => {
  const {token} = await (await page.request.post('/__e2e/invite?org=a')).json();
  await page.goto(`/#token=${token}`);
  await page.getByLabel('New project').fill('Combined engagement');
  await page.getByRole('button', {name:'Create project', exact:true}).click();
  await page.getByRole('button', {name:'Procurement', exact:true}).click();
  for (const discipline of ['consultant.structural', 'consultant.civil']) {
    await page.getByRole('button', {name:'Add discipline', exact:true}).click();
    await page.getByLabel('Discipline', {exact:true}).selectOption(discipline);
    await page.getByRole('button', {name:'Add row', exact:true}).click();
  }
  await expect(page.getByRole('button', {name:'Remove Civil', exact:true})).toBeVisible();
  // Opening the cost plan gives each row its draft fee line.
  await page.getByRole('button', {name:'Costs', exact:true}).click();
  await expect(page.getByRole('button', {name:'Delete Civil', exact:true})).toBeVisible();
  await page.getByRole('button', {name:'Procurement', exact:true}).click();

  await page.getByLabel('Select Structural to combine').check();
  await expect(page.getByRole('button', {name:/^Combine [0-9]/})).toHaveCount(0);
  await page.getByLabel('Select Civil to combine').check();
  await page.getByRole('button', {name:'Combine 2', exact:true}).click();
  const dialog = page.getByRole('dialog');
  await expect(dialog).toContainText('Civil and Structural');
  await expect(dialog).toContainText('These fee lines move under it');
  const start = Date.now();
  await dialog.getByRole('button', {name:'Combine', exact:true}).click();
  await expect(page.getByRole('button', {name:'Uncombine Civil and Structural', exact:true})).toBeVisible();
  testInfo.annotations.push({type:'timing', description:`combine UI: ${Date.now() - start}ms`});
  await expect(page.getByRole('button', {name:'Remove Structural from group', exact:true})).toBeVisible();
  await expect(page.getByLabel('Select Civil to combine')).toHaveCount(0);

  await page.getByTitle('Double-click to rename').dblclick();
  const name = page.getByLabel('Name of Civil and Structural');
  await name.fill('Structural and civil engineering');
  await name.press('Enter');
  await expect(page.getByRole('button', {name:'Uncombine Structural and civil engineering', exact:true})).toBeVisible();
  await page.screenshot({path:testInfo.outputPath('combined.png')});

  // One parent fee line holds both members' lines.
  await page.getByRole('button', {name:'Costs', exact:true}).click();
  await expect(page.getByRole('button', {name:/^(Collapse|Expand) Civil and Structural$/})).toBeVisible();
  await expect(page.getByRole('button', {name:'Delete Civil', exact:true})).toBeVisible();

  await page.getByRole('button', {name:'Procurement', exact:true}).click();
  await page.getByRole('button', {name:'Uncombine Structural and civil engineering', exact:true}).click();
  await expect(page.getByLabel('Select Civil to combine')).toBeVisible();
  await expect(page.getByRole('button', {name:/^Uncombine/})).toHaveCount(0);
  await page.getByRole('button', {name:'Costs', exact:true}).click();
  await expect(page.getByRole('button', {name:'Delete Civil', exact:true})).toBeVisible();
  await expect(page.getByRole('button', {name:/^(Collapse|Expand) Civil and Structural$/})).toHaveCount(0);
});
