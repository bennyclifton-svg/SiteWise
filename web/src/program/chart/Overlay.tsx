// Overlay layer (plan 9.2): the only layer that changes during a drag.
// useDrag writes these nodes directly from a requestAnimationFrame callback
// (no React state per frame) through the refs this component hands back:
// the ghost (a resized or drawn bar), the rubber-band line, the snap guide,
// the selection box and the tooltip with the snapped date and badge.

import { forwardRef, useImperativeHandle, useRef } from "react";

export interface OverlayHandle {
  /** Ghost bar for a resize or a bar being drawn. */
  ghost: SVGRectElement | null;
  /** Rubber-band line from a link dot to the pointer. */
  band: SVGLineElement | null;
  /** Vertical snap guide line. */
  guide: SVGLineElement | null;
  /** Shift-drag selection box. */
  box: SVGRectElement | null;
  /** Tooltip group: snapped date, badge, "Held by Slab". */
  tip: SVGGElement | null;
}

export const Overlay = forwardRef<OverlayHandle, { height: number }>(function Overlay({ height }, ref) {
  const ghost = useRef<SVGRectElement>(null);
  const band = useRef<SVGLineElement>(null);
  const guide = useRef<SVGLineElement>(null);
  const box = useRef<SVGRectElement>(null);
  const tip = useRef<SVGGElement>(null);
  useImperativeHandle(ref, () => ({ ghost: ghost.current, band: band.current, guide: guide.current, box: box.current, tip: tip.current }), []);
  return (
    <g className="pg-layer-overlay" aria-hidden="true">
      <rect ref={ghost} className="pg-ghost" x={0} y={0} width={0} height={0} rx={2} visibility="hidden" />
      <line ref={band} className="pg-band" x1={0} x2={0} y1={0} y2={0} visibility="hidden" />
      <line ref={guide} className="pg-guide" x1={0} x2={0} y1={0} y2={height} visibility="hidden" />
      <rect ref={box} className="pg-selbox" x={0} y={0} width={0} height={0} visibility="hidden" />
      <g ref={tip} className="pg-tip" visibility="hidden">
        <rect x={0} y={-18} width={0} height={20} rx={3} />
        <text x={6} y={-4} />
      </g>
    </g>
  );
});

/** Shows the tooltip at (x, y) with text, or hides it when text is empty. */
export function showTip(tip: SVGGElement | null, x: number, y: number, text: string) {
  if (!tip) return;
  if (!text) {
    tip.setAttribute("visibility", "hidden");
    return;
  }
  const t = tip.querySelector("text")!;
  if (t.textContent !== text) {
    t.textContent = text;
    tip.querySelector("rect")!.setAttribute("width", String(Math.ceil(text.length * 6.4 + 12)));
  }
  tip.setAttribute("transform", `translate(${Math.round(x)},${Math.round(Math.max(20, y))})`);
  tip.setAttribute("visibility", "visible");
}
