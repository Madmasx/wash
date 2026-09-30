package qrcode

import (
	"strings"
	"testing"

	"github.com/rivo/tview"
	goqrcode "github.com/skip2/go-qrcode"
)

// parseAnsiQR reconstructs the module matrix from the ANSI-rendered QR.
// Handles only the SGR sequences this renderer emits. true = black module.
func parseAnsiQR(s string) ([][]bool, int, error) {
	var fg, bg byte = 'x', 'x' // 'k' black, 'w' white, 'x' unknown
	var rows [][]bool
	var row []bool
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '\033' && i+1 < len(s) && s[i+1] == '[' {
			j := strings.IndexByte(s[i+2:], 'm')
			if j < 0 {
				return nil, 0, nil
			}
			seq := s[i+2 : i+2+j]
			i += 2 + j + 1
			parts := strings.Split(seq, ";")
			if len(parts) == 3 && parts[1] == "5" && (parts[0] == "38" || parts[0] == "48") {
				bit := byte('k')
				if parts[2] == "255" {
					bit = 'w'
				}
				if parts[0] == "38" {
					fg = bit
				} else {
					bg = bit
				}
			} else if strings.HasPrefix(seq, "38;2;") || strings.HasPrefix(seq, "48;2;") {
				ps := strings.Split(seq, ";")
				r, g, b := ps[2] == "255", ps[3] == "255", ps[4] == "255"
				bit := byte('w')
				if !(r && g && b) {
					bit = 'k'
				}
				if ps[0] == "38" {
					fg = bit
				} else {
					bg = bit
				}
			} else if seq == "0" {
				fg, bg = 'x', 'x'
			}
			continue
		}
		if c == '\n' {
			rows = append(rows, row)
			row = nil
			fg, bg = 'x', 'x'
			i++
			continue
		}
		// one rendered cell: glyph + active colors -> top/bottom module bits
		if fg == 'x' || bg == 'x' {
			return nil, 0, nil // renderer must set colors before every glyph
		}
		rest := s[i:]
		var glyphLen int
		top, bottom := false, false
		switch {
		case strings.HasPrefix(rest, "█"):
			glyphLen, top, bottom = 3, fg == 'k', fg == 'k'
		case strings.HasPrefix(rest, "▀"):
			glyphLen, top, bottom = 3, fg == 'k', bg == 'k'
		case strings.HasPrefix(rest, "▄"):
			glyphLen, top, bottom = 3, bg == 'k', fg == 'k'
		case rest[0] == ' ':
			glyphLen, top, bottom = 1, bg == 'k', bg == 'k'
		default:
			return nil, 0, nil
		}
		row = append(row, top, bottom)
		i += glyphLen
	}
	return rows, i, nil
}

func TestRenderMatchesSourceMatrix(t *testing.T) {
	for _, n := range []int{100, 180, 220, 300} {
		payload := "2@" + strings.Repeat("abcdefghij0123456789ABCDEFGHIJ-_", n)[:n]
		qr, err := goqrcode.New(payload, goqrcode.Medium)
		if err != nil {
			t.Fatalf("payload %d: %v", n, err)
		}
		src := qr.Bitmap() // includes 4-module quiet zone; renderer crops 3
		got, _, err := parseAnsiQR(string(*New().Get(payload)))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		cropped := make([][]bool, 0, len(src)-7)
		for ir := 3; ir < len(src)-3; {
			r2 := make([]bool, 0, len(src[0])-7)
			for ic := 3; ic < len(src[0])-3; ic++ {
				tp := src[ir][ic]
				bt := ir+1 < len(src)-3 && src[ir+1][ic]
				r2 = append(r2, tp, bt)
			}
			cropped = append(cropped, r2)
			ir += 2
		}
		if len(got) != len(cropped) {
			t.Fatalf("payload %d: %d rows rendered, want %d", n, len(got), len(cropped))
		}
		for r := range cropped {
			if len(got[r]) != len(cropped[r]) {
				t.Fatalf("payload %d: row %d width %d, want %d", n, r, len(got[r]), len(cropped[r]))
			}
			for c := range cropped[r] {
				if got[r][c] != cropped[r][c] {
					t.Fatalf("payload %d: mismatch at (%d,%d)", n, r, c)
				}
			}
		}
		t.Logf("payload %3d chars: %d cols x %d rows reconstruidas idénticas a la fuente", n, len(got[0]), len(got))
	}
}

func TestRenderANSICompact(t *testing.T) {
	payload := "2@" + strings.Repeat("abcdefghij0123456789ABCDEFGHIJ-_", 11)[:220]
	qr, _ := goqrcode.New(payload, goqrcode.Medium)
	s := string(*New().Get(payload))
	const maxBytes = 25000
	if len(s) > maxBytes {
		t.Fatalf("payload %d chars: %d bytes ANSI, want <= %d", len(payload), len(s), maxBytes)
	}
	_ = qr
	t.Logf("payload %d chars (%dx%d): %d bytes ANSI", len(payload), len(qr.Bitmap()), len(qr.Bitmap()), len(s))
}

// TestANSIWriterPath proves the app's real rendering path (tview.ANSIWriter)
// survives the compact SGR codes with the right colors.
func TestANSIWriterPath(t *testing.T) {
	payload := "2@" + strings.Repeat("abcdefghij0123456789ABCDEFGHIJ-_", 11)[:220]
	s := string(*New().Get(payload))
	var b strings.Builder
	if _, err := tview.ANSIWriter(&b).Write([]byte(s)); err != nil {
		t.Fatalf("ANSIWriter: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, "[black:") || !strings.Contains(out, ":black") {
		t.Fatalf("falta color black en la traducción ANSIWriter")
	}
	if !strings.Contains(out, "[#ffffff:") && !strings.Contains(out, "[white:") {
		t.Fatalf("falta color blanco en la traducción ANSIWriter")
	}
	glyphs := strings.Count(out, "█") + strings.Count(out, "▀") + strings.Count(out, "▄")
	if glyphs == 0 {
		t.Fatalf("sin glifos QR tras ANSIWriter")
	}
	t.Logf("ANSIWriter: %d glifos traducidos OK, %d bytes ANSI -> %d bytes tags", glyphs, len(s), len(out))
}
