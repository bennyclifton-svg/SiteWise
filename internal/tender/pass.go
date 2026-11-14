package tender

import (
	"context"
	"errors"
	"time"

	"sitewise/internal/llm"
)

// ErrNoRenderer: page images cannot be made on this server, and the trial
// showed the pass is not good enough without them.
var ErrNoRenderer = errors.New("tender: page rendering is not set up")

// ErrEmpty: the model listed nothing though pages with prices were sent.
// It happens now and then on a first try (seen in the corpus survey, 14
// November 2026), so the caller asks again instead of storing an empty read.
var ErrEmpty = errors.New("tender: the model listed no prices")

// Renderer makes JPEGs (base64) of the listed pages of the file at path.
type Renderer func(ctx context.Context, path string, pages []int) (map[int]string, error)

// PassInfo is what the pass cost and how it ended.
type PassInfo struct {
	Sel       Selection
	Stop      string
	Cut       bool
	Dropped   int
	InTokens  int64
	OutTokens int64
	Latency   time.Duration
}

// Pass lists one file's prices. paged is false for a file with no pages
// (Word, Excel, Markdown): then only text is sent.
func Pass(ctx context.Context, ask llm.Asker, render Renderer, model, path string, paged bool, pages []Page, rows []Row) (RawReading, PassInfo, error) {
	info := PassInfo{}
	in := Input{Model: model, Pages: pages, Rows: rows}
	if paged {
		in.Sel = SelectPages(pages)
		if len(in.Sel.Images) > 0 {
			if render == nil {
				return RawReading{}, info, ErrNoRenderer
			}
			images, err := render(ctx, path, in.Sel.Images)
			if err != nil || len(images) == 0 {
				return RawReading{}, info, ErrNoRenderer
			}
			in.Images = images
		}
	} else {
		for _, p := range pages {
			in.Sel.Texts = append(in.Sel.Texts, p.Number)
		}
	}
	info.Sel = in.Sel
	res, err := ask.Ask(ctx, BuildRequest(in))
	info.Stop, info.InTokens, info.OutTokens, info.Latency = res.StopReason, res.InTokens, res.OutTokens, res.Latency
	if err != nil {
		return RawReading{}, info, err
	}
	info.Cut = res.StopReason == "max_tokens"
	raw, dropped, err := ParseAnswer(res.Text, info.Cut)
	info.Dropped = dropped
	if err == nil && len(raw.Lines) == 0 && len(in.Sel.Images) > 0 && !info.Cut {
		return raw, info, ErrEmpty
	}
	return raw, info, err
}
