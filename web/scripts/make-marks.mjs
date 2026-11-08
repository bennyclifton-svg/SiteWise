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
// String-seeded generator for the newer marks, so each mark is named by its seed.
function seeded(str) {
  let h = 2166136261;
  for (const c of String(str)) h = Math.imul(h ^ c.charCodeAt(0), 16777619);
  return () => { h += 0x6D2B79F5; let t = h; t = Math.imul(t ^ t >>> 15, t | 1); t ^= t + Math.imul(t ^ t >>> 7, t | 61); return ((t ^ t >>> 14) >>> 0) / 4294967296; };
}
const g = n => String(+n.toFixed(2));
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

// The selected navigation item: a dry brush stroke (owner pick B, 8 Nov 2026).
// Solid to 82% of its length, then a short, sharp feather gone by 95%, so the
// paint runs past the longest label (owner, 8 Nov 2026). Broken
// dry streaks run from the first bristle and thicken as the paint runs out.
{
  const r = seeded("dry"), W = 260, T = 6, B = 42, hold = .82, fall = .13, curve = 1.45;
  const load = t => t <= hold ? 1 - .03 * (t / hold) : .97 * (1 - Math.pow(Math.min(1, (t - hold) / fall), curve));
  const bodyEnd = (hold + fall * .78) * W, top = [], bot = [];
  for (let x = 3; x <= bodyEnd; x += 5) { top.push([x, T + (r() - .5) * .7]); bot.push([x, B + (r() - .5) * .7]); }
  const rag = []; for (let i = 0; i <= 9; i++) rag.push([bodyEnd + (r() - .5) * 16, T + (B - T) * i / 9]);
  const body = "M" + [...top, ...rag, ...bot.reverse(), [1.6, B - 6], [1.4, (T + B) / 2], [2.2, T + 6]].map(p => `${g(p[0])} ${g(p[1])}`).join(" L") + " Z";
  let stops = `<stop offset="0" stop-opacity="1"/><stop offset="${g(hold)}" stop-opacity=".97"/>`;
  for (let i = 1; i <= 12; i++) { const t = hold + fall * i / 12; stops += `<stop offset="${g(Math.min(1, t))}" stop-opacity="${g(load(t))}"/>`; }
  let streaks = "";
  for (let i = 0; i < 46; i++) {
    const y = T + 1 + r() * (B - T - 2), w = .3 + r() * .85, base = .18 + r() * .32;
    for (let x = r() * W * .12; x < W;) {
      const x2 = Math.min(W, x + 18 + r() * 110), op = Math.min(.55, base * (1 + x / W));
      streaks += `<path d="M${g(x)} ${g(y)} L${g(x2)} ${g(y + (r() - .5) * .5)}" stroke-opacity="${g(op)}" stroke-width="${g(w)}"/>`;
      x = x2 + r() * 26 * (1 - .5 * (x / W)) + .5;
    }
  }
  let bristles = "";
  for (let i = 0; i < 18; i++) {
    const y = T + 1 + r() * (B - T - 2), x0 = (hold - .06) * W, x1 = (hold + fall) * W + r() * 6;
    bristles += `<path d="M${g(x0)} ${g(y)} Q${g((x0 + x1) / 2)} ${g(y + (r() - .5) * .6)} ${g(x1)} ${g(y + (r() - .5))}" stroke-width="${g(.6 + r() * .7)}"/>`;
  }
  write("brush-nav.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="260" height="48" viewBox="0 0 260 48" preserveAspectRatio="none">
    <defs>
      <linearGradient id="load" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="260" y2="0">${stops}</linearGradient>
      <mask id="m"><rect width="260" height="48" fill="#fff"/><g fill="none" stroke="#000" stroke-linecap="round">${streaks}</g></mask>
      <filter id="e" x="-2%" y="-10%" width="104%" height="120%"><feTurbulence type="fractalNoise" baseFrequency=".04 .6" numOctaves="2" seed="5"/><feDisplacementMap in="SourceGraphic" scale="1.6"/></filter>
    </defs>
    <g mask="url(#m)" filter="url(#e)"><path d="${body}" fill="url(#load)"/><g fill="none" stroke="url(#load)" stroke-linecap="round">${bristles}</g></g></svg>`);
}

// The ground-line button (owner pick 8 Nov 2026): a draftsperson's pen. Lines are
// straight with a slight bow, thicker mid-stroke and lighter where the pen lands
// and lifts. CSS stretches the edge and ground strokes to the button and repeats
// the hatch tile, so no button is drawn at runtime.
{
  const hand = { tremor: .06, bow: .32, taper: .6, wvar: .18 };
  const penShape = (pts, w) => {
    const L = [], R = [], n = pts.length;
    for (let i = 0; i < n; i++) {
      const t = i / (n - 1), [x, y] = pts[i], a = pts[Math.max(0, i - 1)], b = pts[Math.min(n - 1, i + 1)];
      const dx = b[0] - a[0], dy = b[1] - a[1], l = Math.hypot(dx, dy) || 1, nx = -dy / l, ny = dx / l;
      const ww = w * (1 - hand.taper + hand.taper * Math.pow(Math.sin(Math.PI * t), .4)) / 2;
      L.push([x + nx * ww, y + ny * ww]); R.push([x - nx * ww, y - ny * ww]);
    }
    return "M" + [...L, ...R.reverse()].map(p => `${g(p[0])} ${g(p[1])}`).join("L") + "Z";
  };
  const line = (r, x0, y0, x1, y1, bowScale = 1) => {
    const len = Math.hypot(x1 - x0, y1 - y0) || 1, n = Math.max(5, Math.ceil(len / 2.5));
    const nx = -(y1 - y0) / len, ny = (x1 - x0) / len, bow = (r() - .5) * 2 * hand.bow * Math.min(1, len / 40) * bowScale;
    const p1 = r() * 6.28, p2 = r() * 6.28, k = len / (7 + r() * 4), pts = [];
    for (let i = 0; i <= n; i++) {
      const t = i / n, o = bow * 4 * t * (1 - t) + hand.tremor * (Math.sin(t * k + p1) * .6 + Math.sin(t * k * 2.3 + p2) * .4);
      pts.push([x0 + (x1 - x0) * t + nx * o, y0 + (y1 - y0) * t + ny * o]);
    }
    return pts;
  };
  const weight = (r, w) => w * (1 + (r() - .5) * hand.wvar);
  const svg = (w, h, body, stretch = true) => `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}"${stretch ? ' preserveAspectRatio="none"' : ""}>${body}</svg>`;
  const fill = (d, op) => `<path d="${d}" fill-opacity="${op}"/>`;

  // Edges are drawn long and thin so stretching changes their length, never
  // their weight. The frame sits 2.5px outside the button, so each edge runs
  // past the corners like a sketched box.
  for (const [name, seed] of [["pen-edge-top.svg", "top"], ["pen-edge-bottom.svg", "bottom"]]) {
    const r = seeded(seed);
    write(name, svg(240, 5, fill(penShape(line(r, r() * 2.5, 2.5, 240 - r() * 2.5, 2.5 + (r() - .5) * .4, .6), weight(r, 1.05)), .72)));
  }
  for (const [name, seed] of [["pen-edge-left.svg", "left"], ["pen-edge-right.svg", "right"]]) {
    const r = seeded(seed);
    write(name, svg(5, 41, fill(penShape(line(r, 2.5, r() * 1.5, 2.5 + (r() - .5) * .4, 41 - r() * 1.5), weight(r, 1.05)), .72)));
  }
  { const r = seeded("ground"); write("pen-ground.svg", svg(240, 5, fill(penShape(line(r, 0, 2.5, 240, 2.5 + (r() - .5) * .5, .6), weight(r, 1.6)), .88))); }

  // Hatch: a 54px tile of 45 degree strokes on a 4.5px set-square spacing.
  // A stroke crossing the tile edge is drawn again one tile over, so it repeats cleanly.
  {
    const r = seeded("hatch"), TW = 54, TH = 8;
    let d = "";
    for (let i = 0; i < 12; i++) {
      const x = i * 4.5 + (r() - .5) * .44, len = TH - .4 - r() * 1.2, w = weight(r, .72), op = g(.42 + (r() - .5) * .08), own = r() * 1e9;
      for (const s of [0, -TW]) d += fill(penShape(line(seeded(own), x + s, .4, x + s + len, .4 + len), w), op);
    }
    write("pen-hatch.svg", svg(TW, TH, d, false));
  }

  // Icons in the same hand at 16px.
  {
    const r = seeded("refresh"), cx = 8, cy = 8, R = 5.6, a0 = 40 * Math.PI / 180, a1 = 328 * Math.PI / 180, p = r() * 6.28, pts = [];
    for (let i = 0; i <= 28; i++) { const t = i / 28, a = a0 + (a1 - a0) * t, rr = R + hand.bow * .5 * Math.sin(t * 3.1 + p) + hand.tremor * .5 * Math.sin(t * 17 + p); pts.push([cx + Math.cos(a) * rr, cy + Math.sin(a) * rr]); }
    let d = fill(penShape(pts, weight(r, 1.55)), 1);
    const ex = cx + Math.cos(a1) * R, ey = cy + Math.sin(a1) * R, dir = Math.atan2(Math.cos(a1), -Math.sin(a1));
    for (const sg of [1, -1]) { const ang = dir + Math.PI + sg * .62, L = 3.6 + (r() - .5) * .7; d += fill(penShape(line(r, ex, ey, ex + Math.cos(ang) * L, ey + Math.sin(ang) * L), weight(r, 1.4)), 1); }
    write("pen-refresh.svg", svg(16, 16, d, false));
  }
  {
    const r = seeded("back"), cy = 8, tip = 2.2;
    let d = fill(penShape(line(r, 14, cy + (r() - .5) * .4, tip, cy), weight(r, 1.5)), 1);
    for (const sg of [1, -1]) d += fill(penShape(line(r, tip, cy, tip + 4.6, cy + sg * 4.4), weight(r, 1.4)), 1);
    write("pen-back.svg", svg(16, 16, d, false));
  }
}

// Pencil lines: a graphite rule, an ember underline, and a dotted blue underline.
// `wear` sets how much grain eats the line; the summary underlines wear less
// so they read at a glance (owner, 8 Nov 2026).
function pencil(name, color, width, dots, seed, wear = "2.2 -.5") {
  const W = 480, H = 6, r = rng(seed), a = wave(seed, W);
  let d = "";
  for (let x = 0; x <= W; x += 6) d += (x ? " L" : "M") + `${x} ${f(3 + a(x) * .55 + (x && x < W ? (r() - .5) * .3 : 0))}`;
  const dash = dots ? ` stroke-dasharray="1.8 3.6" stroke-linecap="round"` : "";
  write(name, `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}"><filter id="p" x="0" y="-60%" width="100%" height="220%"><feTurbulence type="fractalNoise" baseFrequency=".95" numOctaves="2" seed="${seed}" result="n"/><feDisplacementMap in="SourceGraphic" in2="n" scale=".9" result="d"/><feColorMatrix in="n" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 ${wear}" result="g"/><feComposite in="d" in2="g" operator="in"/></filter><path d="${d}" fill="none" stroke="${color}" stroke-width="${width}"${dash} filter="url(#p)"/></svg>`);
}
pencil("pencil-graphite.svg", "#2f2924", 1.7, false, 3);
pencil("pencil-ember.svg", "#a85e2c", 1.9, false, 11, "2.6 -.25");
pencil("pencil-azure.svg", "#2d5e9e", 2.1, true, 19, "2.6 -.25");
// Dark theme: the same lines in the dark sheet's ember and azure.
pencil("pencil-ember-dark.svg", "#d9935e", 1.9, false, 11, "2.6 -.25");
pencil("pencil-azure-dark.svg", "#93cedd", 2.1, true, 19, "2.6 -.25");

// Grain: a barely-there sandstone for the navigation band.
write("stone.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="360" height="360"><filter id="s"><feTurbulence type="fractalNoise" baseFrequency=".7" numOctaves="3" seed="8" stitchTiles="stitch" result="fine"/><feColorMatrix in="fine" values="0 0 0 0 .42  0 0 0 0 .38  0 0 0 0 .32  0 0 0 .55 -.24" result="a"/><feTurbulence type="fractalNoise" baseFrequency=".012 .03" numOctaves="3" seed="2" stitchTiles="stitch" result="bed"/><feColorMatrix in="bed" values="0 0 0 0 .55  0 0 0 0 .5  0 0 0 0 .42  0 0 0 .16 -.05" result="b"/><feMerge><feMergeNode in="b"/><feMergeNode in="a"/></feMerge></filter><rect width="100%" height="100%" filter="url(#s)"/></svg>`);

console.log("marks written to", out);
