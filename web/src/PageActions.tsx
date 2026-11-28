import { createContext, useContext, useEffect, useId, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { GroundButton } from "./GroundButton";
import { IconConfirmed, IconMore } from "./icons";

/** The page's Update: the same button on Profile and on an RFP or RFT. */
export function UpdateButton({ onClick, disabled, title, busy, children = "Update" }: { onClick: () => void; disabled?: boolean; title?: string; busy?: boolean; children?: ReactNode }) {
  return <GroundButton primary icon="refresh" className="btn-update" onClick={onClick} disabled={disabled} title={title} busy={busy}>{children}</GroundButton>;
}

/** One step of a breadcrumb trail above the one that is open. */
export interface Crumb { label: string; onClick: () => void }

/**
 * The path to a drilled-in view (owner, 25 November 2026): every step above
 * it is a link back, the last names what is open. `to` is the step it came
 * from, kept as "Back to …" so the name says what the click does; `before`
 * are the steps above that, oldest first.
 */
export function BackButton({ to, onClick, here, before = [], disabled, title, className }: {
  to: string; onClick: () => void; here?: string; before?: Crumb[]; disabled?: boolean; title?: string; className?: string;
}) {
  return <nav className={`crumbs${className ? ` ${className}` : ""}`} aria-label="Breadcrumb">
    <ol>
      {before.map(c => <li key={c.label}><button type="button" className="crumb" disabled={disabled} title={disabled ? title : undefined} onClick={c.onClick}>{c.label}</button></li>)}
      <li><button type="button" className="crumb" disabled={disabled} title={title} aria-label={`Back to ${to}`} onClick={onClick}>{to}</button></li>
      {here && <li aria-current="page"><span className="crumb-here" title={here}>{here}</span></li>}
    </ol>
  </nav>;
}

/** A menu item; one with `checked` is a switch that stays open when toggled. */
export type MenuItem = { label: string; icon: ReactNode; onSelect?: () => void; href?: string; download?: boolean; newTab?: boolean; checked?: boolean; disabled?: boolean };

/** The page's other actions behind three dots; each item shows its icon. */
export function MoreMenu({ label, items }: { label: string; items: MenuItem[] }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const id = useId();
  useEffect(() => {
    if (!open) return;
    root.current?.querySelector<HTMLElement>('[role^="menuitem"]:not(:disabled)')?.focus();
    const close = (e: PointerEvent) => { if (!root.current?.contains(e.target as Node)) setOpen(false); };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [open]);
  const pick = (item: MenuItem) => () => { if (item.checked === undefined) setOpen(false); item.onSelect?.(); };
  return <div className="more-menu" ref={root} onKeyDown={e => {
    if (e.key === "Escape") { e.stopPropagation(); setOpen(false); trigger.current?.focus(); }
    if (open && ["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) {
      e.preventDefault();
      const all = Array.from(root.current?.querySelectorAll<HTMLElement>('[role^="menuitem"]:not(:disabled)') ?? []);
      const at = all.indexOf(document.activeElement as HTMLElement);
      const next = e.key === "Home" ? 0 : e.key === "End" ? all.length - 1 : (at + (e.key === "ArrowUp" ? -1 : 1) + all.length) % all.length;
      all[next]?.focus();
    }
  }}>
    <button ref={trigger} type="button" className="more-menu-btn" aria-label={label} title={label} aria-haspopup="menu" aria-expanded={open} aria-controls={id} onClick={() => setOpen(!open)}><IconMore /></button>
    {open && <div id={id} className="page-action-menu more-menu-list" role="menu" aria-label={label}>
      {items.map(item => item.href
        ? <a key={item.label} role="menuitem" href={item.href} download={item.download || undefined} target={item.newTab ? "_blank" : undefined} rel={item.newTab ? "noreferrer" : undefined} onClick={pick(item)}><span className="more-menu-icon" aria-hidden="true">{item.icon}</span>{item.label}</a>
        : <button key={item.label} type="button" role={item.checked === undefined ? "menuitem" : "menuitemcheckbox"} aria-checked={item.checked} disabled={item.disabled} onClick={pick(item)}>
          <span className="more-menu-icon" aria-hidden="true">{item.icon}</span>{item.label}
          {item.checked && <span className="more-menu-tick" aria-hidden="true"><IconConfirmed /></span>}</button>)}
    </div>}
  </div>;
}

/** Which view is on screen, and the right-hand column a view may borrow. */
export const PageActionsContext = createContext<{ surface: string; sideTarget?: HTMLDivElement | null; openSide?: () => void }>({ surface: "" });

/** Details about the thing in the working view (not a document) take the
 * right-hand column in place of the register while they are open. */
export function PageSidePanel({ children }: { children: ReactNode }) {
  const { sideTarget, openSide } = useContext(PageActionsContext);
  useEffect(() => { openSide?.(); }, [openSide]);
  return sideTarget ? createPortal(children, sideTarget) : null;
}
