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

// Wear on the smoked perspex ribbon: a few long scores, cloth-buffing swirls and
// dust. Each mark is a pale line (light scattering out of the groove) over a faint
// dark twin half a pixel down, so it reads on both the light and dark sheet.
// Marks near a side edge are drawn again one tile over, so the tile repeats cleanly.
{
  const W = 1600, H = 260, r = rng(41), rand = (lo, hi) => lo + r() * (hi - lo);
  const defs = [], marks = [];
  const twin = (d, w, pale, dark) => `<path d="${d}" stroke="#3c3228" stroke-opacity="${dark}" stroke-width="${w}" transform="translate(.6 .6)"/><path d="${d}" stroke="${pale}" stroke-width="${w}"/>`;
  const wrap = (x, span, draw) => { draw(0); if (x - span < 0) draw(W); if (x + span > W) draw(-W); };
  for (let i = 0; i < 58; i++) {
    const x = rand(0, W), y = rand(0, H), ang = r() * Math.PI, len = rand(120, 420), a = rand(.05, .14);
    const dx = Math.cos(ang) * len, dy = Math.sin(ang) * len, bend = (r() - .5) * len * .08, w = f(.3 + r() * .55);
    const peak = (.25 + r() * .5).toFixed(2);
    wrap(Math.min(x, x + dx), Math.abs(dx) + 1, s => {
      const id = `g${defs.length}`, x0 = x + s, x1 = x0 + dx, y1 = y + dy;
      defs.push(`<linearGradient id="${id}" gradientUnits="userSpaceOnUse" x1="${f(x0)}" y1="${f(y)}" x2="${f(x1)}" y2="${f(y1)}"><stop offset="0" stop-color="#fff" stop-opacity="0"/><stop offset="${peak}" stop-color="#fff" stop-opacity="${a.toFixed(3)}"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></linearGradient>`);
      const d = `M${f(x0)} ${f(y)}Q${f(x0 + dx / 2 - Math.sin(ang) * bend)} ${f(y + dy / 2 + Math.cos(ang) * bend)} ${f(x1)} ${f(y1)}`;
      marks.push(twin(d, w, `url(#${id})`, (a * .45).toFixed(3)));
    });
  }
  for (let i = 0; i < 20; i++) {
    const cx = rand(0, W), cy = rand(-H * .5, H * 1.5), R = rand(60, 260);
    for (let j = 0; j < 10; j++) {
      const rad = R + rand(-28, 28), st = rand(0, 6.283), sw = rand(.08, .45), a = rand(.03, .09), w = f(rand(.3, .7));
      const x0 = cx + Math.cos(st) * rad, y0 = cy + Math.sin(st) * rad;
      wrap(cx, rad + 1, s => {
        const d = `M${f(x0 + s)} ${f(y0)}A${f(rad)} ${f(rad)} 0 0 1 ${f(cx + s + Math.cos(st + sw) * rad)} ${f(cy + Math.sin(st + sw) * rad)}`;
        marks.push(twin(d, w, `rgba(255,255,255,${a.toFixed(3)})`, (a * .5).toFixed(3)));
      });
    }
  }
  const dust = (n, fill, lo, hi) => {
    let d = "";
    for (let i = 0; i < n; i++) { const x = rand(1, W - 1), y = rand(0, H), q = f(rand(.25, .7)); d += `M${f(x)} ${f(y)}m-${q} 0a${q} ${q} 0 1 0 ${f(q * 2)} 0a${q} ${q} 0 1 0 -${f(q * 2)} 0`; }
    marks.push(`<path d="${d}" fill="${fill}" fill-opacity="${rand(lo, hi).toFixed(2)}" stroke="none"/>`);
  };
  for (let k = 0; k < 2; k++) { dust(80, "#fff", .1, .3); dust(80, "#3c3228", .04, .1); }
  write("perspex-wear.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" fill="none" stroke-linecap="round"><defs>${defs.join("")}</defs>${marks.join("")}</svg>`);
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

// The selected navigation item: one squared brush stroke. It starts fully
// loaded, carries faint bristle hairlines along its length, and runs dry toward
// the end, where the paint thins and breaks into separate bristle streaks.
{
  const W = 260, H = 48, top = 6.5, bot = 41.5, r = rng(77), tw = wave(81, W), bw = wave(83, W);
  const dry = W * .6; // where the brush starts to run out
  // Body: near-rectangular, a touch fuller at the start, edges only slightly uneven.
  const pts = [];
  for (let x = 3; x <= dry; x += 5) pts.push([x, top + tw(x) * .7 - (x < 16 ? (16 - x) * .06 : 0)]);
  for (let y = top + 2; y <= bot - 2; y += 3.5) pts.push([dry - 4 + r() * 18, y]); // ragged where the load gives out
  for (let x = dry; x >= 3; x -= 5) pts.push([x, bot + bw(x) * .7 + (x < 16 ? (16 - x) * .06 : 0)]);
  pts.push([1.6 + r() * .8, bot - 6], [1.2 + r() * .8, (top + bot) / 2], [1.8 + r() * .8, top + 6]);
  // Bristles: one per hair across the width, each running out at its own point.
  let bristles = "";
  const n = 64;
  for (let k = 0; k < n; k++) {
    const y = top + .6 + (k + r() * .6) * (bot - top - 1.2) / n;
    const end = W * (.62 + Math.pow(r(), .75) * .36), drift = (r() - .5) * 1.4;
    bristles += `<path d="M${f(W * .3)} ${f(y)} Q${f((W * .3 + end) / 2)} ${f(y + drift * .3)} ${f(end)} ${f(y + drift)}" stroke-width="${(.45 + r() * .8).toFixed(2)}"/>`;
  }
  // Hairlines: barely-visible streaks where a bristle carried less paint.
  let hairs = "";
  for (let k = 0; k < 18; k++) {
    const y = top + 1.5 + r() * (bot - top - 3), x0 = r() * W * .35;
    hairs += `<path d="M${f(x0)} ${f(y)} L${W} ${f(y + (r() - .5) * 1.2)}" stroke-opacity="${(.1 + r() * .22).toFixed(2)}" stroke-width="${(.3 + r() * .35).toFixed(2)}"/>`;
  }
  write("brush-nav.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}" preserveAspectRatio="none">
    <defs>
      <linearGradient id="load" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="${W}" y2="0"><stop offset="0" stop-opacity="1"/><stop offset=".5" stop-opacity=".96"/><stop offset=".7" stop-opacity=".72"/><stop offset=".88" stop-opacity=".32"/><stop offset="1" stop-opacity="0"/></linearGradient>
      <mask id="m"><rect width="${W}" height="${H}" fill="#fff"/><g fill="none" stroke="#000" stroke-linecap="round">${hairs}</g></mask>
      <filter id="e" x="-2%" y="-10%" width="104%" height="120%"><feTurbulence type="fractalNoise" baseFrequency=".04 .6" numOctaves="2" seed="5"/><feDisplacementMap in="SourceGraphic" scale="1.3"/></filter>
    </defs>
    <g mask="url(#m)" filter="url(#e)">
      <path d="M${pts.map(p => `${f(p[0])} ${f(p[1])}`).join(" L")} Z" fill="url(#load)"/>
      <g fill="none" stroke="url(#load)" stroke-linecap="round">${bristles}</g>
    </g></svg>`);
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

// Grain: a barely-there sandstone for the navigation band.
write("stone.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="360" height="360"><filter id="s"><feTurbulence type="fractalNoise" baseFrequency=".7" numOctaves="3" seed="8" stitchTiles="stitch" result="fine"/><feColorMatrix in="fine" values="0 0 0 0 .42  0 0 0 0 .38  0 0 0 0 .32  0 0 0 .55 -.24" result="a"/><feTurbulence type="fractalNoise" baseFrequency=".012 .03" numOctaves="3" seed="2" stitchTiles="stitch" result="bed"/><feColorMatrix in="bed" values="0 0 0 0 .55  0 0 0 0 .5  0 0 0 0 .42  0 0 0 .16 -.05" result="b"/><feMerge><feMergeNode in="b"/><feMergeNode in="a"/></feMerge></filter><rect width="100%" height="100%" filter="url(#s)"/></svg>`);

console.log("marks written to", out);
