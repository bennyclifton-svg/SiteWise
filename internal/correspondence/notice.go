// Package correspondence holds the record's Go rules that are not storage:
// Jev's one notice question on an email body, the outbound MIME message, and
// the send path's sender (docs/plans/2026-11-17-correspondence-transmittals.md).
package correspondence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sitewise/internal/jev"
)

// NoticeField is the question's id in the filing call.
const NoticeField = "notice"

// NoticeVersion names the question's wording. It is added to the filing's
// question version ("+notice-1") so a stored decision says which call it
// came from, and a cut-off approved for another wording is refused.
const NoticeVersion = "notice-1"

// NoticeOption is one choice, sent in this order.
type NoticeOption struct {
	ID     string
	Covers string
	NotFor string
}

// NoticeOptions are the eight picks (owner decision 5), in a fixed order
// with "none" first: Jev "leans toward the option that comes first"
// (https://docs.typesafe.ai/model-jaggedness/jev-1.13), so an unsure pick
// leans to nothing. Each description has the same two fields, what it
// covers and what it is not for, naming the traps
// (https://docs.typesafe.ai/concepts/how-to-build-with-system-one). The
// answer key is also to be run with the options reversed to measure the
// lean; it is the owner's (about 50 real messages, data/eval/notice/).
var NoticeOptions = []NoticeOption{
	{"none", "Ordinary correspondence: sending or acknowledging documents, arranging a meeting, a question in passing, a quote or fee proposal, an update with nothing asked or claimed under the contract.",
		"A message that formally asks for information, claims time or money, directs work, or gives notice under the contract."},
	{"rfi", "A formal request for information or clarification about the design or the works, asking someone to answer.",
		"A question in passing, a request for a quote or price, or a reply that answers an earlier request."},
	{"direction", "A direction or instruction to carry out, change, stop or not do work, given by someone entitled to direct it.",
		"A suggestion, a request, a discussion of options, or a direction that is only quoted or forwarded for information."},
	{"variation_claim", "A claim or request for a variation: added, omitted or changed work, with or without a price.",
		"A mention of a variation already agreed, a request for a quote before any change is directed, or a forwarded claim already answered."},
	{"eot_delay", "A notice of delay or a claim for an extension of time, saying the works are or will be delayed and why.",
		"A program update or a revised date with no claim, a delay mentioned in passing, or a forwarded notice already answered."},
	{"payment_claim", "A payment claim or progress claim asking to be paid an amount for work done.",
		"An invoice query, a remittance or payment advice, a quote, or a fee proposal."},
	{"dispute", "A notice of dispute, a formal rejection of an assessment, or a statement that the matter is disputed under the contract.",
		"A disagreement in passing, a complaint, or a request to talk something through."},
	{"other_notice", "Another formal notice under the contract: of a defect, a latent condition, a breach, suspension, practical completion or termination.",
		"Anything that fits one of the options above, or a notice only mentioned, quoted or forwarded for information."},
}

// NoticeQuestion is the one choice question joined to the filing call of an
// email body: one judgement, its options in the fixed order (a Go map would
// sort them). It never asks Jev to draft, count or compare dates.
func NoticeQuestion() jev.Question {
	type means struct {
		Covers string `json:"covers"`
		NotFor string `json:"not_for"`
	}
	var b bytes.Buffer
	b.WriteByte('{')
	for i, o := range NoticeOptions {
		if i > 0 {
			b.WriteByte(',')
		}
		k, _ := json.Marshal(o.ID)
		v, _ := json.Marshal(means{o.Covers, o.NotFor})
		b.Write(k)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return jev.Question{
		Type: jev.TypeChoice,
		Instructions: struct {
			Question string `json:"question"`
			NotFor   string `json:"not_for"`
		}{
			Question: "Which of these is this message, under the contract? Judge the newest message at the top, written by its sender, not quoted or forwarded text below it.",
			NotFor:   "Text the sender only quotes or forwards, a signature, a disclaimer, or an attachment's contents.",
		},
		Criteria: json.RawMessage(b.Bytes()),
	}
}

// KnownNotice reports one of the eight option ids.
func KnownNotice(id string) bool {
	for _, o := range NoticeOptions {
		if o.ID == id {
			return true
		}
	}
	return false
}

// NoticeRead is Jev's raw pick for one email body, and its band.
type NoticeRead struct {
	Pick       string
	Confidence *float64
	// Band is amber past an owner-approved cut-off, blank below it or with
	// no cut-off, grey when the answer did not come.
	Band string
}

// NoticeThresholds holds the owner-approved cut-off. Zero is no cut-off:
// every pick is stored blank and never shown, as the register's picks are
// until their key is reviewed (https://docs.typesafe.ai/confidence: act only
// in a calibrated band). Nothing is ever green and nothing is created.
type NoticeThresholds struct {
	Amber float64
}

// Read bands one answer of a call that asked the notice question. failed is
// a call that did not answer.
func (t NoticeThresholds) Read(result jev.Result, failed bool) NoticeRead {
	if failed {
		return NoticeRead{Band: "grey"}
	}
	a, ok := result.Answers[NoticeField]
	if !ok || a.Type != jev.TypeChoice || !KnownNotice(a.Choice) {
		return NoticeRead{Band: "grey"}
	}
	p := a.Probabilities[a.Choice]
	if a.Confidence != nil {
		p = *a.Confidence
	}
	r := NoticeRead{Pick: a.Choice, Confidence: &p, Band: "blank"}
	if t.Amber > 0 && p >= t.Amber && a.Choice != "none" {
		r.Band = "amber"
	}
	return r
}

// LoadNoticeThresholds reads dir/thresholds.json. A missing file is no
// cut-off; a file for another wording is refused, so an old cut-off never
// bands new answers.
func LoadNoticeThresholds(dir string) (NoticeThresholds, error) {
	body, err := os.ReadFile(filepath.Join(dir, "thresholds.json"))
	if os.IsNotExist(err) {
		return NoticeThresholds{}, nil
	}
	if err != nil {
		return NoticeThresholds{}, err
	}
	var t struct {
		QuestionVersion string  `json:"question_version"`
		Amber           float64 `json:"amber"`
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return NoticeThresholds{}, err
	}
	if t.QuestionVersion != NoticeVersion || t.Amber <= 0 || t.Amber > 1 {
		return NoticeThresholds{}, fmt.Errorf("notice thresholds: want question_version %q and an amber cut-off in (0, 1]", NoticeVersion)
	}
	return NoticeThresholds{Amber: t.Amber}, nil
}
