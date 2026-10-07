// works-eval scores an offline canonical work-item export. It never calls Jev.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sitewise/internal/eval"
	"sitewise/internal/knowledge"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	manifest := flag.String("manifest", "data/eval/profile/manifest.json", "Pinned private corpus manifest")
	keyPath := flag.String("key", "", "Version-2 work-item answer key")
	actualPath := flag.String("actual", "", "Offline canonical snapshot JSON")
	versionsPath := flag.String("versions", "", "Expected run version pins JSON, required for scoring")
	knowledgeRoot := flag.String("knowledge", "knowledge", "Building catalogue directory")
	root := flag.String("root", "", "Local corpus root override")
	validate := flag.Bool("validate-only", false, "Check key and corpus without scoring")
	gate := flag.Bool("gate", false, "Fail unless owner-reviewed precision and recall reach 90 percent")
	flag.Parse()
	if *keyPath == "" || (*validate && *gate) {
		return fmt.Errorf("key required; validate-only and gate are mutually exclusive")
	}
	key, hash, err := eval.LoadWorkKey(*keyPath)
	if err != nil {
		return err
	}
	cat, err := knowledge.Load(*knowledgeRoot)
	if err != nil {
		return err
	}
	for _, item := range key.Items {
		if _, ok := cat.System(item.System); !ok {
			return fmt.Errorf("unknown system on keyed item %s", item.ID)
		}
	}
	// An empty snapshot validates every keyed expectation without fabricating a result.
	if _, err := eval.ScoreWorkItems(key, eval.WorkSnapshot{Project: key.Project}); err != nil {
		return err
	}
	m, err := eval.LoadProfileManifest(*manifest)
	if err != nil {
		return err
	}
	corpus, err := eval.VerifyWorkCorpus(m, key, *root)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if *validate {
		return encoder.Encode(map[string]any{"project": key.Project, "reviewed": key.Reviewed, "key_sha256": hash, "corpus": corpus, "key": key, "validation": "passed", "scored": false})
	}
	if *actualPath == "" {
		return fmt.Errorf("actual snapshot required for scoring")
	}
	if *versionsPath == "" {
		return fmt.Errorf("expected version pins required for scoring")
	}
	var versions eval.WorkVersions
	if err := eval.ReadStrictJSON(*versionsPath, &versions); err != nil {
		return err
	}
	if versions.KnowledgeVersion != cat.Version() {
		return fmt.Errorf("expected knowledge version differs from loaded catalogue")
	}
	var actual eval.WorkSnapshot
	if err := eval.ReadStrictJSON(*actualPath, &actual); err != nil {
		return err
	}
	if err := eval.VerifyWorkSnapshot(actual, hash, corpus); err != nil {
		return err
	}
	if err := eval.VerifyWorkVersions(actual, versions); err != nil {
		return err
	}
	score, err := eval.ScoreWorkItems(key, actual)
	if err != nil {
		return err
	}
	if err := encoder.Encode(map[string]any{"score": score, "model": actual.Model, "question_version": actual.QuestionVersion, "knowledge_version": actual.KnowledgeVersion, "thresholds_version": actual.ThresholdsVersion, "app_build": actual.AppBuild, "key_sha256": hash, "corpus": corpus}); err != nil {
		return err
	}
	if *gate && !score.GatePassed {
		return fmt.Errorf("work extraction gate failed or key awaits owner review")
	}
	return nil
}
