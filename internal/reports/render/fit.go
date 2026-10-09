package render

import (
	"errors"

	"sitewise/internal/reports"
)

// PageFit tells the draft screen how the issued PDF would paginate, using the
// same renderer as issue so the count cannot disagree with the export gate.
type PageFit struct {
	Pages  int     `json:"pages"`
	Limit  int     `json:"limit"`
	Target float64 `json:"target"`
	Fits   bool    `json:"fits"`
	Error  string  `json:"error,omitempty"`
}

func Fit(s reports.IssueSnapshot) PageFit {
	limit, target := PageLimit(s)
	fit := PageFit{Limit: limit, Target: target}
	r, err := PDF(s)
	fit.Pages = r.Pages
	switch {
	case err == nil:
		fit.Fits = true
	case errors.Is(err, ErrOverflow):
	default:
		fit.Error = err.Error()
	}
	return fit
}
