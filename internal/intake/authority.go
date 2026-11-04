package intake

import (
	"regexp"
	"strings"

	"sitewise/internal/identity"
)

var (
	// A council's own certificate names itself. Section 149 is the old name
	// of the s10.7 planning certificate.
	planningCertificate = regexp.MustCompile(`(?i)\bplanning\s+certificate\b|\b(?:s|sec|section)\.?\s*(?:10\.7|149)\b[^.]{0,20}\bcertificate\b`)
	// A line that names the certificate without being it: a request for one,
	// advice about it, or a consultant's deliverable that discusses it.
	aboutCertificate = regexp.MustCompile(`(?i)\b(?:request|application|apply|review|advice|summary|memo|report|proposal|fee|statement|assessment)\b`)
	// A council letterhead: the whole line is the council's name.
	councilName = regexp.MustCompile(`(?i)^(?:the\s+)?(?:(?:city|shire|municipality|town)\s+of\s+[a-z][a-z' .-]{1,40}|[a-z][a-z' .-]{1,40}?\s+(?:city\s+|shire\s+|municipal\s+|regional\s+)?council)$`)
)

// identityLines bounds where a certificate title or letterhead may sit:
// the top of the identity page, not its body prose.
const (
	identityLines   = 12
	maxIdentityLine = 80
)

// authorityIssuer files a council or authority document with no consultant
// discipline (F7, testing/synthetic/FINDINGS.md). A council is not one of the
// discipline options, so asking Jev forces the nearest consultant (a planning
// certificate became Town Planner). Code reads what the document says it is:
// a planning certificate by its own title, or a council letterhead as its
// first line. A consultant's document that discusses a certificate, or is
// addressed to a council, keeps the question. Code settles what it can
// read before the one fan-out, which then asks one question fewer:
// https://docs.typesafe.ai/concepts/how-to-build-with-system-one
// https://docs.typesafe.ai/model-jaggedness/jev-1.13
func authorityIssuer(filename string, text identity.Text) []Result {
	if hasDrawingIdentity(text) || feeProposalName.MatchString(filename) {
		return nil
	}
	none := []Result{{Field: FieldDiscipline, Settled: true, Rule: "authority_issuer"}}
	if certificateTitle(strings.TrimSuffix(filename, ".")) {
		return none
	}
	lines := topLines(text)
	if len(lines) > 0 && councilName.MatchString(strings.TrimSpace(strings.Trim(lines[0], "#*_ "))) {
		return none
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") && feeProposalName.MatchString(line) {
			return nil
		}
		if certificateTitle(strings.TrimLeft(line, "# ")) {
			return none
		}
	}
	return nil
}

func certificateTitle(line string) bool {
	return planningCertificate.MatchString(line) && !aboutCertificate.MatchString(line)
}

// topLines returns the first short non-empty lines of the identity page. A
// Markdown document's lines stop at its first heading, which is its title.
func topLines(text identity.Text) []string {
	md := text.Format == "md" || text.Format == "markdown"
	var out []string
	for _, run := range text.Runs {
		line := strings.TrimSpace(run.Text)
		if line == "" {
			continue
		}
		if len(out) == identityLines {
			break
		}
		if len(line) <= maxIdentityLine {
			out = append(out, line)
		}
		if md && strings.HasPrefix(line, "# ") {
			break
		}
	}
	return out
}
