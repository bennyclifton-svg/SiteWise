// titleblock-experiment exports existing identity/OCR candidates and filing
// requests for a local, review-only comparison. It has no database write path.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	manifest := flag.String("manifest", "", "private manifest JSON with id, filename, sha256")
	blobs := flag.String("blobs", ".tools/dev-files", "content-addressed source directory")
	output := flag.String("output", "", "private output directory")
	python := flag.String("python", "python", "existing OCR Python")
	tess := flag.String("tesseract", "tesseract", "existing Tesseract")
	data := flag.String("tessdata", ".tools/tesseract/tessdata", "existing language data")
	visual := flag.Bool("visual", false, "export production visual request as well")
	visualOnly := flag.Bool("visual-only", false, "skip the slower legacy OCR baseline")
	flag.Parse()
	if *manifest == "" || *output == "" {
		return fmt.Errorf("manifest and output required")
	}
	raw, err := os.ReadFile(*manifest)
	if err != nil {
		return err
	}
	var items []struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
		SHA256   string `json:"sha256"`
	}
	if err = json.Unmarshal(raw, &items); err != nil {
		return err
	}
	if len(items) == 0 || len(items) > 40 {
		return fmt.Errorf("scope must contain 1-40 documents")
	}
	cat, err := intake.LoadCatalog("data/intake")
	if err != nil {
		return err
	}
	ocr, err := identity.NewOCR(*tess, *data, *python)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(*output, 0700); err != nil {
		return err
	}
	for _, item := range items {
		if filepath.Base(item.ID) != item.ID || item.ID == "" || item.ID == "." {
			return fmt.Errorf("invalid id")
		}
		sum, err := hex.DecodeString(item.SHA256)
		if err != nil || len(sum) != 32 {
			return fmt.Errorf("invalid hash")
		}
		path := filepath.Join(*blobs, item.SHA256[:2], item.SHA256)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		actual := sha256.Sum256(content)
		if hex.EncodeToString(actual[:]) != item.SHA256 {
			return fmt.Errorf("hash mismatch")
		}
		results := map[string]any{"id": item.ID, "filename": item.Filename, "sha256": item.SHA256}
		if *visual {
			started := time.Now()
			f, e := os.Open(path)
			if e != nil {
				return e
			}
			limits := identity.DefaultLimits()
			limits.MaxPages, limits.MaxRuns = 3, 1200
			text, e := identity.Extract(context.Background(), "pdf", f, int64(len(content)), limits)
			f.Close()
			if e != nil {
				return e
			}
			evidence, e := ocr.Visual(context.Background(), path, false, len(text.Runs) < 5)
			if e != nil {
				results["visual"] = map[string]any{"error": e.Error()}
			} else {
				candidates, call := intake.PrepareVisualReview(item.Filename, text, evidence)
				results["visual"] = map[string]any{"candidates": candidates, "prepare_ms": float64(time.Since(started).Microseconds()) / 1000, "request": map[string]any{"model": "jev-1.14.0", "state": call.State, "questions": call.Questions, "assets": call.Images}}
			}
		}
		for _, mode := range []string{"native", "ocr"} {
			if *visualOnly {
				break
			}
			started := time.Now()
			var text identity.Text
			if mode == "native" {
				f, e := os.Open(path)
				if e != nil {
					return e
				}
				text, err = identity.Extract(context.Background(), "pdf", f, int64(len(content)), identity.DefaultLimits())
				f.Close()
			} else {
				text, err = ocr.Extract(context.Background(), path)
			}
			ms := float64(time.Since(started).Microseconds()) / 1000
			if err != nil {
				results[mode] = map[string]any{"error": err.Error(), "extract_ms": ms}
				continue
			}
			started = time.Now()
			candidates := intake.Harvest(item.Filename, text)
			draft := intake.NewDraft(cat, item.Filename, text, candidates, nil)
			draft.Plan(nil, item.ID)
			call, needed := draft.Call()
			results[mode] = map[string]any{"text": text, "candidates": candidates, "extract_ms": ms, "harvest_ms": float64(time.Since(started).Microseconds()) / 1000, "decisions": draft.Decisions(), "needed": needed, "request": map[string]any{"model": "jev-1.13.0", "state": call.State, "questions": call.Questions}}
		}
		encoded, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(*output, item.ID+".json"), encoded, 0600); err != nil {
			return err
		}
		fmt.Println(item.Filename, "extracted")
	}
	return nil
}
