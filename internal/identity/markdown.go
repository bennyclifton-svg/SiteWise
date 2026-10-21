package identity

import (
	"context"
	"io"
	"strings"
	"unicode/utf8"
)

// Markdown is evidence text, never rendered HTML. Keep links, tables and code
// verbatim; reading a link or image must not fetch another resource.
func extractMarkdown(ctx context.Context, r io.ReaderAt, size int64, limits Limits) (Text, error) {
	if size == 0 {
		return Text{}, nil
	}
	raw, err := readAtMost(r, size, limits.MaxBytes)
	if err != nil {
		return Text{}, err
	}
	if !utf8.Valid(raw) || strings.ContainsRune(string(raw), 0) {
		return Text{}, ErrMalformed
	}
	body := strings.TrimPrefix(string(raw), "\ufeff")
	var out Text
	for line := range strings.SplitSeq(body, "\n") {
		if err := ctx.Err(); err != nil {
			return Text{}, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(out.Runs) == limits.MaxRuns {
			out.Incomplete = true
			break
		}
		// Recognise ATX headings without changing the source text.
		title := strings.TrimLeft(line, "#")
		level := len(line) - len(title)
		heading := level >= 1 && level <= 6 && (title == "" || title[0] == ' ' || title[0] == '\t')
		out.Runs = append(out.Runs, Run{Text: line, Source: Source{Heading: heading}})
	}
	return out, nil
}
