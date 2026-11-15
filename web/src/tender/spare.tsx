// Both tender tables keep room for three firms (owner, 15 Nov 2026), as the
// Procurement board does: with fewer quotes in, the spare columns stand
// empty, headed "Firm 2", "Firm 3", so one quote does not fill the width.
export const FIRM_COLUMNS = 3;

/** The numbers of the spare columns after `count` firms: [2, 3] after one. */
export function spareFirms(count: number): number[] {
  return Array.from({ length: Math.max(0, FIRM_COLUMNS - count) }, (_, i) => count + i + 1);
}

export const spareCols = (spare: number[]) => spare.map(n => <col key={`spare-${n}`} />);

export const spareHeads = (spare: number[]) => spare.map(n =>
  <th scope="col" key={`spare-${n}`} className="prices-firm tender-spare"><span className="tender-spare-name">Firm {n}</span></th>);

export const spareCells = (spare: number[]) => spare.map(n => <td key={`spare-${n}`} className="tender-spare" />);
