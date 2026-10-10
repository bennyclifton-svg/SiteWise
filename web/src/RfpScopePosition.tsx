import type { ReportBlock } from "./reportApi";

interface ScopeItem {
  key: string;
  label: string;
  value: "I" | "O" | "X" | "ask";
  party?: string;
  assumption?: string;
  why?: string;
}

const groups = [
  { value: "ask", title: "To confirm before issue", position: "To confirm" },
  { value: "I", title: "Included in the fee", position: "Included" },
  { value: "O", title: "Separately priced options", position: "Option" },
  { value: "X", title: "Excluded from this appointment", position: "Excluded" },
] as const;

function isScopeItem(value: unknown): value is ScopeItem {
  if (!value || typeof value !== "object") return false;
  const item = value as Record<string, unknown>;
  return typeof item.key === "string" && typeof item.label === "string"
    && groups.some(group => group.value === item.value)
    && (item.party === undefined || typeof item.party === "string")
    && (item.assumption === undefined || typeof item.assumption === "string")
    && (item.why === undefined || typeof item.why === "string");
}

export function RfpScopePosition({ block }: { block: ReportBlock }) {
  const items = block.basis?.positions;
  // Saved wording wins. Never reconstruct an edited or unsupported block from
  // today's controls, which could disagree with the report the user saved.
  if (block.edited || block.source_missing || block.basis?.method !== "rfp_discipline"
    || !Array.isArray(items) || !items.length || !items.every(isScopeItem)) {
    return <p className="report-wording">{block.text}</p>;
  }
  const lines = block.text.split("\n");
  const prefixes = ["Included in the fee:", "Optional: price each on its own line in R1:", "Excluded: the party named does it:", "To confirm before issue:"];
  const first = lines.findIndex(line => prefixes.some(prefix => line.startsWith(prefix)));
  if (first < 0) return <p className="report-wording">{block.text}</p>;
  const before = lines.slice(0, first).join("\n");
  const after = lines.slice(first).filter(line => !prefixes.some(prefix => line.startsWith(prefix))).join("\n");

  return <div className="rfp-scope-position">
    <p className="report-wording">Specific inclusions, separately priced options and responsibilities for this appointment.</p>
    {before && <p className="report-wording report-muted">{before}</p>}
    {groups.map(group => {
      const rows = items.filter(item => item.value === group.value);
      if (!rows.length) return null;
      return <div className="report-table-scroll" key={group.value}>
        <table className="report-table rfp-scope-table">
          <caption>{group.title}</caption>
          <colgroup><col className="rfp-scope-item" /><col className="rfp-scope-state" /><col /></colgroup>
          <thead><tr><th scope="col">Scope item</th><th scope="col">Position</th><th scope="col">Conditions / responsibility</th></tr></thead>
          <tbody>{rows.map(item => <tr key={item.key}>
            <th scope="row">{item.label}</th>
            <td>{group.position}</td>
            <td>{item.assumption && <p>{item.assumption}</p>}{item.value === "ask" && !item.assumption && item.why && <p>{item.why}</p>}{item.value === "X" && <p>{item.party || "Party not named"}</p>}{!item.assumption && item.value !== "X" && !(item.value === "ask" && item.why) && <span aria-label="No additional condition">—</span>}</td>
          </tr>)}</tbody>
        </table>
        {group.value === "O" && <p className="rfp-scope-note">Price each option on its own line in R1.</p>}
      </div>;
    })}
    {after && <p className="report-wording">{after}</p>}
  </div>;
}
