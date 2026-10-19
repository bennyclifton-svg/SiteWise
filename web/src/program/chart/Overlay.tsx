// Overlay layer (plan 9.2): the only layer that changes during a drag. In
// WP-P7 it draws the hover tooltip and reserves the drag ghost, snap guide
// and selection box groups for WP-P8, which writes their transforms directly
// from a requestAnimationFrame callback (no React state per frame) through
// the refs this component hands back.

import { forwardRef, useImperativeHandle, useRef } from "react";

export interface OverlayHandle {
  /** Drag ghost group: WP-P8 sets transform on its children each frame. */
  ghosts: SVGGElement | null;
  /** Vertical snap guide line. */
  guide: SVGLineElement | null;
  /** Shift-drag selection box. */
  box: SVGRectElement | null;
}

export interface OverlayProps {
  tooltip: { x: number; y: number; text: string } | null;
  height: number;
}

export const Overlay = forwardRef<OverlayHandle, OverlayProps>(function Overlay({ tooltip, height }, ref) {
  const ghosts = useRef<SVGGElement>(null);
  const guide = useRef<SVGLineElement>(null);
  const box = useRef<SVGRectElement>(null);
  useImperativeHandle(ref, () => ({ ghosts: ghosts.current, guide: guide.current, box: box.current }), []);
  return (
    <g className="pg-layer-overlay" aria-hidden="true">
      <g ref={ghosts} className="pg-ghosts" />
      <line ref={guide} className="pg-guide" x1={0} x2={0} y1={0} y2={height} visibility="hidden" />
      <rect ref={box} className="pg-selbox" x={0} y={0} width={0} height={0} visibility="hidden" />
      {tooltip && (
        <g className="pg-tip" transform={`translate(${Math.round(tooltip.x)},${Math.round(tooltip.y)})`}>
          <rect x={0} y={-18} width={tooltip.text.length * 6.6 + 12} height={20} rx={3} />
          <text x={6} y={-4}>
            {tooltip.text}
          </text>
        </g>
      )}
    </g>
  );
});
