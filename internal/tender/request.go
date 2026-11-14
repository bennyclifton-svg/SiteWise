package tender

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"sitewise/internal/llm"
)

const (
	pageTextClip = 7000
	passTokens   = 40000
	passTimeout  = 120 * time.Second
)

// Input is everything one pass sends. It holds no user, org or project.
type Input struct {
	Model  string
	Pages  []Page
	Sel    Selection
	Images map[int]string // page number -> base64 JPEG
	Rows   []Row
}

func BuildRequest(in Input) llm.Request {
	byNumber := map[int]Page{}
	for _, p := range in.Pages {
		byNumber[p.Number] = p
	}
	image := map[int]bool{}
	for _, n := range in.Sel.Images {
		image[n] = true
	}
	numbers := append(append([]int{}, in.Sel.Images...), in.Sel.Texts...)
	sort.Ints(numbers)
	var parts []llm.Part
	for _, n := range numbers {
		p := byNumber[n]
		parts = append(parts, llm.Part{Text: fmt.Sprintf("=== Page %d of %d ===", n, len(in.Pages))})
		if image[n] && in.Images[n] != "" {
			parts = append(parts, llm.Part{ImageJPEG: in.Images[n]})
		}
		if p.Text != "" {
			head := fmt.Sprintf("Text of page %d:\n", n)
			if p.OCR {
				head = fmt.Sprintf("Text of page %d (read by OCR):\n", n)
			}
			parts = append(parts, llm.Part{Text: head + clip(p.Text, pageTextClip)})
		}
	}
	rows, _ := json.Marshal(in.Rows)
	parts = append(parts, llm.Part{Text: "The schedule rows to map each line to:\n" + string(rows) + "\n\nList the prices in this submission."})
	return llm.Request{Name: "tender_pass", Model: in.Model, System: Prompt, Parts: parts, Schema: Schema(), MaxTokens: passTokens, Timeout: passTimeout}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
