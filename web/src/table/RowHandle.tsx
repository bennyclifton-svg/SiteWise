import { type PointerEvent as ReactPointerEvent } from "react";
import "./interaction.css";

export function RowHandle({ disabled = false, onPointerDown }: { disabled?: boolean; onPointerDown: (e: ReactPointerEvent<HTMLElement>) => void }) {
  return <span className="table-row-handle" title={disabled ? "Clear sorting to reorder" : "Drag to reorder"} aria-hidden="true" onPointerDown={disabled ? undefined : onPointerDown} data-disabled={disabled || undefined} />;
}

// Both grids share the threshold, pointer lifecycle and Escape cancellation.
// The owning surface supplies its geometry and domain-specific move operation.
export function startRowDrag(e: ReactPointerEvent<HTMLElement>, actions: {
  onStart: () => void; onMove: (e: PointerEvent) => void; onDrop: () => void; onStop: () => void;
}) {
  if (e.pointerType === "mouse" && e.button !== 0) return () => {};
  e.preventDefault(); e.stopPropagation();
  const y0 = e.clientY;
  let started = false;
  const stop = () => {
    window.removeEventListener("pointermove", move);
    window.removeEventListener("pointerup", up);
    window.removeEventListener("pointercancel", cancel);
    window.removeEventListener("keydown", key, true);
    actions.onStop();
  };
  const move = (ev: PointerEvent) => {
    if (!started && Math.abs(ev.clientY - y0) < 4) return;
    if (!started) { started = true; actions.onStart(); }
    actions.onMove(ev);
  };
  const up = () => { stop(); if (started) actions.onDrop(); };
  const cancel = () => stop();
  const key = (ev: KeyboardEvent) => {
    if (ev.key !== "Escape") return;
    ev.preventDefault(); ev.stopPropagation(); stop();
  };
  window.addEventListener("pointermove", move);
  window.addEventListener("pointerup", up);
  window.addEventListener("pointercancel", cancel);
  window.addEventListener("keydown", key, true);
  return stop;
}
