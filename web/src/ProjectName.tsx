import { useRef, useState } from "react";

export function ProjectName({ name, onSave }: { name: string; onSave: (name: string) => Promise<void> }) {
  const [draft, setDraft] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const cancel = useRef(false);
  const pending = useRef(false);

  async function save() {
    if (cancel.current) { cancel.current = false; return; }
    if (pending.current || draft === null) return;
    const next = draft.trim();
    if (!next) { setError("Enter a project name."); return; }
    if (next === name) { setDraft(null); setError(""); return; }
    pending.current = true;
    setSaving(true);
    setError("");
    try {
      await onSave(next);
      setDraft(null);
    } catch {
      setError("Name wasn’t saved. Press Enter or click out to retry.");
    } finally {
      pending.current = false;
      setSaving(false);
    }
  }

  return <>
    <h1 className="cockpit-title" aria-label={name}>
      {/* Keep the same native input mounted so the browser places the caret at the click. */}
      <input className="project-name-editor" aria-label="Project name"
        aria-invalid={!!error} aria-describedby={error ? "project-name-error" : undefined}
        title="Edit project name" value={draft ?? name} readOnly={saving}
        onChange={e => { setDraft(e.target.value); setError(""); }}
        onBlur={() => void save()}
        onKeyDown={e => {
          if (e.nativeEvent.isComposing) return;
          if (e.key === "Enter") { e.preventDefault(); e.currentTarget.blur(); }
          if (e.key === "Escape") {
            e.preventDefault(); cancel.current = true; setDraft(null); setError(""); e.currentTarget.blur();
          }
        }} />
    </h1>
    {error && <p id="project-name-error" className="project-name-error" role="alert">{error}</p>}
    <span className="sr-only" role="status">{saving ? "Saving project name…" : ""}</span>
  </>;
}
