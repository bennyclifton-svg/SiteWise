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

/** Select all: list rules and a tick over the set. */
export const IconSelectAll = () => (
  <Icon>
    <path d="M2.5 4.5h5.5M2.5 8h5.5M2.5 11.5h3.5" />
    <path d="M9.2 9.2l1.7 1.7 3.2-3.6" />
  </Icon>
);

/** Response form: a sheet ruled as a table. */
export const IconResponseForm = () => (
  <Icon>
    <rect x="2.5" y="2.5" width="11" height="11" rx="1" />
    <path d="M2.5 6.5h11M2.5 10h11M6.5 2.5v11" />
  </Icon>
);

/** Update: an arrow turning back onto the page. */
export const IconUpdate = () => (
  <Icon>
    <path d="M13 8.2a5 5 0 1 1-1.5-3.6" />
    <path d="M13.2 2.4v3.2H10" />
  </Icon>
);

/** Documents panel: a sheet split by a rule, the register listed on the right. */
export const IconDocumentsPanel = () => (
  <Icon>
    <rect x="2.5" y="2.5" width="11" height="11" rx="1.5" />
    <path d="M9 2.5v11M10.8 5.5h1M10.8 8h1M10.8 10.5h1" />
  </Icon>
);

/** Log out: an arrow leaving an open door frame. */
export const IconLogOut = () => (
  <Icon>
    <path d="M6.5 2.5h-4v11h4" />
    <path d="M6 8h7.5M10.5 5l3 3-3 3" />
  </Icon>
);

/** Close: two strokes crossing at the corner of a sheet. */
export const IconClose = () => (
  <Icon>
    <path d="M4 4l8 8M12 4L4 12" />
  </Icon>
);

/** Download: an arrow into the tray, the reverse of upload. */
export const IconDownload = () => (
  <Icon>
    <path d="M8 2.5v9M4.8 8.2L8 11.5l3.2-3.3" />
    <path d="M2.5 11.5v2.5h11V11.5" />
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

/** Undo: an arrow turning back to the left. */
export const IconUndo = () => (
  <Icon strokeLinecap="round" strokeLinejoin="round">
    <path d="M5.5 3.5l-3 3 3 3" />
    <path d="M2.5 6.5h7a3.5 3.5 0 0 1 0 7H7" />
  </Icon>
);

/** Redo: the undo arrow mirrored. */
export const IconRedo = () => (
  <Icon strokeLinecap="round" strokeLinejoin="round">
    <path d="M10.5 3.5l3 3-3 3" />
    <path d="M13.5 6.5h-7a3.5 3.5 0 0 0 0 7H9" />
  </Icon>
);

/** Link in sequence: three bars stepping down, each tied to the next. */
export const IconSequence = () => (
  <Icon>
    <path d="M2 3.5h4M7.5 8H11M12.5 12.5H14" strokeWidth="2.5" />
    <path d="M6 3.5V8h1.5M11 8v4.5h1.5" strokeWidth="1" />
  </Icon>
);

/** Calendar: a page with rings. */
export const IconCalendar = () => (
  <Icon>
    <path d="M2.5 4h11v9.5h-11zM2.5 7h11M5.5 2.5v3M10.5 2.5v3" />
  </Icon>
);

/** Fit to screen: four corners pushed out. */
export const IconFitScreen = () => (
  <Icon>
    <path d="M2.5 6V2.5H6M10 2.5h3.5V6M13.5 10v3.5H10M6 13.5H2.5V10" />
  </Icon>
);

/** Fit to selection: corners around a smaller box. */
export const IconFitSelection = () => (
  <Icon>
    <path d="M2.5 5V2.5H5M11 2.5h2.5V5M13.5 11v2.5H11M5 13.5H2.5V11" />
    <path d="M6 6h4v4H6z" />
  </Icon>
);

/** Fit height: a double arrow between two rules. */
export const IconFitHeight = () => (
  <Icon>
    <path d="M3 2.5h10M3 13.5h10M8 4.5v7M6 6.5l2-2 2 2M6 9.5l2 2 2-2" />
  </Icon>
);

/** Links: a finish-to-start connector with its arrow. */
export const IconLinks = () => (
  <Icon>
    <path d="M2.5 4.5h4.5v7h4" />
    <path d="M9.5 9.5l2 2-2 2" />
  </Icon>
);

/** Delete: a bin with its lid. */
export const IconBin = () => (
  <Icon className="icon-bin" width="20" height="20" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2.5 4.5h11M6 4.5V2.5h4v2" />
    <path d="M4 4.5l.7 9h6.6l.7-9" />
  </Icon>
);

/** Add: a plus. */
export const IconPlus = () => (
  <Icon>
    <path d="M8 3v10M3 8h10" />
  </Icon>
);

/** More actions: three dots, as in a file browser row. */
export const IconMore = () => (
  <Icon className="icon-more" fill="currentColor" stroke="none">
    <circle cx="8" cy="3.5" r="1.25" />
    <circle cx="8" cy="8" r="1.25" />
    <circle cx="8" cy="12.5" r="1.25" />
  </Icon>
);

/** Not read: an eye struck through. */
export const IconEyeOff = () => (
  <Icon className="icon-eye-off" strokeLinecap="round" strokeLinejoin="round">
    <path d="M1.5 8s2.4-4.5 6.5-4.5S14.5 8 14.5 8s-2.4 4.5-6.5 4.5S1.5 8 1.5 8z" />
    <circle cx="8" cy="8" r="2" />
    <path d="M2.5 13.5l11-11" />
  </Icon>
);

/** Open: a page with an arrow out of its corner. */
export const IconOpen = () => (
  <Icon strokeLinecap="round" strokeLinejoin="round">
    <path d="M7 3H3.5v9.5H13V9" />
    <path d="M9.5 2.5h4v4M13.5 2.5L7.5 8.5" />
  </Icon>
);

/** Rename: a pencil. */
export const IconRename = () => (
  <Icon strokeLinecap="round" strokeLinejoin="round">
    <path d="M10.5 2.5l3 3-8 8h-3v-3z" />
    <path d="M9 4l3 3" />
  </Icon>
);

/** Discipline: a tag. */
export const IconTag = () => (
  <Icon strokeLinecap="round" strokeLinejoin="round">
    <path d="M2.5 2.5h5.5l5.5 5.5-5.5 5.5-5.5-5.5z" />
    <circle cx="5.5" cy="5.5" r="1" />
  </Icon>
);

/** Read: an open eye. */
export const IconEye = () => (
  <Icon strokeLinecap="round" strokeLinejoin="round">
    <path d="M1.5 8s2.4-4.5 6.5-4.5S14.5 8 14.5 8s-2.4 4.5-6.5 4.5S1.5 8 1.5 8z" />
    <circle cx="8" cy="8" r="2" />
  </Icon>
);

/** Print: a sheet leaving a printer. */
export const IconPrint = () => (
  <Icon strokeLinejoin="round">
    <path d="M4.5 6V2.5h7V6" />
    <path d="M4.5 11.5h-2v-5.5h11v5.5h-2" />
    <path d="M4.5 9.5h7v4h-7z" />
  </Icon>
);

/** Quantities: a dimension line with end ticks. */
export const IconQuantity = () => (
  <Icon strokeLinecap="round">
    <path d="M2.5 8h11M2.5 5v6M13.5 5v6" />
    <path d="M6 6.5v3M10 6.5v3" />
  </Icon>
);

/** Decimals: a point between two figures. */
export const IconDecimals = () => (
  <Icon strokeLinecap="round">
    <path d="M3 4.5v7" />
    <circle cx="6.5" cy="11.2" r=".4" fill="currentColor" />
    <rect x="9" y="4.5" width="4" height="7" rx="2" />
  </Icon>
);

/** Freeze: a padlock, the revision locked as the baseline. */
export const IconLock = () => (
  <Icon strokeLinejoin="round">
    <rect x="3.5" y="7" width="9" height="6.5" rx="1" />
    <path d="M5.5 7V5a2.5 2.5 0 0 1 5 0v2" />
  </Icon>
);
