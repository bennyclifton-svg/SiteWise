// Correspondence (owner, 25 November 2026): the register and the project's
// mail, in the middle panel, from the left nav. The right-hand panel is the
// documents only. Register is the default tab; Messages is the mail that
// came in and went out, each message opening under its row. A register row
// drills in to its sheet, with a breadcrumb back to here.

import type { ReactNode } from "react";
import { RegisterList } from "./Delivery";
import { PanelHead } from "./PanelHead";
import type { WorkbenchView } from "./LeftNav";
import { MailRepository, type MailState } from "./MailIn";
import type { RegisterData } from "./registerData";

export type CorrespondenceTab = "register" | "messages";

interface Props {
  tab: CorrespondenceTab;
  onTab: (tab: CorrespondenceTab) => void;
  /** Whether the view is on screen: chat moves its tables only then. */
  shown: boolean;
  data: RegisterData;
  projectId: string;
  /** Moves when a mail event arrives. */
  mailTick: number;
  mail: MailState;
  onMail: (state: MailState) => void;
  raise: { documentId: string; title: string; n: number } | null;
  onOpenRow: (id: string) => void;
  onOpenDocument: (id: string) => boolean;
  onShowView: (view: WorkbenchView) => void;
  documents: { id: string; title: string; kind: string }[];
}

export function Correspondence({ tab, onTab, shown, data, projectId, mailTick, mail, onMail, raise, onOpenRow, onOpenDocument, onShowView, documents }: Props) {
  // The heading every view wears (PanelHead): Register | Messages as large
  // words, the open one underlined; each tab brings its own three dots.
  const head = (more?: ReactNode) => <PanelHead<CorrespondenceTab>
    title="Correspondence"
    tabs={{ label: "Correspondence", idPrefix: "correspondence", value: tab, onChange: onTab, items: [
      { id: "register", label: "Register" },
      { id: "messages", label: <>Messages{mail.held > 0 ? <span className="panel-switch-held"> · {mail.held} held</span> : null}</> },
    ] }}
    more={more}
  />;
  return <div className="correspondence">
    <div role="tabpanel" id="correspondence-panel-register" aria-labelledby="correspondence-tab-register" hidden={tab !== "register"}>
      <RegisterList data={data} shown={shown && tab === "register"} head={head} raise={raise} onOpen={onOpenRow} onShowView={onShowView} documents={documents} />
    </div>
    {/* Messages stays mounted once read, so an open message survives the switch. */}
    <section className="correspondence-messages" role="tabpanel" id="correspondence-panel-messages" aria-labelledby="correspondence-tab-messages" hidden={tab !== "messages"}>
      {head()}
      <MailRepository projectId={projectId} tick={mailTick} documents={null} side="messages" onMail={onMail} onOpenDocument={onOpenDocument} />
    </section>
  </div>;
}
