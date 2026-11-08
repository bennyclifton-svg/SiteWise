import { useLayoutEffect, useRef, type ReactNode } from "react";

/**
 * Wording edited where it is read (owner direction, 7 Nov 2026). The text
 * and its edit box share type, padding and width, and the box grows with its
 * text, so nothing moves when editing starts. A click puts the cursor where
 * it landed; Save or Cancel sit underneath, and clicking away saves.
 */
export function InlineText({ text, label, editing, value, disabled, busy, saveDisabled, compact, maxLength, onStart, onChange, onSave, onCancel, actions }: {
  text: string;
  label: string;
  editing: boolean;
  value: string;
  disabled?: boolean;
  busy?: boolean;
  saveDisabled?: boolean;
  onStart: () => void;
  onChange: (text: string) => void;
  onSave: () => void;
  onCancel: () => void;
  /** Shown under the text when not editing, such as an Edit button. */
  actions?: ReactNode;
  /**
   * A table cell (the RFP scope lines): one line of text with no buttons.
   * Enter or clicking away saves; Escape cancels; a Ctrl or Shift click
   * selects the row instead of editing it.
   */
  compact?: boolean;
  maxLength?: number;
}) {
  const box = useRef<HTMLTextAreaElement>(null);
  const caret = useRef<number | null>(null);

  useLayoutEffect(() => {
    const el = box.current;
    if (!el) return;
    grow(el);
    if (caret.current !== null || document.activeElement !== el) {
      const at = caret.current ?? el.value.length;
      el.focus({ preventScroll: true });
      el.setSelectionRange(at, at);
      caret.current = null;
    }
  }, [editing]);

  function start(e: React.MouseEvent<HTMLParagraphElement>) {
    if (disabled) return;
    if (compact && (e.ctrlKey || e.metaKey || e.shiftKey)) return;
    // Selecting text to copy is not a request to edit it.
    if (String(window.getSelection() ?? "")) return;
    caret.current = offsetAt(e.currentTarget, e.clientX, e.clientY);
    onStart();
  }

  if (!editing) return <>
    <p className={`report-wording inline-text${compact ? " inline-text-cell" : ""}`} data-editable={!disabled || undefined} onClick={start}>{text}</p>
    {actions}
  </>;

  const leave = () => {
    // A window switch also blurs: only a click elsewhere on the page saves.
    window.setTimeout(() => {
      if (!document.hasFocus() || busy) return;
      if (value === text) onCancel(); else if (!saveDisabled) onSave();
    }, 0);
  };
  return <form className="inline-text-form" onSubmit={e => { e.preventDefault(); onSave(); }}
    onBlur={e => { if (!e.currentTarget.contains(e.relatedTarget as Node | null)) leave(); }}
    onKeyDown={e => { if (e.key === "Escape") { e.stopPropagation(); onCancel(); } }}>
    <textarea ref={box} className={`report-wording inline-text${compact ? " inline-text-cell" : ""}`} aria-label={label} maxLength={maxLength ?? 20000} value={value} disabled={busy}
      onChange={e => { onChange(compact ? e.target.value.replace(/\n/g, " ") : e.target.value); grow(e.target); }}
      onKeyDown={e => {
        if (!compact || e.key !== "Enter") return;
        e.preventDefault();
        if (value === text) onCancel(); else if (!saveDisabled) onSave();
      }} />
    {!compact && <div className="report-actions">
      <button className="btn btn-small" disabled={busy || saveDisabled}>Save wording</button>
      <button type="button" className="btn btn-small" disabled={busy} onClick={onCancel}>Cancel edit</button>
    </div>}
  </form>;
}

/** The box is as tall as its text: no inner scroll bar. */
function grow(el: HTMLTextAreaElement) {
  el.style.height = "auto";
  el.style.height = `${el.scrollHeight + el.offsetHeight - el.clientHeight}px`;
}

/** The character offset under a click, so the cursor lands there. */
function offsetAt(root: HTMLElement, x: number, y: number): number | null {
  const doc = document as Document & { caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null; caretRangeFromPoint?: (x: number, y: number) => Range | null };
  let node: Node | null = null, offset = 0;
  const pos = doc.caretPositionFromPoint?.(x, y);
  if (pos) { node = pos.offsetNode; offset = pos.offset; }
  else { const range = doc.caretRangeFromPoint?.(x, y); if (range) { node = range.startContainer; offset = range.startOffset; } }
  if (!node || !root.contains(node)) return null;
  // Count the text before the clicked node, in case the text is split.
  let total = 0;
  const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  for (let n = walk.nextNode(); n; n = walk.nextNode()) {
    if (n === node) return total + offset;
    total += n.textContent?.length ?? 0;
  }
  return null;
}
