// Package tender reads a tender submission into priced lines and reconciles
// them to the firm's printed tender sum
// (docs/design/2026-11-14-tender-comparison.md).
package tender

// Row is one schedule row offered to the model.
type Row struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

// Page is one page of a submission as SiteWise extracted it.
type Page struct {
	Number int
	Text   string
	OCR    bool
}

// RawLine is one line exactly as the model returned it.
type RawLine struct {
	P     int    `json:"p"`
	Label string `json:"label"`
	Amt   string `json:"amt"`
	G     string `json:"g"`
	St    string `json:"st"`
	Kind  string `json:"kind"`
	Where string `json:"where"`
	Group string `json:"group"`
	Row   string `json:"row"`
	Note  string `json:"note"`
	Sure  bool   `json:"sure"`
}

type RawExclusion struct {
	P    int    `json:"p"`
	Text string `json:"text"`
}

// RawReading is the model's whole answer.
type RawReading struct {
	Firm       string         `json:"firm"`
	Basis      string         `json:"basis"`
	GST        string         `json:"gst"`
	Lines      []RawLine      `json:"lines"`
	Exclusions []RawExclusion `json:"exclusions"`
	Notes      []string       `json:"notes"`
}

// Flag is one thing to look at, in plain words written by Go.
type Flag struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

// Line is one line after Go has checked it.
type Line struct {
	ID        string
	Ordinal   int
	Page      int
	Label     string
	Printed   string
	Cents     *int64
	GST       string
	State     string
	Kind      string
	Placed    string
	Heading   string
	Row       string
	Note      string
	Counted   bool
	Flags     []Flag
	DecidedBy string
	SplitOf   string
}

// Reading is one submission after reconciling.
type Reading struct {
	Firm       string
	Basis      string
	GST        string
	TenderSum  *int64
	SumHow     string
	Lines      []Line
	Exclusions []RawExclusion
	Notes      []string
	Flags      []Flag
	// Gap is the tender sum less every counted line: what the lines do not
	// explain. Remainder is the system row's cell: Gap plus the counted
	// lines that have no row.
	Gap       int64
	Remainder int64
	Why       string // "", "rounding", "margin:15", "unbroken", "differs"
}

// Line and column flag codes.
const (
	FlagAmountNotFound = "amount_not_found"
	FlagFromScan       = "from_scan"
	FlagRowUnsure      = "row_unsure"
	FlagRowUnknown     = "row_unknown"
	FlagCountedToMatch = "counted_to_match"
	FlagBreakdown      = "treated_as_breakdown"
	FlagRepeat         = "repeat"

	FlagNotFixedPrice  = "not_fixed_price"
	FlagGSTUnknown     = "gst_unknown"
	FlagNoTotalPrinted = "no_total_printed"
	FlagCheckTenderSum = "check_tender_sum"
	FlagLinesShort     = "lines_short"
	FlagLinesOver      = "lines_over"
	FlagGroupDiffers   = "group_differs"
	FlagSummaryCounted = "summary_counted"
	FlagOptionsOffered = "options_offered"
	FlagIncomplete     = "reading_incomplete"
)

// States of a line.
const (
	StatePriced    = "priced"
	StateIncluded  = "included"
	StateExcluded  = "excluded"
	StateNotPriced = "not_priced"
)

// SystemRow is the schedule key of the "Not broken down" row.
const SystemRow = "tender:not_broken_down"
