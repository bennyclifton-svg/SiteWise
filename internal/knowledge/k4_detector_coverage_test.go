package knowledge_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestK4PreciseDetectorRoutingAndAnswerKeys(t *testing.T) {
	cat := loadRepo(t)
	raw, err := os.ReadFile("../../data/eval/profile/k4-detector-answer-keys.json")
	if err != nil {
		t.Fatal(err)
	}
	var keys []struct {
		QuestionID string   `json:"question_id"`
		Labels     []string `json:"labels"`
		Text       string   `json:"text"`
		Expected   bool     `json:"expected"`
	}
	if err = json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	coverage := map[string]map[bool]bool{}
	for _, key := range keys {
		if key.Text == "" {
			t.Fatal("empty answer key")
		}
		if coverage[key.QuestionID] == nil {
			coverage[key.QuestionID] = map[bool]bool{}
		}
		coverage[key.QuestionID][key.Expected] = true
		found := false
		for _, q := range cat.EvidenceQuestions(key.Labels) {
			if q.ID == key.QuestionID {
				found = true
			}
		}
		if !found {
			t.Fatalf("key question not routed:%s", key.QuestionID)
		}
		for _, q := range cat.EvidenceQuestions([]string{"hydraulic.gas"}) {
			if q.ID == key.QuestionID {
				t.Fatal("unrelated label routes question", q.ID)
			}
		}
	}
	for id, bits := range coverage {
		if !bits[true] || !bits[false] {
			t.Fatal("missing polarity", id)
		}
	}
	for _, label := range []string{"mechanical.air-conditioning", "mechanical.air-distribution", "hydraulic.cold-water", "hydraulic.hot-water", "hydraulic.non-potable-water"} {
		found := false
		for _, q := range cat.EvidenceQuestions([]string{label}) {
			if q.ID == "fm:fm.services-plant-maintenance-access" {
				found = true
			}
		}
		if !found {
			t.Fatal("maintenance source system not routed", label)
		}
	}
}
