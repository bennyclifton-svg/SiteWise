// Covers for the two tender pictures, measured by hand from the pictures
// themselves (both captured at 1000 CSS px, 2x): procurement.webp has a
// 68.7 px row pitch under a header ending at 153 px; tender.webp has its
// stage rows at 60 px from 178 px. Colours are the surfaces read off the
// pictures. Times are ms from when each picture starts to play.
import type { Box, Shot } from "./shotMotion";

const PLAIN = "#fefeff";

// ---------- Procurement: quotes land on their rows, one at a time ----------

const P = { w: 2000, h: 1000, head: 153, pitch: 68.7 };
const FIRM_X = [311, 676, 1041]; // left edge of Firm 1, 2, 3 cells
const FIRM_W = 363;
// [row, firm] of each cell that holds a quote in the picture.
const QUOTES: [number, number][] = [[0, 0], [0, 1], [1, 0], [3, 0], [5, 0], [6, 0], [9, 0]];

export function procurementShot(order: number[]): Shot {
  const boxes: Box[] = [];
  QUOTES.forEach(([row, firm], i) => {
    const x = FIRM_X[firm], y = P.head + row * P.pitch + 1.5, h = P.pitch - 3;
    const at = 250 + order.indexOf(i) * 480;
    boxes.push([x / P.w, y / P.h, FIRM_W / P.w, h / P.h, "", at, "glint"]);
    boxes.push([x / P.w, y / P.h, FIRM_W / P.w, h / P.h, PLAIN, at + 260, "write", "upload"]);
  });
  return { width: P.w, height: P.h, cssWidth: 1000, boxes };
}
export const QUOTE_COUNT = QUOTES.length;

// ---------- Tender: the architect's quote read stage by stage ----------

const T = { w: 2000, h: 1039, x: 638, cw: 456 };
const tb = (y: number, h: number, colour: string, at: number, how: Box[6], x = T.x, w = T.cw): Box =>
  [x / T.w, y / T.h, w / T.w, h / T.h, colour, at, how];

export function tenderShot(): Shot {
  const boxes: Box[] = [tb(16, 86, "#e8effa", 200, "write", 645, 440)]; // the firm's name heads its column
  // Stage 1 to 4: each line is spotted in the quote, then its price is written.
  [179, 239, 299, 359].forEach((y, i) => {
    boxes.push(tb(y, 57, "", 650 + i * 420, "glint"));
    boxes.push(tb(y, 57, "#ffffff", 850 + i * 420, "write"));
  });
  // The total is Go's sum of the lines; then the award.
  boxes.push(tb(419, 57, "#eaedf1", 2600, "write", 638, 160));
  boxes.push(tb(419, 57, "#eaedf1", 2600, "write", 798, 296));
  boxes.push(tb(428, 40, "#eaedf1", 3250, "pop", 800, 150));
  // The option and the stages it did not price.
  [[479, 91], ...Array.from({ length: 8 }, (_, i) => [572 + i * 60, 58])].forEach(([y, h], i) => {
    boxes.push(tb(y, Math.min(h, T.h - y), "#ffffff", 1500 + i * 110, "write"));
  });
  return { width: T.w, height: T.h, cssWidth: 1000, boxes };
}
