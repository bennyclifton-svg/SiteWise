package specifications

import (
	"strings"
	"testing"
)

func TestImportColumnGeometryCodesAndContinuedText(t *testing.T) {
	pages := []Page{{Page: 1, Location: "Page 1", Text: "CODE\nTDK1 (A) Timber\nDecking\nMaterial: Timber\nCSR\nSupplier note", Regions: []Region{{Text: "CODE", X: 50}, {Text: "TDK1 (A) Timber", X: 50}, {Text: "CSR", X: 300}}}, {Page: 2, Location: "Page 2", Text: "Continued pedestal requirement\nCONC1\nConcrete\nFinish: Honed", Regions: []Region{{Text: "CONC1", X: 50}}}}
	candidates := Harvest(pages)
	codes := []string{}
	continued := false
	for _, c := range candidates {
		if c.Code != "" {
			codes = append(codes, c.Code)
		}
		continued = continued || strings.Contains(c.Text, "Continued pedestal requirement")
	}
	if strings.Join(codes, ",") != "TDK1 (A),CONC1" || !continued {
		t.Fatal("geometry, inline code or continuation lost", codes, continued)
	}
}

func TestRepeatedCodeInDescriptionDoesNotSplitRow(t *testing.T) {
	p := Page{Page: 1, Text: "CODE\nWC\nNatspec ref: 0555\nWC\nToilet suite", Regions: []Region{{Text: "CODE", X: 30}, {Text: "WC", X: 30}, {Text: "Natspec ref: 0555", X: 30}, {Text: "WC", X: 170}, {Text: "Toilet suite", X: 170}}}
	var coded []Candidate
	for _, c := range Harvest([]Page{p}) {
		if c.Code != "" {
			coded = append(coded, c)
		}
	}
	if len(coded) != 1 || !strings.Contains(coded[0].Text, "Toilet suite") {
		t.Fatal("description occurrence split the code row", coded)
	}
}
