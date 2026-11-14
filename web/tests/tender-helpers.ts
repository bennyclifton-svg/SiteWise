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
