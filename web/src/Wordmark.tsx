// The datum lockup: the surveyor's level datum, with "sitewise" standing on
// its datum line, which runs on as one graphite pencil stroke. Drawn in the
// current text colour so it reads on both themes.

const RATIO = 132 / 30;

export function Wordmark({ height = 30 }: { height?: number }) {
  return (
    <span className="wordmark" style={{ width: Math.round(height * RATIO), height }} aria-hidden="true">
      <svg viewBox="0 0 132 30" width="100%" height="100%">
        <defs>
          <filter id="wm-graphite" x="-2%" y="-80%" width="104%" height="260%">
            <feTurbulence type="fractalNoise" baseFrequency=".95" numOctaves="2" seed="4" result="n" />
            <feDisplacementMap in="SourceGraphic" in2="n" scale=".7" result="d" />
            <feColorMatrix in="n" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 2.4 -.45" result="g" />
            <feComposite in="d" in2="g" operator="in" />
          </filter>
        </defs>
        <circle cx="9.6" cy="19.5" r="8" fill="none" stroke="currentColor" strokeWidth="1.5" />
        <path d="M9.6 19.5V11.5A8 8 0 0 0 1.6 19.5Z M9.6 19.5v8A8 8 0 0 0 17.6 19.5Z" fill="currentColor" />
        <path d="M9.6 11.5v16" stroke="currentColor" strokeWidth="1.1" />
        <path
          d="M0.3 19.6 L12 19.4 L30 19.7 L50 19.4 L72 19.65 L92 19.45 L112 19.7 L126 19.4 L131.5 19.75"
          fill="none" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" filter="url(#wm-graphite)"
        />
        <text x="23.5" y="16.6" fill="currentColor" fontFamily="Instrument Sans, ui-sans-serif, system-ui" fontWeight="600" fontSize="22" letterSpacing="-0.5">sitewise</text>
      </svg>
    </span>
  );
}
