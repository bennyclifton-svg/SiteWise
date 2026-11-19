package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"sitewise/internal/invoices"
)

// The progress claim key (docs/plans/2026-11-17-variations-progress-claims.md,
// Task 12; owner decision 7). The owner supplies 20 or more real
// head-contract claims in data/eval/claim/private/answer-key.json; until a
// key is reviewed and data/claim/thresholds.json is fitted, every claim-1
// pick stays blank in the product. This checks the key's shape and counts
// it; the live run, its replay and the fit follow the invoice key's when
// the first cases arrive.

type claimKey struct {
	Version         int         `json:"version"`
	Reviewed        bool        `json:"reviewed"`
	QuestionVersion string      `json:"question_version"`
	CorpusRoot      string      `json:"corpus_root"`
	Cases           []claimCase `json:"cases"`
}

type claimCase struct {
	ID       string     `json:"id"`
	Path     string     `json:"path"`
	SHA256   string     `json:"sha256"`
	Contract claimLines `json:"contract"`
	Number   *string    `json:"number"`
	Date     *string    `json:"date"`
	Ref      *string    `json:"reference_date"`
	Rows     []struct {
		Label string `json:"label"`
		// Line is the contract line or variation the row charges for, as
		// labelled in Contract, or "none".
		Line string `json:"line"`
	} `json:"rows"`
}

type claimLines struct {
	Lines      []string `json:"lines"`
	Variations []string `json:"variations"`
}

// evaluateClaims loads and checks the claim key.
func evaluateClaims(o options, stdout io.Writer) error {
	body, err := os.ReadFile(o.key)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stdout, "No claim answer key at %s. Claim reading is not measured; claim-1 picks stay blank until the owner supplies a reviewed key.\n", o.key)
		return nil
	}
	if err != nil {
		return err
	}
	var key claimKey
	if err := json.Unmarshal(body, &key); err != nil {
		return err
	}
	if key.QuestionVersion != invoices.ClaimQuestionVersion {
		return fmt.Errorf("claim key is for %q, this build asks %q", key.QuestionVersion, invoices.ClaimQuestionVersion)
	}
	rows := 0
	for _, c := range key.Cases {
		if c.ID == "" || c.Path == "" || len(c.Contract.Lines) == 0 {
			return fmt.Errorf("case %q needs an id, a path and its contract's lines", c.ID)
		}
		rows += len(c.Rows)
	}
	fmt.Fprintf(stdout, "Claim key: %d cases, %d rows, reviewed %v. Fit needs at least 20 reviewed cases.\n", len(key.Cases), rows, key.Reviewed)
	return nil
}
