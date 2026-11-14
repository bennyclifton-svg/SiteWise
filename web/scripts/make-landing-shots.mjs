// Captures the landing page's three animated screens from a running dev
// server, with the boxes its motion plays over (web/src/landing/shots.json).
//
//   node scripts/make-landing-shots.mjs <project-url> <out-dir>
//
// The project must be a sample one (the page is public): the pictures in the
// tree are T05 House Alterations from testing/synthetic. Each PNG is then
// converted to WebP by hand and placed in src/landing/. Run it again only
// when a screen's layout changes; the boxes are measured from the live DOM
// at the same moment as the picture, so the two always agree.
//
// A box is [x, y, w, h, colour, at, how]: position and size as fractions of
// the picture, the colour that hides it (the commonest colour inside it in
// the picture, which is the surface behind the ink), when it is revealed in
// milliseconds, and how ("write" wipes left to right, "pop" fades a mark in,
// "glint" is the cyan wash of a passage being read).
import { writeFileSync } from "node:fs";
import { chromium } from "playwright";

const [projectUrl, outDir] = process.argv.slice(2);
if (!projectUrl || !outDir) {
  console.error("usage: node scripts/make-landing-shots.mjs <project-url> <out-dir>");
  process.exit(1);
}
const origin = new URL(projectUrl).origin;

const browser = await chromium.launch();
const page = await (await browser.newContext({ viewport: { width: 1600, height: 1000 }, deviceScaleFactor: 2, colorScheme: "light" })).newPage();
await page.goto(origin + "/dev/login", { waitUntil: "networkidle" });
await page.goto(projectUrl, { waitUntil: "networkidle" });
await page.waitForTimeout(2000);
await page.click("button[aria-label='Hide documents']");
await page.waitForTimeout(800);

// Runs in the page. Returns boxes in CSS pixels relative to the clip.
function measure(view, clip) {
  const inClip = r => r.width > 0 && r.top >= clip.y - 1 && r.bottom <= clip.y + clip.height + 1 && r.left >= clip.x - 1 && r.right <= clip.x + clip.width + 1;
  // "surface" is filled in afterwards from the picture itself (see surfaces).
  const ground = () => "surface";
  const box = (r, colour, at, how, grow = 1.5) => [r.left - clip.x - grow, r.top - clip.y - grow, r.width + grow * 2, r.height + grow * 2, colour, Math.round(at), how];
  const out = [];

  if (view === "profile") {
    // 1. Title block: each row is read (glint), then its value is written.
    const rows = [...document.querySelectorAll(".pf-titleblock .pf-row, .pf-row")]
      .filter((row, i, all) => all.indexOf(row) === i)
      .map(row => {
        const label = row.querySelector("label");
        const field = label && document.getElementById(label.htmlFor);
        return { row, field, r: row.getBoundingClientRect() };
      })
      .filter(x => x.field && inClip(x.r) && String(x.field.value ?? x.field.textContent ?? "").trim() !== "")
      .sort((a, b) => Math.round(a.r.left / 40) - Math.round(b.r.left / 40) || a.r.top - b.r.top);
    rows.forEach(({ row, field, r }, i) => {
      const f = field.getBoundingClientRect();
      const value = { left: f.left, top: r.top + 1, width: r.right - f.left - 1, height: r.height - 2 };
      out.push(box(r, "", 200 + i * 75, "glint", 0));
      out.push(box(value, ground(field), 420 + i * 75, "write", 0));
    });
    // 2. The summary sentence: each phrase the documents settled is written in.
    let at = 1650;
    for (const el of document.querySelectorAll(".pf-summary .pf-t")) {
      for (const r of el.getClientRects()) {
        if (!inClip(r)) continue;
        out.push(box({ left: r.left, top: r.top, width: r.width, height: r.height + 5 }, ground(el), at, "write"));
        at += 95;
      }
    }
    // 3. Scope: the systems the documents call for are shaded in, row by row.
    const empty = document.querySelector(".sw-cell:not([data-on])");
    const cs = empty && getComputedStyle(empty);
    out.emptyCell = cs ? { background: cs.backgroundColor, border: cs.borderTopColor, radius: cs.borderTopLeftRadius } : null;
    at += 150;
    for (const entry of document.querySelectorAll(".sw-entry")) {
      const er = entry.getBoundingClientRect();
      if (!inClip(er)) continue;
      let t = at;
      for (const cell of entry.querySelectorAll(".sw-cell[data-on], .sw-cell[data-band], .sw-cell[data-check]")) {
        out.push(box(cell.getBoundingClientRect(), "cell", t, "pop", 0.5));
        t += 22;
      }
      for (const el of entry.querySelectorAll(".sw-count, .sw-flag, .sw-check")) {
        const r = el.getBoundingClientRect();
        if (r.width) out.push(box(r, ground(el), t + 40, "write"));
      }
      at += 85;
    }
  }

  if (view === "cost") {
    // Lines are set up first, section by section; then an award drops the
    // firm's fee into the contract column and the totals follow.
    const rows = [...document.querySelectorAll(".cost-grid tbody tr")].filter(tr => inClip(tr.getBoundingClientRect()));
    let line = 0, fig = 0;
    for (const tr of rows) {
      if (!tr.classList.contains("cost-row")) continue;
      const cells = [...tr.children];
      const first = cells[0].getBoundingClientRect(), item = cells[2].getBoundingClientRect();
      out.push(box({ left: first.left + 2, top: first.top + 1, width: item.right - first.left - 4, height: first.height - 2 }, ground(cells[2]), 150 + line * 55, "write", 0));
      line++;
    }
    const start = 150 + line * 55 + 350;
    for (const tr of rows) {
      const figures = [...tr.querySelectorAll("td.cost-num")].filter(td => /[1-9]/.test(td.textContent));
      if (!figures.length) continue;
      figures.forEach((td, k) => {
        const r = td.getBoundingClientRect();
        const inner = { left: r.left + 2, top: r.top + 1, width: r.width - 4, height: r.height - 2 };
        if (k === 0) out.push(box(inner, "", start + fig * 150, "glint", 0));
        out.push(box(inner, ground(td), start + fig * 150 + 120 + k * 110, "write", 0));
      });
      fig++;
    }
  }

  if (view === "program") {
    // Activities are listed, then each bar is drawn against its dates.
    const chartLeft = Math.max(...[...document.querySelectorAll(".pg-row")].map(r => r.getBoundingClientRect().right));
    const chart = document.querySelector(".pg-layer-bars");
    let i = 0;
    for (const row of document.querySelectorAll(".pg-row")) {
      const r = row.getBoundingClientRect();
      // A row the picture cuts through is still covered, down to the edge.
      const bottom = Math.min(r.bottom, clip.y + clip.height);
      if (r.top < clip.y - 1 || bottom - r.top < 4) continue;
      const left = Math.max(r.left, clip.x);
      out.push(box({ left: left + 1, top: r.top + 1, width: chartLeft - left - 2, height: bottom - r.top - 1 }, ground(row), 150 + i * 60, "write", 0));
      // The chart's cover overlaps its neighbours by a pixel, so a link that
      // runs along a row's edge stays hidden until that row is drawn.
      out.push(box({ left: chartLeft + 1, top: r.top - 1, width: clip.x + clip.width - chartLeft - 2, height: bottom - r.top + 2 }, ground(chart), 420 + i * 85, "write", 0));
      i++;
    }
  }
  return { boxes: [...out], emptyCell: out.emptyCell ?? null };
}

// Replaces each "surface" with the commonest colour inside the box, read
// from the picture, so a cover matches what is painted there whatever paints it.
const sampler = await (await browser.newContext({ deviceScaleFactor: 1 })).newPage();
async function surfaces(png, boxes) {
  const colours = await sampler.evaluate(async ({ b64, boxes }) => {
    const img = new Image();
    img.src = "data:image/png;base64," + b64;
    await img.decode();
    const canvas = Object.assign(document.createElement("canvas"), { width: img.width, height: img.height });
    const ctx = canvas.getContext("2d");
    ctx.drawImage(img, 0, 0);
    return boxes.map(([x, y, w, h, colour]) => {
      if (colour !== "surface") return colour;
      const sx = Math.max(0, Math.round(x * 2)), sy = Math.max(0, Math.round(y * 2));
      const d = ctx.getImageData(sx, sy, Math.max(1, Math.min(img.width - sx, Math.round(w * 2))), Math.max(1, Math.min(img.height - sy, Math.round(h * 2)))).data;
      const seen = new Map();
      for (let i = 0; i < d.length; i += 4) {
        const key = (d[i] << 16) | (d[i + 1] << 8) | d[i + 2];
        seen.set(key, (seen.get(key) ?? 0) + 1);
      }
      const [top] = [...seen].sort((a, b) => b[1] - a[1])[0];
      return "#" + top.toString(16).padStart(6, "0");
    });
  }, { b64: png.toString("base64"), boxes });
  boxes.forEach((box, i) => { box[4] = colours[i]; });
}

const shots = [
  { name: "profile", nav: null, clip: { x: 235, y: 140, width: 1350, height: 810 } },
  { name: "cost", nav: "Cost Plan", clip: { x: 245, y: 135, width: 1340, height: 715 } },
  { name: "program", nav: "Program", clip: { x: 264, y: 360, width: 1272, height: 624 }, prepare: ["Expand all", "Fit to screen"] },
];
const result = {};
for (const shot of shots) {
  if (shot.nav) {
    await page.click(`nav button:has-text('${shot.nav}')`);
    await page.waitForTimeout(2500);
  }
  for (const label of shot.prepare ?? []) {
    await page.click(`button[aria-label='${label}']`);
    await page.waitForTimeout(1200);
  }
  await page.mouse.move(5, 5); // no hover state in the picture
  const { boxes, emptyCell } = await page.evaluate(`(${measure})(${JSON.stringify(shot.name)}, ${JSON.stringify(shot.clip)})`);
  const png = await page.screenshot({ path: `${outDir}/${shot.name}.png`, clip: shot.clip });
  await surfaces(png, boxes);
  const f = n => Math.round(n * 1e5) / 1e5;
  result[shot.name] = {
    width: shot.clip.width * 2,
    height: shot.clip.height * 2,
    // The CSS width the screen was captured at: one of its pixels is
    // 1/cssWidth of the picture, which keeps a drawn line its true weight.
    cssWidth: shot.clip.width,
    ...(emptyCell ? { emptyCell } : {}),
    boxes: boxes.map(([x, y, w, h, colour, at, how]) => [f(x / shot.clip.width), f(y / shot.clip.height), f(w / shot.clip.width), f(h / shot.clip.height), colour, at, how]),
  };
  console.log(shot.name, boxes.length, "boxes");
}
writeFileSync(`${outDir}/shots.json`, JSON.stringify(result));
await browser.close();
