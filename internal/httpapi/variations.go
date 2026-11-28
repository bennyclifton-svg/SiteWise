package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"sitewise/internal/store"
)

// Variations and progress claims on the Cost Plan's "Variations and claims"
// tab (docs/plans/2026-11-17-variations-progress-claims.md). Every write
// answers with the ledger, which carries the tab, so the screen shows Go's
// figures without a second read. No route here calls Jev or the model.

// Budget paths (bench/budgets.json).
const (
	pathVariationWrite = "variation_write"
	pathVariationsRead = "variations_read"
)

func postVariation(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.VariationInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil || !uuidPattern.MatchString(in.Package) {
		http.Error(w, "invalid body: package is the contract's package id", http.StatusBadRequest)
		return
	}
	if _, err := deps.Store.AddVariation(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, in, time.Now()); err != nil {
		variationError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

func patchVariation(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !uuidPattern.MatchString(r.PathValue("v")) {
		http.NotFound(w, r)
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.VariationInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if _, err := deps.Store.EditVariation(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("v"), in, time.Now()); err != nil {
		variationError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

func variationError(w http.ResponseWriter, deps Deps, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidVariation):
		msg := strings.TrimPrefix(err.Error(), store.ErrInvalidVariation.Error()+": ")
		http.Error(w, strings.ToUpper(msg[:1])+msg[1:]+".", http.StatusUnprocessableEntity)
	case errors.Is(err, store.ErrVariationNumber):
		http.Error(w, "That variation number is already used on this contract.", http.StatusConflict)
	case errors.Is(err, store.ErrVersionConflict):
		http.Error(w, "This variation changed while it was being edited. Read it again.", http.StatusConflict)
	default:
		invoiceError(w, deps, err)
	}
}

// The tab's JSON, riding on the ledger.

type contractLineJSON struct {
	ID     string     `json:"id"`
	Label  string     `json:"label"`
	Amount moneyField `json:"amount"`
}

type contractJSON struct {
	AwardID       string             `json:"award_id"`
	PackageID     string             `json:"package_id"`
	Title         string             `json:"title"`
	Firm          string             `json:"firm"`
	Claims        bool               `json:"claims"`
	Lines         []contractLineJSON `json:"lines"`
	Sum           moneyField         `json:"contract_sum"`
	Approved      moneyField         `json:"approved_variations"`
	SumToDate     moneyField         `json:"contract_sum_to_date"`
	Certified     moneyField         `json:"certified_to_date"`
	RetentionHeld moneyField         `json:"retention_held"`
	// Adjusted is the sum to date with provisional sums marked final taken
	// at what was completed; Paid is the payment list's total ex GST.
	Provisional moneyField       `json:"provisional_adjustment"`
	Adjusted    moneyField       `json:"adjusted"`
	Paid        moneyField       `json:"paid_to_date"`
	Latest      *scheduleRefJSON `json:"latest_schedule"`
	// Breakdown is what the contract's payment schedules print; gaps are
	// the cost lines it does not add up to.
	Breakdown []breakdownLineJSON `json:"breakdown"`
	Gaps      []breakdownGapJSON  `json:"breakdown_gaps"`
	// What a schedule prints of the contract: its date, and its retention
	// as a percent of a base. "" is not entered.
	ContractDate     string `json:"contract_date"`
	RetentionPercent string `json:"retention_percent"`
	RetentionBase    string `json:"retention_base"`
	// Services: a consultant's contract. ActOn: its schedules name the Act;
	// ActApplies is the PM's switch ("yes", "no", "" for the default).
	Services      bool   `json:"services"`
	ActOn         bool   `json:"act_on"`
	ActApplies    string `json:"act_applies"`
	FirmABN       string `json:"firm_abn"`
	ScheduleDays  *int   `json:"schedule_days"`
	ScheduleBasis string `json:"schedule_basis"`
	PaymentDays   *int   `json:"payment_days"`
	PaymentBasis  string `json:"payment_basis"`
}

type variationJSON struct {
	ID            string     `json:"id"`
	PackageID     string     `json:"package_id"`
	LineID        string     `json:"line_id"`
	Line          string     `json:"line"`
	Number        string     `json:"number"`
	Title         string     `json:"title"`
	Details       string     `json:"details"`
	Status        string     `json:"status"`
	Claimed       moneyField `json:"claimed"`
	Estimate      moneyField `json:"estimate"`
	Approved      moneyField `json:"approved"`
	Forecast      moneyField `json:"forecast"`
	ForecastBasis string     `json:"forecast_basis"`
	TimeDays      *int       `json:"time_claimed_days"`
	SubmittedOn   string     `json:"submitted_on"`
	DecidedOn     string     `json:"decided_on"`
	Documents     []docRef   `json:"documents"`
	Version       int64      `json:"version"`
}

type docRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func tabBody(v store.LedgerView) ([]contractJSON, []variationJSON) {
	contracts := []contractJSON{}
	lineLabel := map[string]string{}
	summaries := store.ContractSummaries(v)
	for i, c := range v.Contracts {
		j := contractJSON{AwardID: c.AwardID, PackageID: c.PackageID, Title: c.Title, Firm: c.Firm, Claims: c.Claims(), Lines: []contractLineJSON{},
			ScheduleDays: c.ScheduleDays, ScheduleBasis: c.ScheduleBasis, PaymentDays: c.PaymentDays, PaymentBasis: c.PaymentBasis,
			ContractDate: c.ContractDate, RetentionPercent: ratePercent(c.RetentionBP), RetentionBase: c.RetentionBase, FirmABN: c.FirmABN,
			Services: c.Services(), ActOn: c.ActOn(), ActApplies: yesNoPtr(c.ActApplies)}
		for _, l := range c.Lines {
			j.Lines = append(j.Lines, contractLineJSON{ID: l.ID, Label: l.Label, Amount: cents(l.Amount)})
			lineLabel[l.ID] = l.Label
		}
		s := summaries[i]
		j.Sum, j.Approved, j.SumToDate = cents(s.Original), centsValue(s.Variations), cents(s.TotalLet)
		j.Certified, j.RetentionHeld = centsValue(s.Certified), centsValue(s.Retention)
		j.Provisional, j.Adjusted, j.Paid, j.Latest = centsValue(s.Provisional), cents(s.Adjusted), centsValue(s.Paid), scheduleRefBody(s.Latest)
		j.Breakdown, j.Gaps = breakdownBody(c)
		contracts = append(contracts, j)
	}
	vars := []variationJSON{}
	for _, va := range v.Variations {
		f, basis := va.Forecast()
		j := variationJSON{ID: va.ID, PackageID: va.PackageID, LineID: va.LineID, Line: lineLabel[va.LineID], Number: va.Number, Title: va.Title,
			Details: va.Details, Status: va.Status, Claimed: cents(va.Claimed), Estimate: cents(va.Estimate), Approved: cents(va.Approved),
			Forecast: cents(f), ForecastBasis: basis, TimeDays: va.TimeDays, SubmittedOn: va.SubmittedOn, DecidedOn: va.DecidedOn,
			Documents: []docRef{}, Version: va.Version}
		for i, d := range va.Documents {
			name := ""
			if i < len(va.DocumentNames) {
				name = va.DocumentNames[i]
			}
			j.Documents = append(j.Documents, docRef{ID: d, Name: name})
		}
		vars = append(vars, j)
	}
	return contracts, vars
}

func yesNoPtr(b *bool) string {
	switch {
	case b == nil:
		return ""
	case *b:
		return "yes"
	}
	return "no"
}
