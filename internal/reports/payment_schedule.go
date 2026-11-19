package reports

// KindPaymentSchedule is a progress claim's payment schedule
// (docs/plans/2026-11-17-variations-progress-claims.md, Task 9): one report
// per claim, issued and frozen, never a package's RFP or RFT.
const KindPaymentSchedule = "payment_schedule"

// ScheduleRow is one row of the schedule as printed: amounts already
// formatted by Go from cents.
type ScheduleRow struct {
	Label, Claimed, Scheduled, Reason string
}

// ScheduleInput is everything a payment schedule prints. It reads no
// template and no profile: the store passes what it holds.
type ScheduleInput struct {
	Project, Client, Firm, Contract string
	ClaimNumber, ClaimDate, Served  string
	ReferenceDate                   string
	// Act names the Act; Section is printed only from a verified row.
	Act, Section                        string
	Rows                                []ScheduleRow
	Gross, Retention, Net, GST, Payable string
	IssuedBy, IssuedOn, DueOn           string
	Late                                bool
}

// PaymentScheduleSections are the schedule's sections, in order: the
// parties, the claim it answers, the Act, the table, the totals, the
// reasons for withholding, and who issued it when.
func PaymentScheduleSections(in ScheduleInput) []Section {
	block := func(id, label, text string) Block {
		return Block{ID: id, Label: label, Text: text, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated"}
	}
	or := func(s, missing string) string {
		if s == "" {
			return missing
		}
		return s
	}
	claim := "Payment claim " + or(in.ClaimNumber, "(no number)")
	if in.ClaimDate != "" {
		claim += " dated " + LongDate(in.ClaimDate)
	}
	claim += ", served " + LongDate(in.Served)
	if in.ReferenceDate != "" {
		claim += ", for the reference date " + LongDate(in.ReferenceDate)
	}
	act := "This is a payment schedule"
	if in.Act != "" {
		act += " under the " + in.Act
		if in.Section != "" {
			act += ", " + in.Section
		}
	}
	act += "."
	table := &Table{Columns: []string{"Item", "Claimed (ex GST)", "Scheduled (ex GST)", "Reason"}, Display: "document"}
	var reasons []string
	for _, r := range in.Rows {
		table.Rows = append(table.Rows, []string{r.Label, r.Claimed, r.Scheduled, r.Reason})
		if r.Reason != "" {
			reasons = append(reasons, r.Label+": "+r.Reason)
		}
	}
	totals := &Table{Columns: []string{"", "Amount"}, Display: "document", Rows: [][]string{
		{"Scheduled amount (ex GST)", in.Gross}, {"Retention withheld", in.Retention}, {"Net", in.Net}, {"GST", in.GST}, {"Amount the respondent proposes to pay", in.Payable}}}
	why := "Nothing is withheld."
	if len(reasons) > 0 {
		why = "The scheduled amount is less than the claimed amount for these reasons. " + joinSentences(reasons)
	}
	issued := "Issued by " + or(in.IssuedBy, "the project manager") + " on " + LongDate(in.IssuedOn) + "."
	if in.Late && in.DueOn != "" {
		issued += " It is issued after its due date of " + LongDate(in.DueOn) + "."
	}
	return []Section{
		{ID: "parties", Title: "Parties", Essential: true, Blocks: []Block{block("parties", "Parties",
			"Claimant: "+or(in.Firm, "the contractor")+". Respondent: "+or(in.Client, "the principal")+". Contract: "+or(in.Contract, "")+". Project: "+in.Project+".")}},
		{ID: "claim", Title: "The claim answered", Essential: true, Blocks: []Block{block("claim", "Claim", claim+"."), block("act", "Act", act)}},
		{ID: "schedule", Title: "Schedule", Essential: true, Blocks: []Block{{ID: "rows", Label: "Rows", Table: table, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated"},
			{ID: "totals", Label: "Totals", Table: totals, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated"}}},
		{ID: "reasons", Title: "Reasons for withholding", Essential: true, Blocks: []Block{block("reasons", "Reasons", why)}},
		{ID: "issued", Title: "Issued", Essential: true, Blocks: []Block{block("issued", "Issued", issued)}},
	}
}

func joinSentences(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
		if n := len(p); n > 0 && p[n-1] != '.' {
			out += "."
		}
	}
	return out
}
