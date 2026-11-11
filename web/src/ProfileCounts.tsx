// Bedrooms, bathrooms and car parks on one card row, as RFP page one prints
// them (owner, 10 Nov 2026): "2 bed · 3 bath · 1 car". Each count is still its
// own field and its own input; how it was decided shows as the summary's
// pencil under the number (the origin rule) and in its tooltip. The row ends,
// like every row, with one mark and the sources behind it. Clearing a count
// hands it back to the documents.
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Sources, type Setter } from "./ProfileFields";
import type { ProfileField } from "./profileApi";
import { PendingCount, RevisionMark } from "./RunMarks";

/** The counts the row holds, in its order; garage spaces stand in for car parks when they differ. */
export const COUNT_KEYS = ["hdr.scale.bedrooms", "hdr.scale.bathrooms", "hdr.scale.car_parks", "hdr.scale.garage_spaces"];
const WORD: Record<string, string> = {
  "hdr.scale.bedrooms": "bed",
  "hdr.scale.bathrooms": "bath",
  "hdr.scale.car_parks": "car",
  "hdr.scale.garage_spaces": "garage",
};
const DECIDED: Record<string, string> = {
  user: "set by you",
  amber: "picked by Jev from the documents",
  green: "from the documents",
  red: "the documents disagree",
  suggested: "typical for this building",
};
// The row's mark shows the count that most needs a look.
const RANK = ["red", "amber", "suggested", "green", "user"];

export function CountsRow({ fields, onSet, onJump, changed, pending }: {
  fields: ProfileField[];
  onSet: Setter;
  onJump: (id: string) => void;
  changed?: (f: ProfileField) => string | undefined;
  pending?: (f: ProfileField) => number;
}) {
  const marks = fields.map((f) => changed?.(f)).filter((m): m is string => !!m);
  const waiting = fields.reduce((n, f) => n + (pending?.(f) ?? 0), 0);
  const band = RANK.find((b) => fields.some((f) => f.band === b && f.value)) ?? "";
  // One sources list for the row: every passage behind any of its counts.
  const merged = { value: "", band: band as ProfileField["band"], sources: fields.flatMap((f) => f.sources), alternatives: [] };
  return (
    <div className="pf-row pf-pair pf-counts" data-key="counts" data-band={band || undefined} data-changed={marks.length ? "true" : undefined}>
      {marks.length > 0 && <RevisionMark text={marks.join("\n")} />}
      <div className="pf-pair-head">
        <span className="pf-label" id="pf-counts-label">Counts</span>
        <span className="pf-pair-why">
          <span className="pf-why">
            <Sources cell={merged} onJump={onJump} />
            {waiting > 0 && <PendingCount n={waiting} />}
          </span>
        </span>
      </div>
      <div className="pf-value pf-count-cells" role="group" aria-labelledby="pf-counts-label">
        {fields.map((f, i) => (
          <span key={f.key + f.part_id} className="pf-count-item">
            {i > 0 && <span className="pf-count-sep" aria-hidden="true">·</span>}
            <CountCell field={f} onSet={onSet} changed={changed?.(f)} />
          </span>
        ))}
      </div>
    </div>
  );
}

function CountCell({ field, onSet, changed }: { field: ProfileField; onSet: Setter; changed?: string }) {
  const [draft, setDraft] = useState(field.value);
  const dirty = useRef(false);
  useEffect(() => {
    // A reading that lands while a count is being typed waits for it.
    if (!dirty.current || field.value === draft) {
      dirty.current = false;
      setDraft(field.value);
    }
  }, [field.value, draft]);
  const commit = (e?: FormEvent) => {
    e?.preventDefault();
    const v = draft.trim();
    if (v === field.value) return;
    onSet(field.key, v === "" ? { part_id: field.part_id, reset: true } : { part_id: field.part_id, value: v });
  };
  const n = field.sources.length;
  const said = field.value
    ? `${field.label}: ${DECIDED[field.band] ?? "not decided"}${n ? `, ${n} source${n === 1 ? "" : "s"}` : ""}${field.band === "user" ? ". Clear it to use the documents" : ""}`
    : `${field.label}: not yet known`;
  return (
    <form className="pf-count" data-key={field.key} data-part={field.part_id} data-band={field.band || undefined} data-changed={changed ? "true" : undefined} onSubmit={commit}>
      <input
        id={`pf-${field.key}-${field.part_id}`}
        className="pf-count-input field-plain"
        inputMode="numeric"
        aria-label={field.label}
        title={said}
        placeholder="—"
        size={Math.max(1, draft.length)}
        value={draft}
        onChange={(e) => { dirty.current = true; setDraft(e.target.value); }}
        onBlur={() => commit()}
      />
      <span className="pf-count-word" aria-hidden="true">{WORD[field.key] ?? field.label.toLowerCase()}</span>
    </form>
  );
}
