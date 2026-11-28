// The Datum lockup, Section version (owner pick 8 Nov 2026). Its one source is
// public/brand/sitewise-wordmark.svg; brand.css paints it as a mask in the
// current text colour, so this and the static pages in /legal show the same
// logo and it reads on both themes.

// The lockup's own proportions (its viewBox is 238 x 51).
const RATIO = 238 / 51;

export function Wordmark({ height = 30 }: { height?: number }) {
  return <span className="wordmark" style={{ width: Math.round(height * RATIO), height }} aria-hidden="true" />;
}
