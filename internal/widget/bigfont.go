package widget

import (
	"strings"
	"unicode"
)

// bigFont is a compact 3x5 block font used by the text widget's content zoom
// (see Zoomer / SetZoom in text.go). It covers A-Z, 0-9, space, and a handful
// of punctuation — enough for a short label or status line to read as "a few
// giant letters" next to a pane packed with normal-size text. Lowercase is
// upper-cased before lookup; anything still unknown falls back to a solid
// glyph the same size so column spacing stays even rather than collapsing.
var bigFont = map[rune][bigGlyphH]string{
	' ':  {"   ", "   ", "   ", "   ", "   "},
	'A':  {" # ", "# #", "###", "# #", "# #"},
	'B':  {"## ", "# #", "## ", "# #", "## "},
	'C':  {" ##", "#  ", "#  ", "#  ", " ##"},
	'D':  {"## ", "# #", "# #", "# #", "## "},
	'E':  {"###", "#  ", "## ", "#  ", "###"},
	'F':  {"###", "#  ", "## ", "#  ", "#  "},
	'G':  {" ##", "#  ", "# #", "# #", " ##"},
	'H':  {"# #", "# #", "###", "# #", "# #"},
	'I':  {"###", " # ", " # ", " # ", "###"},
	'J':  {"  #", "  #", "  #", "# #", " # "},
	'K':  {"# #", "## ", "#  ", "## ", "# #"},
	'L':  {"#  ", "#  ", "#  ", "#  ", "###"},
	'M':  {"# #", "###", "###", "# #", "# #"},
	'N':  {"# #", "###", "###", "###", "# #"},
	'O':  {" # ", "# #", "# #", "# #", " # "},
	'P':  {"## ", "# #", "## ", "#  ", "#  "},
	'Q':  {" # ", "# #", "# #", "###", "  #"},
	'R':  {"## ", "# #", "## ", "## ", "# #"},
	'S':  {" ##", "#  ", " # ", "  #", "## "},
	'T':  {"###", " # ", " # ", " # ", " # "},
	'U':  {"# #", "# #", "# #", "# #", " # "},
	'V':  {"# #", "# #", "# #", "# #", " # "},
	'W':  {"# #", "# #", "###", "###", "# #"},
	'X':  {"# #", "# #", " # ", "# #", "# #"},
	'Y':  {"# #", "# #", " # ", " # ", " # "},
	'Z':  {"###", "  #", " # ", "#  ", "###"},
	'0':  {" # ", "# #", "# #", "# #", " # "},
	'1':  {" # ", "## ", " # ", " # ", "###"},
	'2':  {"## ", "  #", " # ", "#  ", "###"},
	'3':  {"## ", "  #", " # ", "  #", "## "},
	'4':  {"# #", "# #", "###", "  #", "  #"},
	'5':  {"###", "#  ", "## ", "  #", "## "},
	'6':  {" ##", "#  ", "## ", "# #", " # "},
	'7':  {"###", "  #", " # ", "#  ", "#  "},
	'8':  {" # ", "# #", " # ", "# #", " # "},
	'9':  {" # ", "# #", " ##", "  #", " # "},
	'.':  {"   ", "   ", "   ", "   ", " # "},
	',':  {"   ", "   ", "   ", " # ", "#  "},
	'!':  {" # ", " # ", " # ", "   ", " # "},
	'?':  {"## ", "  #", " # ", "   ", " # "},
	'-':  {"   ", "   ", "###", "   ", "   "},
	':':  {"   ", " # ", "   ", " # ", "   "},
	'\'': {" # ", " # ", "   ", "   ", "   "},
}

const (
	bigGlyphW = 3
	bigGlyphH = 5
)

var bigFallback = [bigGlyphH]string{"###", "###", "###", "###", "###"}

func bigGlyph(r rune) [bigGlyphH]string {
	if g, ok := bigFont[unicode.ToUpper(r)]; ok {
		return g
	}
	return bigFallback
}

// renderBig re-renders s as big block glyphs, scale cells per source
// "pixel" (scale < 1 is treated as 1 — there's no shrinking below the font's
// own 3x5 size). One column of blank spacing separates glyphs.
func renderBig(s string, scale int) string {
	if scale < 1 {
		scale = 1
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		var rows [bigGlyphH]strings.Builder
		for _, r := range line {
			g := bigGlyph(r)
			for row := 0; row < bigGlyphH; row++ {
				rows[row].WriteString(g[row])
				rows[row].WriteByte(' ')
			}
		}
		for row := 0; row < bigGlyphH; row++ {
			scaled := scaleGlyphLine(rows[row].String(), scale)
			for k := 0; k < scale; k++ {
				out = append(out, scaled)
			}
		}
	}
	return strings.Join(out, "\n")
}

// scaleGlyphLine repeats each cell scale times horizontally, turning a '#'
// into a solid block glyph so a bigger zoom reads as filled letterforms
// rather than a sparser dot pattern.
func scaleGlyphLine(s string, scale int) string {
	var b strings.Builder
	for _, r := range s {
		ch := r
		if r == '#' {
			ch = '█'
		}
		for k := 0; k < scale; k++ {
			b.WriteRune(ch)
		}
	}
	return b.String()
}
