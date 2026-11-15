/** The one disclosure chevron of every table — Project Profile, Cost Plan,
 * Program and Procurement (owner, 15 Nov 2026): right when closed, down when
 * open. Its size and colour are set once, in table-heads.css. */
export function Chevron({ open }: { open: boolean }) {
  return <svg className="disclosure-chevron" width="20" height="20" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true" focusable="false">
    <path d={open ? "m4 6 4 4 4-4" : "m6 4 4 4-4 4"} />
  </svg>;
}
