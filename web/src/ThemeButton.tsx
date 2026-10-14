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

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <circle cx="12" cy="12" r="4" fill="none" stroke="currentColor" strokeWidth="1.7" />
      <path d="M12 2.5v2.2M12 19.3v2.2M4.8 4.8l1.6 1.6M17.6 17.6l1.6 1.6M2.5 12h2.2M19.3 12h2.2M4.8 19.2l1.6-1.6M17.6 6.4l1.6-1.6" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M15.5 3.5a8.2 8.2 0 1 0 5 12.2A7 7 0 0 1 15.5 3.5z" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" />
    </svg>
  );
}
