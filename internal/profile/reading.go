package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"sort"
)

// Document reading settings. Automatic follows the document's kind; read
// and skip are the user's word and are never changed by reclassification.
const (
	ReadAuto = "auto"
	ReadYes  = "read"
	ReadSkip = "skip"
)

// ValidReadSetting reports whether s is a stored reading setting.
func ValidReadSetting(s string) bool {
	return s == ReadAuto || s == ReadYes || s == ReadSkip
}

// ReadPolicy decides which documents the profile reads. Code routes; no Jev
// call decides it (https://docs.typesafe.ai/patterns/intent-routing).
type ReadPolicy struct {
	kinds map[string]bool
}

// LoadReadPolicy reads data/profile/reading.json. An empty kind list is an
// error: it would silently read nothing.
func LoadReadPolicy(path string) (ReadPolicy, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return ReadPolicy{}, err
	}
	var f struct {
		Version   int      `json:"version"`
		Note      string   `json:"note"`
		ReadKinds []string `json:"read_kinds"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return ReadPolicy{}, err
	}
	if f.Version != 1 {
		return ReadPolicy{}, errors.New("profile reading policy needs version 1")
	}
	if len(f.ReadKinds) == 0 {
		return ReadPolicy{}, errors.New("profile reading policy lists no document kinds")
	}
	p := ReadPolicy{kinds: map[string]bool{}}
	for _, k := range f.ReadKinds {
		p.kinds[k] = true
	}
	return p, nil
}

// Loaded is false for the zero policy, which reads every document: tools and
// tests that predate the setting keep their behaviour.
func (p ReadPolicy) Loaded() bool { return p.kinds != nil }

// Reads reports whether the profile reads a document of this kind with this
// setting. An unclassified document is not read automatically.
func (p ReadPolicy) Reads(kind, setting string) bool {
	switch setting {
	case ReadYes:
		return true
	case ReadSkip:
		return false
	}
	return p.kinds[kind]
}

// Kinds lists the automatically read kinds, sorted.
func (p ReadPolicy) Kinds() []string {
	out := make([]string, 0, len(p.kinds))
	for k := range p.kinds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
