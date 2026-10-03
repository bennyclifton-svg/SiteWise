package identity

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

func extractDOCX(ctx context.Context, r io.ReaderAt, size int64, limits Limits) (Text, error) {
	zr, err := openPackage(r, size, limits.MaxBytes)
	if err != nil {
		return Text{}, err
	}
	member := zipMember(zr, "word/document.xml")
	if member == nil {
		return Text{}, ErrMalformed
	}
	rc, br, err := openMember(ctx, member, limits.MaxBytes)
	if err != nil {
		return Text{}, err
	}
	defer rc.Close()
	runs, err := docxRuns(ctx, br, limits)
	if err != nil {
		return Text{}, err
	}
	return Text{Runs: runs}, nil
}

func docxRuns(ctx context.Context, r io.Reader, limits Limits) ([]Run, error) {
	dec := newXML(r)
	var (
		runs       []Run
		tableDepth int
		tableIndex int
		row        int
		col        int
		inCell     bool
		inText     bool
		heading    bool
		para       strings.Builder
		cell       strings.Builder
		tokens     int
	)
	flushPara := func() {
		text := trimKept(para.String())
		para.Reset()
		if text == "" {
			heading = false
			return
		}
		if inCell && tableDepth == 1 {
			if cell.Len() > 0 {
				cell.WriteByte(' ')
			}
			cell.WriteString(text)
			heading = false
			return
		}
		if len(runs) < limits.MaxRuns && tableDepth == 0 {
			runs = append(runs, Run{Text: text, Source: Source{Heading: heading}})
		}
		heading = false
	}
	flushCell := func() {
		text := trimKept(cell.String())
		cell.Reset()
		if text == "" || tableDepth != 1 {
			return
		}
		if row > limits.MaxRows || col > limits.MaxCols || len(runs) >= limits.MaxRuns {
			return
		}
		runs = append(runs, Run{Text: text, Source: Source{
			Table: tableIndex,
			Row:   row,
			Col:   col,
		}})
	}
	for {
		tokens++
		if tokens%32 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, malformed(err)
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "tbl":
				if limits.RequireComplete && tableDepth > 0 {
					return nil, fmt.Errorf("%w: nested table requires source review", ErrMalformed)
				}
				flushPara()
				if tableDepth == 0 {
					tableIndex++
					row = 0
				}
				tableDepth++
			case "tr":
				if tableDepth == 1 {
					row++
					col = 0
				}
			case "tc":
				if tableDepth == 1 {
					col++
					inCell = true
					cell.Reset()
				}
			case "p":
				para.Reset()
				heading = false
			case "pStyle":
				if styleIsHeading(el) {
					heading = true
				}
			case "t":
				inText = true
			case "tab":
				para.WriteByte(' ')
			}
		case xml.EndElement:
			switch el.Name.Local {
			case "t":
				inText = false
			case "p":
				flushPara()
			case "tc":
				if tableDepth == 1 && inCell {
					flushPara()
					flushCell()
					inCell = false
				}
			case "tbl":
				if tableDepth > 0 {
					tableDepth--
				}
			}
		case xml.CharData:
			if inText {
				para.Write(el)
			}
		}
		if len(runs) >= limits.MaxRuns && tableDepth == 0 {
			break
		}
	}
	return runs, nil
}

func styleIsHeading(el xml.StartElement) bool {
	for _, attr := range el.Attr {
		if attr.Name.Local != "val" {
			continue
		}
		return strings.HasPrefix(strings.ToLower(attr.Value), "heading")
	}
	return false
}
