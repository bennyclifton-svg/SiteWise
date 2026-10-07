// profile-eval records or replays bounded Jev profile checks. Only -live
// contacts TypeSafe; replay refuses changed questions or source text.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

type sample struct {
	ID, Text, Section string
	Context           string
	// SourcePDF exercises the same extraction and routing as an uploaded brief.
	SourcePDF, Match    string
	Expected, Forbidden map[string]string
}
type recorded struct {
	Fingerprint         string
	EvidenceFingerprint string
	Label, Evidence     jev.Result
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	fixtures := flag.String("cases", "data/eval/profile/private/source-cases.json", "checked source cases")
	recording := flag.String("recording", "data/eval/profile/private/source-recording.json", "Jev recording")
	live := flag.Bool("live", false, "make paid Jev requests and save responses")
	measure := flag.Bool("measure", false, "measure current requests using recorded labels; no calls or accuracy claim")
	flag.Parse()
	if *live && *measure {
		return fmt.Errorf("measure and live are separate runs")
	}
	raw, err := os.ReadFile(*fixtures)
	if err != nil {
		return err
	}
	var cases []sample
	if err = json.Unmarshal(raw, &cases); err != nil {
		return err
	}
	if len(cases) == 0 {
		return fmt.Errorf("no cases")
	}
	cat, err := knowledge.Load("knowledge")
	if err != nil {
		return err
	}
	saved := map[string]recorded{}
	if !*live {
		raw, err = os.ReadFile(*recording)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &saved); err != nil {
			return err
		}
	}
	var client *jev.Client
	if *live {
		client, err = jev.New(jev.Options{APIKey: os.Getenv("SITEWISE_JEV_API_KEY"), Model: "jev-1.13.0", Deadline: 15 * time.Second})
		if err != nil {
			return err
		}
	}
	correct, total, wrong, automatic := 0, 0, 0, 0
	thresholds, err := profile.LoadThresholds("data/profile/thresholds.json")
	if err != nil {
		return err
	}
	var durations []time.Duration
	inputTokens, outputTokens := 0, 0
	for _, c := range cases {
		p := jobs.Passage{Text: c.Text, Section: c.Section, Kind: "specification", Ordinal: 30}
		p.Context = c.Context
		if c.SourcePDF != "" {
			p, err = sourcePassage(c)
			if err != nil {
				return err
			}
		}
		call := jobs.LabelCall(cat, p)
		call.Deadline = 15 * time.Second
		b, _ := json.Marshal(call)
		hash := sha256.Sum256(b)
		fingerprint := hex.EncodeToString(hash[:])
		rec := saved[c.ID]
		if *live {
			rec.Fingerprint = fingerprint
			rec.Label, err = client.Ask(context.Background(), call)
			if err != nil {
				return fmt.Errorf("%s: %w", c.ID, err)
			}
		} else if !*measure && rec.Fingerprint != fingerprint {
			return fmt.Errorf("%s: recording is stale", c.ID)
		}
		p.Labels = jobs.AcceptLabels(cat, rec.Label, 0.5)
		evidence, ok := jobs.EvidenceCall(cat, p)
		if ok {
			evidence.Deadline = 15 * time.Second
			raw, _ := json.Marshal(evidence)
			sum := sha256.Sum256(raw)
			fingerprint := hex.EncodeToString(sum[:])
			if *live {
				rec.EvidenceFingerprint = fingerprint
			} else if !*measure && rec.EvidenceFingerprint != fingerprint {
				return fmt.Errorf("%s: evidence recording is stale", c.ID)
			}
		}
		if *measure {
			for stage, request := range map[string]jev.Call{"label": call, "evidence": evidence} {
				if len(request.Questions) == 0 {
					continue
				}
				size, sizeErr := jev.CheckRequestSize(request)
				data, _ := json.Marshal(size)
				before := request
				before.Questions = map[string]jev.Question{}
				for id, q := range request.Questions {
					if !strings.HasSuffix(id, ".action") {
						before.Questions[id] = q
					}
				}
				prior, _ := jev.MeasureRequest(before)
				fmt.Printf("%s %s %s allowed=%v action_questions=%d action_bytes=%d\n", c.ID, stage, data, sizeErr == nil, size.Questions-prior.Questions, size.Bytes-prior.Bytes)
				if sizeErr != nil {
					return sizeErr
				}
			}
			continue
		}
		if *live && ok {
			rec.Evidence, err = client.Ask(context.Background(), evidence)
			if err != nil {
				return fmt.Errorf("%s evidence: %w", c.ID, err)
			}
		}
		saved[c.ID] = rec
		cands, _ := call.State.(map[string]any)["candidates"].(map[string][]profile.Candidate)
		values := profile.Readings(rec.Label, call.Questions, cands, p.Text)
		if ok {
			values = append(values, profile.Readings(rec.Evidence, evidence.Questions, nil, p.Text)...)
		}
		got := map[string]string{}
		applied := map[string]string{}
		for _, v := range values {
			got[v.QuestionID] = v.Value
			accepted := v.Confidence != nil && *v.Confidence >= 0.6
			if strings.HasSuffix(v.QuestionID, ".action") {
				accepted = thresholds.ActionApplied(v)
			}
			if accepted {
				applied[v.QuestionID] = v.Value
			}
		}
		for key, want := range c.Expected {
			total++
			if got[key] == want {
				correct++
				if applied[key] == want {
					automatic++
				}
			} else {
				fmt.Printf("MISS %s %s: want %q got %q\n", c.ID, key, want, got[key])
			}
		}
		for key, bad := range c.Forbidden {
			if v, ok := applied[key]; ok && (bad == "*" || v == bad) {
				wrong++
				fmt.Printf("WRONG %s %s=%q\n", c.ID, key, v)
			}
		}
		for _, r := range []jev.Result{rec.Label, rec.Evidence} {
			if r.Latency > 0 {
				durations = append(durations, r.Latency)
			}
			inputTokens += r.Usage.InputTokens
			outputTokens += r.Usage.OutputTokens
		}
	}
	if *measure {
		return nil
	}
	if *live {
		b, _ := json.MarshalIndent(saved, "", "  ")
		if err = os.WriteFile(*recording, b, 0600); err != nil {
			return err
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	fmt.Printf("Correct captured readings: %d/%d; automatically applied: %d/%d; forbidden applied readings: %d; input/output tokens: %d/%d\n", correct, total, automatic, total, wrong, inputTokens, outputTokens)
	if len(durations) > 0 {
		fmt.Printf("Recorded Jev call latency p50=%s p90=%s\n", durations[len(durations)/2], durations[(len(durations)-1)*9/10])
	}
	if correct != total || wrong > 0 {
		return fmt.Errorf("profile accuracy gate failed")
	}
	return nil
}

func sourcePassage(c sample) (jobs.Passage, error) {
	if c.Match == "" {
		return jobs.Passage{}, fmt.Errorf("%s: source match required", c.ID)
	}
	f, err := os.Open(c.SourcePDF)
	if err != nil {
		return jobs.Passage{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return jobs.Passage{}, err
	}
	text, err := identity.Extract(context.Background(), "pdf", f, info.Size(), identity.Limits{RequireComplete: true, MaxPages: 10000, MaxBytes: 256 << 20, MaxRuns: 200000})
	if err != nil {
		return jobs.Passage{}, err
	}
	src := jobs.SourceFromText(text)
	var found []jobs.Passage
	for i, u := range src.Units {
		if strings.Contains(u.Body, c.Match) {
			found = append(found, jobs.Passage{Text: u.Body, Section: u.Section, Context: u.Context, Page: u.Page, Ordinal: i + 1, Kind: "design_brief"})
		}
	}
	if len(found) != 1 {
		return jobs.Passage{}, fmt.Errorf("%s: expected one source passage, found %d", c.ID, len(found))
	}
	return found[0], nil
}
