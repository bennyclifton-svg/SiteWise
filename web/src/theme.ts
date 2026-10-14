// Light and dark follow the Clerk product palette (web/src/clerk-colours.css).
// The choice is local to this browser; nothing is sent to the server.

export type ColorTheme = "dark" | "light";

const STORAGE_KEY = "sitewise.colorTheme.v1";
const CHANGE_EVENT = "sitewise:color-theme-change";

export function isColorTheme(value: unknown): value is ColorTheme {
  return value === "dark" || value === "light";
}

export function readStoredTheme(): ColorTheme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return isColorTheme(stored) ? stored : "dark";
  } catch {
    return "dark";
  }
}

export function applyTheme(theme: ColorTheme, root: HTMLElement = document.documentElement): void {
  root.dataset.theme = theme;
  root.style.colorScheme = theme;
  document.querySelector('meta[name="color-scheme"]')?.setAttribute("content", theme);
}

export function setTheme(theme: ColorTheme): void {
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    /* Storage is optional; the attribute still switches this tab. */
  }
  applyTheme(theme);
  window.dispatchEvent(new Event(CHANGE_EVENT));
}

export function toggleTheme(): ColorTheme {
  const next = readStoredTheme() === "light" ? "dark" : "light";
  setTheme(next);
  return next;
}

export function subscribeTheme(onStoreChange: () => void): () => void {
  const onStorage = (event: StorageEvent) => {
    if (event.key === STORAGE_KEY || event.key === null) {
      applyTheme(readStoredTheme());
      onStoreChange();
    }
  };
  window.addEventListener("storage", onStorage);
  window.addEventListener(CHANGE_EVENT, onStoreChange);
  return () => {
    window.removeEventListener("storage", onStorage);
    window.removeEventListener(CHANGE_EVENT, onStoreChange);
  };
}
