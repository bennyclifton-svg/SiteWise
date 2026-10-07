package reports

import "sort"

type SourceState struct {
	Domains                  map[string]int64 `json:"domains"`
	ProfileRevision          int64            `json:"profile_revision"`
	ProfileFingerprint       string           `json:"profile_fingerprint"`
	ProfileKnowledgeVersion  string           `json:"profile_knowledge_version"`
	ProfileQuestionVersion   string           `json:"profile_question_version"`
	ProfileThresholdsVersion string           `json:"profile_thresholds_version"`
	KnowledgeVersion         string           `json:"knowledge_version"`
	QuestionVersion          string           `json:"question_version"`
	ThresholdsVersion        string           `json:"thresholds_version"`
	AppBuild                 string           `json:"app_build"`
	TemplateID               string           `json:"template_id"`
	TemplateVersion          int              `json:"template_version"`
}

// M1 RFP assembly has no cost dependency or cost-table access. The M2 assembler
// must add its pricing dependencies when those sections become available.
var draftDependencies = map[string][]string{
	"rfp": {"profile_inputs", "works", "packages", "delivery"},
}

func StaleReasons(kind string, saved, current SourceState) []string {
	deps, ok := draftDependencies[kind]
	if !ok {
		return []string{"unsupported_report_kind"}
	}
	out := []string{}
	for _, name := range deps {
		old, oldOK := saved.Domains[name]
		now, nowOK := current.Domains[name]
		if !oldOK || !nowOK || old != now {
			out = append(out, name)
		}
	}
	if saved.ProfileRevision != current.ProfileRevision || saved.ProfileFingerprint != current.ProfileFingerprint {
		out = append(out, "profile_build")
	}
	for name, pair := range map[string][2]string{"knowledge_version": {saved.KnowledgeVersion, current.KnowledgeVersion}, "question_version": {saved.QuestionVersion, current.QuestionVersion}, "thresholds_version": {saved.ThresholdsVersion, current.ThresholdsVersion}, "app_build": {saved.AppBuild, current.AppBuild}} {
		if pair[0] != pair[1] {
			out = append(out, name)
		}
	}
	if saved.TemplateID != current.TemplateID || saved.TemplateVersion != current.TemplateVersion {
		out = append(out, "template")
	}
	sort.Strings(out)
	return out
}
