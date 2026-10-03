import {
  Callout,
  Code,
  computeDAGLayout,
  Divider,
  Grid,
  H1,
  H2,
  H3,
  Pill,
  Row,
  Stack,
  Stat,
  Swatch,
  Text,
  UsageBar,
  useCanvasState,
  useHostTheme,
  type DAGLayoutEdge,
} from "cursor/canvas";

type View =
  | "machine"
  | "file"
  | "later"
  | "profile"
  | "roles"
  | "building"
  | "commands"
  | "records";

const TABS: { id: View; label: string }[] = [
  { id: "machine", label: "The machine" },
  { id: "file", label: "Drop a file" },
  { id: "later", label: "After it lands" },
  { id: "profile", label: "The profile" },
  { id: "roles", label: "Code or Jev" },
  { id: "building", label: "The building" },
  { id: "commands", label: "Commands" },
  { id: "records", label: "Records" },
];

export default function SiteWisePicture() {
  const [view, setView] = useCanvasState<View>("view", "machine");

  return (
    <Stack gap={18}>
      <Stack gap={4}>
        <H1>SiteWise, as a picture</H1>
        <Text tone="secondary">
          One program, one database, one judgement service. Pick a picture.
        </Text>
      </Stack>
      <Row gap={6} wrap>
        {TABS.map((tab) => (
          <span key={tab.id}>
            <Pill active={view === tab.id} onClick={() => setView(tab.id)}>
              {tab.label}
            </Pill>
          </span>
        ))}
      </Row>
      {view === "machine" && <Machine />}
      {view === "file" && <DropFile />}
      {view === "later" && <AfterLanding />}
      {view === "profile" && <ProfileFlow />}
      {view === "roles" && <Roles />}
      {view === "building" && <Building />}
      {view === "commands" && <Commands />}
      {view === "records" && <Records />}
      <Text size="small" tone="tertiary">
        Source: the SiteWise program and the 29 September 2026 foundation design.
      </Text>
    </Stack>
  );
}

function Machine() {
  const theme = useHostTheme();
  return (
    <Stack gap={16}>
      <Row gap={16} align="end" wrap>
        <Stat value="1" label="Program you run" />
        <Stat value="1" label="Jev call while you wait" />
        <Stat value="1 s" label="Typical filing budget" />
        <Stat value="2 s" label="Slow filing budget" />
      </Row>
      <Stack gap={0} style={{ alignItems: "center" }}>
        <Box
          title="You, in the browser"
          sub="React screens, built with Vite and packed inside the program"
          accent
          width={560}
        >
          <Row gap={8} wrap>
            <Mini label="Sign in" detail="Invite link" />
            <Mini label="Projects" detail="List and create" />
            <Mini label="Register" detail="Drop files, chips" />
            <Mini label="Profile" detail="Facts about the building" />
          </Row>
        </Box>
        <Down label="clicks down · live updates up" />
        <Box
          title="One Go program — sitewise serve"
          sub="The pages, the API, filing, and the background reading all live here"
          width={560}
        >
          <Row gap={8} wrap>
            <Mini label="API" detail="/api" />
            <Mini label="Filing" detail="While you wait" />
            <Mini label="Workers" detail="After the file lands" />
            <Mini label="Live wire" detail="Event stream" />
          </Row>
        </Box>
        <Fan />
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
            gap: 12,
            width: 560,
          }}
        >
          <Box title="PostgreSQL 17" sub="Records and full-text search. Same machine." />
          <Box title="Files on disk" sub="PDFs and spreadsheets, named by fingerprint." />
          <Box title="Jev" sub="TypeSafe System One. The only judgement." jev />
        </div>
        <Down label="loaded when the program starts" />
        <Box
          title="knowledge/"
          sub="The building, as data: systems, determinants, rules, interfaces, failure modes"
          width={560}
        />
      </Stack>
      <Grid columns={2} gap={16}>
        <Stack gap={6}>
          <H3>On this PC</H3>
          <Text size="small" tone="secondary">
            <Code>tools/dev.ps1</Code> starts PostgreSQL on port 5433 and the program at{" "}
            <Code>127.0.0.1:8080</Code>. <Code>/dev/login</Code> signs you in on this machine only.
          </Text>
        </Stack>
        <Stack gap={6}>
          <H3>On the server</H3>
          <Text size="small" tone="secondary">
            Caddy is the front door. systemd keeps the one program running. PostgreSQL sits beside
            it. Files stay on the machine’s disk.
          </Text>
        </Stack>
      </Grid>
      <div style={{ height: 1, background: theme.stroke.tertiary }} />
      <Text size="small" tone="tertiary">
        Every record belongs to one organisation. A session cookie is the key. Invite links are
        the way in.
      </Text>
    </Stack>
  );
}

function DropFile() {
  return (
    <Stack gap={16}>
      <H2>From the drop to the chip</H2>
      <Text tone="secondary">
        You wait on this path. It is budgeted at a typical time under 1 second and a slow time
        under 2 seconds. The build fails when that slips.
      </Text>
      <UsageBar
        total={1000}
        topLeftLabel="Where a typical second goes"
        topRightLabel="441 ms used of a 1,000 ms budget"
        segments={[
          { id: "read", value: 80, color: "blue" },
          { id: "find", value: 5, color: "cyan" },
          { id: "rules", value: 1, color: "green" },
          { id: "jev", value: 350, color: "purple" },
          { id: "save", value: 5, color: "gray" },
        ]}
      />
      <Row gap={14} wrap>
        <Legend color="blue" label="Read title pages · 80 ms" />
        <Legend color="cyan" label="Find candidates · 5 ms" />
        <Legend color="green" label="Rules · 1 ms" />
        <Legend color="purple" label="One Jev call · 350 ms" />
        <Legend color="gray" label="Save and notify · 5 ms" />
      </Row>
      <Flow
        marker="file"
        nodes={[
          { id: "drop", title: "You drop a file", sub: "PDF, Word, or Excel", tone: "path" },
          { id: "save", title: "Bytes are saved", sub: "Named by fingerprint", tone: "path" },
          { id: "same", title: "Already in this project", sub: "Same fingerprint. Stop.", tone: "exit" },
          { id: "read", title: "Read the title pages", sub: "Typical 80 ms, slow 250 ms", tone: "path" },
          { id: "quiet", title: "Kept, not filed", sub: "No text, unreadable, or too large", tone: "exit" },
          { id: "list", title: "List candidates", sub: "Number, title, revision, date", tone: "path" },
          { id: "rules", title: "Rules fill what they can", sub: "About 1 ms. Your edits stay.", tone: "path" },
          { id: "jev", title: "One Jev call", sub: "Only the fields still open", tone: "jev" },
          { id: "write", title: "Write the register", sub: "Decisions, then a live event", tone: "path" },
          { id: "chips", title: "Chips appear", sub: "Green, amber, blank, or grey", tone: "path" },
          { id: "sheets", title: "Split a drawing set", sub: "Each sheet is filed on its own", tone: "exit" },
          { id: "queue", title: "Queue the deep read", sub: "Does not hold this screen", tone: "exit" },
        ]}
        edges={[
          { from: "drop", to: "save" },
          { from: "save", to: "same" },
          { from: "save", to: "read" },
          { from: "read", to: "quiet" },
          { from: "read", to: "list" },
          { from: "list", to: "rules" },
          { from: "rules", to: "jev" },
          { from: "jev", to: "write" },
          { from: "write", to: "chips" },
          { from: "write", to: "sheets" },
          { from: "write", to: "queue" },
        ]}
      />
      <H3>What the chip colours mean</H3>
      <Grid columns={4} gap={10}>
        <Chip name="Green" color="green" detail="Above that question’s bar. Applied." />
        <Chip name="Amber" color="yellow" detail="Applied, and flagged for a look." />
        <Chip name="Blank" detail="Below the band. Left empty." />
        <Chip name="Grey" color="gray" detail="Jev did not answer in time. Rules kept. Asked again later." />
      </Grid>
      <Callout tone="info">
        Supersedes — “this replaces that document” — has the strictest bar. A closed tab does not
        cancel a filing whose bytes are already stored.
      </Callout>
    </Stack>
  );
}

function AfterLanding() {
  return (
    <Stack gap={16}>
      <H2>You can keep working. Reading continues.</H2>
      <Grid columns="minmax(0, 220px) minmax(0, 1fr)" gap={20}>
        <Stack gap={8}>
          <Box title="Reserved for you" sub="Filing keeps its own Jev slots" accent />
          <Text size="small" tone="secondary">
            A person dropping files stays ahead of the deep read. Background calls cannot take
            those slots.
          </Text>
        </Stack>
        <Flow
          marker="later"
          nodes={[
            { id: "done", title: "Filing is committed", sub: "The register already shows it", tone: "path" },
            { id: "text", title: "Full text", sub: "Every passage, then search", tone: "path" },
            { id: "label", title: "Label the systems", sub: "One Jev call per passage", tone: "jev" },
            { id: "ask", title: "Ask the knowledge", sub: "One Jev call for those systems", tone: "jev" },
            { id: "rows", title: "Rebuild the profile", sub: "Precomputed rows for the page", tone: "path" },
            { id: "live", title: "The screen updates", sub: "Same live wire as filing", tone: "path" },
          ]}
          edges={[
            { from: "done", to: "text" },
            { from: "text", to: "label" },
            { from: "label", to: "ask" },
            { from: "ask", to: "rows" },
            { from: "rows", to: "live" },
          ]}
        />
      </Grid>
      <Grid columns={3} gap={12}>
        <Box title="Two queues" sub="Text extraction runs on its own. Labelling and evidence run on another. Each org is worked one job at a time." />
        <Box title="If the program stops" sub="Jobs resume from the jobs table. A filing that had not finished starts again." />
        <Box title="If Jev is down" sub="Filing still lands. Grey chips stay. A later job asks again, with a pause between tries." />
      </Grid>
    </Stack>
  );
}

function ProfileFlow() {
  return (
    <Stack gap={16}>
      <H2>How a project profile is built</H2>
      <Text tone="secondary">
        The profile column reads one prepared view. It does not ask Jev when you open the page.
      </Text>
      <Flow
        marker="profile"
        direction="horizontal"
        nodes={[
          { id: "docs", title: "Filed documents", sub: "Passages of their text", tone: "path" },
          { id: "facts", title: "Facts", sub: "Jev readings, or a rule", tone: "jev" },
          { id: "yours", title: "Your values", sub: "Final. Background jobs skip these.", tone: "path" },
          { id: "rows", title: "Profile rows", sub: "Code reconciles facts and edits", tone: "path" },
          { id: "page", title: "The profile column", sub: "One query paints the page", tone: "path" },
        ]}
        edges={[
          { from: "docs", to: "facts" },
          { from: "facts", to: "rows" },
          { from: "yours", to: "rows" },
          { from: "rows", to: "page" },
        ]}
      />
      <Grid columns={3} gap={12}>
        <Box title="Parts of the building" sub="Whole project, building, part, storey, compartment, tenancy, outbuilding. A fact keeps its scope." />
        <Box title="A blank row" sub="Thresholds the owner has not approved store the reading and apply nothing." />
        <Box title="You can ask again" sub="“Read the profile” queues the work. It does not run a chain of calls in the click." />
      </Grid>
    </Stack>
  );
}

function Roles() {
  const theme = useHostTheme();
  return (
    <Stack gap={14}>
      <H2>Who does which job</H2>
      <Text tone="secondary">
        The program owns the steps. Jev is called where a choice needs judgement. One call carries
        every open question about that state.
      </Text>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: 0,
          border: `1px solid ${theme.stroke.secondary}`,
        }}
      >
        <RoleHead title="The program" detail="Parsing, maths, lookups, control" />
        <RoleHead title="Jev" detail="Picks. Does not calculate." jev />
        <RoleBody
          items={[
            "Read PDF, Word, and Excel text",
            "Over-find numbers, titles, revisions, dates",
            "Order revisions and compare dates",
            "NCC table lookups: type of construction, FRLs",
            "Which interfaces the present systems imply",
            "Spot conflicts between documents",
            "Fingerprint a file and keep each org’s rows apart",
            "Decide which questions are still open",
          ]}
        />
        <RoleBody
          jev
          items={[
            "Document kind, among 16",
            "Discipline, among 57, asked even when a rule might settle it",
            "Number, revision, and title — one of the candidates, or none",
            "Which earlier document this one replaces",
            "Which building systems a passage is about",
            "Whether a passage addresses a rule or a failure mode",
            "A determinant, chosen from candidates the program already found",
          ]}
        />
      </div>
      <Callout tone="neutral">
        A document number or an exact title goes straight to a lookup. Jev is reserved for a real
        ambiguity. Jev does not count, do arithmetic, compare dates, or write the prose.
      </Callout>
    </Stack>
  );
}

function Building() {
  const layers = [
    {
      n: "1",
      title: "Systems",
      detail: "What physically exists: ground, structure, envelope, interiors, fire, mechanical, hydraulic, electrical, vertical transport, access.",
    },
    {
      n: "2",
      title: "Determinants",
      detail: "The few facts that choose the rules: NCC class, rise in storeys, height, sprinklers, state, climate, wind, flood, heritage, new or existing.",
    },
    {
      n: "3",
      title: "Rules",
      detail: "NCC provisions, Australian Standards, state instruments. Many are table lookups. A number stays unverified until it is checked against the instrument itself.",
    },
    {
      n: "4",
      title: "Interfaces",
      detail: "Where systems meet: penetrates, sequences, loads, supplies, controls, depends on, shares space. Most failures sit here.",
    },
    {
      n: "5",
      title: "Failure modes",
      detail: "Known pitfalls, attached to a rule or an interface.",
    },
  ];
  return (
    <Stack gap={14}>
      <H2>The building is the model</H2>
      <Text tone="secondary">
        Knowledge is these five layers, stored as YAML and loaded at startup. Consultant
        disciplines are a way of looking across them.
      </Text>
      <div style={{ display: "grid", gridTemplateColumns: "28px 1fr", gap: 10, alignItems: "stretch" }}>
        <ViewBracket />
        <Stack gap={8}>
          {layers.map((layer) => (
            <div key={layer.n}>
              <Layer n={layer.n} title={layer.title} detail={layer.detail} />
            </div>
          ))}
        </Stack>
      </div>
      <Text size="small" tone="tertiary">
        The bracket is the consultant view: architecture, fire, hydraulic, and the rest read the
        same building. They are not five separate models.
      </Text>
    </Stack>
  );
}

function Commands() {
  return (
    <Stack gap={16}>
      <H2>What you run</H2>
      <Flow
        marker="commands"
        direction="horizontal"
        nodes={[
          { id: "look", title: "Look at it", sub: "tools/dev.ps1", tone: "path" },
          { id: "change", title: "A change lands", sub: "One lane: core, edge, or security", tone: "path" },
          { id: "prove", title: "Prove it", sub: "tools/check.ps1", tone: "path" },
          { id: "ship", title: "Run it", sub: "sitewise serve", tone: "path" },
        ]}
        edges={[
          { from: "look", to: "change" },
          { from: "change", to: "prove" },
          { from: "prove", to: "ship" },
        ]}
      />
      <Grid columns={3} gap={16}>
        <Stack gap={8}>
          <H3>Start</H3>
          <Cmd name="tools/dev.ps1" does="Postgres, the program, and a local sign-in." />
          <Cmd name="sitewise serve" does="The program: pages, API, filing, workers." />
          <Cmd name="sitewise bootstrap" does="The first organisation and an invite." />
        </Stack>
        <Stack gap={8}>
          <H3>Prove</H3>
          <Cmd name="tools/check.ps1" does="Knowledge, tests, accuracy replay, speed." />
          <Cmd name="check_knowledge.py" does="The building files still fit the schema." />
          <Cmd name="intake-eval" does="Filing answers, replayed from a recording." />
          <Cmd name="profile-eval" does="Profile answers, replayed the same way." />
          <Cmd name="intake-bench" does="Filing time against the 1 s / 2 s gate." />
          <Cmd name="sitewise gate" does="The same speed gate, as a program command." />
        </Stack>
        <Stack gap={8}>
          <H3>Look after</H3>
          <Cmd name="sitewise restore-check" does="A backup can actually be restored." />
          <Cmd name="/healthz" does="Database, Jev, queue depth, oldest job." />
          <Cmd name="corpus-eval" does="A specialist pass over a document corpus." />
          <Cmd name="ocr-trial" does="A trial reader for scanned title blocks." />
        </Stack>
      </Grid>
    </Stack>
  );
}

function Records() {
  const theme = useHostTheme();
  return (
    <Stack gap={14}>
      <H2>Where a fact lives</H2>
      <Text tone="secondary">
        PostgreSQL holds the records. The file bytes live on disk, under the fingerprint. Search
        uses the database’s own full-text index.
      </Text>
      <div style={{ display: "grid", gridTemplateColumns: "1.4fr 0.8fr", gap: 16 }}>
        <div
          style={{
            border: `1px solid ${theme.accent.primary}`,
            padding: 12,
          }}
        >
          <Text weight="semibold">Your organisation</Text>
          <Text size="small" tone="tertiary">
            Every row carries this org. The queries use it.
          </Text>
          <div style={{ marginTop: 10 }}>
            <Tree />
          </div>
        </div>
        <div
          style={{
            border: `1px solid ${theme.stroke.secondary}`,
            padding: 12,
            opacity: 0.72,
          }}
        >
          <Text weight="semibold" tone="secondary">
            Another organisation
          </Text>
          <Text size="small" tone="tertiary">
            Its own projects, files, decisions, and profile. No line joins the two.
          </Text>
          <Stack gap={6} style={{ marginTop: 12 }}>
            <Ghost label="Projects" />
            <Ghost label="Documents" />
            <Ghost label="Profile" />
          </Stack>
        </div>
      </div>
      <Divider />
      <Grid columns={4} gap={12}>
        <Cluster
          title="People"
          rows={["orgs", "users", "memberships", "invites", "sessions"]}
        />
        <Cluster
          title="The register"
          rows={["projects", "files", "documents", "decisions", "supersessions", "drawing_sheets"]}
        />
        <Cluster
          title="The reading"
          rows={["passages", "passage_systems", "passage_evidence", "document_sources", "jobs", "events"]}
        />
        <Cluster
          title="The profile"
          rows={["project_parts", "profile_facts", "profile_user_values", "profile_rows"]}
        />
      </Grid>
    </Stack>
  );
}

function Tree() {
  const theme = useHostTheme();
  const line = `1px solid ${theme.stroke.secondary}`;
  return (
    <Stack gap={0}>
      <TreeRow label="Project" />
      <div style={{ marginLeft: 14, borderLeft: line, paddingLeft: 12 }}>
        <TreeRow label="File on disk" hint="fingerprint" />
        <TreeRow label="Document" hint="the register row" />
        <div style={{ marginLeft: 14, borderLeft: line, paddingLeft: 12 }}>
          <TreeRow label="Decisions" hint="one per field: value, colour, who decided" />
          <TreeRow label="Sheets" hint="when a drawing set is split" />
          <TreeRow label="Passages" hint="the text, searchable" />
          <div style={{ marginLeft: 14, borderLeft: line, paddingLeft: 12 }}>
            <TreeRow label="Facts" hint="a Jev or rule reading" />
          </div>
        </div>
        <TreeRow label="Profile rows" hint="what the page shows" />
        <TreeRow label="Your values" hint="sit beside the facts" />
      </div>
    </Stack>
  );
}

function TreeRow({ label, hint }: { label: string; hint?: string }) {
  const theme = useHostTheme();
  return (
    <div style={{ display: "flex", gap: 8, alignItems: "baseline", padding: "3px 0" }}>
      <span style={{ width: 8, height: 8, background: theme.fill.primary, display: "inline-block" }} />
      <span style={{ fontSize: 13, color: theme.text.primary, fontWeight: 600 }}>{label}</span>
      {hint && <span style={{ fontSize: 12, color: theme.text.tertiary }}>{hint}</span>}
    </div>
  );
}

function Cluster({ title, rows }: { title: string; rows: string[] }) {
  const theme = useHostTheme();
  return (
    <Stack gap={4}>
      <Text size="small" weight="semibold">
        {title}
      </Text>
      {rows.map((row) => (
        <div key={row} style={{ fontSize: 12, color: theme.text.secondary, fontFamily: "ui-monospace, monospace" }}>
          {row}
        </div>
      ))}
    </Stack>
  );
}

function Ghost({ label }: { label: string }) {
  const theme = useHostTheme();
  return (
    <div
      style={{
        border: `1px solid ${theme.stroke.tertiary}`,
        padding: "6px 8px",
        color: theme.text.tertiary,
        fontSize: 12,
      }}
    >
      {label}
    </div>
  );
}

function RoleHead({ title, detail, jev }: { title: string; detail: string; jev?: boolean }) {
  const theme = useHostTheme();
  return (
    <div
      style={{
        padding: "10px 12px",
        background: theme.fill.tertiary,
        borderBottom: `1px solid ${theme.stroke.secondary}`,
        borderLeft: jev ? `1px solid ${theme.stroke.secondary}` : undefined,
      }}
    >
      <Row gap={8} align="center">
        {jev && <Swatch color="purple" />}
        <Text weight="semibold">{title}</Text>
      </Row>
      <Text size="small" tone="tertiary">
        {detail}
      </Text>
    </div>
  );
}

function RoleBody({ items, jev }: { items: string[]; jev?: boolean }) {
  const theme = useHostTheme();
  return (
    <div
      style={{
        padding: "8px 12px 12px",
        borderLeft: jev ? `1px solid ${theme.stroke.secondary}` : undefined,
      }}
    >
      <Stack gap={6}>
        {items.map((item) => (
          <div key={item}>
            <Text size="small">{item}</Text>
          </div>
        ))}
      </Stack>
    </div>
  );
}

function Layer({ n, title, detail }: { n: string; title: string; detail: string }) {
  const theme = useHostTheme();
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "28px 140px 1fr",
        gap: 12,
        alignItems: "center",
        padding: "10px 12px",
        background: theme.bg.elevated,
        border: `1px solid ${theme.stroke.secondary}`,
      }}
    >
      <span style={{ fontSize: 13, color: theme.accent.primary, fontWeight: 600 }}>{n}</span>
      <span style={{ fontSize: 14, color: theme.text.primary, fontWeight: 600 }}>{title}</span>
      <span style={{ fontSize: 12, color: theme.text.secondary, lineHeight: "17px" }}>{detail}</span>
    </div>
  );
}

function ViewBracket() {
  const theme = useHostTheme();
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center" }}>
      <div style={{ width: 10, height: 10, borderTop: `1px solid ${theme.accent.primary}`, borderLeft: `1px solid ${theme.accent.primary}` }} />
      <div style={{ width: 1, flex: 1, background: theme.accent.primary }} />
      <div
        style={{
          writingMode: "vertical-rl",
          transform: "rotate(180deg)",
          fontSize: 11,
          color: theme.accent.primary,
          padding: "8px 0",
        }}
      >
        Disciplines look across
      </div>
      <div style={{ width: 1, flex: 1, background: theme.accent.primary }} />
      <div style={{ width: 10, height: 10, borderBottom: `1px solid ${theme.accent.primary}`, borderLeft: `1px solid ${theme.accent.primary}` }} />
    </div>
  );
}

function Cmd({ name, does }: { name: string; does: string }) {
  return (
    <Stack gap={2}>
      <Code>{name}</Code>
      <Text size="small" tone="secondary">
        {does}
      </Text>
    </Stack>
  );
}

function Chip({ name, detail, color }: { name: string; detail: string; color?: "green" | "yellow" | "gray" }) {
  return (
    <Stack gap={4}>
      <Row gap={6} align="center">
        {color ? <Swatch color={color} /> : <BlankSwatch />}
        <Text weight="semibold" size="small">
          {name}
        </Text>
      </Row>
      <Text size="small" tone="secondary">
        {detail}
      </Text>
    </Stack>
  );
}

function BlankSwatch() {
  const theme = useHostTheme();
  return (
    <span
      style={{
        width: 14,
        height: 14,
        border: `1px solid ${theme.stroke.primary}`,
        display: "inline-block",
      }}
    />
  );
}

function Legend({ color, label }: { color: "blue" | "cyan" | "green" | "purple" | "gray"; label: string }) {
  return (
    <Row gap={6} align="center">
      <Swatch color={color} />
      <Text size="small" tone="secondary">
        {label}
      </Text>
    </Row>
  );
}

function Box({
  title,
  sub,
  children,
  width,
  accent,
  jev,
}: {
  title: string;
  sub?: string;
  children?: ReturnType<typeof Row>;
  width?: number | string;
  accent?: boolean;
  jev?: boolean;
}) {
  const theme = useHostTheme();
  const top = accent ? theme.accent.primary : jev ? theme.category.purple : theme.stroke.primary;
  return (
    <div
      style={{
        width,
        boxSizing: "border-box",
        background: theme.bg.elevated,
        border: `1px solid ${theme.stroke.secondary}`,
        borderTop: `3px solid ${top}`,
        padding: "10px 12px",
      }}
    >
      <div style={{ fontSize: 14, fontWeight: 600, color: theme.text.primary }}>{title}</div>
      {sub && (
        <div style={{ fontSize: 12, color: theme.text.secondary, marginTop: 2, lineHeight: "16px" }}>{sub}</div>
      )}
      {children && <div style={{ marginTop: 8 }}>{children}</div>}
    </div>
  );
}

function Mini({ label, detail }: { label: string; detail: string }) {
  const theme = useHostTheme();
  return (
    <div
      style={{
        minWidth: 108,
        padding: "6px 8px",
        background: theme.fill.tertiary,
        border: `1px solid ${theme.stroke.tertiary}`,
      }}
    >
      <div style={{ fontSize: 12, fontWeight: 600, color: theme.text.primary }}>{label}</div>
      <div style={{ fontSize: 11, color: theme.text.tertiary }}>{detail}</div>
    </div>
  );
}

function Down({ label }: { label: string }) {
  const theme = useHostTheme();
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 2, padding: "4px 0" }}>
      <div style={{ width: 1, height: 10, background: theme.stroke.primary }} />
      <span style={{ fontSize: 11, color: theme.text.tertiary }}>{label}</span>
      <div style={{ width: 1, height: 8, background: theme.stroke.primary }} />
      <div
        style={{
          width: 0,
          height: 0,
          borderLeft: "4px solid transparent",
          borderRight: "4px solid transparent",
          borderTop: `5px solid ${theme.text.tertiary}`,
        }}
      />
    </div>
  );
}

function Fan() {
  const theme = useHostTheme();
  const w = 560;
  const h = 28;
  const mid = w / 2;
  const color = theme.stroke.primary;
  return (
    <svg width={w} height={h} aria-hidden="true">
      <path d={`M ${mid} 0 L ${mid} 12`} stroke={color} fill="none" />
      <path d={`M ${w * 0.16} ${h} L ${mid} 12 L ${w * 0.84} ${h}`} stroke={color} fill="none" />
    </svg>
  );
}

type FlowNode = { id: string; title: string; sub: string; tone: "path" | "exit" | "jev" };

function Flow({
  nodes,
  edges,
  marker,
  direction = "vertical",
}: {
  nodes: FlowNode[];
  edges: { from: string; to: string }[];
  marker: string;
  direction?: "vertical" | "horizontal";
}) {
  const theme = useHostTheme();
  const nodeWidth = direction === "horizontal" ? 168 : 188;
  const nodeHeight = 58;
  const layout = computeDAGLayout({
    nodes: nodes.map((n) => ({ id: n.id })),
    edges,
    direction,
    nodeWidth,
    nodeHeight,
    rankGap: direction === "horizontal" ? 44 : 36,
    nodeGap: 16,
    padding: 2,
  });
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const markerId = `arrow-${marker}`;

  return (
    <div style={{ overflowX: "auto" }}>
      <div style={{ position: "relative", width: layout.width, height: layout.height }}>
        <svg width={layout.width} height={layout.height} style={{ position: "absolute", inset: 0 }}>
          <defs>
            <marker id={markerId} markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
              <path d="M 0 0 L 8 4 L 0 8 Z" fill={theme.text.tertiary} />
            </marker>
          </defs>
          {layout.edges.map((edge) => (
            <path
              key={`${edge.from}-${edge.to}`}
              d={curve(edge, direction)}
              stroke={theme.stroke.primary}
              fill="none"
              markerEnd={`url(#${markerId})`}
            />
          ))}
        </svg>
        {layout.nodes.map((pos) => {
          const node = byId.get(pos.id);
          if (!node) return null;
          const top =
            node.tone === "jev"
              ? theme.category.purple
              : node.tone === "path"
                ? theme.accent.primary
                : theme.stroke.secondary;
          return (
            <div
              key={pos.id}
              style={{
                position: "absolute",
                left: pos.x,
                top: pos.y,
                width: nodeWidth,
                height: nodeHeight,
                boxSizing: "border-box",
                background: node.tone === "exit" ? theme.fill.tertiary : theme.bg.elevated,
                border: `1px solid ${theme.stroke.secondary}`,
                borderTop: `3px solid ${top}`,
                padding: "6px 8px",
                overflow: "hidden",
              }}
            >
              <div style={{ fontSize: 12, fontWeight: 600, color: theme.text.primary, lineHeight: "15px" }}>
                {node.title}
              </div>
              <div style={{ fontSize: 11, color: theme.text.secondary, lineHeight: "14px", marginTop: 2 }}>
                {node.sub}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function curve(edge: DAGLayoutEdge, direction: "vertical" | "horizontal") {
  const { sourceX, sourceY, targetX, targetY } = edge;
  if (direction === "horizontal") {
    const mid = (sourceX + targetX) / 2;
    return `M ${sourceX} ${sourceY} C ${mid} ${sourceY}, ${mid} ${targetY}, ${targetX} ${targetY}`;
  }
  const mid = (sourceY + targetY) / 2;
  return `M ${sourceX} ${sourceY} C ${sourceX} ${mid}, ${targetX} ${mid}, ${targetX} ${targetY}`;
}
