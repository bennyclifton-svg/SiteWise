// The two tender tables as the server sends them (contract C5 of
// docs/plans/2026-11-14-tender-comparison-plan.md) and the named edits the
// screen sends back (contract C6). Money is an exact decimal string or
// null; the browser never adds.
import { request } from "../api";
import type { Money, PriceTable } from "../submissionApi";

export interface TenderFlag { code: string; text: string }
export interface TenderLine {
  id: string; ordinal: number; page: number; label: string; printed: string; amount: Money;
  gst: "ex" | "inc" | "unknown"; state: "priced" | "included" | "excluded" | "not_priced";
  kind: "base" | "ps" | "pc" | "option" | "saving" | "rate" | "subtotal" | "total" | "gst";
  placed: "sum" | "inside" | "outside"; heading: string; row: string; note: string; counted: boolean;
  flags: TenderFlag[]; decided_by: "model" | "rule" | "user"; split_of: string;
}
export interface TenderCell { state: "" | "priced" | "included" | "excluded" | "not_priced"; amount: Money; lines: number; mark: "" | "high" | "low" | "gap"; to_check: boolean; note?: string }
export interface TenderColumn {
  submission_id: string; state: "pending" | "reading" | "read" | "failed"; error: string; basis: string;
  tender_sum: Money; tender_sum_how: string; tender_sum_by: "" | "rule" | "user"; remainder_why: string;
  like_for_like: Money; lowest: boolean; checked: boolean; checked_at: string | null;
  flags: TenderFlag[]; notes: string[]; exclusions: { p: number; text: string }[];
  cells: Record<string, TenderCell>; lines: TenderLine[];
}
export interface TenderRow { key: string; label: string; system: boolean }
export interface TenderAmount { amount: Money; state: "suggested" | "applied"; source: "firm" | "others_highest" | "others_average" | "typed" | "none"; line_id: string; page: number }
export interface TenderRange { low: Money; high: Money; average: Money; count: number }
export interface TenderAdjustment {
  id: string; label: string; note: string; origin: "system" | "user"; reason: string; row: string;
  amounts: Record<string, TenderAmount>; others: Record<string, TenderRange>;
}
export interface TenderTable { on: boolean; version: number; rows: TenderRow[]; columns: TenderColumn[]; adjustments: TenderAdjustment[]; suggested: number }

/** The key of the "Not broken down" row. A line with no row sits there. */
export const SYSTEM_ROW = "tender:not_broken_down";

export type TenderOp =
  | { op: "check_column"; submission: string; checked: boolean }
  | { op: "move_line"; line: string; row: string }
  | { op: "split_line"; line: string; parts: { row: string; amount: string }[]; rest_row: string }
  | { op: "unsplit_line"; line: string }
  | { op: "set_line"; line: string; label?: string; amount?: string; gst?: string; kind?: string; placed?: string; state?: string }
  | { op: "add_line"; submission: string; label: string; amount: string; gst: string; row: string; page?: number }
  | { op: "count_line"; line: string; counted: boolean }
  | { op: "delete_line"; line: string }
  | { op: "set_tender_sum"; submission: string; amount: string; gst: string }
  | { op: "add_adjustment"; label: string; note?: string; row?: string }
  | { op: "set_adjustment"; adjustment: string; label?: string; note?: string }
  | { op: "remove_adjustment"; adjustment: string }
  | { op: "set_adjustment_amount"; adjustment: string; submission: string; amount: string | null }
  | { op: "apply_adjustment"; adjustment: string; submission: string; applied: boolean; source?: string; amount?: string }
  | { op: "take_out_line"; line: string }
  | { op: "apply_suggested"; submission: string };

/** The table an edit answers with, and the change it made (for Undo). */
export type EditedTable = PriceTable & { change?: string };

export const tenderApi = {
  ops: (project: string, pkg: string, version: number, ops: TenderOp[]) =>
    request<EditedTable>("POST", `/projects/${project}/procurement/packages/${pkg}/tender/ops`, { version, ops }),
  readAgain: (project: string, sub: string) =>
    request<PriceTable>("POST", `/projects/${project}/submissions/${sub}/tender-read`),
  undo: (project: string, change: string) =>
    request<{ undone: boolean; areas: string[] }>("POST", `/projects/${project}/changes/${change}/undo`),
};
