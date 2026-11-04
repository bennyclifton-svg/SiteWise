// Generates the Trace hand marks as small static SVG files in src/assets/marks.
// Run once with `node scripts/make-marks.mjs` when a mark needs to change.
// Seeded so the same files come out every time; nothing is drawn at runtime.

import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const out = resolve(import.meta.dirname, "../src/assets/marks");
mkdirSync(out, { recursive: true });

function rng(seed) { let s = seed >>> 0; return () => { s = (s * 1664525 + 1013904223) >>> 0; return s / 4294967296; }; }
const f = n => n.toFixed(1);
// Periodic wave over width w, so tiles join without a seam.
function wave(seed, w) {
  const r = rng(seed);
  const parts = [[1, 1.0], [3, .55], [7, .35], [13, .22], [29, .12]].map(([k, a]) => [k, a * (.7 + r() * .6), r() * 6.283]);
  return x => parts.reduce((s, [k, a, p]) => s + Math.sin(x / w * 6.283 * k + p) * a, 0) / 1.6;
}
function write(name, svg) { writeFileSync(resolve(out, name), svg.replace(/\n\s*/g, "")); }

// Torn edge of the butter paper. Black is kept, so these are masks.
// The sheet keeps everything above its edge; the fringe is the paler fibrous band below it.
{
  const W = 1600, H = 30, edge = 10;
  const a = wave(5, W), r = rng(9), r2 = rng(17);
  const sheet = [], fringe = [];
  for (let x = 0; x <= W; x += 2) {
    const y = edge + a(x) * 2.6;
    sheet.push([x, y + (x === 0 || x === W ? 0 : (r() - .5) * .6)]);
    fringe.push([x, y + 6.5 + a(x * 1.7) * 1.2 + (x === 0 || x === W ? 0 : (r2() - .5) * 2.6)]);
  }
  const line = pts => pts.map(p => `${f(p[0])} ${f(p[1])}`).join(" L");
  write("tear-sheet.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}"><path d="M0 0 L${W} 0 L${line(sheet.slice().reverse())} Z"/></svg>`);
  write("tear-fringe.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}"><filter id="f"><feTurbulence type="fractalNoise" baseFrequency=".9" numOctaves="2" seed="3"/><feDisplacementMap in="SourceGraphic" scale="2.2"/></filter><path filter="url(#f)" d="M${line(sheet.map(([x, y]) => [x, y - 1.5]))} L${line(fringe.slice().reverse())} Z"/></svg>`);
}

// Highlighter: squared off, even height, with a slight wobble and uneven ink.
for (const [i, seed] of [[1, 31], [2, 47], [3, 63]]) {
  const W = 300, H = 40, r = rng(seed), top = wave(seed, W), bot = wave(seed + 5, W);
  const pts = [];
  for (let x = 3; x <= W - 3; x += 6) pts.push([x, 5 + top(x) * 1.6]);
  pts.push([W - 1.5 - r() * 2, 12], [W - 1, 24], [W - 2 - r() * 2, 34]);
  for (let x = W - 3; x >= 3; x -= 6) pts.push([x, 35 + bot(x) * 1.6]);
  pts.push([1.5 + r() * 2, 28], [1, 16]);
  let streaks = "";
  for (let k = 0; k < 6; k++) {
    const y = 9 + k * 4.6 + (r() - .5) * 1.4, x0 = 4 + r() * 30, x1 = W - 4 - r() * 40;
    streaks += `<path d="M${f(x0)} ${f(y)} L${f(x1)} ${f(y + (r() - .5) * 1.2)}" stroke="#000" stroke-opacity="${(.18 + r() * .2).toFixed(2)}" stroke-width="${f(1.6 + r() * 2)}" stroke-linecap="round"/>`;
  }
  if (i === 2) write("brush-check.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" preserveAspectRatio="none"><filter id="g" x="-2%" y="-10%" width="104%" height="120%"><feTurbulence type="fractalNoise" baseFrequency=".02 .6" numOctaves="2" seed="${seed}" result="t"/><feColorMatrix in="t" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1.1 .25" result="a"/><feComposite in="SourceGraphic" in2="a" operator="in"/></filter><g filter="url(#g)" opacity=".62"><path d="M${pts.map(p => `${f(p[0])} ${f(p[1])}`).join(" L")} Z" fill="#E2AA48" fill-opacity=".74"/>${streaks.replaceAll('stroke="#000"', 'stroke="#E2AA48"')}</g></svg>`);
  write(`brush-${i}.svg`, `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" preserveAspectRatio="none"><filter id="g" x="-2%" y="-10%" width="104%" height="120%"><feTurbulence type="fractalNoise" baseFrequency=".02 .6" numOctaves="2" seed="${seed}" result="t"/><feColorMatrix in="t" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1.1 .25" result="a"/><feComposite in="SourceGraphic" in2="a" operator="in"/></filter><g filter="url(#g)"><path d="M${pts.map(p => `${f(p[0])} ${f(p[1])}`).join(" L")} Z" fill="#000" fill-opacity=".74"/>${streaks}</g></svg>`);
}

// The selected navigation item: one broad paint stroke with a dry tail.
{
  const W = 260, H = 48, r = rng(77), top = wave(81, W), bot = wave(83, W);
  const pts = [];
  for (let x = 4; x <= W - 26; x += 5) pts.push([x, 7 + top(x) * 2.2 + (x < 20 ? (20 - x) * .25 : 0)]);
  pts.push([W - 16, 12], [W - 6, 17], [W - 14, 24], [W - 3, 30], [W - 18, 36]);
  for (let x = W - 26; x >= 4; x -= 5) pts.push([x, 41 + bot(x) * 2.2 - (x < 20 ? (20 - x) * .25 : 0)]);
  pts.push([1, 30], [2.5, 18]);
  let gaps = "";
  for (let k = 0; k < 9; k++) {
    const y = 10 + r() * 28, x0 = W * (.45 + r() * .4);
    gaps += `<path d="M${f(x0)} ${f(y)} L${W} ${f(y + (r() - .5) * 2)}" stroke="#fff" stroke-opacity="${(.35 + r() * .5).toFixed(2)}" stroke-width="${f(.6 + r() * 1.4)}"/>`;
  }
  write("brush-nav.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" preserveAspectRatio="none"><mask id="m"><rect width="${W}" height="${H}" fill="#fff"/>${gaps}</mask><filter id="e" x="-2%" y="-10%" width="104%" height="120%"><feTurbulence type="fractalNoise" baseFrequency=".03 .5" numOctaves="2" seed="5"/><feDisplacementMap in="SourceGraphic" scale="2.4"/></filter><g mask="url(#m)" filter="url(#e)"><path d="M${pts.map(p => `${f(p[0])} ${f(p[1])}`).join(" L")} Z" fill="#000"/></g></svg>`);
}

// Pencil lines: a graphite rule, an ember underline, and a dotted blue underline.
function pencil(name, color, width, dots, seed) {
  const W = 480, H = 6, r = rng(seed), a = wave(seed, W);
  let d = "";
  for (let x = 0; x <= W; x += 6) d += (x ? " L" : "M") + `${x} ${f(3 + a(x) * .55 + (x && x < W ? (r() - .5) * .3 : 0))}`;
  const dash = dots ? ` stroke-dasharray="1.2 4.2" stroke-linecap="round"` : "";
  write(name, `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}"><filter id="p" x="0" y="-60%" width="100%" height="220%"><feTurbulence type="fractalNoise" baseFrequency=".95" numOctaves="2" seed="${seed}" result="n"/><feDisplacementMap in="SourceGraphic" in2="n" scale=".9" result="d"/><feColorMatrix in="n" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 2.2 -.5" result="g"/><feComposite in="d" in2="g" operator="in"/></filter><path d="${d}" fill="none" stroke="${color}" stroke-width="${width}"${dash} filter="url(#p)"/></svg>`);
}
pencil("pencil-graphite.svg", "#2f2924", 1.7, false, 3);
pencil("pencil-ember.svg", "#a85e2c", 1.45, false, 11);
pencil("pencil-azure.svg", "#2d5e9e", 1.6, true, 19);

// Grain: butter-paper fibre and a barely-there sandstone for the navigation band.
write("paper-fibre.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="420" height="420"><filter id="f"><feTurbulence type="fractalNoise" baseFrequency=".012 .55" numOctaves="2" seed="3" stitchTiles="stitch"/><feColorMatrix values="0 0 0 0 1  0 0 0 0 1  0 0 0 0 1  0 0 0 .9 -.42"/></filter><rect width="100%" height="100%" filter="url(#f)"/></svg>`);
write("stone.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="360" height="360"><filter id="s"><feTurbulence type="fractalNoise" baseFrequency=".7" numOctaves="3" seed="8" stitchTiles="stitch" result="fine"/><feColorMatrix in="fine" values="0 0 0 0 .42  0 0 0 0 .38  0 0 0 0 .32  0 0 0 .55 -.24" result="a"/><feTurbulence type="fractalNoise" baseFrequency=".012 .03" numOctaves="3" seed="2" stitchTiles="stitch" result="bed"/><feColorMatrix in="bed" values="0 0 0 0 .55  0 0 0 0 .5  0 0 0 0 .42  0 0 0 .16 -.05" result="b"/><feMerge><feMergeNode in="b"/><feMergeNode in="a"/></feMerge></filter><rect width="100%" height="100%" filter="url(#s)"/></svg>`);

console.log("marks written to", out);
