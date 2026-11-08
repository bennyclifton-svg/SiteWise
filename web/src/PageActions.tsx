import { Children, cloneElement, createContext, isValidElement, useContext, useEffect, useId, useRef, useState, type ReactElement, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { GroundButton } from "./GroundButton";
import { IconConfirmed, IconDownload, IconMore } from "./icons";

/** The page's Update: the same button on Profile and on an RFP or RFT. */
export function UpdateButton({ onClick, disabled, title, busy, children = "Update" }: { onClick: () => void; disabled?: boolean; title?: string; busy?: boolean; children?: ReactNode }) {
  return <GroundButton primary icon="refresh" className="btn-update" onClick={onClick} disabled={disabled} title={title} busy={busy}>{children}</GroundButton>;
}

/** Back out of a drilled-in view: the same drawn button, ground line only. */
export function BackButton({ to, onClick, disabled, title, className }: { to: string; onClick: () => void; disabled?: boolean; title?: string; className?: string }) {
  return <GroundButton small icon="back" className={className} onClick={onClick} disabled={disabled} title={title} aria-label={`Back to ${to}`}>{to}</GroundButton>;
}

/** A menu item; one with `checked` is a switch that stays open when toggled. */
export type MenuItem = { label: string; icon: ReactNode; onSelect?: () => void; href?: string; download?: boolean; checked?: boolean; disabled?: boolean };

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
        ? <a key={item.label} role="menuitem" href={item.href} download={item.download || undefined} onClick={pick(item)}><span className="more-menu-icon" aria-hidden="true">{item.icon}</span>{item.label}</a>
        : <button key={item.label} type="button" role={item.checked === undefined ? "menuitem" : "menuitemcheckbox"} aria-checked={item.checked} disabled={item.disabled} onClick={pick(item)}>
          <span className="more-menu-icon" aria-hidden="true">{item.icon}</span>{item.label}
          {item.checked && <span className="more-menu-tick" aria-hidden="true"><IconConfirmed /></span>}</button>)}
    </div>}
  </div>;
}

export const PageActionsContext = createContext<{ surface: string; target: HTMLDivElement | null; primaryTarget: HTMLDivElement | null; besideTarget: HTMLDivElement | null; sideTarget?: HTMLDivElement | null; openSide?: () => void }>({ surface: "", target: null, primaryTarget: null, besideTarget: null });

/** Details about the thing in the working view (not a document) take the
 * right-hand column in place of the register while they are open. */
export function PageSidePanel({ children }: { children: ReactNode }) {
  const { sideTarget, openSide } = useContext(PageActionsContext);
  useEffect(() => { openSide?.(); }, [openSide]);
  return sideTarget ? createPortal(children, sideTarget) : null;
}

export function PagePrimaryAction({ surface, children }: { surface: string; children: ReactNode }) {
  const { surface: active, primaryTarget } = useContext(PageActionsContext);
  return active === surface && primaryTarget ? createPortal(children, primaryTarget) : null;
}

/** Sits in the top-right action row, immediately left of Download. */
export function PageBesideDownload({ surface, children }: { surface: string; children: ReactNode }) {
  const { surface: active, besideTarget } = useContext(PageActionsContext);
  return active === surface && besideTarget ? createPortal(children, besideTarget) : null;
}

/** A file download in the same top-right place as the profile Download control. */
export function PageDownload({ surface, href, label }: { surface: string; href: string; label: string }) {
  const { surface: active, target } = useContext(PageActionsContext);
  return active === surface && target ? createPortal(
    <a className="page-download" href={href} aria-label={label} title={label}><IconDownload /><span>Download</span></a>,
    target,
  ) : null;
}

/** Each mounted workspace keeps its own actions; only the visible one reaches the shell. */
export function PageActions({ surface, children }: { surface: string; children: ReactNode }) {
  const { surface: active, target } = useContext(PageActionsContext);
  return active === surface && target ? createPortal(<DownloadMenu key={surface}>{children}</DownloadMenu>, target) : null;
}

function DownloadMenu({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const id = useId();
  useEffect(() => {
    if (!open) return;
    root.current?.querySelector<HTMLElement>(".page-action-menu a, .page-action-menu button")?.focus();
    const close = (event: PointerEvent) => { if (!root.current?.contains(event.target as Node)) setOpen(false); };
    document.addEventListener("pointerdown", close);
    return () => document.removeEventListener("pointerdown", close);
  }, [open]);
  return <div className="page-actions" ref={root} onKeyDown={event => {
    if (event.key === "Escape") { event.stopPropagation(); setOpen(false); trigger.current?.focus(); }
    if (open && ["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
      event.preventDefault();
      const items = Array.from(root.current?.querySelectorAll<HTMLElement>('[role^="menuitem"]:not(:disabled)') ?? []);
      const current = items.indexOf(document.activeElement as HTMLElement);
      const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1 : (current + (event.key === "ArrowUp" ? -1 : 1) + items.length) % items.length;
      items[next]?.focus();
    }
  }}>
    <button ref={trigger} type="button" className="page-download" aria-label="Download" title="Download and print" aria-haspopup="menu" aria-expanded={open} aria-controls={id} onClick={() => setOpen(value => !value)}><IconDownload /><span>Download</span></button>
    {open && <div id={id} className="page-action-menu" role="menu" aria-label="Download" onClick={event => {
      if ((event.target as HTMLElement).closest("a, button")) window.setTimeout(() => setOpen(false), 0);
    }}>{Children.map(children, child => isValidElement(child) ? cloneElement(child as ReactElement<{ role?: string }>, { role: "menuitem" }) : child)}</div>}
  </div>;
}
