// A modal confirmation for an action that cannot be undone. Cancel has the
// initial focus so Enter never confirms by accident; Escape cancels.

import { useEffect, useRef, type ReactNode } from "react";

interface Props {
  title: string;
  message: string;
  confirmLabel: string;
  busy?: boolean;
	busyLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
  /** Extra fields between the message and the buttons (a reason, say). */
  children?: ReactNode;
  confirmDisabled?: boolean;
  /** "plain" for a confirm that is not destructive. */
  tone?: "danger" | "plain";
}

export function ConfirmDialog({ title, message, confirmLabel, busy, busyLabel = "Deleting…", onConfirm, onCancel, children, confirmDisabled, tone = "danger" }: Props) {
  const ref = useRef<HTMLDialogElement>(null);
  const cancel = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    const d = ref.current;
    if (d && !d.open) d.showModal();
    cancel.current?.focus();
    return () => d?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      className="confirm"
      aria-labelledby="confirm-title"
      aria-describedby="confirm-message"
      onCancel={(e) => {
        e.preventDefault();
        if (!busy) onCancel();
      }}
    >
      <h2 id="confirm-title">{title}</h2>
      <p id="confirm-message">{message}</p>
      {children}
      <div className="confirm-actions">
        <button ref={cancel} type="button" className="btn btn-small" onClick={onCancel} disabled={busy}>
          Cancel
        </button>
        <button type="button" className={`btn btn-small${tone === "danger" ? " btn-danger" : ""}`} onClick={onConfirm} disabled={busy || confirmDisabled}>
          {busy ? busyLabel : confirmLabel}
        </button>
      </div>
    </dialog>
  );
}
