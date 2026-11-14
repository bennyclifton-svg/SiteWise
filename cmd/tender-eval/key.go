package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// answerKey is the private answer key. It names real firms and prices.
type answerKey struct {
	Version    int       `json:"version"`
	Reviewed   bool      `json:"reviewed"`    // true once the owner has checked it
	CorpusRoot string    `json:"corpus_root"` // "testing/rft"
	Cases      []keyCase `json:"cases"`
}

type keyCase struct {
	ID        string    `json:"id"`         // e.g. "camp-b1"; never printed without -show
	Path      string    `json:"path"`       // under CorpusRoot
	SHA256    string    `json:"sha256"`     // of the file; a changed file fails the run
	Schedules []string  `json:"schedules"`  // RFP ids for the rows, e.g. ["main-works"]
	Split     string    `json:"split"`      // "tune" or "hold"
	Scan      bool      `json:"scan"`       // the file has no text layer
	TenderSum string    `json:"tender_sum"` // decimal, ex GST; "" when the firm prints no single price
	Lines     []keyLine `json:"lines"`
}

// keyLine is one priced line of scope, an option or a saving. Subtotals,
// totals, GST lines and rates are not keyed.
type keyLine struct {
	Label   string `json:"label"`
	Amount  string `json:"amount"`  // decimal, ex GST, signed
	Row     string `json:"row"`     // schedule key; "" for not broken down
	Counted bool   `json:"counted"` // true: part of the tender sum
	Kind    string `json:"kind"`    // base | ps | pc | option | saving
}

// loadKey reads the key and checks every file against its hash. An error
// names the case id, never the path: paths name firms.
func loadKey(path, corpusRoot string) (answerKey, error) {
	var key answerKey
	b, err := os.ReadFile(path)
	if err != nil {
		return key, fmt.Errorf("answer key: %w", err)
	}
	if err := json.Unmarshal(b, &key); err != nil {
		return key, fmt.Errorf("answer key: %w", err)
	}
	if corpusRoot != "" {
		key.CorpusRoot = corpusRoot
	}
	for _, c := range key.Cases {
		sum, err := fileHash(filepath.Join(key.CorpusRoot, filepath.FromSlash(c.Path)))
		if err != nil {
			return key, fmt.Errorf("case %s: its file cannot be read", c.ID)
		}
		if sum != c.SHA256 {
			return key, fmt.Errorf("case %s: its file changed since it was labelled", c.ID)
		}
	}
	return key, nil
}

func fileHash(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
