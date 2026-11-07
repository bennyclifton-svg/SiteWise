package knowledge

import (
	"fmt"
	"strings"
)

// DeliveryStage is one of the project stages every RFP, package, Program and
// the Cost Plan share (works/stages.yaml), or `other`.
type DeliveryStage struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Novation string `json:"novation"`
	Status   string `json:"status"`
}

func (c *Catalog) DeliveryStages() []DeliveryStage { return append([]DeliveryStage(nil), c.stages...) }
func (c *Catalog) DeliveryStage(id string) (DeliveryStage, bool) {
	for _, s := range c.stages {
		if s.ID == id {
			return s, true
		}
	}
	return DeliveryStage{}, false
}

func (c *Catalog) loadDeliveryStages(path string) error {
	if ok, err := exists(path); !ok {
		return err
	}
	var file struct {
		Version int    `yaml:"version"`
		Status  string `yaml:"status"`
		Stages  []struct {
			ID       string `yaml:"id"`
			Label    string `yaml:"label"`
			Novation string `yaml:"novation"`
			Status   string `yaml:"status"`
		} `yaml:"stages"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 || (file.Status != "draft" && file.Status != "reviewed") || len(file.Stages) == 0 {
		return fmt.Errorf("%s: invalid stage catalogue", path)
	}
	seen := map[string]bool{}
	for _, s := range file.Stages {
		phase := s.Novation
		if s.ID == OtherStageID {
			// `other` is never a default, so it has no novation side.
			if phase != "" {
				return fmt.Errorf("%s: %s takes no novation side", path, OtherStageID)
			}
			phase = "none"
		}
		if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.Label) == "" || seen[s.ID] || (phase != "none" && phase != "pre" && phase != "post") {
			return fmt.Errorf("%s: invalid or duplicate stage %q", path, s.ID)
		}
		seen[s.ID] = true
		c.stages = append(c.stages, DeliveryStage{ID: s.ID, Label: s.Label, Novation: phase, Status: file.Status})
	}
	return nil
}
