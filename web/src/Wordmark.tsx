// The lockup PNG is a silhouette. Masking it with the theme text colour
// keeps one asset readable on both the light and dark canvases.

const RATIO = 584 / 129;

export function Wordmark({ height = 28 }: { height?: number }) {
  const width = Math.round(height * RATIO);
  return (
    <span className="wordmark" style={{ width, height }} aria-hidden="true">
      <span className="wordmark-ink" />
    </span>
  );
}
