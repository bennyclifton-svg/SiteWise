package tender

import (
	"regexp"
	"sort"
	"strings"
)

// Selection is which pages go to the model, by page number, in page order.
type Selection struct {
	Images []int // sent as the page image and its text
	Texts  []int // sent as text only
}

const (
	MaxImagePages = 30
	MaxTextPages  = 20
	TextClip      = 5000 // characters of a text-only page
	tokenCeiling  = 95000
	// Measured in the trial: a page sent as image and text costs about this.
	tokensPerImagePage = 2900
)

var (
	moneyLine    = regexp.MustCompile(`\$\s?\d[\d,]*(?:\.\d\d)?|\b\d{1,3}(?:,\d{3})+(?:\.\d\d)?\b`)
	priceWords   = regexp.MustCompile(`(?i)tender (sum|price)|lump sum|contract sum|fee (proposal|schedule|summary|table|breakdown)|schedule of (rates|prices|fees)|price schedule|returnable|sub-?\s?total|\btotal\b|provisional sum|prime cost|pc sum|quotation|quote summary|remuneration|professional fee|hourly rate|costings`)
	contextWords = regexp.MustCompile(`(?i)exclusion|excluded|qualification|clarification|assumption|not included|no allowance|departure|\boption|alternative|value management|cost saving`)
	noiseWords   = regexp.MustCompile(`(?i)certificate of currency|policy (no|number)|sum insured|the insured|underwriter|limit of (liability|indemnity)`)
)

type scored struct {
	number, score int
}

// SelectPages decides, by rule, which pages the model sees. No model is asked.
func SelectPages(pages []Page) Selection {
	var cands []scored
	noise := map[int]bool{}
	scanned := 0
	for _, p := range pages {
		text := strings.TrimSpace(p.Text)
		if len(text) < 40 {
			scanned++
			cands = append(cands, scored{p.Number, 0})
			continue
		}
		if noiseWords.MatchString(text) {
			noise[p.Number] = true
			continue
		}
		money := 0
		for _, l := range strings.Split(text, "\n") {
			if moneyLine.MatchString(l) {
				money++
			}
		}
		score := money
		if priceWords.MatchString(text) {
			score += 3
		}
		if (money >= 3 && score >= 4) || (money >= 1 && score >= 5) {
			cands = append(cands, scored{p.Number, score})
		}
	}
	if scanned == len(pages) {
		cands = cands[:min(len(cands), MaxImagePages)]
	} else {
		sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
		cands = cands[:min(len(cands), MaxImagePages)]
	}
	image := map[int]bool{}
	for _, c := range cands {
		image[c.number] = true
	}
	var sel Selection
	for _, p := range pages {
		switch {
		case image[p.Number]:
			sel.Images = append(sel.Images, p.Number)
		case noise[p.Number] || strings.TrimSpace(p.Text) == "":
		case len(sel.Texts) < MaxTextPages && (p.Number <= 2 || len(pages) <= 12 || contextWords.MatchString(p.Text)):
			sel.Texts = append(sel.Texts, p.Number)
		}
	}
	// Shed the weakest image pages until the request fits.
	for EstimateTokens(pages, sel) > tokenCeiling && len(sel.Images) > 1 {
		weakest := cands[len(cands)-1].number
		cands = cands[:len(cands)-1]
		sel.Images = without(sel.Images, weakest)
	}
	return sel
}

// EstimateTokens is a deliberately high guess of the request's size.
func EstimateTokens(pages []Page, sel Selection) int {
	byNumber := map[int]Page{}
	for _, p := range pages {
		byNumber[p.Number] = p
	}
	n := len(sel.Images) * tokensPerImagePage
	for _, t := range sel.Texts {
		n += min(len(byNumber[t].Text), TextClip) * 10 / 35
	}
	return n
}

func without(list []int, n int) []int {
	out := list[:0]
	for _, v := range list {
		if v != n {
			out = append(out, v)
		}
	}
	return out
}
