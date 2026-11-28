// A replay of the Profile's Update over the sample project, the way a real
// run looks: documents are taken three at a time, each one's passages are
// labelled and then read for evidence, and a value is written the moment the
// first passage that settles it is read. So values land out of order, in a
// different order every time, wherever the document that holds them happens
// to be. The documents, values and quotes are T05's own
// (testing/synthetic/scenarios/ST05-base/corpus). The timings are a replay,
// shortened; the page shows no elapsed time, so it claims no speed.

import type { Box, Shot } from "./shotMotion";

export interface ReplayDoc { name: string; short: string; units: number }
/** One document's state at a moment, as the workbench strip draws it. */
export interface DocState { state: "queued" | "reading" | "done"; pass: "label" | "evidence"; labelled: number; evidence: number }
export interface Find { at: number; what: string; excerpt: string; where: string }
export interface Replay {
  shot: Shot;
  times: number[];
  docs: ReplayDoc[];
  /** Ends of each passage, per document: [label ends, evidence ends]. */
  passes: { start: number; label: number[]; evidence: number[] }[];
  finds: Find[];
  /** Values the update writes in the title block. */
  filled: number;
  end: number;
}

const DOCS: { name: string; short: string }[] = [
  { name: "Owner brief - 41 Tallowwood Avenue", short: "Owner brief" },
  { name: "Architectural design statement", short: "Design statement" },
  { name: "Detail and boundary survey report", short: "Survey report" },
  { name: "Planning certificate s10.7 - Lot 23 DP 30418", short: "s10.7" },
  { name: "A-201 Proposed ground floor plan Rev B", short: "A-201 B" },
  { name: "Statement of environmental effects", short: "SEE" },
  { name: "Geotechnical report SG-26-0742", short: "SG-26-0742" },
  { name: "Hazardous materials survey CS-26-0388", short: "CS-26-0388" },
  { name: "RE Timing for the building work", short: "RE Timing" },
  { name: "Fee proposal - Architectural services", short: "Fee proposal" },
];
const [BRIEF, STATEMENT, SURVEY, S107, A201, SEE, GEOTECH, HAZMAT, TIMING, FEE] = DOCS.map((_, i) => i);

// Title-block rows by their cover pair in shots.json (glint, write), the
// summary phrase each one fills, and where the corpus says it: each pair is a
// document and words that are in it.
type Said = [doc: number, words: string][];
const ADDRESS = "41 Tallowwood Avenue, Concord NSW 2137";
const FIELDS: { boxes: [number, number]; summary?: number; what: string; said: Said }[] = [
  { boxes: [0, 1], what: "Site address", said: [[SURVEY, ADDRESS], [BRIEF, ADDRESS], [STATEMENT, ADDRESS], [S107, ADDRESS]] },
  { boxes: [2, 3], what: "Lot and DP", said: [[S107, "Lot 23 DP 30418"], [SURVEY, "Lot 23 DP 30418"], [SEE, "Lot 23 DP 30418"]] },
  { boxes: [4, 5], summary: 35, what: "Local government area", said: [[S107, "CITY OF CANADA BAY"], [SEE, "a development application to the City of Canada Bay"]] },
  { boxes: [6, 7], summary: 36, what: "Zone", said: [[S107, "Zone R2 Low Density Residential under Canada Bay Local Environmental Plan 2013"], [SEE, "zoned R2 Low Density Residential"]] },
  { boxes: [8, 9], summary: 34, what: "Site sqm", said: [[SURVEY, "Site area by survey: 557.4 m2."]] },
  { boxes: [10, 11], summary: 39, what: "Planning", said: [[SEE, "This statement accompanies a development application to the City of Canada Bay."]] },
  { boxes: [12, 13], summary: 30, what: "Work type", said: [[STATEMENT, "The proposal is a rear addition and internal alterations to a brick house"]] },
  { boxes: [14, 15], what: "Building category", said: [[BRIEF, "The existing building is a single-storey brick house."]] },
  { boxes: [16, 17], summary: 31, what: "Building class", said: [[STATEMENT, "Building classification: Class 1a dwelling."], [FEE, "This is a Class 1a house"]] },
  { boxes: [18, 19], summary: 32, what: "Storeys", said: [[BRIEF, "a single-storey rear addition"], [SEE, "The addition is single storey"]] },
  { boxes: [20, 21], summary: 33, what: "Proposed GFA sqm", said: [[A201, "Total after the works 195 m2"], [STATEMENT, "Proposed gross floor area after the works: 195 m2"]] },
  { boxes: [22, 23], summary: 37, what: "Quality level", said: [[BRIEF, "Mid-range finishes: good quality and durable, nothing high end."]] },
  { boxes: [24, 25], summary: 38, what: "Client", said: [[BRIEF, "Client: Tomas and Leah Ferreira"], [FEE, "Client: Tomas and Leah Ferreira"], [SURVEY, "Client: Tomas and Leah Ferreira"]] },
  { boxes: [26, 27], what: "Procurement route", said: [[FEE, "a tender list of three or four builders"], [TIMING, "a construction certificate and a tender with three or four builders"]] },
];
// The construction budget is the user's (rust square): it is on the page
// before the update and the update reads it again without moving it.
const HELD = new Set([28, 29, 40]);

// Scope rows in building order, with where the corpus calls for each system.
// A row with nothing quotable still shades; it just has no "latest find" line.
const SCOPE: { what: string; said: Said }[] = [
  { what: "Planning and approvals", said: [[SEE, "A stormwater concept plan accompanies the application."], [S107, "The land is not identified as bush fire prone land."]] },
  { what: "Site, ground and external works", said: [[A201, "Paved terrace and rear deck, with the steps down to the garden moved"], [GEOTECH, "footprint and footing recommendations"], [HAZMAT, "Rear skillion and laundry lean-to (to be demolished)"]] },
  { what: "Substructure", said: [[SURVEY, "a suspended timber floor about 450 mm above ground on brick piers"], [GEOTECH, "footprint and footing recommendations"]] },
  { what: "Structure", said: [[A201, "the removed internal walls on steel beams (refer to structural)"]] },
  { what: "Building envelope", said: [[A201, "Demolition outline: rear skillion, laundry lean-to and two internal walls"], [STATEMENT, "rear skillion and laundry lean-to are removed"]] },
  { what: "Hydraulic services", said: [[A201, "New kitchen, living and dining, with the bathroom rebuilt"], [STATEMENT, "heat-pump hot water and LED lighting throughout"]] },
  { what: "Mechanical services", said: [[BRIEF, "Ducted or split air-conditioning to the new living area"]] },
  { what: "Electrical services", said: [[BRIEF, "More power points and a data point in the kitchen and living area"], [STATEMENT, "LED lighting throughout"]] },
  { what: "Active fire protection", said: [[STATEMENT, ""], [A201, ""]] },
];

const CONCURRENT = 3;
const LEAD = 700; // the Update press, then the first document is taken

/** A small seeded random, so one replay is one consistent run. */
function random(seed: number) {
  let s = seed >>> 0 || 1;
  return () => { s ^= s << 13; s ^= s >>> 17; s ^= s << 5; return (s >>> 0) / 4294967296; };
}

export function makeReplay(profile: Shot, seed = Date.now()): Replay {
  const rnd = random(seed);
  const pick = <T,>(xs: T[]) => xs[Math.floor(rnd() * xs.length)];

  // The documents in the order the worker happens to take them.
  const order = DOCS.map((_, i) => i).sort(() => rnd() - 0.5);
  const docs: ReplayDoc[] = DOCS.map(d => ({ ...d, units: 6 + Math.floor(rnd() * 10) }));
  const passes: Replay["passes"] = DOCS.map(() => ({ start: 0, label: [], evidence: [] }));
  const free = Array.from({ length: CONCURRENT }, (_, i) => LEAD + i * 180);
  for (const d of order) {
    const lane = free.indexOf(Math.min(...free));
    let t = free[lane];
    passes[d].start = t;
    for (let u = 0; u < docs[d].units; u++) passes[d].label.push((t += 50 + rnd() * 90));
    t += 120;
    for (let u = 0; u < docs[d].units; u++) passes[d].evidence.push((t += 90 + rnd() * 170));
    free[lane] = t + 140;
  }
  // When a document first says something: one of its evidence passages.
  const saidIn = (d: number) => pick(passes[d].evidence);
  const firstOf = (said: Said) => said
    .map(([d, words]) => ({ d, words, at: saidIn(d) }))
    .reduce((a, b) => (b.at < a.at ? b : a));

  const times = profile.boxes.map(b => b[5]);
  const finds: Find[] = [];
  for (const f of FIELDS) {
    const { d, words, at } = firstOf(f.said);
    times[f.boxes[0]] = at;
    times[f.boxes[1]] = at + 220;
    if (f.summary !== undefined) times[f.summary] = at + 520;
    finds.push({ at, what: f.what, excerpt: words, where: docs[d].short });
  }

  // Scope: each system's cells shade as passages that call for them are
  // read; its count is written with its last cell.
  const rows = scopeRows(profile.boxes);
  rows.forEach((row, r) => {
    const s = SCOPE[r] ?? SCOPE[SCOPE.length - 1];
    const said = row.cells.map(i => {
      const first = firstOf(s.said);
      times[i] = first.at;
      return first;
    });
    if (said.length === 0) return;
    const last = Math.max(...said.map(x => x.at));
    for (const i of row.counts) times[i] = last + 120;
    const first = said.reduce((a, b) => (b.at < a.at ? b : a));
    if (first.words) finds.push({ at: first.at, what: s.what, excerpt: first.words, where: docs[first.d].short });
  });

  finds.sort((a, b) => a.at - b.at);
  const boxes = profile.boxes.filter((_, i) => !HELD.has(i));
  const kept = times.filter((_, i) => !HELD.has(i));
  const end = Math.max(...passes.map(p => p.evidence[p.evidence.length - 1]), ...kept) + 500;
  return { shot: { ...profile, boxes }, times: kept, docs, passes, finds, filled: FIELDS.length, end };
}

/** Groups the scope's covers into rows: its cells ("pop") and its count and check ("write"). */
function scopeRows(boxes: Box[]): { cells: number[]; counts: number[] }[] {
  const rows: { y: number; cells: number[]; counts: number[] }[] = [];
  const firstScope = boxes.findIndex(b => b[6] === "pop");
  if (firstScope < 0) return [];
  boxes.forEach((b, i) => {
    if (i < firstScope) return;
    const mid = b[1] + b[3] / 2;
    let row = rows.find(r => Math.abs(r.y - mid) < 0.015);
    if (!row) rows.push((row = { y: mid, cells: [], counts: [] }));
    (b[6] === "pop" ? row.cells : row.counts).push(i);
  });
  return rows.sort((a, b) => a.y - b.y);
}

/** Each document's state at time t. */
export function docsAt(r: Replay, t: number): DocState[] {
  return r.passes.map(p => {
    const labelled = p.label.filter(x => x <= t).length;
    const evidence = p.evidence.filter(x => x <= t).length;
    const units = p.label.length;
    if (t < p.start) return { state: "queued", pass: "label", labelled: 0, evidence: 0 };
    if (evidence >= units) return { state: "done", pass: "evidence", labelled, evidence };
    return { state: "reading", pass: labelled >= units ? "evidence" : "label", labelled, evidence };
  });
}

/** The latest find at time t, as the strip's "latest find" line shows it. */
export function findAt(r: Replay, t: number): Find | null {
  let latest: Find | null = null;
  for (const f of r.finds) if (f.at <= t) latest = f;
  return latest;
}
