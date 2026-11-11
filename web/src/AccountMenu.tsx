// Foot of the left nav: the signed-in person's initials, opening one menu for
// Account, Cost settings, the colour theme and Log out. Initials come from the account's
// email; navigation and theme never call the server.

import { useEffect, useId, useRef, useState, useSyncExternalStore, type KeyboardEvent } from "react";
import { api } from "./api";
import { IconLogOut } from "./icons";
import { endSession } from "./SignOutButton";
import { readStoredTheme, setTheme, subscribeTheme, type ColorTheme } from "./theme";
import { MoonIcon, SunIcon } from "./ThemeButton";

/** "jane.smith@x.com" -> "JS"; "orlan@x.com" -> "OR". */
export function initialsOf(email: string): string {
  const local = email.split("@")[0] ?? "";
  const words = local.split(/[._\-+\s]+/).filter(Boolean);
  const letters = words.length > 1 ? words[0][0] + words[1][0] : (words[0] ?? "").slice(0, 2);
  return letters.toUpperCase();
}

const THEMES: { id: ColorTheme; label: string; icon: typeof SunIcon }[] = [{ id: "light", label: "Light", icon: SunIcon }, { id: "dark", label: "Dark", icon: MoonIcon }];

/** onCostSettings is present only inside a project, where a cost plan exists.
    "bar" places the menu in the top bar of the Projects and Account pages,
    opening downward; the default sits at the foot of the left nav. */
export function AccountMenu({ onCostSettings, placement = "nav" }: { onCostSettings?: () => void; placement?: "nav" | "bar" }) {
  const [email, setEmail] = useState("");
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const theme = useSyncExternalStore(subscribeTheme, readStoredTheme, () => "dark" as ColorTheme);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const menuId = useId();

  useEffect(() => {
    let live = true;
    // A failed read only costs the initials; the menu still works.
    api.account().then(account => { if (live) setEmail(account.email); }, () => undefined);
    return () => { live = false; };
  }, []);

  useEffect(() => {
    if (!open) return;
    root.current?.querySelector<HTMLElement>('[role^="menuitem"]')?.focus();
    const close = (event: PointerEvent) => { if (!root.current?.contains(event.target as Node)) setOpen(false); };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [open]);

  function onKeyDown(event: KeyboardEvent) {
    if (event.key === "Escape" && open) {
      event.stopPropagation();
      setOpen(false);
      trigger.current?.focus();
    }
    if (open && ["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
      event.preventDefault();
      const items = Array.from(root.current?.querySelectorAll<HTMLElement>('[role^="menuitem"]') ?? []);
      const at = items.indexOf(document.activeElement as HTMLElement);
      const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : (at + (event.key === "ArrowUp" ? -1 : 1) + items.length) % items.length;
      items[next]?.focus();
    }
  }

  async function signOut() {
    setBusy(true);
    setError("");
    const failed = await endSession();
    if (failed) { setError(failed); setBusy(false); }
  }

  return <div className={placement === "bar" ? "account-menu account-menu-bar" : "account-menu"} ref={root} onKeyDown={onKeyDown}>
    <button ref={trigger} type="button" className="account-trigger" aria-label="Account menu" title={email || "Account menu"} aria-haspopup="menu" aria-expanded={open} aria-controls={menuId} onClick={() => setOpen(value => !value)}>
      <span className="account-initials" aria-hidden="true">{initialsOf(email) || "?"}</span>
    </button>
    {open && <div id={menuId} className="account-popover" role="menu" aria-label="Account">
      <a role="menuitem" className="account-item" href="/account" onClick={event => {
        event.preventDefault();
        setOpen(false);
        window.dispatchEvent(new CustomEvent("sitewise:navigate", { detail: "/account" }));
      }}>Account</a>
      {onCostSettings && <button type="button" role="menuitem" className="account-item" onClick={() => { setOpen(false); onCostSettings(); }}>Cost settings</button>}
      <div className="account-group account-theme" role="group" aria-label="Theme">
        {THEMES.map(item => <button key={item.id} type="button" role="menuitemradio" aria-checked={theme === item.id} aria-label={item.label} title={item.label} className="account-theme-option" onClick={() => setTheme(item.id)}><item.icon /></button>)}
      </div>
      <button type="button" role="menuitem" className="account-item" disabled={busy} onClick={() => void signOut()}><span className="account-icon" aria-hidden="true"><IconLogOut /></span>{busy ? "Logging out…" : "Log out"}</button>
      {error && <p role="alert" className="error-text">{error}</p>}
    </div>}
  </div>;
}
