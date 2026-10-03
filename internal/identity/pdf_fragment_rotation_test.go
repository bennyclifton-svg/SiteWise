package identity

import (
	"fmt"
	"testing"
)

// PDFium emits breaks between touching CAD glyphs even when the printed
// title is one line. Exercise those breaks, not preassembled title runs.
func TestFragmentedTitleAtEveryRotation(t *testing.T) {
	for _, rotation := range []int{0, 90, 180, 270} {
		t.Run(fmt.Sprint(rotation), func(t *testing.T) {
			chars := []rune("M\r\nE\r\nCHANICAL\nSERVICES\nLEVEL 1")
			boxes := make([]charBox, len(chars))
			x := 10.0
			for i, r := range chars {
				if r == '\r' || r == '\n' {
					if i > 0 && (chars[i-1] == 'L' || chars[i-1] == 'S') {
						x += 3
					}
					continue
				}
				boxes[i] = turnTestBox(charBox{left: x, right: x + 6, bottom: 20, top: 30}, rotation)
				x += 6
			}
			got := pdfLines(chars, boxes, func(int) int { return rotation })
			if len(got) != 1 || got[0].text != "MECHANICAL SERVICES LEVEL 1" {
				t.Fatalf("fragmented title: %+v", got)
			}
			mixed := pdfLines(chars, boxes, func(index int) int {
				if index == 0 {
					return rotation
				}
				return (rotation + 90) % 360
			})
			if len(mixed) < 2 || mixed[0].text != "M" {
				t.Fatalf("joined different text orientations: %+v", mixed)
			}
			// Nearby rows and distant cells must remain separate at every angle.
			for _, next := range []charBox{{left: 16, right: 22, bottom: 5, top: 15}, {left: 50, right: 56, bottom: 20, top: 30}, {left: 4, right: 10, bottom: 20, top: 30}} {
				pair := []charBox{turnTestBox(charBox{left: 10, right: 16, bottom: 20, top: 30}, rotation), {}, turnTestBox(next, rotation)}
				if got := pdfLines([]rune("A\nB"), pair, func(int) int { return rotation }); len(got) != 2 {
					t.Fatalf("joined distinct cells: %+v", got)
				}
			}
		})
	}
}

func turnTestBox(b charBox, rotation int) charBox {
	switch rotation {
	case 90:
		return charBox{left: b.bottom, right: b.top, bottom: -b.right, top: -b.left}
	case 180:
		return charBox{left: -b.right, right: -b.left, bottom: -b.top, top: -b.bottom}
	case 270:
		return charBox{left: -b.top, right: -b.bottom, bottom: b.left, top: b.right}
	}
	return b
}
