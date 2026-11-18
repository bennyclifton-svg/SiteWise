// Package chatfix files synthetic documents straight into the store, for
// the chat tests and the command key's fixtures. Nothing here is real
// project data, and no Jev or model is called.
package chatfix

import (
	"context"
	"crypto/rand"
	"fmt"

	"sitewise/internal/store"
)

// Document is one synthetic filed document. Pages holds the text of each
// page; a document with no pages is a scan with no searchable text.
type Document struct {
	Title      string   `json:"title"`
	Kind       string   `json:"kind"`
	Discipline string   `json:"discipline,omitempty"`
	Number     string   `json:"number,omitempty"`
	Revision   string   `json:"revision,omitempty"`
	Supersedes string   `json:"supersedes,omitempty"` // the title of the document this one replaces
	Pages      []string `json:"pages"`
	// Systems label every passage with these building systems, as the
	// profile's reading would.
	Systems []string `json:"systems,omitempty"`
}

// SeedDocuments files the documents in order and returns each one's id by
// title. A page is split into passages at blank lines, each keeping its
// page number.
func SeedDocuments(ctx context.Context, st *store.Store, org, project string, docs []Document) (map[string]string, error) {
	ids := map[string]string{}
	for _, d := range docs {
		sum := make([]byte, 32)
		_, _ = rand.Read(sum)
		// The same path a dropped file takes, without the bytes or Jev.
		up, err := st.CommitIntake(ctx, org, store.CommitIntake{ProjectID: project, SHA256: sum, ByteSize: 1000, MediaType: "application/pdf", Filename: d.Title + ".pdf"})
		if err != nil {
			return nil, fmt.Errorf("%s: intake: %w", d.Title, err)
		}
		docID := up.DocumentID
		decisions := []store.DecisionWrite{{Field: "title", Value: d.Title, Band: "green", DecidedBy: "rule"}}
		for field, value := range map[string]string{"kind": d.Kind, "discipline": d.Discipline, "number": d.Number, "revision": d.Revision} {
			if value != "" {
				decisions = append(decisions, store.DecisionWrite{Field: field, Value: value, Band: "green", DecidedBy: "rule"})
			}
		}
		if _, err := st.CommitFiling(ctx, org, docID, store.CommitFiling{PDFPages: len(d.Pages), Decisions: decisions, PriorID: ids[d.Supersedes]}); err != nil {
			return nil, fmt.Errorf("%s: filing: %w", d.Title, err)
		}
		if len(d.Pages) > 0 {
			src := store.DocumentSource{}
			for i, page := range d.Pages {
				src.Source = append(src.Source, store.SourcePage{Text: page})
				for _, para := range paragraphs(page) {
					src.Units = append(src.Units, store.SourceUnit{Body: para, Page: i + 1})
				}
			}
			if err := st.ReplaceSource(ctx, org, docID, src); err != nil {
				return nil, fmt.Errorf("%s: text: %w", d.Title, err)
			}
			if err := label(ctx, st, org, project, docID, d.Systems); err != nil {
				return nil, fmt.Errorf("%s: labels: %w", d.Title, err)
			}
		}
		ids[d.Title] = docID
	}
	return ids, nil
}

func label(ctx context.Context, st *store.Store, org, project, docID string, systems []string) error {
	if len(systems) == 0 {
		return nil
	}
	passages, err := st.ChatDocumentPassages(ctx, org, project, docID, 0, 0)
	if err != nil {
		return err
	}
	for _, p := range passages {
		if err := st.SetPassageSystems(ctx, org, p.ID, systems); err != nil {
			return err
		}
	}
	return nil
}

func paragraphs(page string) []string {
	var out []string
	start := 0
	for i := 0; i+1 < len(page); i++ {
		if page[i] == '\n' && page[i+1] == '\n' {
			if s := page[start:i]; len(s) > 0 {
				out = append(out, s)
			}
			start = i + 2
		}
	}
	if s := page[start:]; len(s) > 0 {
		out = append(out, s)
	}
	return out
}
