package specifications

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Field struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Meaning string `json:"meaning"`
	Value   string `json:"value"`
	Source  Source `json:"source"`
}
type Row struct {
	Authority     string   `json:"authority"`
	TypeID        string   `json:"type_id"`
	ApplicationID string   `json:"application_id,omitempty"`
	Code          string   `json:"code"`
	Title         string   `json:"title"`
	SystemID      string   `json:"system_id"`
	Location      string   `json:"location"`
	Surface       string   `json:"surface"`
	Action        string   `json:"action"`
	Selection     string   `json:"selection"`
	Inclusion     string   `json:"inclusion"`
	Scheme        string   `json:"scheme"`
	OptionGroup   string   `json:"option_group"`
	Description   string   `json:"description"`
	Quantity      *float64 `json:"quantity,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	ReviewStatus  string   `json:"review_status,omitempty"`
	Fields        []Field  `json:"fields"`
	Pending       int      `json:"pending"`
	Relationships []string `json:"relationships"`
}

func ValueText(v Value) string {
	if v.State != "known" {
		return strings.ReplaceAll(v.State, "_", " ")
	}
	text := v.Text
	if v.Number != nil {
		text = strconv.FormatFloat(*v.Number, 'f', -1, 64)
		if v.Upper != nil {
			text += "–" + strconv.FormatFloat(*v.Upper, 'f', -1, 64)
		}
		if v.Unit != "" {
			text += " " + v.Unit
		}
	}
	if v.Boolean != nil {
		if *v.Boolean {
			text = "Yes"
		} else {
			text = "No"
		}
	}
	if v.Qualifier != "" && v.Qualifier != "exact" {
		text = strings.ReplaceAll(v.Qualifier, "_", " ") + " " + text
	}
	for _, s := range []string{v.Datum, v.Basis, v.Method} {
		if s != "" {
			text += "; " + s
		}
	}
	return text
}

// Assemble applies only accepted values. Local decisions override the matching
// type default, never unrelated meanings such as an offered capability.
func Assemble(s Snapshot) []Row {
	byType := map[string][]Assertion{}
	byApplication := map[string][]Application{}
	names := map[string]string{}
	for _, a := range s.Assertions {
		byType[a.TypeID] = append(byType[a.TypeID], a)
	}
	for _, a := range s.Applications {
		byApplication[a.TypeID] = append(byApplication[a.TypeID], a)
	}
	for _, t := range s.Types {
		names[t.ID] = strings.TrimSpace(t.Code + " " + t.Title)
	}
	out := []Row{}
	for _, t := range s.Types {
		if t.Status == "retired" {
			continue
		}
		apps := byApplication[t.ID]
		if len(apps) == 0 {
			apps = []Application{{Location: "Not assigned", Selection: "undecided", Inclusion: "unknown"}}
		}
		for _, a := range apps {
			r := Row{TypeID: t.ID, ApplicationID: a.ID, Code: t.Code, Title: t.Title, SystemID: t.SystemID, Location: a.Location, Surface: a.Surface, Action: a.Action, Selection: a.Selection, Inclusion: a.Inclusion, Scheme: t.Scheme, OptionGroup: t.OptionGroup, Description: t.Description, Fields: []Field{}, Relationships: []string{}}
			r.Authority = "sitewise"
			if External(s, t) {
				r.Authority = "external"
				if t.Origin != "document" {
					r.Description = ""
				}
			}
			r.Quantity, r.Unit, r.ReviewStatus = a.Quantity, a.Unit, t.ReviewStatus
			values := map[string]Assertion{}
			for _, local := range []bool{false, true} {
				for _, v := range byType[t.ID] {
					if (v.ApplicationID != "") != local || (local && v.ApplicationID != a.ID) {
						continue
					}
					if v.Status == "proposed" {
						r.Pending++
						continue
					}
					if v.Status == "accepted" {
						// A later responsibility transfer does not turn old local
						// decisions into statements by the external maintainer.
						if r.Authority == "external" && v.Origin != "document" {
							r.Pending++
							continue
						}
						values[v.Key+"/"+v.Meaning] = v
					}
				}
			}
			for _, v := range values {
				d, _ := DefinitionFor(v.Key)
				r.Fields = append(r.Fields, Field{Key: v.Key, Label: d.Label, Meaning: v.Meaning, Value: ValueText(v.Value), Source: v.Source})
			}
			sort.Slice(r.Fields, func(i, j int) bool {
				return r.Fields[i].Key+"/"+r.Fields[i].Meaning < r.Fields[j].Key+"/"+r.Fields[j].Meaning
			})
			for _, link := range s.Relationships {
				if r.Authority == "external" {
					continue
				}
				if link.FromID == t.ID || (link.ToID == t.ID && link.Kind == "component_of") {
					label := strings.ReplaceAll(link.Kind, "_", " ") + ": " + names[link.ToID]
					if link.ToID == t.ID {
						label = "component: " + names[link.FromID]
					}
					if link.Sequence > 0 {
						label = fmt.Sprintf("%d. %s", link.Sequence, label)
					}
					if link.Proportion != nil {
						label += fmt.Sprintf(" (%g%% %s)", *link.Proportion, link.Basis)
					}
					if link.Note != "" {
						label += "; " + link.Note
					}
					r.Relationships = append(r.Relationships, label)
				}
			}
			sort.Strings(r.Relationships)
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Code+out[i].Title+out[i].Location+out[i].ApplicationID < out[j].Code+out[j].Title+out[j].Location+out[j].ApplicationID
	})
	return out
}

func CSV(s Snapshot) ([]byte, error) {
	var out bytes.Buffer
	w := csv.NewWriter(&out)
	_ = w.Write([]string{"Code", "Item", "System", "Location", "Surface", "Work action", "Inclusion", "Selection", "Scheme", "Option group", "Specification", "Relationships", "Pending decisions", "Evidence", "Quantity", "Unit", "Item review"})
	for _, c := range s.Coverage {
		cells := make([]string, 17)
		cells[1] = "Specification coverage"
		cells[2] = c.SystemID
		cells[3] = c.Extent
		cells[10] = CoverageText(c)
		cells[16] = c.Mode
		for i, v := range cells {
			cells[i] = safeCSV(v)
		}
		if err := w.Write(cells); err != nil {
			return nil, err
		}
	}
	for _, r := range Assemble(s) {
		values := []string{}
		refs := []string{}
		if r.Authority == "external" {
			values = append(values, "Maintained externally; confirmed source readings only. Refer to the governing document.")
		}
		if r.Description != "" {
			values = append(values, r.Description)
		}
		for _, f := range r.Fields {
			values = append(values, fmt.Sprintf("%s (%s): %s", f.Label, f.Meaning, f.Value))
			if f.Source.DocumentID != "" {
				ref := fmt.Sprintf("%s: document %s, page %d, %s; sha256 %s", f.Label, f.Source.DocumentID, f.Source.Page, f.Source.Location, f.Source.Hash)
				if f.Source.Unavailable {
					ref += " (source unavailable)"
				}
				if f.Source.Stale {
					ref += " (revision review required)"
				}
				refs = append(refs, ref)
			}
		}
		cells := []string{r.Code, r.Title, r.SystemID, r.Location, r.Surface, r.Action, r.Inclusion, r.Selection, r.Scheme, r.OptionGroup, strings.Join(values, "\n"), strings.Join(r.Relationships, "\n"), strconv.Itoa(r.Pending), strings.Join(refs, "\n")}
		quantity := ""
		if r.Quantity != nil {
			quantity = strconv.FormatFloat(*r.Quantity, 'f', -1, 64)
		}
		cells = append(cells, quantity, r.Unit, r.ReviewStatus)
		for i, c := range cells {
			cells[i] = safeCSV(c)
		}
		if err := w.Write(cells); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return out.Bytes(), w.Error()
}

func safeCSV(c string) string {
	v := strings.TrimLeft(c, " \t\r\n")
	if len(v) > 0 && strings.ContainsAny(v[:1], "=+-@") {
		return "'" + c
	}
	return c
}

func CoverageText(c Coverage) string {
	lines := []string{"Coverage: " + strings.ReplaceAll(c.Mode, "_", " "), "Extent: " + c.Extent, "Maintained by: " + c.Maintainer, "Reviewed by: " + c.Reviewer}
	if c.Exclusions != "" {
		lines = append(lines, "Gaps / exclusions: "+c.Exclusions)
	}
	for _, r := range c.References {
		line := fmt.Sprintf("%s; %s; revision %s; %s; document %s; sha256 %s", r.Filename, r.Number, r.Revision, r.Location, r.DocumentID, r.Hash)
		if r.Unavailable {
			line += "; SOURCE UNAVAILABLE"
		}
		if r.Stale {
			line += "; REVISION REVIEW REQUIRED"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
