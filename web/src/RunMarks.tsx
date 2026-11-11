// Marks a run leaves until they are seen, for any surface (web/DESIGN.md,
// "Run-time motion and marks"; styles in trace-motion.css). The surface
// places them beside the value they speak for.

/**
 * A drawing's revision triangle: this value changed in the last update and
 * you have not cleared it yet. The tooltip says what it was and what it is.
 */
export function RevisionMark({ text }: { text: string }) {
  return <span className="rev-mark" role="img" aria-label={text} title={text} />;
}

/** Passages read for a field during an update; they count as sources once committed. */
export function PendingCount({ n }: { n: number }) {
  const words = `${n} passage${n === 1 ? "" : "s"} read for this, not yet applied`;
  return <sup className="pend-count" title={words} aria-label={words}>+{n}</sup>;
}
