// Command ocr-trial runs the opt-in Bankstown OCR trial outside the web server.
// The manifest scopes exact organisation/document/hash triples. No directory
// scanning, automatic crop detection, production rollout or threshold changes.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

type item struct {
	ID        string `json:"id"`
	OrgID     string `json:"org_id"`
	ProjectID string `json:"project_id"`
	Filename  string `json:"filename"`
	SHA256    string `json:"sha256"`
}
type evidence struct {
	SHA256      string        `json:"sha256"`
	ModelSHA256 string        `json:"model_sha256"`
	Text        identity.Text `json:"text"`
	RenderMS    float64       `json:"render_ms"`
	TotalMS     float64       `json:"total_ms"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	manifest := flag.String("manifest", "", "exact document/hash scope JSON")
	python := flag.String("python", "python", "Python with pypdfium2 and Pillow")
	tesseract := flag.String("tesseract", "tesseract", "Tesseract executable")
	tessdata := flag.String("tessdata", ".tools/tesseract/tessdata", "fast English model directory")
	output := flag.String("output", "", "new audit directory (must not exist)")
	apply := flag.Bool("apply", false, "commit review-band decisions to the scoped documents")
	flag.Parse()
	if *manifest == "" || *output == "" {
		return errors.New("manifest and output are required")
	}
	body, err := os.ReadFile(*manifest)
	if err != nil {
		return err
	}
	var items []item
	if err = json.Unmarshal(body, &items); err != nil {
		return err
	}
	if len(items) == 0 || len(items) > 5 {
		return errors.New("trial scope must contain 1-5 documents")
	}
	if err = os.Mkdir(*output, 0700); err != nil {
		return err
	}
	ctx := context.Background()
	st, err := store.Open(ctx, os.Getenv("SITEWISE_DATABASE_URL"))
	if err != nil {
		return err
	}
	defer st.Close()
	blobs, err := files.Open(os.Getenv("SITEWISE_FILE_DIR"), 32<<20)
	if err != nil {
		return err
	}
	cat, err := intake.LoadCatalog("data/intake")
	if err != nil {
		return err
	}
	th, err := intake.LoadThresholds("data/intake")
	if err != nil {
		return err
	}
	cl, err := jev.New(jev.Options{APIKey: os.Getenv("SITEWISE_JEV_API_KEY"), Model: "jev-1.13.0"})
	if err != nil {
		return err
	}
	svc, err := intake.NewService(st, cl, cat, th)
	if err != nil {
		return err
	}
	var timings []float64
	for _, it := range items {
		if it.ID == "" || filepath.Base(it.ID) != it.ID {
			return errors.New("invalid document id")
		}
		doc, err := st.GetDocument(ctx, it.OrgID, it.ID)
		if err != nil {
			return err
		}
		f, err := st.GetFile(ctx, it.OrgID, doc.FileID)
		if err != nil {
			return err
		}
		if doc.ProjectID != it.ProjectID || doc.Filename != it.Filename || hex.EncodeToString(f.SHA256) != it.SHA256 {
			return errors.New("manifest does not match stored document")
		}
		before, err := st.DocumentView(ctx, it.OrgID, it.ID)
		if err != nil {
			return err
		}
		if before.Status != store.StatusNotFiled || before.Reason != intake.ReasonNoText {
			return errors.New("trial only accepts not-filed textless documents")
		}
		if err = save(filepath.Join(*output, it.ID+"-before.json"), before); err != nil {
			return err
		}
		path, err := blobs.Path(f.SHA256)
		if err != nil {
			return err
		}
		started := time.Now()
		callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		cmd := exec.CommandContext(callCtx, *python, "tools/ocr_titleblock.py", "--pdf", path, "--tesseract", *tesseract, "--tessdata", *tessdata)
		cmd.Stderr = os.Stderr
		raw, err := cmd.Output()
		cancel()
		if err != nil {
			return err
		}
		var e evidence
		if err = json.Unmarshal(raw, &e); err != nil {
			return err
		}
		if e.SHA256 != it.SHA256 || e.ModelSHA256 != "7d4322bd2a7749724879683fc3912cb542f19906c83bcc1a52132556427170b2" {
			return errors.New("OCR source or model mismatch")
		}
		if err = save(filepath.Join(*output, it.ID+"-evidence.json"), e); err != nil {
			return err
		}
		decisions, err := svc.TrialOCR(ctx, it.OrgID, it.ID, e.Text)
		if err != nil {
			return err
		}
		if err = save(filepath.Join(*output, it.ID+"-decisions.json"), decisions); err != nil {
			return err
		}
		if *apply {
			if _, err = st.CommitOCRTrial(ctx, it.OrgID, it.ID, decisions); err != nil {
				return err
			}
			after, err := st.DocumentView(ctx, it.OrgID, it.ID)
			if err != nil {
				return err
			}
			if err = save(filepath.Join(*output, it.ID+"-after.json"), after); err != nil {
				return err
			}
		}
		elapsed := time.Since(started)
		timings = append(timings, float64(elapsed.Microseconds())/1000)
		fmt.Printf("%s OCR=%.0fms total=%s applied=%t\n", it.Filename, e.TotalMS, elapsed.Round(time.Millisecond), *apply)
		for _, d := range decisions {
			fmt.Printf("  %s=%q (%s)\n", d.Field, d.Value, d.Band)
		}
	}
	sort.Float64s(timings)
	p50, p90 := timings[(len(timings)-1)/2], timings[(9*len(timings)+9)/10-1]
	summary := map[string]any{"samples": len(timings), "total_ms": timings, "p50_ms": p50, "p90_ms": p90, "budget_p50_ms": 1500, "budget_p90_ms": 2000, "applied": *apply}
	if err := save(filepath.Join(*output, "timing.json"), summary); err != nil {
		return err
	}
	fmt.Printf("Trial p50=%.0fms p90=%.0fms (budget 1500/2000ms)\n", p50, p90)
	if p50 > 1500 || p90 > 2000 {
		return errors.New("trial latency budget exceeded; see audit directory for completed results")
	}
	return nil
}
func save(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(path, b, 0600)
}
