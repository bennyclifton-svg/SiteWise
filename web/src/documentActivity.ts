import type { RowModel } from "./DocumentRow";
import { ocrStage } from "./OCRStatus";

export function documentActivity(row: RowModel): { state: "active" | "queued"; label: string } | null {
  if (row.uploadError || row.interrupted) return null;
  const doc = row.doc;
  if (!doc) {
    if (row.uploadQueued) return { state: "queued", label: "Queued for upload" };
    const pct = Math.round((row.progress ?? 0) * 100);
    return { state: "active", label: pct < 100 ? `Uploading ${pct}%` : "Finishing upload…" };
  }
  const stage = ocrStage(doc);
  if (stage) return { state: stage.reason === "ocr_queued" ? "queued" : "active", label: stage.title };
  if (doc.visual_status === "reading") return { state: "active", label: "Reading document images…" };
  // Pending initial filing does not expose a worker-start event.
  if (doc.status === "pending") return { state: "queued", label: "Waiting for filing…" };
  if (doc.visual_status === "queued") return { state: "queued", label: "Waiting for image reading…" };
  return null;
}
