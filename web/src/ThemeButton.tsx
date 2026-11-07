import { useSyncExternalStore } from "react";
import { readStoredTheme, subscribeTheme, toggleTheme } from "./theme";

export function ThemeButton() {
  const theme = useSyncExternalStore(subscribeTheme, readStoredTheme, () => "dark");
  const next = theme === "light" ? "Dark" : "Light";
  return (
    <button type="button" className="theme-button" aria-label={next} title={next} onClick={() => toggleTheme()}>
      {theme === "light" ? <MoonIcon /> : <SunIcon />}
    </button>
  );
}

// A small sun with short, separate rays and a round crescent moon, matching
// the owner's reference (7 Nov 2026).
export function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="3.6" fill="none" stroke="currentColor" strokeWidth="1.6" />
      <path d="M12 3.6v1.6M12 18.8v1.6M3.6 12h1.6M18.8 12h1.6M6.06 6.06l1.13 1.13M16.81 16.81l1.13 1.13M6.06 17.94l1.13-1.13M16.81 7.19l1.13-1.13" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
    </svg>
  );
}

export function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12.2 4a8 8 0 1 0 7.8 9.8 6.2 6.2 0 0 1-7.8-9.8z" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" />
    </svg>
  );
}
