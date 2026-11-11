// One Scope and Works value, answered in one click (owner, 11 Nov 2026).
// Yes/no and short lists are a row of buttons: click to set, click the lit
// one again to clear what you set. Long lists and typed values read as
// quiet text and show their control on hover. Same save call as FieldRow.

import { useEffect, useRef, useState, type FormEvent } from "react";
import { ConflictPop, Mark, ResetButton, Sources, type Setter } from "./ProfileFields";
import type { Profile, ProfileField, ProfileOption } from "./profileApi";
import "./quick-field.css";

// More options than this, or longer names, and a drop-down is quieter.
const MAX_BUTTONS = 8;
const MAX_OPTION_CHARS = 16;

/** The label without what the heading already says: "Declarations selected"
 * reads "Declarations" under "Consultant appointment choices", and a cited
 * standard in brackets moves to the hover. */
export function shortLabel(f: ProfileField): string {
  let l = f.label;
  if (f.scope_decision) l = l.replace(/\s+(selected|allocated)$/i, "");
  return l.replace(/\s*\([^)]*\)$/, "") || f.label;
}

/** Option names without a prefix every option repeats: "BAL-LOW, BAL-12.5"
 * become "LOW, 12.5". The prefix must end at a hyphen or space. */
export function optionNames(options: ProfileOption[]): string[] {
  const labels = options.map((o) => o.label);
  if (labels.length < 2) return labels;
  let common = labels[0];
  for (const l of labels) while (!l.startsWith(common)) common = common.slice(0, -1);
  const cut = Math.max(common.lastIndexOf("-"), common.lastIndexOf(" "));
  if (cut < 0) return labels;
  const rest = labels.map((l) => l.slice(cut + 1).trim());
  return rest.every((r) => r !== "") ? rest : labels;
}

const asButtons = (f: ProfileField) =>
  f.kind === "choice" && !!f.options && f.options.length > 0 && f.options.length <= MAX_BUTTONS &&
  f.options.every((o) => o.label.length <= MAX_OPTION_CHARS);

const unitText = (u?: string) => (u === "m2" ? "m²" : u ?? "");

export function QuickField({ field, onSet, onJump, parts }: { field: ProfileField; onSet: Setter; onJump: (id: string) => void; parts?: Profile["parts"] }) {
  const part = parts && parts.length > 1 ? parts.find((p) => p.id === field.part_id) : undefined;
  const suffix = part && part.kind !== "whole" ? ` · ${part.label}` : "";
  const label = shortLabel(field) + suffix;
  const hint = field.stated_in && field.stated_in.length > 0 ? `Usually in: ${field.stated_in.join(", ")}` : "";
  const title = [field.label + suffix, hint].filter(Boolean).join("\n");
  const id = `qf-${field.key}-${field.part_id}`;
  const set = (value: string) => onSet(field.key, { part_id: field.part_id, value });
  const reset = () => onSet(field.key, { part_id: field.part_id, reset: true });
  const buttons = asButtons(field);

  let control;
  if (field.derived) {
    control = <span className="qf-text">{field.value ? `${field.value} ${unitText(field.unit)}`.trim() : "Unknown"}</span>;
  } else if (buttons) {
    const names = optionNames(field.options!);
    control = (
      <span className="qf-buttons" role="group" aria-label={field.label + suffix}>
        {field.options!.map((o, i) => {
          const on = field.value === o.id;
          // Clearing hands the value back to the documents, so only an
          // answer you set clears; a read one stays lit.
          const click = () => {
            if (!on) set(o.id);
            else if (field.band === "user") reset();
          };
          return (
            <button
              key={o.id}
              type="button"
              aria-pressed={on}
              data-band={on ? field.band || undefined : undefined}
              title={on && field.band === "user" ? `${o.label}: click again to clear` : o.label}
              onClick={click}
            >
              {names[i]}
            </button>
          );
        })}
      </span>
    );
  } else if (field.kind === "choice" && field.options) {
    control = (
      <select id={id} className="field-plain qf-quiet" value={field.value} onChange={(e) => (e.target.value === "" ? reset() : set(e.target.value))}>
        <option value="">—</option>
        {field.options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.label}
          </option>
        ))}
      </select>
    );
  } else {
    control = <QuietInput id={id} field={field} placeholder={field.typical || hint.replace(/^Usually in: /, "In ")} onCommit={(v) => (v === "" ? reset() : set(v))} />;
  }

  return (
    <div className="qf-row" data-band={field.band || undefined} data-key={field.key} data-part={field.part_id}>
      <label className="qf-label" htmlFor={buttons || field.derived ? undefined : id} title={title}>
        {label}
      </label>
      <span className="qf-control">{control}</span>
      <span className="qf-why">
        <Mark cell={field} />
        <Sources cell={field} onJump={onJump} />
        <ConflictPop cell={field} onPick={set} />
        {!buttons && field.band === "user" && <ResetButton onClick={reset} />}
      </span>
    </div>
  );
}

function QuietInput({ id, field, placeholder, onCommit }: { id: string; field: ProfileField; placeholder: string; onCommit: (v: string) => void }) {
  const [draft, setDraft] = useState(field.value);
  const dirty = useRef(false);
  useEffect(() => {
    // A streamed reading must not discard a value being typed.
    if (!dirty.current || field.value === draft) {
      dirty.current = false;
      setDraft(field.value);
    }
  }, [field.value, draft]);
  const commit = (e?: FormEvent) => {
    e?.preventDefault();
    if (draft.trim() !== field.value) onCommit(draft.trim());
  };
  const numeric = field.kind === "number";
  return (
    <form onSubmit={commit} className="qf-input" data-kind={field.kind}>
      <input
        id={id}
        className="field-plain qf-quiet"
        inputMode={numeric ? "decimal" : undefined}
        value={draft}
        placeholder={field.kind === "multi" ? "e.g. 2, 6" : placeholder || "—"}
        onChange={(e) => {
          dirty.current = true;
          setDraft(e.target.value);
        }}
        onBlur={() => commit()}
      />
      {field.unit && <span className="qf-unit">{unitText(field.unit)}</span>}
    </form>
  );
}
