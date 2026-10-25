// One profile value on one line: label, control, then how it was decided
// (mark, sources, conflict picks, reset). The Project card and every system
// pane use the same row so a value reads the same wherever it sits.

import { useEffect, useRef, useState, type FormEvent } from "react";
import { IconCheck, IconConfirmed, IconNotChecked, IconYou } from "./icons";
import type { Cell, Profile, ProfileField } from "./profileApi";

export type Setter = (key: string, body: { part_id?: string; value?: string | null; note?: string; reset?: boolean }) => void;

const ASSERTIONS: Record<string, string> = {
  stated: "Stated",
  required: "Required",
  allowance: "Allowance",
};

const REASONS: Record<string, string> = {
  unreviewed: "Awaiting owner review of the rule and table",
  pending: "Pending: table not verified",
  missing_evidence: "Needs stated inputs",
  conflict: "Inputs disagree",
  mixed_scope: "Inputs belong to different parts",
  no_row: "No table row for these inputs",
  not_built: "Not built yet",
};

export function Mark({ cell }: { cell: Cell }) {
  switch (cell.band) {
    case "user":
      return (
        <span className="pf-mark" data-band="user" title="Set by you">
          <IconYou />
          <span className="sr-only">Set by you</span>
        </span>
      );
    case "green":
      return (
        <span className="pf-mark" data-band="green" title="From the documents">
          <IconConfirmed />
          <span className="sr-only">From the documents</span>
        </span>
      );
    case "amber":
      // A Jev reading shows only its source count: the owner found the
      // check mark added friction without an action to take.
      return null;
    case "red":
      return (
        <span className="pf-mark" data-band="red" title={cell.tenders === "differ" ? "Tenderers differ" : "Sources differ"}>
          <IconCheck />
          <span className="pf-mark-text">{cell.tenders === "differ" ? "Tenderers differ" : "Sources differ"}</span>
        </span>
      );
    case "suggested":
      return (
        <span className="pf-mark" data-band="suggested" title="Typical for this building type; not read from a document">
          <span className="pf-mark-text">Typical</span>
        </span>
      );
    case "unchecked":
      return (
        <span className="pf-mark" data-band="unchecked" title="Not checked: Jev didn't answer in time">
          <IconNotChecked />
          <span className="sr-only">Not checked</span>
        </span>
      );
    case "blank":
      return cell.sources.length > 0 ? (
        <span className="pf-mark" data-band="blank" title="Read, but not sure enough to fill in">
          <span className="pf-mark-text">Not sure</span>
        </span>
      ) : null;
    default:
      return null;
  }
}

export function Sources({ cell, onJump }: { cell: Cell; onJump: (id: string) => void }) {
  const ref = useRef<HTMLDetailsElement>(null);
  const [open, setOpen] = useState(false);
  // Clicking anywhere outside the popup, or Escape, closes it.
  useEffect(() => {
    if (!open) return;
    const close = () => {
      if (ref.current) ref.current.open = false;
    };
    const onDown = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    document.addEventListener("pointerdown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);
  if (cell.sources.length === 0) return null;
  return (
    <details className="pf-sources" ref={ref} onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary aria-label={`${cell.sources.length} source${cell.sources.length === 1 ? "" : "s"}`}>
        {cell.sources.length}
      </summary>
      <ul>
        {cell.sources.map((s, i) => (
          <li key={i}>
            {s.excerpt ? <q>{s.excerpt}</q> : <span className="muted">No excerpt</span>}{" "}
            <button type="button" className="cell-link" onClick={() => onJump(s.document_id)}>
              Show in register
            </button>
          </li>
        ))}
      </ul>
    </details>
  );
}

export function Conflict({ cell, onPick }: { cell: Cell; onPick: (v: string) => void }) {
  if (cell.band !== "red" || cell.alternatives.length === 0) return null;
  return (
    <span className="pf-alts">
      {cell.alternatives.map((a) => (
        <button type="button" className="btn btn-small" key={a.value} onClick={() => onPick(a.value)}>
          Use {a.value} ({a.sources.length})
        </button>
      ))}
    </span>
  );
}

/** Hands a value back to the documents; an icon so the row stays one line. */
export function ResetButton({ onClick }: { onClick: () => void }) {
  return (
    <button type="button" className="pf-reset" aria-label="Reset" title="Reset: use what the documents say" onClick={onClick}>
      ↺
    </button>
  );
}

export function FieldRow({
  field,
  onSet,
  onJump,
  parts,
  compact,
  stacked,
}: {
  field: ProfileField;
  onSet: Setter;
  onJump: (id: string) => void;
  parts?: Profile["parts"];
  /** A card cell: no "usually in" hint beside the value. */
  compact?: boolean;
  /** Label above the value, how it was decided at the label's end. */
  stacked?: boolean;
}) {
  const [draft, setDraft] = useState(field.value);
  useEffect(() => setDraft(field.value), [field.value]);
  const id = `pf-${field.key}-${field.part_id}`;
  const part = parts && parts.length > 1 ? parts.find((p) => p.id === field.part_id) : undefined;
  const label = part && part.kind !== "whole" ? `${field.label} · ${part.label}` : field.label;
  const write = (value: string) =>
    value === "" ? onSet(field.key, { part_id: field.part_id, reset: true }) : onSet(field.key, { part_id: field.part_id, value });
  const derived = field.derived;
  const hint = !field.value && !derived && field.band !== "user" && field.stated_in && field.stated_in.length > 0
    ? `Usually in: ${field.stated_in.join(", ")}` : "";

  let control;
  if (derived) {
    control = (
      <span className="pf-derived">
        {field.value ? `${field.value}${field.unit ? ` ${field.unit}` : ""}` : REASONS[derived.reason ?? ""] ?? "Unknown"}
      </span>
    );
  } else if (field.kind === "choice" && field.options && field.options.length > 0) {
    control = (
      <select id={id} className="select pf-control" value={field.value} onChange={(e) => write(e.target.value)} title={compact ? hint || undefined : undefined}>
        <option value="">{stacked ? "Not stated" : "—"}</option>
        {field.options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.label}
          </option>
        ))}
      </select>
    );
  } else {
    const commit = (e?: FormEvent) => {
      e?.preventDefault();
      if (draft.trim() !== field.value) write(draft.trim());
    };
    control = (
      <form onSubmit={commit} className="pf-inline">
        <input
          id={id}
          className="input pf-control"
          inputMode={field.kind === "number" ? "decimal" : undefined}
          value={draft}
          placeholder={field.kind === "multi" ? "e.g. 2, 6" : field.typical && !field.value ? field.typical : compact && hint ? hint : undefined}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => commit()}
        />
        {field.unit && <span className="pf-unit">{field.unit === "m2" ? "m²" : field.unit}</span>}
      </form>
    );
  }

  const why = (
    <span className="pf-why">
      {field.assertion && ASSERTIONS[field.assertion] && (
        <span className="pf-assert" data-assertion={field.assertion}>
          {ASSERTIONS[field.assertion]}
        </span>
      )}
      <Mark cell={field} />
      <Sources cell={field} onJump={onJump} />
      {field.band === "user" && <ResetButton onClick={() => onSet(field.key, { part_id: field.part_id, reset: true })} />}
    </span>
  );
  const pick = (v: string) => onSet(field.key, { part_id: field.part_id, value: v });

  if (stacked) {
    return (
      <div className="pf-row pf-stack" data-band={field.band || undefined}>
        <div className="pf-stack-head">
          <label htmlFor={derived ? undefined : id} className="pf-label">
            {label}
          </label>
          {why}
        </div>
        {control}
        <Conflict cell={field} onPick={pick} />
      </div>
    );
  }

  return (
    <div className="pf-row" data-band={field.band || undefined}>
      <label htmlFor={derived ? undefined : id} className="pf-label">
        {label}
      </label>
      <div className="pf-value">
        <span className="pf-main">
          {control}
          {why}
        </span>
        <Conflict cell={field} onPick={pick} />
        {!compact && hint && <span className="pf-hint">{hint}</span>}
      </div>
    </div>
  );
}

/**
 * A choice with every option in view: chips for class and work type, a list
 * for subclass. Seeing what the project is not is part of reading what it is.
 * Native radios keep arrow-key movement; the chosen option carries the band.
 */
export function ChoiceGroup({
  field,
  variant,
  title,
  onSet,
  onJump,
}: {
  field: ProfileField;
  variant: "chips" | "list";
  title: string;
  onSet: Setter;
  onJump: (id: string) => void;
}) {
  const name = `pf-${field.key}-${field.part_id}`;
  return (
    <div className={`pf-choice pf-choice-${variant}`} data-band={field.band || undefined} role="radiogroup" aria-label={field.label}>
      <div className="pf-stack-head">
        <span className="pf-rule">{title}</span>
        <span className="pf-why">
          <Mark cell={field} />
          <Sources cell={field} onJump={onJump} />
          {field.band === "user" && <ResetButton onClick={() => onSet(field.key, { part_id: field.part_id, reset: true })} />}
        </span>
      </div>
      <div className="pf-options">
        {(field.options ?? []).map((o) => {
          const on = field.value === o.id;
          return (
            <label key={o.id} className="pf-option" data-on={on || undefined} data-band={on ? field.band || undefined : undefined}>
              <input type="radio" name={name} value={o.id} checked={on} onChange={() => onSet(field.key, { part_id: field.part_id, value: o.id })} />
              <span>{o.label}</span>
            </label>
          );
        })}
      </div>
      <Conflict cell={field} onPick={(v) => onSet(field.key, { part_id: field.part_id, value: v })} />
    </div>
  );
}
