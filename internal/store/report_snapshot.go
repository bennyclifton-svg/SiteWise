package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
)

// The caller owns the transaction so sources, revision counters and the draft
// write belong to one snapshot. No method here queues reading or recomputes it.
func (s *Store) reportSnapshot(ctx context.Context, tx pgx.Tx, org, project, pkg, appBuild string, template knowledge.ReportTemplate) (reports.Snapshot, reports.SourceState, error) {
	var snap reports.Snapshot
	var state reports.SourceState
	if s.profileBuild == nil {
		return snap, state, fmt.Errorf("profile build configuration unavailable")
	}
	snap.ProjectID = project
	err := tx.QueryRow(ctx, `SELECT p.name,jsonb_build_object('project',to_jsonb(p),'site',to_jsonb(s)) FROM projects p JOIN sites s ON s.org_id=p.org_id AND s.id=p.site_id WHERE p.org_id=$1::uuid AND p.id=$2::uuid`, org, project).Scan(&snap.ProjectName, &snap.ProjectBasis)
	if errors.Is(err, pgx.ErrNoRows) {
		return snap, state, ErrNotFound
	}
	if err != nil {
		return snap, state, err
	}
	snap.Package, err = readPackage(ctx, tx, org, project, pkg)
	if err != nil {
		return snap, state, err
	}
	v, err := readProfileTx(ctx, tx, org, project, s.profileBuild.ReadKinds)
	if err != nil {
		return snap, state, err
	}
	snap.PendingDocuments, snap.FailedDocuments, snap.UnreadDocuments = v.PendingDocuments, v.FailedDocuments, v.UnreadDocuments
	snap.ProfileMissing = v.Revision == 0
	snap.ProfileStale = v.StaleFor(*s.profileBuild)
	snap.PartLabels = map[string]string{}
	for _, p := range v.Parts {
		snap.PartLabels[p.ID] = p.Label
	}
	labels := reportBriefLabels(s.workCatalog())
	audit, err := reportUserAudit(ctx, tx, org, project)
	if err != nil {
		return snap, state, err
	}
	for _, row := range v.Rows {
		if !strings.HasPrefix(row.Key, "hdr.") && !strings.HasPrefix(row.Key, "det.") {
			continue
		}
		label := labels[row.Key]
		if label == "" {
			label = strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(row.Key, "hdr."), "det."), "_", " ")
		}
		if part := snap.PartLabels[row.PartID]; part != "" {
			label = part + " — " + label
		}
		basis, err := reportRowBasis(row, audit)
		if err != nil {
			return snap, state, err
		}
		unknown := row.Value == "" || row.ValueState == "unknown" || row.ValueState == "cleared" || row.Band == "red" || row.Band == "blank" || row.Band == "unchecked" || row.Band == "suggested"
		snap.Brief = append(snap.Brief, reports.BriefValue{ID: row.PartID + ":" + row.Key, Label: label, Text: row.Value, Origin: row.Origin, ReviewStatus: row.ReviewStatus, Meaning: row.Meaning, Basis: basis, Unknown: unknown})
	}
	snap.Works, err = readWorkItems(ctx, tx, org, project)
	if err != nil {
		return snap, state, err
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT jsonb_build_object(
 'scope',COALESCE((SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM package_scope_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL),'[]'::jsonb),
 'delivery',COALESCE((SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM project_delivery_items i WHERE org_id=$1::uuid AND project_id=$2::uuid AND retired_at IS NULL),'[]'::jsonb),
 'proposals',COALESCE((SELECT jsonb_agg(to_jsonb(p) ORDER BY rank,key) FROM proposals p WHERE org_id=$1::uuid AND project_id=$2::uuid),'[]'::jsonb))`, org, project).Scan(&raw)
	if err != nil {
		return snap, state, err
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return snap, state, err
	}
	packages, err := readPackages(ctx, tx, org, project)
	if err != nil {
		return snap, state, err
	}
	assignments := []procurement.Assignment{}
	for _, row := range snap.Scope {
		assignments = append(assignments, procurement.Assignment{ID: row.ID, PackageID: row.PackageID, ScopeContent: row.ScopeContent, RetiredAt: row.RetiredAt})
	}
	snap.Gaps, err = procurement.CheckGaps(snap.Works, packages, assignments, s.workCatalog())
	if err != nil {
		return snap, state, err
	}
	state = s.reportSourceState(v, template, appBuild)
	return snap, state, nil
}

func (s *Store) reportSourceState(v ProfileView, template knowledge.ReportTemplate, appBuild string) reports.SourceState {
	r := v.CurrentInputs
	return reports.SourceState{Domains: map[string]int64{"profile_inputs": r.ProfileInputs, "works": r.Works, "packages": r.Packages, "delivery": r.Delivery}, ProfileRevision: v.Revision, ProfileFingerprint: v.InputFingerprint, ProfileKnowledgeVersion: v.KnowledgeVersion, ProfileQuestionVersion: v.QuestionVersion, ProfileThresholdsVersion: v.ThresholdsVersion, KnowledgeVersion: s.profileBuild.KnowledgeVersion, QuestionVersion: s.profileBuild.QuestionVersion, ThresholdsVersion: s.profileBuild.ThresholdsVersion, AppBuild: appBuild, TemplateID: template.ID, TemplateVersion: template.Version}
}

func reportBriefLabels(cat *knowledge.Catalog) map[string]string {
	labels := map[string]string{"hdr.building_class": "Building class", "hdr.subclass": "Building type", "hdr.work_type": "Work type"}
	if cat == nil {
		return labels
	}
	for _, condition := range cat.Taxonomy().Conditions {
		labels["hdr."+condition.Key] = condition.Label
	}
	for _, d := range cat.ProfileDeterminants() {
		labels["det."+d.ID] = d.Label
	}
	return labels
}
