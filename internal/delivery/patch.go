package delivery

import "encoding/json"

// DateChange distinguishes an omitted edit from explicit null (clear date).
type DateChange struct {
	Present bool
	Value   *string
}

func (d *DateChange) UnmarshalJSON(raw []byte) error {
	d.Present = true
	return json.Unmarshal(raw, &d.Value)
}

type Patch struct {
	Version      int64           `json:"version"`
	Kind         *string         `json:"kind"`
	Title        *string         `json:"title"`
	OwnerText    *string         `json:"owner_text"`
	OwnerUserID  *string         `json:"owner_user_id"`
	BaselineDate DateChange      `json:"baseline_date"`
	TargetDate   DateChange      `json:"target_date"`
	ForecastDate DateChange      `json:"forecast_date"`
	ActualDate   DateChange      `json:"actual_date"`
	AsOf         DateChange      `json:"as_of"`
	Status       *string         `json:"status"`
	PackageID    *string         `json:"package_id"`
	WorkItemID   *string         `json:"work_item_id"`
	StageID      *string         `json:"stage_id"`
	Details      json.RawMessage `json:"details"`
	Retired      *bool           `json:"retired"`
}

func (p Patch) Apply(c *Content) {
	for dst, src := range map[*string]*string{&c.Kind: p.Kind, &c.Title: p.Title, &c.OwnerText: p.OwnerText, &c.OwnerUserID: p.OwnerUserID, &c.Status: p.Status, &c.PackageID: p.PackageID, &c.WorkItemID: p.WorkItemID, &c.StageID: p.StageID} {
		if src != nil {
			*dst = *src
		}
	}
	for dst, src := range map[**string]DateChange{&c.BaselineDate: p.BaselineDate, &c.TargetDate: p.TargetDate, &c.ForecastDate: p.ForecastDate, &c.ActualDate: p.ActualDate, &c.AsOf: p.AsOf} {
		if src.Present {
			*dst = src.Value
		}
	}
	if p.Details != nil {
		c.Details = p.Details
	}
}
