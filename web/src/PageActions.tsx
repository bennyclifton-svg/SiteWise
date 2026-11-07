import { Children, cloneElement, createContext, isValidElement, useContext, useEffect, useId, useRef, useState, type ReactElement, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { IconDownload } from "./icons";

export const PageActionsContext = createContext<{ surface: string; target: HTMLDivElement | null; primaryTarget: HTMLDivElement | null; besideTarget: HTMLDivElement | null }>({ surface: "", target: null, primaryTarget: null, besideTarget: null });

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
      const items = Array.from(root.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? []);
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
