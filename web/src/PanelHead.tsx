// The one heading every view in the middle panel wears (owner, 25 November
// 2026). Top line, only when drilled in: the breadcrumb back. Main line: the
// view's title, or its tabs as large words with the open one underlined in
// pencil. Right end: at most one primary action as a ground-line button
// (Update, Update program, New), then the three dots for everything else.
// Anything that only shows (a saving note, a selection count) sits between.

import type { KeyboardEvent, ReactNode } from "react";
import { GroundButton } from "./GroundButton";
import "./panel-head.css";

export interface PanelTab<T extends string> {
  id: T;
  label: ReactNode;
  /** The tab's accessible name when its label holds more than words. */
  name?: string;
}

interface Props<T extends string> {
  /** A BackButton trail; drilled-in views only. */
  crumbs?: ReactNode;
  /** The view's name: shown when it has no tabs, read aloud when it has. */
  title?: ReactNode;
  tabs?: {
    label: string;
    items: PanelTab<T>[];
    value: T;
    onChange: (id: T) => void;
    /** Ties each tab to its panel; the panel is `${idPrefix}-panel-${id}`. */
    idPrefix: string;
  };
  /** The one primary action: an UpdateButton or a PrimaryButton. */
  action?: ReactNode;
  /** The MoreMenu. */
  more?: ReactNode;
  children?: ReactNode;
  /** A surface's own hook for its tests and styles. */
  className?: string;
}

export function PanelHead<T extends string>({ crumbs, title, tabs, action, more, children, className }: Props<T>) {
  return (
    <header className={className ? `panel-head ${className}` : "panel-head"}>
      {crumbs && <div className="panel-head-crumbs">{crumbs}</div>}
      <div className="panel-head-row">
        {/* With tabs the title is still the page's heading, for a screen reader. */}
        {tabs ? <>{title && <h1 className="sr-only">{title}</h1>}<Tabs {...tabs} /></> : <h1 className="panel-head-title">{title}</h1>}
        {children && <div className="panel-head-note">{children}</div>}
        {(action || more) && <div className="panel-head-actions">{action}{more}</div>}
      </div>
    </header>
  );
}

function Tabs<T extends string>({ label, items, value, onChange, idPrefix }: NonNullable<Props<T>["tabs"]>) {
  const step = (e: KeyboardEvent, by: number) => {
    e.preventDefault();
    const at = items.findIndex(t => t.id === value);
    const next = items[(at + by + items.length) % items.length];
    onChange(next.id);
    document.getElementById(`${idPrefix}-tab-${next.id}`)?.focus();
  };
  return (
    <div className="panel-tabs" role="tablist" aria-label={label}>
      {items.map(t => (
        <button
          key={t.id}
          type="button"
          role="tab"
          id={`${idPrefix}-tab-${t.id}`}
          aria-controls={`${idPrefix}-panel-${t.id}`}
          aria-selected={t.id === value}
          aria-label={t.name}
          tabIndex={t.id === value ? 0 : -1}
          className="panel-tab"
          onKeyDown={e => { if (e.key === "ArrowRight") step(e, 1); if (e.key === "ArrowLeft") step(e, -1); }}
          onClick={() => onChange(t.id)}
        >
          {t.label}
        </button>
      ))}
    </div>
  );
}

/** A view's primary action that is not an Update: the same ground-line button. */
export function PrimaryButton({ children, ...rest }: Parameters<typeof GroundButton>[0]) {
  return <GroundButton primary className="btn-update" {...rest}>{children}</GroundButton>;
}
