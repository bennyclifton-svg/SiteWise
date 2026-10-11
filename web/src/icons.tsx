// One authored icon set: 16px grid, 1.5px stroke, square caps, drafted like
// title-block symbols. Icons are decorative; the text beside them carries the
// meaning for assistive technology.

import type { SVGProps } from "react";

function Icon({ children, ...rest }: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 16 16"
      width="16"
      height="16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="square"
      strokeLinejoin="miter"
      aria-hidden="true"
      focusable="false"
      {...rest}
    >
      {children}
    </svg>
  );
}

/** Confirmed: a checked box, the drafter's tick. */
export const IconConfirmed = () => (
  <Icon>
    <path d="M3.5 8.5l3 3 6-7" />
  </Icon>
);

/** Check: a revision cloud's warning triangle. */
export const IconCheck = () => (
  <Icon>
    <path d="M8 2.5l6 11H2z" strokeLinejoin="round" />
    <path d="M8 6.5v3.2M8 11.6v.2" />
  </Icon>
);

/** Not set: an open, dashed box. */
export const IconNotSet = () => (
  <Icon>
    <path d="M3 3h2.5M7 3h2M10.5 3H13v2.5M13 7v2M13 10.5V13h-2.5M9 13H7M5.5 13H3v-2.5M3 9V7M3 5.5V3" />
  </Icon>
);

/** Not checked: a clock stopped by a stroke. */
export const IconNotChecked = () => (
  <Icon>
    <circle cx="8" cy="8" r="5.5" />
    <path d="M8 5v3l1.8 1.2" />
    <path d="M2.5 13.5l11-11" />
  </Icon>
);

/** Set by you: a drafting pencil. */
export const IconYou = () => (
  <Icon>
    <path d="M10.5 2.5l3 3-7.5 7.5H3v-3z" />
    <path d="M9 4l3 3" />
  </Icon>
);

export const IconUpload = () => (
  <Icon>
    <path d="M8 10.5V2.5M4.8 5.5L8 2.3l3.2 3.2" />
    <path d="M2.5 9.5v4h11v-4" />
  </Icon>
);

export const IconRetry = () => (
  <Icon>
    <path d="M13 8a5 5 0 1 1-1.6-3.7" />
    <path d="M13 2.5v3h-3" />
  </Icon>
);

export const IconSupersedes = () => (
  <Icon>
    <path d="M2.5 8h9M8.5 4.8L11.7 8l-3.2 3.2" />
    <path d="M13.5 3v10" />
  </Icon>
);

export const IconStored = () => (
  <Icon>
    <path d="M2.5 4.5h11v9h-11z" />
    <path d="M2.5 4.5l1.5-2h8l1.5 2M6 8h4" />
  </Icon>
);

/** Delete: a bin with its lid. */
export const IconBin = () => (
  <Icon>
    <path d="M2.5 4.5h11M6 4.5V2.5h4v2" />
    <path d="M4 4.5l.7 9h6.6l.7-9M6.8 7v4M9.2 7v4" />
  </Icon>
);
