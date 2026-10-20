package rfp

import (
	"strconv"

	"sitewise/internal/xlsx"
)

// FormRef is the header of column A on the fee and rate sheets. A returned
// form is read by these references, item by item, with no judgement.
const FormRef = "Ref"

// R7Columns is the header of the R7 Program table, cell for cell. The
// program read settles an R7 row by rule only under exactly this header
// (internal/programread, plan 10.3).
var R7Columns = []string{"Stage", "Your weeks", "Authority weeks", "Inputs you need, and by when"}

// FormFirmFields are the labelled cells at the top of the fee sheet, in order.
var FormFirmFields = []string{"Firm", "ABN", "Contact", "Email", "Phone", "Address"}

// FormHeader names the RFP the form answers.
type FormHeader struct {
	Title   string // e.g. "Town planning RFP"
	Project string
	Ref     string
}

// ResponseForm is the returnable schedule as a workbook a firm fills in: the
// fee rows of section 3 and the R3 roles keyed by schedule item, then one
// table for each non-price schedule.
func ResponseForm(h FormHeader, items []ScheduleItem, returnables []Row) []xlsx.Sheet {
	bold := func(s string) xlsx.Cell { return xlsx.Cell{Text: s, Bold: true} }
	text := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
	fees := xlsx.Sheet{Name: "R1 Fees", Widths: []float64{22, 60, 18, 10, 10, 40}}
	fees.Rows = append(fees.Rows, []xlsx.Cell{bold(h.Title + " — response form")}, []xlsx.Cell{text(h.Project)})
	if h.Ref != "" {
		fees.Rows = append(fees.Rows, []xlsx.Cell{text("RFP reference " + h.Ref)})
	}
	fees.Rows = append(fees.Rows, nil)
	for _, f := range FormFirmFields {
		fees.Rows = append(fees.Rows, []xlsx.Cell{bold(f), {}})
	}
	fees.Rows = append(fees.Rows, nil, []xlsx.Cell{text("Fees exclude GST. Enter \"Not priced\" or \"Included in <ref>\" instead of an amount where that applies.")}, nil)
	fees.Rows = append(fees.Rows, []xlsx.Cell{bold(FormRef), bold("Item"), bold("Fee $ excl. GST"), bold("Weeks"), bold("Hours"), bold("Notes")})
	fees.TitleRow = len(fees.Rows)
	first := len(fees.Rows) + 1
	var options []ScheduleItem
	for _, item := range items {
		if !item.Price() {
			continue
		}
		if !item.Base {
			options = append(options, item)
			continue
		}
		fees.Rows = append(fees.Rows, []xlsx.Cell{text(item.Key), text(item.Label), {}, {}, {}, {}})
	}
	last := len(fees.Rows)
	fees.Rows = append(fees.Rows, []xlsx.Cell{bold("base_total"), bold("Base fee total"), sum("C", first, last), {}, {}, {}})
	if len(options) > 0 {
		first = len(fees.Rows) + 1
		for _, item := range options {
			fees.Rows = append(fees.Rows, []xlsx.Cell{text(item.Key), text("Option: " + item.Label), {}, {}, {}, {}})
		}
		fees.Rows = append(fees.Rows, []xlsx.Cell{bold("options_total"), bold("Options total"), sum("C", first, len(fees.Rows)), {}, {}, {}})
	}
	rates := xlsx.Sheet{Name: "R3 Rates", Widths: []float64{28, 40, 30, 18}, TitleRow: 1}
	rates.Rows = append(rates.Rows, []xlsx.Cell{bold(FormRef), bold("Role"), bold("Name"), bold("Rate $/h excl. GST")})
	for _, item := range items {
		if item.Kind == ItemRate {
			rates.Rows = append(rates.Rows, []xlsx.Cell{text(item.Key), text(item.Label), {}, {}})
		}
	}
	sheets := []xlsx.Sheet{fees, rates}
	byKey := map[string]Row{}
	for _, r := range returnables {
		byKey[r.Key] = r
	}
	blank := func(n, cols int) [][]xlsx.Cell {
		out := make([][]xlsx.Cell, n)
		for i := range out {
			out[i] = make([]xlsx.Cell, cols)
		}
		return out
	}
	table := func(key, name string, columns []string, rows [][]xlsx.Cell) {
		s := xlsx.Sheet{Name: name}
		if r, ok := byKey[key]; ok {
			s.Rows = append(s.Rows, []xlsx.Cell{text(r.Text)}, nil)
		}
		head := []xlsx.Cell{}
		for _, c := range columns {
			head = append(head, bold(c))
			s.Widths = append(s.Widths, 28)
		}
		s.Rows = append(s.Rows, head)
		s.TitleRow = len(s.Rows)
		s.Rows = append(s.Rows, rows...)
		sheets = append(sheets, s)
	}
	var stages [][]xlsx.Cell
	for _, item := range items {
		if item.Kind == ItemStage || (item.Kind == ItemOption && len(item.Key) > 6 && item.Key[:6] == "stage:") {
			stages = append(stages, []xlsx.Cell{text(item.Label), {}, {}, {}})
		}
	}
	table("r2", "R2 Deliverables", []string{"Deliverable", "Format", "Stage", "Review rounds", "Recipient", "Editable files and reliance"}, blank(10, 6))
	table("r4", "R4 Team", []string{"Role", "Name", "% time by stage", "Subconsultant firm"}, blank(8, 4))
	table("r5", "R5 Experience", []string{"Project", "Client", "Value $", "Year", "Your role", "Referee and phone"}, blank(3, 6))
	table("r6", "R6 Departures", []string{"Clause or item", "Proposed change", "Effect on fee", "Effect on time"}, append([][]xlsx.Cell{{text("Nil, or list each departure")}}, blank(6, 4)...))
	table("r7", "R7 Program", R7Columns, stages)
	insurance := [][]xlsx.Cell{}
	for _, row := range []string{"Professional indemnity", "Public liability", "Workers compensation", "Quality system", "Safety system", "Environmental system", "Conflicts of interest"} {
		insurance = append(insurance, []xlsx.Cell{text(row), {}, {}, {}})
	}
	table("r8", "R8 Insurance", []string{"Item", "Insurer or detail", "Amount $", "Expiry"}, insurance)
	table("r9", "R9 Reimbursables", []string{"Item", "Rate, or \"Included in the fee\""}, append([][]xlsx.Cell{{text("All reimbursables"), text("Included in the fee")}}, blank(5, 2)...))
	return sheets
}

func sum(col string, first, last int) xlsx.Cell {
	if last < first {
		return xlsx.Cell{}
	}
	return xlsx.Cell{Formula: "SUM(" + col + strconv.Itoa(first) + ":" + col + strconv.Itoa(last) + ")", Bold: true}
}
