import { useEffect, useRef, useState, type ButtonHTMLAttributes, type ReactNode } from "react";
import "./ground-button.css";

// The page-level button: a pen-drawn box standing on a ground line, with trailing
// section hatch under the primary one. Every line is a static mark from
// scripts/make-marks.mjs applied by CSS; nothing is drawn or measured here.

type Props = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> & {
  icon?: "refresh" | "back";
  /** Primary carries the hatch; secondary keeps the ground line only. */
  primary?: boolean;
  small?: boolean;
  busy?: boolean;
  children: ReactNode;
};

// Fast work would only flicker; the same wait the app uses before "Saving…".
const BUSY_DELAY_MS = 400;
// Longest a running cycle takes to finish before the button settles.
const CYCLE_MS = 2400;

export function GroundButton({ icon, primary = false, small = false, busy = false, className, children, type = "button", ...rest }: Props) {
  const sketching = useSketching(busy);
  const classes = ["ground-btn", primary ? "is-primary" : "", small ? "is-small" : "", className ?? ""].filter(Boolean).join(" ");
  return (
    <button type={type} className={classes} aria-busy={busy || undefined} data-sketching={sketching.on || undefined} {...rest}>
      <span className="ground-frame" aria-hidden="true" />
      {primary && <span className="ground-hatch" aria-hidden="true" onAnimationIteration={sketching.cycleEnded} />}
      {icon && <span className={`ground-icon ground-icon-${icon}`} aria-hidden="true" />}
      <span className="ground-label">{children}</span>
    </button>
  );
}

/** Starts the sketch once work has run past the delay; when work ends, lets the current cycle finish. */
function useSketching(busy: boolean) {
  const [on, setOn] = useState(false);
  const stopping = useRef(false); // work has ended; settle at the end of this cycle
  useEffect(() => {
    if (busy) {
      stopping.current = false;
      const t = setTimeout(() => setOn(true), BUSY_DELAY_MS);
      return () => clearTimeout(t);
    }
    if (!on) return;
    if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) { setOn(false); return; }
    stopping.current = true;
    const t = setTimeout(() => setOn(false), CYCLE_MS);
    return () => clearTimeout(t);
  }, [busy, on]);
  return { on, cycleEnded: () => { if (stopping.current) { stopping.current = false; setOn(false); } } };
}
