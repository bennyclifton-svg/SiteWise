import { useState } from "react";
import { Pending } from "./Activity";
import type { Profile, ProfileField } from "./profileApi";
import "./profile-confirm.css";

/** One value a document stated that no one has accepted yet. */
interface Unconfirmed {
  field: ProfileField;
  shown: string;
  documents: number;
  differ: boolean;
  site: boolean;
}

// Site and client first, then what the building is, then the rest: the
// order a PM checks a new project in.
const FIRST = ["site.address", "proj.client", "site.lot", "hdr.building_class", "hdr.subclass", "hdr.work_type", "det.state"];
const rank = (key: string) => {
  const i = FIRST.indexOf(key);
  return i >= 0 ? i : key.startsWith("hdr.") ? 20 : key.startsWith("proj.") ? 30 : 40;
};

function shownValue(f: ProfileField): string {
  const option = f.options?.find((o) => o.id === f.value);
  return option?.label ?? f.value;
}

/** unconfirmed lists amber document readings on the whole project, not yet the user's. */
function unconfirmed(data: Profile, siteFields: ProfileField[]): Unconfirmed[] {
  const whole = data.parts[0]?.id ?? "";
  const fields = [...siteFields, ...data.details, ...data.header, ...data.facts, ...data.compliance.flatMap((g) => g.rows)];
  const seen = new Set<string>();
  const out: Unconfirmed[] = [];
  for (const f of fields) {
    if (f.band !== "amber" || !f.value || (f.part_id && whole && f.part_id !== whole) || seen.has(f.key)) continue;
    seen.add(f.key);
    out.push({
      field: f,
      shown: shownValue(f),
      documents: new Set(f.sources.map((s) => s.document_id)).size,
      differ: f.alternatives.length > 0,
      site: f.key === "site.address" || f.key === "site.lot",
    });
  }
  return out.sort((a, b) => rank(a.field.key) - rank(b.field.key));
}

type Save = (key: string, body: { part_id?: string; value?: string | null }) => void | Promise<void>;

/** ProfileConfirm is one list of the values documents stated that wait for
 * the PM's word, most important first, each with how many documents gave
 * it. Accepting writes the PM's value through the usual routes; the site
 * address goes onto the site record. */
export function ProfileConfirm({ data, siteFields, onSet, onSetSite }: { data: Profile; siteFields: ProfileField[]; onSet: Save; onSetSite: Save }) {
  const [busy, setBusy] = useState("");
  const items = unconfirmed(data, siteFields);
  if (items.length === 0) return null;
  const agreeing = items.filter((i) => i.documents > 1 && !i.differ);

  const accept = (i: Unconfirmed) => (i.site ? onSetSite : onSet)(i.field.key, { part_id: i.field.part_id || undefined, value: i.field.value });
  const acceptAll = async () => {
    setBusy("all");
    try {
      // One at a time: each write returns the whole profile.
      for (const i of agreeing) await accept(i);
    } finally {
      setBusy("");
    }
  };

  return (
    <details className="pf-confirm" open>
      <summary>
        <strong>To confirm</strong> · {items.length} value{items.length === 1 ? "" : "s"} read from documents
      </summary>
      {agreeing.length > 1 && (
        <p className="pf-confirm-all">
          <button type="button" className="btn btn-small" disabled={busy !== ""} onClick={() => void acceptAll()}>
            {busy === "all" ? <Pending delayed>Accepting…</Pending> : `Accept the ${agreeing.length} that several documents agree on`}
          </button>
        </p>
      )}
      <ul>
        {items.map((i) => (
          <li key={i.field.key}>
            <span className="pf-confirm-label">{i.field.label}</span>
            <span className="pf-confirm-value">{i.shown}</span>
            <span className="pf-confirm-basis" data-differ={i.differ || undefined}>
              {i.differ ? "Documents differ" : `Read in ${i.documents} document${i.documents === 1 ? "" : "s"}${i.documents > 1 ? ", none disagree" : ""}`}
            </span>
            <button type="button" className="table-action" disabled={busy !== ""} aria-label={`Accept ${i.field.label}: ${i.shown}`} onClick={() => accept(i)}>
              Accept
            </button>
          </li>
        ))}
      </ul>
    </details>
  );
}
