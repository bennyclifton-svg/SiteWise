package knowledge

import (
	"fmt"
	"strings"
)

type DeliveryStage struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	ParentID string `json:"parent_id,omitempty"`
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
	type substage struct {
		ID       string `yaml:"id"`
		Label    string `yaml:"label"`
		Novation string `yaml:"novation"`
	}
	var file struct {
		Version int    `yaml:"version"`
		Status  string `yaml:"status"`
		Stages  []struct {
			ID        string     `yaml:"id"`
			Label     string     `yaml:"label"`
			Substages []substage `yaml:"substages"`
		} `yaml:"stages"`
	}
	if err := c.unmarshal(path, &file); err != nil {
		return err
	}
	if file.Version != 1 || (file.Status != "draft" && file.Status != "reviewed") || len(file.Stages) == 0 {
		return fmt.Errorf("%s: invalid stage catalogue", path)
	}
	seen := map[string]bool{}
	add := func(id, label, parent, phase string) error {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(label) == "" || seen[id] || (phase != "none" && phase != "pre" && phase != "post") {
			return fmt.Errorf("%s: invalid or duplicate stage %q", path, id)
		}
		seen[id] = true
		c.stages = append(c.stages, DeliveryStage{ID: id, Label: label, ParentID: parent, Novation: phase, Status: file.Status})
		return nil
	}
	for _, stage := range file.Stages {
		if err := add(stage.ID, stage.Label, "", "none"); err != nil {
			return err
		}
		for _, sub := range stage.Substages {
			if err := add(sub.ID, sub.Label, stage.ID, sub.Novation); err != nil {
				return err
			}
		}
	}
	return nil
}
