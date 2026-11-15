import { createContext } from "react";

// The comparison's Decimals switch, from the price table's ⋮ menu. Off by
// default: whole dollars (owner, 15 Nov 2026), as the Cost Plan offers.
export const TenderCents = createContext(false);

const KEY = "sitewise.procurement.decimals";

export function storedCents(): boolean {
  try { return localStorage.getItem(KEY) === "true"; } catch { return false; }
}

export function storeCents(on: boolean) {
  try { localStorage.setItem(KEY, String(on)); } catch { /* Storage is optional. */ }
}
