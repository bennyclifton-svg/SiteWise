package rfp

import (
	"strconv"
	"strings"

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
	// Works is a trade RFT: the same sheets and keys, in tender words.
	Works bool
}

// ResponseForm is the returnable schedule as a workbook a firm fills in: the
// fee rows of section 3 and the R3 roles keyed by schedule item, then one
// table for each non-price schedule.
func ResponseForm(h FormHeader, items []ScheduleItem, returnables []Row) []xlsx.Sheet {
	fees := feeSheetHeader(h)
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
		fees.Rows = append(fees.Rows, []xlsx.Cell{formText(item.Key), formText(item.Label), {}, {}})
	}
	last := len(fees.Rows)
	total := "Base fee total"
	if h.Works {
		total = "Tender price total"
	}
	fees.Rows = append(fees.Rows, []xlsx.Cell{formBold("base_total"), formBold(total), sum("C", first, last), {}})
	appendOptions(&fees, options)
	return formSheets(fees, items, returnables, h.Works)
}

func formText(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
func formBold(s string) xlsx.Cell { return xlsx.Cell{Text: s, Bold: true} }

// feeSheetHeader is the top of the R1 fee sheet, down to its column titles.
func feeSheetHeader(h FormHeader) xlsx.Sheet {
	bold, text := formBold, formText
	// Fee only: weeks and hours per item cost a firm effort the comparison
	// does not use (owner direction, 2 November 2026). R7 holds durations.
	fees := xlsx.Sheet{Name: "R1 Fees", Widths: []float64{22, 60, 18, 40}}
	fees.Rows = append(fees.Rows, []xlsx.Cell{bold(h.Title + " — response form")}, []xlsx.Cell{text(h.Project)})
	doc, amount, note := "RFP", "Fee $ excl. GST", "Fees exclude GST."
	if h.Works {
		doc, amount, note = "RFT", "Price $ excl. GST", "Prices exclude GST."
	}
	if h.Ref != "" {
		fees.Rows = append(fees.Rows, []xlsx.Cell{text(doc + " reference " + h.Ref)})
	}
	fees.Rows = append(fees.Rows, nil)
	for _, f := range FormFirmFields {
		fees.Rows = append(fees.Rows, []xlsx.Cell{bold(f), {}})
	}
	fees.Rows = append(fees.Rows, nil, []xlsx.Cell{text(note + " Enter \"Not priced\" or \"Included in <ref>\" instead of an amount where that applies.")}, nil)
	fees.Rows = append(fees.Rows, []xlsx.Cell{bold(FormRef), bold("Item"), bold(amount), bold("Notes")})
	fees.TitleRow = len(fees.Rows)
	return fees
}

// appendOptions lists the options below the base fee, with their own total.
func appendOptions(fees *xlsx.Sheet, options []ScheduleItem) {
	if len(options) == 0 {
		return
	}
	first := len(fees.Rows) + 1
	for _, item := range options {
		fees.Rows = append(fees.Rows, []xlsx.Cell{formText(item.Key), formText("Option: " + item.Label), {}, {}})
	}
	fees.Rows = append(fees.Rows, []xlsx.Cell{formBold("options_total"), formBold("Options total"), sum("C", first, len(fees.Rows)), {}})
}

// GroupResponseForm is a combined engagement's response form: a sum for
// each discipline and the engagement total (owner decision 2). Each key is
// the discipline's key and the item's own, so a returned form reads back
// discipline by discipline; a stage priced as one amount is entered once
// and marked "Included in" against the others.
func GroupResponseForm(h FormHeader, fees []FeeSection, returnables []Row) []xlsx.Sheet {
	sheet := feeSheetHeader(h)
	sheet.Rows = append(sheet.Rows[:sheet.TitleRow-1], []xlsx.Cell{formText(CombinedStageNote)}, nil, sheet.Rows[sheet.TitleRow-1])
	sheet.TitleRow = len(sheet.Rows)
	sums := []string{}
	var options, all []ScheduleItem
	for _, f := range fees {
		sheet.Rows = append(sheet.Rows, []xlsx.Cell{{}, formBold(f.Title), {}, {}})
		first := len(sheet.Rows) + 1
		for _, item := range f.Items {
			item.Key, item.Label = f.Key+":"+item.Key, f.Title+" — "+item.Label
			all = append(all, item)
			if !item.Base {
				options = append(options, item)
				continue
			}
			sheet.Rows = append(sheet.Rows, []xlsx.Cell{formText(item.Key), formText(item.Label), {}, {}})
		}
		sheet.Rows = append(sheet.Rows, []xlsx.Cell{formBold("sum:" + f.Key), formBold("Sum for " + f.Title), sum("C", first, len(sheet.Rows)), {}})
		sums = append(sums, "C"+strconv.Itoa(len(sheet.Rows)))
		for _, rate := range f.Rates {
			rate.Key, rate.Label = f.Key+":"+rate.Key, f.Title+" — "+rate.Label
			all = append(all, rate)
		}
	}
	total := xlsx.Cell{Bold: true}
	if len(sums) > 0 {
		total.Formula = "SUM(" + strings.Join(sums, ",") + ")"
	}
	sheet.Rows = append(sheet.Rows, []xlsx.Cell{formBold("engagement_total"), formBold("Engagement total"), total, {}})
	appendOptions(&sheet, options)
	return formSheets(sheet, all, returnables, false)
}

// formSheets follows the fee sheet with the rate sheet and one table for
// each non-price schedule.
func formSheets(fees xlsx.Sheet, items []ScheduleItem, returnables []Row, works bool) []xlsx.Sheet {
	bold, text := formBold, formText
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
	insurance := [][]xlsx.Cell{}
	if works {
		// A trade RFT asks for the tender schedules (skeleton-works.yaml).
		table("r2", "R2 Provisional sums", []string{"Item", "Provisional sum or prime cost item", "Amount $ excl. GST", "Margin on adjustment %"}, append([][]xlsx.Cell{{text("Nil, or list each item")}}, blank(8, 4)...))
		table("r4", "R4 Team", []string{"Role", "Name", "Time on this project"}, blank(6, 3))
		table("r5", "R5 Experience", []string{"Project", "Client", "Value $", "Year", "Your role", "Referee and phone"}, blank(3, 6))
		table("r6", "R6 Departures", []string{"Clause or item", "Proposed change", "Effect on price", "Effect on time"}, append([][]xlsx.Cell{{text("Nil, or list each departure")}}, blank(6, 4)...))
		table("r7", "R7 Program", R7Columns, stages)
		for _, row := range []string{"Contractor licence", "Public liability", "Contract works insurance", "Workers compensation", "Safety system", "Quality system"} {
			insurance = append(insurance, []xlsx.Cell{text(row), {}, {}, {}})
		}
		table("r8", "R8 Insurance", []string{"Item", "Insurer, licence or detail", "Amount $", "Expiry"}, insurance)
		table("r9", "R9 Subcontractors", []string{"Trade", "Subcontractor or supplier", "Licence"}, blank(8, 3))
		return sheets
	}
	table("r2", "R2 Deliverables", []string{"Deliverable", "Format", "Stage", "Review rounds", "Recipient", "Editable files and reliance"}, blank(10, 6))
	table("r4", "R4 Team", []string{"Role", "Name", "% time by stage", "Subconsultant firm"}, blank(8, 4))
	table("r5", "R5 Experience", []string{"Project", "Client", "Value $", "Year", "Your role", "Referee and phone"}, blank(3, 6))
	table("r6", "R6 Departures", []string{"Clause or item", "Proposed change", "Effect on fee", "Effect on time"}, append([][]xlsx.Cell{{text("Nil, or list each departure")}}, blank(6, 4)...))
	table("r7", "R7 Program", R7Columns, stages)
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
