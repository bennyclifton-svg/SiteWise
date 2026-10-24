import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import "./interaction.css";

export type CommitKey = "enter" | "tab" | "shift-tab" | "blur";

interface Props {
  value: string;
  label: string;
  error: string;
  onCommit: (text: string, key: CommitKey) => void;
  onCancel: () => void;
  className?: string;
  busy?: boolean;
  maxLength?: number;
  options?: readonly { id: string; label: string }[];
}

// Program's editor shared with the register: one commit per gesture, with
// typing isolated from row shortcuts. Callers own validation and persistence.
export function InlineCellEditor({ value, label, error, onCommit, onCancel, className = "", busy = false, maxLength, options }: Props) {
  const [text, setText] = useState(value);
  const input = useRef<HTMLInputElement>(null);
  const select = useRef<HTMLSelectElement>(null);
  const done = useRef(false);
  useEffect(() => {
    const control = input.current ?? select.current;
    control?.focus({ preventScroll: true });
    input.current?.select();
    // Native vocabulary picker, where supported and permitted by the browser.
    try { select.current?.showPicker?.(); } catch { /* Focused select remains keyboard accessible. */ }
  }, []);
  useEffect(() => {
    if (error) {
      done.current = false;
      (input.current ?? select.current)?.focus({ preventScroll: true });
    }
  }, [error]);
  const finish = (key: CommitKey, next = text) => {
    if (done.current || busy) return;
    done.current = true;
    onCommit(next, key);
  };
  const keys = (e: KeyboardEvent) => {
    e.stopPropagation();
    if (busy) return;
    if (e.key === "Enter") { e.preventDefault(); finish("enter"); }
    else if (e.key === "Escape") { e.preventDefault(); done.current = true; onCancel(); }
    else if (e.key === "Tab") { e.preventDefault(); finish(e.shiftKey ? "shift-tab" : "tab"); }
  };
  const blur = () => {
    if (done.current || busy) return;
    if (text === value) { done.current = true; onCancel(); }
    else finish("blur");
  };
  const common = {
    className: `inline-cell-editor ${className}`, "aria-label": label, "aria-invalid": error ? true as const : undefined,
    disabled: busy, value: text, onKeyDown: keys, onBlur: blur,
    onClick: (e: React.MouseEvent) => e.stopPropagation(),
    onDoubleClick: (e: React.MouseEvent) => e.stopPropagation(),
  };
  return options ? <select {...common} ref={select} onChange={e => {
    done.current = false;
    setText(e.target.value);
    finish("enter", e.target.value);
  }}>
    <option value="">None</option>
    {text && !options.some(option => option.id === text) && <option value={text}>{text}</option>}
    {options.map(option => <option key={option.id} value={option.id}>{option.label}</option>)}
  </select> : <input {...common} ref={input} maxLength={maxLength} onChange={e => {
    done.current = false;
    setText(e.target.value);
  }} />;
}
