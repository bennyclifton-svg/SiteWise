# Trace hand marks: how they reach the front end

Status: proposal for the owner, 3 November 2026. Nothing is built.

The mockups draw the marks with script and SVG noise filters so they could
be explored quickly. Production should not do that: noise filters are slow to
repaint on scroll and would differ on every render. Each mark ships instead
as a small static file, coloured by a CSS token and applied by one CSS class.
No GIFs: nothing moves, and GIF is heavy and low quality for this.

## The files

Kept in `web/src/assets/marks/`, imported through Vite so they are hashed and
cached once.

| File | Size (est.) | Used for | How it is applied |
| --- | --- | --- | --- |
| `butter-paper.webp` | 512 px tile, ~15 KB | the butter paper layer | `background-image`, repeat |
| `tear-edge.svg` | 1600 × 24, ~3 KB | the torn edge and its fibrous band | `mask-image` on the paper layer, `repeat-x` |
| `brush-1.svg` to `brush-3.svg` | ~2 KB each | the dry brush stroke | `mask-image` on a `::before`, colour from a token; the variant is picked from the field id so a field always gets the same stroke |
| `pencil-line.svg`, `pencil-dots.svg` | ~1.5 KB each | ruled lines and provenance underlines | `background` repeat-x; on inline text with `box-decoration-break: clone` so wrapped lines each get their own underline |
| `cloud.svg` | ~1 KB | the revision cloud | `border-image: url(cloud.svg) 10 round`, so it fits any box size with no script |
| `grain.webp` | 256 px tile, ~8 KB | the paper grain on the sheet | `background-image` at low opacity |

That comes to about 35 KB in total, loaded once and cached.

## The code

- Tokens in `cockpit.css`: `--sw-paper-butter`, `--sw-mark-check`,
  `--sw-mark-here`, `--sw-pencil`, `--sw-pencil-jev`. The dark theme reuses the
  same files and changes only the colours.
- Five classes: `.paper-torn`, `.mark-brush`, `.mark-cloud`, `.pencil-under`,
  `.pencil-rule`. Optionally one small React wrapper (`<Mark kind="check">`)
  that only sets the class.
- No new dependency and no Jev call. The marks are CSS, so they add no time
  to the 100 ms read and write budgets.

## Rules that keep it restrained

- **One meaning per mark.** The brush marks either "needs your decision" or
  "you are here", not both. Pencil marks provenance underlines and the
  title-block rule. Torn butter paper marks a working layer laid over the
  sheet, at most once per screen.
- **Never decoration on data in bulk.** No marks on whole tables, nothing
  animated.
- **Marks are decorative only.** They are drawn as CSS pseudo-elements, so a
  screen reader hears the words ("2 readings", "To check"), not the mark.
  Colour and texture never carry the meaning alone.
- **They turn off when they would get in the way.** Under
  `prefers-contrast: more` and in print, the marks become a plain underline
  or border.
