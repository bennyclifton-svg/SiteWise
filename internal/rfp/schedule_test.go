package rfp

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// The RFP's fee table and the stored schedule are the same rows: every row a
// firm prices in section 3 is a schedule price item, in the same order, and
// no price item is missing from the table.
func TestScheduleIsTheFeeTable(t *testing.T) {
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	tpl := consultantTemplate(t)
	for _, s := range loadScenarios(t) {
		ids := make([]string, 0, len(s.Expect))
		for id := range s.Expect {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			t.Run(s.Name+"/"+id, func(t *testing.T) {
				d, _ := cat.RFP(id)
				drivers := s.drivers(t, sk, d)
				r := Resolve(d, sk, drivers, s.settings(t, id, drivers))
				sections, err := Sections(r, tpl)
				if err != nil {
					t.Fatal(err)
				}
				var printed []string
				for _, sec := range sections {
					for _, b := range sec.Blocks {
						if b.ID != "rfp:stages" || b.Table == nil {
							continue
						}
						for _, row := range b.Table.Rows {
							if name := feeRowName(row); name != "" {
								printed = append(printed, name)
							}
						}
					}
				}
				var want []string
				for _, item := range Schedule(r) {
					if item.Price() {
						want = append(want, scheduleRowName(item))
					}
				}
				got := printed
				if strings.Join(got, "\n") != strings.Join(want, "\n") {
					t.Fatalf("fee table rows\n%s\nschedule price items\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
				}
			})
		}
	}
}

// feeRowName reduces a printed fee row to what the schedule calls it, or ""
// for a total and for a stage priced by element below.
func feeRowName(row []string) string {
	switch {
	case row[0] == "Base fee total" || row[0] == "Options total" || row[2] == "By element below":
		return ""
	case row[0] == "Service option":
		return "option|" + row[1]
	}
	name, _, _ := strings.Cut(row[0], " — ")
	return name
}

func scheduleRowName(item ScheduleItem) string {
	switch {
	case item.Kind == ItemReimbursable:
		return "Reimbursables"
	case strings.HasPrefix(item.Key, "option:"):
		return "option|" + item.Label
	case strings.HasPrefix(item.Key, "element:"):
		_, element, _ := strings.Cut(item.Label, ": ")
		code, _, _ := strings.Cut(element, " ")
		return code
	}
	return item.Label
}

func TestScheduleFollowsTheSelectedStages(t *testing.T) {
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	d, ok := cat.RFP("town-planning")
	if !ok {
		t.Fatal("town planning RFP missing")
	}
	var scenarioDrivers DriverSet
	for _, s := range loadScenarios(t) {
		if _, ok := s.Expect["town-planning"]; ok {
			scenarioDrivers = s.drivers(t, sk, d)
			break
		}
	}
	base := Schedule(Resolve(d, sk, scenarioDrivers, Settings{}))
	var stage ScheduleItem
	for _, item := range base {
		if item.Kind == ItemStage {
			stage = item
			break
		}
	}
	if stage.Key == "" {
		t.Fatal("no included stage")
	}
	key := strings.TrimPrefix(stage.Key, "stage:")
	raw, _ := json.Marshal("option")
	moved := Schedule(Resolve(d, sk, scenarioDrivers, Settings{"stage:" + key: {Value: raw}}))
	if got := find(moved, stage.Key); got.Kind != ItemOption || got.Base {
		t.Fatalf("stage set to option is %+v", got)
	}
	raw, _ = json.Marshal("exclude")
	gone := Schedule(Resolve(d, sk, scenarioDrivers, Settings{"stage:" + key: {Value: raw}}))
	if got := find(gone, stage.Key); got.Key != "" {
		t.Fatalf("excluded stage still priced: %+v", got)
	}
}

func TestScheduleRatesAndCriteria(t *testing.T) {
	r := Resolved{Returnables: []Row{
		{Key: "r2", Label: "R2 Deliverables list", Text: "Each deliverable."},
		{Key: "r3", Label: "R3 Hourly rates", Text: "Excl. GST. Roles: Director · Senior Planner · Graduate."},
		{Key: "r6", Label: "R6 Departures", Text: "Each departure."},
	}}
	items := Schedule(r)
	var rates, criteria []string
	for _, i := range items {
		switch i.Kind {
		case ItemRate:
			rates = append(rates, i.Key)
		case ItemCriterion:
			criteria = append(criteria, i.Key+"="+i.Label)
		}
	}
	if strings.Join(rates, ",") != "rate:director,rate:senior_planner,rate:graduate" {
		t.Fatalf("rates %v", rates)
	}
	if strings.Join(criteria, ",") != "criterion:r2=Deliverables list,criterion:r6=Departures" {
		t.Fatalf("criteria %v", criteria)
	}
	if got := Roles(Resolved{}); len(got) != len(DefaultRoles) {
		t.Fatalf("default roles %v", got)
	}
	generic := GenericSchedule(nil)
	if generic[0].Key != "stage:lump_sum" || !generic[0].Base || find(generic, ReimbursablesKey).Key == "" {
		t.Fatalf("generic schedule %+v", generic)
	}
}

func find(items []ScheduleItem, key string) ScheduleItem {
	for _, i := range items {
		if i.Key == key {
			return i
		}
	}
	return ScheduleItem{}
}
