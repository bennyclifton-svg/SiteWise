// Package specifications holds project specification records and deterministic
// validation. A source statement and a project decision are different records.
package specifications

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type Type struct {
	Authority        string `json:"authority,omitempty"`
	SourceDocumentID string `json:"source_document_id,omitempty"`
	StarterKey       string `json:"starter_key,omitempty"`
	SuggestedTypeID  string `json:"suggested_type_id,omitempty"`
	Origin           string `json:"origin,omitempty"`
	ReviewStatus     string `json:"review_status,omitempty"`
	ID               string `json:"id"`
	ProjectID        string `json:"project_id"`
	Code             string `json:"code"`
	Title            string `json:"title"`
	Kind             string `json:"kind"`
	SystemID         string `json:"system_id"`
	Description      string `json:"description"`
	Scheme           string `json:"scheme"`
	OptionGroup      string `json:"option_group"`
	Status           string `json:"status"`
	Version          int64  `json:"version"`
}

type Application struct {
	ID         string   `json:"id"`
	ProjectID  string   `json:"project_id"`
	TypeID     string   `json:"type_id"`
	PartID     string   `json:"part_id,omitempty"`
	WorkItemID string   `json:"work_item_id,omitempty"`
	Location   string   `json:"location"`
	Surface    string   `json:"surface"`
	Action     string   `json:"action"`
	Inclusion  string   `json:"inclusion"`
	Selection  string   `json:"selection"`
	Quantity   *float64 `json:"quantity,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	Version    int64    `json:"version"`
}

type Value struct {
	State     string   `json:"state"`
	Text      string   `json:"text,omitempty"`
	Number    *float64 `json:"number,omitempty"`
	Upper     *float64 `json:"upper,omitempty"`
	Boolean   *bool    `json:"boolean,omitempty"`
	Unit      string   `json:"unit,omitempty"`
	Qualifier string   `json:"qualifier,omitempty"`
	Basis     string   `json:"basis,omitempty"`
	Method    string   `json:"method,omitempty"`
	Datum     string   `json:"datum,omitempty"`
}

type Source struct {
	Stale       bool   `json:"stale,omitempty"`
	DocumentID  string `json:"document_id,omitempty"`
	Hash        string `json:"hash,omitempty"`
	Page        int    `json:"page,omitempty"`
	Location    string `json:"location,omitempty"`
	Excerpt     string `json:"excerpt,omitempty"`
	Issue       string `json:"issue,omitempty"`
	Marker      string `json:"marker,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty"`
}

type Assertion struct {
	ID            string    `json:"id"`
	TypeID        string    `json:"type_id"`
	ApplicationID string    `json:"application_id,omitempty"`
	Key           string    `json:"key"`
	Value         Value     `json:"value"`
	Meaning       string    `json:"meaning"`
	Source        Source    `json:"source"`
	Origin        string    `json:"origin"`
	Status        string    `json:"status"`
	Actor         string    `json:"actor"`
	Reason        string    `json:"reason"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

type Relationship struct {
	ID         string   `json:"id"`
	FromID     string   `json:"from_id"`
	ToID       string   `json:"to_id"`
	Kind       string   `json:"kind"`
	Sequence   int      `json:"sequence,omitempty"`
	Proportion *float64 `json:"proportion,omitempty"`
	Basis      string   `json:"basis,omitempty"`
	Note       string   `json:"note,omitempty"`
	Version    int64    `json:"version"`
}

type Snapshot struct {
	Coverage      []Coverage     `json:"coverage"`
	Scope         []Scope        `json:"scope"`
	ProjectID     string         `json:"project_id"`
	ProjectName   string         `json:"project_name"`
	Revision      int64          `json:"revision"`
	Types         []Type         `json:"types"`
	Applications  []Application  `json:"applications"`
	Assertions    []Assertion    `json:"assertions"`
	Relationships []Relationship `json:"relationships"`
}

type Issue struct {
	ExportSHA256 []byte    `json:"-"`
	ID           string    `json:"id"`
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	Purpose      string    `json:"purpose"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
	Snapshot     Snapshot  `json:"snapshot"`
}

var Kinds = []string{"material", "product", "finish", "assembly", "equipment", "unspecified"}
var Actions = []string{"new", "replace", "upgrade", "alter", "repair", "remove", "retain", "investigate"}
var Meanings = []string{"requirement", "selection", "capability", "allowance", "exclusion", "observed", "unclassified"}

func small(s string, max int) bool { return utf8.RuneCountInString(s) <= max }
func finite(n *float64) bool       { return n == nil || (!math.IsNaN(*n) && !math.IsInf(*n, 0)) }

func ValidateType(t Type) error {
	if strings.TrimSpace(t.Title) == "" || !small(t.Title, 200) || !small(t.Code, 80) || !slices.Contains(Kinds, t.Kind) {
		return fmt.Errorf("an item needs a title and a valid kind; code may be blank")
	}
	if !small(t.Description, 4000) || !small(t.Scheme, 120) || !small(t.OptionGroup, 120) || !small(t.SystemID, 160) || !slices.Contains([]string{"active", "retired"}, t.Status) {
		return fmt.Errorf("invalid specification item details")
	}
	return nil
}

func ValidateApplication(a Application) error {
	if strings.TrimSpace(a.Location) == "" || !small(a.Location, 200) || !small(a.Surface, 80) || !slices.Contains(Actions, a.Action) {
		return fmt.Errorf("an application needs a location and work action")
	}
	if !slices.Contains([]string{"included", "excluded", "optional", "unknown"}, a.Inclusion) || !slices.Contains([]string{"undecided", "selected", "rejected"}, a.Selection) {
		return fmt.Errorf("invalid inclusion or selection")
	}
	if !finite(a.Quantity) || (a.Quantity != nil && (*a.Quantity < 0 || strings.TrimSpace(a.Unit) == "")) || (a.Quantity == nil && a.Unit != "") || !small(a.Unit, 32) {
		return fmt.Errorf("quantity and unit must be supplied together")
	}
	return nil
}

func ValidateAssertion(a Assertion) error {
	d, ok := DefinitionFor(a.Key)
	if !ok || !slices.Contains(Meanings, a.Meaning) {
		return fmt.Errorf("unknown attribute or assertion meaning")
	}
	v := a.Value
	if !slices.Contains([]string{"known", "unknown", "not_applicable", "deferred", "not_required"}, v.State) {
		return fmt.Errorf("invalid value state")
	}
	if !small(v.Text, 4000) || !small(v.Basis, 500) || !small(v.Method, 300) || !small(v.Datum, 120) || !finite(v.Number) || !finite(v.Upper) {
		return fmt.Errorf("invalid value")
	}
	if !slices.Contains([]string{"", "exact", "minimum", "maximum", "approximate", "site_measured", "tolerance"}, v.Qualifier) {
		return fmt.Errorf("invalid qualifier")
	}
	if v.State != "known" {
		if v.Text != "" || v.Number != nil || v.Upper != nil || v.Boolean != nil || v.Unit != "" {
			return fmt.Errorf("unresolved values cannot contain a selected value")
		}
	} else {
		switch d.Type {
		case "text":
			if strings.TrimSpace(v.Text) == "" || v.Number != nil || v.Upper != nil || v.Boolean != nil || v.Unit != "" {
				return fmt.Errorf("%s needs text", d.Label)
			}
		case "number":
			if v.Number == nil || v.Text != "" || v.Boolean != nil || !slices.Contains(d.Units, v.Unit) || (v.Upper != nil && *v.Upper < *v.Number) {
				return fmt.Errorf("%s needs a number and a supported unit", d.Label)
			}
		case "boolean":
			if v.Boolean == nil || v.Text != "" || v.Number != nil || v.Upper != nil || v.Unit != "" {
				return fmt.Errorf("%s needs yes or no", d.Label)
			}
		}
	}
	if !small(a.Source.Excerpt, 4000) || !small(a.Source.Location, 200) || !small(a.Source.Issue, 200) || a.Source.Page < 0 || a.Source.Page > 100000 || !small(a.Reason, 1000) {
		return fmt.Errorf("invalid source or reason")
	}
	if !slices.Contains([]string{"", "unreviewed", "normal", "changed", "withdrawn"}, a.Source.Marker) {
		return fmt.Errorf("invalid source marker")
	}
	if a.Source.DocumentID != "" && ((a.Source.Page < 1 && strings.TrimSpace(a.Source.Location) == "") || strings.TrimSpace(a.Source.Excerpt) == "") {
		return fmt.Errorf("document evidence needs a page or source location, and an excerpt")
	}
	return nil
}

func ValidateRelationship(r Relationship) error {
	if r.FromID == r.ToID || !slices.Contains([]string{"component_of", "finished_with", "requires", "matches", "replaces"}, r.Kind) || r.Sequence < 0 || r.Sequence > 1000 || !finite(r.Proportion) || !small(r.Basis, 200) || !small(r.Note, 1000) {
		return fmt.Errorf("invalid item relationship")
	}
	if r.Proportion != nil && (*r.Proportion < 0 || *r.Proportion > 100 || strings.TrimSpace(r.Basis) == "") {
		return fmt.Errorf("proportion needs a percentage and basis")
	}
	if (r.Sequence != 0 || r.Proportion != nil) && r.Kind != "component_of" {
		return fmt.Errorf("only components have order or proportion")
	}
	return nil
}
