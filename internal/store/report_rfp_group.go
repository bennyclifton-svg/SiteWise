package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/rfp"
)

// Combined engagements, slice 2 (docs/plans/2026-11-07-combined-engagements-slice2.md).
// A group's RFP is one report on the group package. Each member keeps its
// own RFP report and settings as the store of its choices: the group's
// build assembles every member's draft in its own transaction, then
// composes the document from those drafts (rfp.GroupSections). A member's
// setting is written to the member's report and the group's draft is
// recomposed in the same transaction, from frozen drivers, with no project
// read. The shared issue details and transmittal are the group's own
// settings. Nothing here calls Jev.

// groupState is what a group draft was composed from, frozen in its source
// state: each member, in Procurement order, and the member report that
// stores its choices ("" for a discipline with no RFP rules).
type groupState struct {
	Members []groupStateMember `json:"members"`
}

type groupStateMember struct {
	PackageID    string `json:"package_id"`
	Title        string `json:"title"`
	DisciplineID string `json:"discipline_id,omitempty"`
	ReportID     string `json:"report_id,omitempty"`
}

// memberPart is one member resolved for the engagement.
type memberPart struct {
	groupStateMember
	Report   Report
	Draft    ReportDraft
	Settings rfp.Settings // the member's own settings
	Disc     knowledge.RFPDiscipline
	Sk       knowledge.RFPSkeleton
	Drivers  rfp.DriverSet
	Resolved rfp.Resolved // resolved with the group's shared settings
}

func (p memberPart) templated() bool { return p.ReportID != "" }

// groupDraft reports whether a draft is a combined engagement's RFP.
func groupDraft(state reports.SourceState) (groupState, bool) {
	var st rfpState
	if len(state.RFP) == 0 || json.Unmarshal(state.RFP, &st) != nil || st.Group == nil {
		return groupState{}, false
	}
	return *st.Group, true
}

func engagementOf(title string, parts []memberPart) rfp.Engagement {
	e := rfp.Engagement{Title: title}
	for _, p := range parts {
		e.Parts = append(e.Parts, rfp.GroupPart{PackageID: p.PackageID, PackageTitle: p.Title, DisciplineID: p.DisciplineID, Templated: p.templated(), Discipline: p.Disc, Resolved: p.Resolved, Settings: p.Settings})
	}
	return e
}

// leadPart is the first member with RFP rules; it supplies the shared parts.
func leadPart(parts []memberPart) (memberPart, bool) {
	for _, p := range parts {
		if p.templated() {
			return p, true
		}
	}
	return memberPart{}, false
}

// readGroupMemberPackages lists a group's live members in Procurement order.
func readGroupMemberPackages(ctx context.Context, tx pgx.Tx, org, project, group string) ([]procurement.Package, error) {
	rows, err := tx.Query(ctx, `SELECT to_jsonb(p) FROM packages p WHERE org_id=$1::uuid AND project_id=$2::uuid AND group_id=$3::uuid AND retired_at IS NULL ORDER BY lower(title),id`, org, project, group)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []procurement.Package{}
	for rows.Next() {
		var raw []byte
		var p procurement.Package
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// packageRFPReport is a package's RFP report, if it has one.
func packageRFPReport(ctx context.Context, tx pgx.Tx, org, project, pkg string) (Report, bool, error) {
	var r Report
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT to_jsonb(r) FROM reports r WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND kind='rfp' ORDER BY created_at,id LIMIT 1`, org, project, pkg).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	return r, true, json.Unmarshal(raw, &r)
}

// ensureMemberReports gives every member of a group an RFP report, so each
// has somewhere to keep its choices. Existing reports are left as they are.
func ensureMemberReports(ctx context.Context, tx pgx.Tx, org, project, group string) error {
	members, err := readGroupMemberPackages(ctx, tx, org, project, group)
	if err != nil {
		return err
	}
	for _, m := range members {
		if _, ok, err := packageRFPReport(ctx, tx, org, project, m.ID); err != nil || ok {
			if err != nil {
				return err
			}
			continue
		}
		if _, err := insertReport(ctx, tx, org, project, "rfp", m.ID, m.Title); err != nil {
			return err
		}
	}
	return nil
}

// groupSectionsTx builds a group's RFP: each member's draft is assembled
// from the project as its own RFP would be, then the document is composed.
// source is the group draft (or last issue) holding the shared settings.
func (s *Store) groupSectionsTx(ctx context.Context, tx pgx.Tx, org string, r Report, group procurement.Package, source string, edits []reports.Edit, appBuild string, useLastCompleted bool, template knowledge.ReportTemplate) ([]reports.Section, reports.SourceState, error) {
	var state reports.SourceState
	members, err := readGroupMemberPackages(ctx, tx, org, r.ProjectID, group.ID)
	if err != nil {
		return nil, state, err
	}
	if len(members) == 0 {
		return nil, state, ErrInvalidReport
	}
	shared, err := readReportSettings(ctx, tx, org, source)
	if err != nil {
		return nil, state, err
	}
	cat := s.workCatalog()
	st := groupState{}
	parts := []memberPart{}
	for _, m := range members {
		member := groupStateMember{PackageID: m.ID, Title: m.Title, DisciplineID: m.DisciplineID}
		if _, ok := cat.RFPForDiscipline(m.DisciplineID); !ok {
			st.Members = append(st.Members, member)
			parts = append(parts, memberPart{groupStateMember: member})
			continue
		}
		mr, ok, err := packageRFPReport(ctx, tx, org, r.ProjectID, m.ID)
		if err != nil {
			return nil, state, err
		}
		if !ok {
			if mr, err = insertReport(ctx, tx, org, r.ProjectID, "rfp", m.ID, m.Title); err != nil {
				return nil, state, err
			}
		}
		if _, err := tx.Exec(ctx, `SELECT id FROM reports WHERE org_id=$1::uuid AND id=$2::uuid FOR UPDATE`, org, mr.ID); err != nil {
			return nil, state, err
		}
		md, err := s.assembleDraftTx(ctx, tx, org, mr, appBuild, useLastCompleted)
		if err != nil {
			return nil, state, err
		}
		mr.CurrentDraftVersionID = md.ID
		member.ReportID = mr.ID
		if state.TemplateID == "" {
			// Every member reads the same project in this transaction; the
			// first member's revisions are the group's.
			state = md.SourceRevisions
		}
		part, err := s.memberPartTx(ctx, tx, org, member, mr, md, shared)
		if err != nil {
			return nil, state, err
		}
		st.Members = append(st.Members, member)
		parts = append(parts, part)
	}
	if state.TemplateID == "" {
		return nil, state, ErrInvalidReport
	}
	sections, err := rfp.GroupSections(engagementOf(group.Title, parts), template)
	if err != nil {
		return nil, state, err
	}
	if sections, err = reports.ApplyEdits(sections, edits); err != nil {
		return nil, state, err
	}
	raw, err := json.Marshal(rfpState{Group: &st})
	if err != nil {
		return nil, state, err
	}
	state.RFP, state.RFPParts = raw, len(parts)
	return sections, state, nil
}

// memberPartTx resolves one member from its draft's frozen drivers and its
// own settings, with the group's shared settings in place of its own.
func (s *Store) memberPartTx(ctx context.Context, tx pgx.Tx, org string, member groupStateMember, mr Report, md ReportDraft, shared rfp.Settings) (memberPart, error) {
	part := memberPart{groupStateMember: member, Report: mr, Draft: md}
	settings, err := readReportSettings(ctx, tx, org, md.ID)
	if err != nil {
		return part, err
	}
	part.Settings = settings
	sk, ok := s.workCatalog().RFPSkeleton()
	if !ok {
		return part, ErrInvalidReport
	}
	part.Disc, part.Sk, part.Drivers, part.Resolved, err = s.storedRFP(md.SourceRevisions, rfp.MergeSettings(sk, settings, shared))
	return part, err
}

// groupEngagementTx resolves a group draft's members from their current
// drafts, without reading the project. It returns the group's name, the
// parts and the group's own (shared) settings.
func (s *Store) groupEngagementTx(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft) (string, []memberPart, rfp.Settings, error) {
	st, ok := groupDraft(d.SourceRevisions)
	if !ok {
		return "", nil, nil, ErrInvalidReport
	}
	group, err := readPackageRecord(ctx, tx, org, r.ProjectID, r.PackageID, true)
	if err != nil {
		return "", nil, nil, err
	}
	shared, err := readReportSettings(ctx, tx, org, d.ID)
	if err != nil {
		return "", nil, nil, err
	}
	drafts, settings, err := memberDraftStates(ctx, tx, org, r.ProjectID, st)
	if err != nil {
		return "", nil, nil, err
	}
	sk, ok := s.workCatalog().RFPSkeleton()
	if !ok {
		return "", nil, nil, ErrInvalidReport
	}
	parts := []memberPart{}
	for _, m := range st.Members {
		if m.ReportID == "" {
			parts = append(parts, memberPart{groupStateMember: m})
			continue
		}
		md, ok := drafts[m.ReportID]
		if !ok {
			return "", nil, nil, ErrInvalidReport
		}
		part := memberPart{groupStateMember: m, Report: Report{ID: m.ReportID, ProjectID: r.ProjectID, PackageID: m.PackageID, Kind: "rfp", CurrentDraftVersionID: md.ID}, Draft: md, Settings: settings[md.ID]}
		if part.Settings == nil {
			part.Settings = rfp.Settings{}
		}
		if part.Disc, part.Sk, part.Drivers, part.Resolved, err = s.storedRFP(md.SourceRevisions, rfp.MergeSettings(sk, part.Settings, shared)); err != nil {
			return "", nil, nil, err
		}
		parts = append(parts, part)
	}
	return group.Title, parts, shared, nil
}

// memberDraftStates reads every member's current draft (its id, version and
// frozen state, not its sections) and settings in two queries, so a setting
// edit recomposes the group without reading each member's whole document.
func memberDraftStates(ctx context.Context, tx pgx.Tx, org, project string, st groupState) (map[string]ReportDraft, map[string]rfp.Settings, error) {
	ids := []string{}
	for _, m := range st.Members {
		if m.ReportID != "" {
			ids = append(ids, m.ReportID)
		}
	}
	drafts := map[string]ReportDraft{}
	rows, err := tx.Query(ctx, `SELECT r.id::text,v.id::text,v.version,v.status,v.template_id,v.source_revisions FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.id=r.current_draft_version_id
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.id=ANY($3::uuid[])`, org, project, ids)
	if err != nil {
		return nil, nil, err
	}
	versions := []string{}
	for rows.Next() {
		var report string
		var d ReportDraft
		var state []byte
		if err := rows.Scan(&report, &d.ID, &d.Version, &d.Status, &d.TemplateID, &state); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if err := json.Unmarshal(state, &d.SourceRevisions); err != nil {
			rows.Close()
			return nil, nil, err
		}
		d.ReportID, d.ProjectID = report, project
		drafts[report] = d
		versions = append(versions, d.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	settings := map[string]rfp.Settings{}
	rows, err = tx.Query(ctx, `SELECT report_version_id::text,key,value,basis FROM report_settings WHERE org_id=$1::uuid AND report_version_id=ANY($2::uuid[])`, org, versions)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var version, key string
		var value, basis []byte
		if err := rows.Scan(&version, &key, &value, &basis); err != nil {
			return nil, nil, err
		}
		if settings[version] == nil {
			settings[version] = rfp.Settings{}
		}
		settings[version][key] = rfp.Setting{Value: value, Basis: basis}
	}
	return drafts, settings, rows.Err()
}

// readDraftState reads a draft without its sections or references: what a
// setting write needs to check and re-resolve it, at a fraction of the cost
// of a combined engagement's whole document.
func readDraftState(ctx context.Context, tx pgx.Tx, org, id string) (ReportDraft, error) {
	var d ReportDraft
	var state []byte
	err := tx.QueryRow(ctx, `SELECT id::text,report_id::text,project_id::text,number,status,reporting_date::text,template_id,template_version,version,source_revisions FROM report_versions WHERE org_id=$1::uuid AND id=$2::uuid`, org, id).
		Scan(&d.ID, &d.ReportID, &d.ProjectID, &d.Number, &d.Status, &d.ReportingDate, &d.TemplateID, &d.TemplateVersion, &d.Version, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	return d, json.Unmarshal(state, &d.SourceRevisions)
}

// recomposeGroupTx re-renders a group's draft from its members' current
// settings. It changes no member and reads no project records.
func (s *Store) recomposeGroupTx(ctx context.Context, tx pgx.Tx, org string, r Report) (ReportDraft, error) {
	var d ReportDraft
	if r.CurrentDraftVersionID == "" {
		return d, ErrInvalidReport
	}
	// Only the draft's frozen state is needed: its old sections are replaced.
	d, err := readDraftState(ctx, tx, org, r.CurrentDraftVersionID)
	if err != nil {
		return d, err
	}
	if d.Status != "draft" {
		return d, ErrInvalidReport
	}
	title, parts, _, err := s.groupEngagementTx(ctx, tx, org, r, d)
	if err != nil {
		return d, err
	}
	template, ok := s.workCatalog().CurrentReportTemplate(rfpTemplateID)
	if !ok {
		return d, ErrInvalidReport
	}
	sections, err := rfp.GroupSections(engagementOf(title, parts), template)
	if err != nil {
		return d, err
	}
	edits, err := readReportEdits(ctx, tx, org, d.ID)
	if err != nil {
		return d, err
	}
	if sections, err = reports.ApplyEdits(sections, edits); err != nil {
		return d, err
	}
	sections, refs, err := reports.Cite(sections)
	if err != nil {
		return d, err
	}
	raw, _ := json.Marshal(sections)
	if err := tx.QueryRow(ctx, `UPDATE report_versions SET sections=$3::jsonb,version=version+1,updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid AND status='draft' RETURNING version`, org, d.ID, raw).Scan(&d.Version); err != nil {
		return d, err
	}
	d.Sections = sections
	if err := saveReportReferences(ctx, tx, org, d.ID, refs); err != nil {
		return d, err
	}
	d.References = refs
	d.MaterialAssumptions = reports.MaterialAssumptions(refs)
	return d, nil
}

// recomposeGroupOf recomposes the draft of the group a member belongs to,
// returning the group's report id ("" when the member is in no group or
// the group has no RFP draft yet).
func (s *Store) recomposeGroupOf(ctx context.Context, tx pgx.Tx, org string, member Report) (string, error) {
	var group *string
	err := tx.QueryRow(ctx, `SELECT group_id::text FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, member.ProjectID, member.PackageID).Scan(&group)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && group == nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	gr, ok, err := packageRFPReport(ctx, tx, org, member.ProjectID, *group)
	if err != nil || !ok || gr.CurrentDraftVersionID == "" {
		return "", err
	}
	d, err := readDraftState(ctx, tx, org, gr.CurrentDraftVersionID)
	if err != nil {
		return "", err
	}
	if st, ok := groupDraft(d.SourceRevisions); !ok || !hasMember(st, member.ID) {
		// The member joined after the group's last build; the next refresh
		// adds its part.
		return gr.ID, nil
	}
	if _, err := s.recomposeGroupTx(ctx, tx, org, gr); err != nil {
		return "", err
	}
	if err := BumpRevision(ctx, tx, org, gr.ProjectID, "reports"); err != nil {
		return "", err
	}
	return gr.ID, nil
}

func hasMember(st groupState, report string) bool {
	for _, m := range st.Members {
		if m.ReportID == report {
			return true
		}
	}
	return false
}

// writeGroupSetting stores one of the engagement's shared settings on the
// group's draft; the caller recomposes it. A discipline's own choice belongs to its
// member's report and is refused here.
func (s *Store) writeGroupSetting(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft, actor, key string, value json.RawMessage, settings rfp.Settings) error {
	sk, ok := s.workCatalog().RFPSkeleton()
	if !ok {
		return ErrInvalidReport
	}
	if !rfp.SharedSetting(sk, key) {
		return ErrInvalidSetting
	}
	_, parts, _, err := s.groupEngagementTx(ctx, tx, org, r, d)
	if err != nil {
		return err
	}
	lead, ok := leadPart(parts)
	if !ok {
		return ErrInvalidReport
	}
	if value == nil {
		if _, ok := settings[key]; !ok {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM report_settings WHERE org_id=$1::uuid AND report_version_id=$2::uuid AND key=$3`, org, d.ID, key); err != nil {
			return err
		}
		delete(settings, key)
	} else {
		if rfp.DocumentListSetting(key) {
			if value, err = transmittalSelection(ctx, tx, org, r.ProjectID, value); err != nil {
				return err
			}
		}
		if err := rfp.ValidateSetting(lead.Disc, lead.Sk, lead.Drivers, key, value); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO report_settings(org_id,report_version_id,key,value,basis,user_id) VALUES($1::uuid,$2::uuid,$3,$4::jsonb,'{}'::jsonb,$5::uuid) ON CONFLICT(org_id,report_version_id,key) DO UPDATE SET value=EXCLUDED.value,basis=EXCLUDED.basis,user_id=EXCLUDED.user_id,version=report_settings.version+1,updated_at=now()`, org, d.ID, key, []byte(value), actor); err != nil {
			return err
		}
		settings[key] = rfp.Setting{Value: value}
	}
	return nil
}

// RFPGroupView is the editor's view of a combined engagement: each member
// with the controls of its own RFP and the draft version to write against,
// and the members that have left since the last build.
type RFPGroupView struct {
	Title   string           `json:"title"`
	Members []RFPGroupMember `json:"members"`
	Left    []string         `json:"left"`
}

type RFPGroupMember struct {
	PackageID string `json:"package_id"`
	Title     string `json:"title"`
	// Section is the member's part in the document.
	Section string `json:"section"`
	// ReportID and DraftVersion address a setting to the member's report;
	// empty for a discipline with no RFP rules.
	ReportID     string        `json:"report_id,omitempty"`
	DraftVersion int64         `json:"draft_version,omitempty"`
	Controls     *rfp.Controls `json:"controls,omitempty"`
}

// groupControls is the editor payload of a group draft: the shared controls
// (page one, terms, proposal dates and the merged checks) and each member's.
func (s *Store) groupControls(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft) (*rfp.Controls, *RFPGroupView, error) {
	title, parts, shared, err := s.groupEngagementTx(ctx, tx, org, r, d)
	if err != nil {
		return nil, nil, err
	}
	lead, ok := leadPart(parts)
	if !ok {
		return nil, nil, ErrInvalidReport
	}
	e := engagementOf(title, parts)
	c := rfp.GroupControls(rfp.BuildControls(lead.Resolved, lead.Disc, lead.Sk, shared), e)
	view := &RFPGroupView{Title: title, Members: []RFPGroupMember{}, Left: []string{}}
	for _, p := range parts {
		m := RFPGroupMember{PackageID: p.PackageID, Title: p.Title, Section: rfp.PartSectionID(p.PackageID)}
		if p.templated() {
			mc := rfp.BuildControls(p.Resolved, p.Disc, p.Sk, p.Settings)
			m.ReportID, m.DraftVersion, m.Controls = p.Report.ID, p.Draft.Version, &mc
		}
		view.Members = append(view.Members, m)
	}
	current, err := readGroupMemberPackages(ctx, tx, org, r.ProjectID, r.PackageID)
	if err != nil {
		return nil, nil, err
	}
	live := map[string]bool{}
	for _, p := range current {
		live[p.ID] = true
	}
	for _, p := range parts {
		if !live[p.PackageID] {
			view.Left = append(view.Left, p.Title)
		}
	}
	return &c, view, nil
}

// groupBlocking are the checks that block a group draft's issue.
func (s *Store) groupBlocking(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft) ([]rfp.Check, error) {
	title, parts, _, err := s.groupEngagementTx(ctx, tx, org, r, d)
	if err != nil {
		return nil, err
	}
	return rfp.GroupBlocking(engagementOf(title, parts)), nil
}

// groupResponseForm is a group draft's response form: a sum per discipline.
func (s *Store) groupResponseForm(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft) ([]rfp.FeeSection, []rfp.Row, string, error) {
	title, parts, _, err := s.groupEngagementTx(ctx, tx, org, r, d)
	if err != nil {
		return nil, nil, "", err
	}
	lead, ok := leadPart(parts)
	if !ok {
		return nil, nil, "", ErrInvalidReport
	}
	e := engagementOf(title, parts)
	shared := rfp.GroupSharedReturnables(e)
	return rfp.GroupSchedule(e), shared, lead.Resolved.Fields["ref"].Value, nil
}

// groupFactBasis is the skeleton and drivers a fact edited on a group's
// page one is checked against: the lead member's.
func (s *Store) groupFactBasis(ctx context.Context, tx pgx.Tx, org string, r Report, d ReportDraft) (knowledge.RFPSkeleton, rfp.DriverSet, error) {
	_, parts, _, err := s.groupEngagementTx(ctx, tx, org, r, d)
	if err != nil {
		return knowledge.RFPSkeleton{}, rfp.DriverSet{}, err
	}
	lead, ok := leadPart(parts)
	if !ok {
		return knowledge.RFPSkeleton{}, rfp.DriverSet{}, ErrInvalidReport
	}
	return lead.Sk, lead.Drivers, nil
}
