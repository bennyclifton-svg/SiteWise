import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import "../register-edit.css";

// A row's actions menu, opened by a ⋮ button, right-click or the menu key.
// Shared by the document register and the RFP scope table.
export type MenuItem = { label: string; icon: ReactNode; onSelect: () => void; disabled?: boolean; danger?: boolean } | { heading: string } | "sep";
/** Where the menu opens, and the control focus returns to. */
export interface RowMenuAt { x: number; y: number; trigger: HTMLElement | null }

/** The row actions menu, shared by the ⋮ button and right-click. Rendered on
 * the body so the side panel's containment cannot clip or offset it. */
export function RowMenu({ at: menu, items, label, head, onClose }: { at: RowMenuAt; items: MenuItem[]; label: string; head?: string; onClose: (restoreFocus: boolean) => void }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ left: menu.x, top: menu.y });
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const left = Math.max(8, Math.min(menu.x, window.innerWidth - r.width - 8));
    const top = menu.y + r.height > window.innerHeight - 8 ? Math.max(8, menu.y - r.height) : menu.y;
    setPos({ left, top });
    el.querySelector<HTMLElement>('[role="menuitem"]:not(:disabled)')?.focus({ preventScroll: true });
  }, [menu]);
  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    const away = (e: PointerEvent) => {
      const target = e.target as Node;
      // The ⋮ button toggles the menu itself on click.
      const toggle = menu.trigger instanceof HTMLButtonElement && menu.trigger.contains(target);
      if (!ref.current?.contains(target) && !toggle) close.current(false);
    };
    const leave = () => close.current(false);
    document.addEventListener("pointerdown", away, true);
    window.addEventListener("resize", leave);
    window.addEventListener("blur", leave);
    return () => {
      document.removeEventListener("pointerdown", away, true);
      window.removeEventListener("resize", leave);
      window.removeEventListener("blur", leave);
    };
  }, [menu]);
  return createPortal(
    <div
      ref={ref}
      className="reg-menu"
      role="menu"
      aria-label={label}
      style={{ left: pos.left, top: pos.top }}
      onContextMenu={e => e.preventDefault()}
      onKeyDown={e => {
        if (e.key === "Escape" || e.key === "Tab") { e.preventDefault(); e.stopPropagation(); onClose(true); return; }
        if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) return;
        e.preventDefault();
        const all = Array.from(ref.current?.querySelectorAll<HTMLElement>('[role="menuitem"]:not(:disabled)') ?? []);
        const at = all.indexOf(document.activeElement as HTMLElement);
        const next = e.key === "Home" ? 0 : e.key === "End" ? all.length - 1 : (at + (e.key === "ArrowUp" ? -1 : 1) + all.length) % all.length;
        all[next]?.focus();
      }}
    >
      {head && <p className="reg-menu-head" aria-hidden>{head}</p>}
      {items.map((item, i) => item === "sep"
        ? (i > 0 && items[i - 1] !== "sep" && i < items.length - 1 ? <div key={`sep-${i}`} role="separator" className="reg-menu-sep" /> : null)
        : "heading" in item
        ? <p key={`head-${i}`} className="reg-menu-head reg-menu-group" aria-hidden>{item.heading}</p>
        : <button key={item.label} type="button" role="menuitem" className="reg-menu-item" data-danger={item.danger || undefined} disabled={item.disabled}
            onClick={() => { onClose(true); item.onSelect(); }}>
            <span className="reg-menu-icon" aria-hidden>{item.icon}</span>
            {item.label}
          </button>)}
    </div>,
    document.body,
  );
}

