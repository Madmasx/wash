package main

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestStripLinkTags(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`["mid1"](05-08-26 10:00:00) [green::b]Alice: [-::-]hola[""]`,
			`(05-08-26 10:00:00) Alice: hola`},
		{`[::d]dim[-::-] normal`, `dim normal`},
		{`[red]rojo[-]fin`, `rojofin`},
		{`[] vacio`, `[] vacio`},                        // [] is not a valid tag: prints literally
		{`a[[]b`, `a[[]b`},                              // "[[]" is not an escape: prints literally
		{`texto [brackets[] ok`, `texto [brackets] ok`}, // tview.Escape output
		{``, ``},
	}
	for _, c := range cases {
		if got := stripLinkTags(c.in); got != c.want {
			t.Errorf("stripLinkTags(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWrapLinkLine(t *testing.T) {
	// Word wrap breaks at a boundary and keeps the boundary chars on the
	// current line (tview semantics: lines carry their trailing space).
	lines := wrapLinkLine("hola mundo esto es un test", 5, true)
	want := []string{"hola ", "mundo ", "esto ", "es un ", "test"}
	if len(lines) != len(want) {
		t.Fatalf("wrapLinkLine = %v, want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("wrapLinkLine = %v, want %v", lines, want)
		}
	}
	// Empty line stays a single empty line (mirrors tview buffer lines).
	if got := wrapLinkLine("", 10, true); len(got) != 1 || got[0] != "" {
		t.Errorf("wrapLinkLine(\"\") = %v, want [\"\"]", got)
	}
}

func TestColorizeLinks(t *testing.T) {
	in := "visita https://example.com/abc y https://e.io/x fin"
	out := colorizeLinks(in)
	if !strings.Contains(out, "[lightblue]https://example.com/abc[-]") {
		t.Errorf("colorizeLinks missing first link tag: %q", out)
	}
	if !strings.Contains(out, "[lightblue]https://e.io/x[-]") {
		t.Errorf("colorizeLinks missing second link tag: %q", out)
	}
	if !strings.Contains(out, "visita ") || !strings.Contains(out, " fin") {
		t.Errorf("colorizeLinks lost surrounding text: %q", out)
	}
}

// buildTestView creates the same TextView configuration WaSh uses.
func buildTestView(width, height int) (*tview.TextView, tcell.SimulationScreen) {
	textView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)
	textView.SetRect(0, 0, width, height)
	textView.SetMouseCapture(linkClickCapture)
	screen := tcell.NewSimulationScreen("UTF-8")
	screen.Init()
	screen.SetSize(width, height)
	return textView, screen
}

// screenRow renders one row of the simulated screen as plain text.
func screenRow(screen tcell.SimulationScreen, y, w int) string {
	cells, _, _ := screen.GetContents()
	var b strings.Builder
	for _, cell := range cells[y*w : y*w+w] {
		b.WriteString(string(cell.Runes))
	}
	return b.String()
}

// findText locates the first occurrence of needle in the screen and returns
// its (row, col).
func findText(screen tcell.SimulationScreen, needle string) (int, int, bool) {
	_, w, h := screen.GetContents()
	for y := 0; y < h; y++ {
		row := screenRow(screen, y, w)
		if x := strings.Index(row, needle); x >= 0 {
			return y, x, true
		}
	}
	return 0, 0, false
}

// TestLinkClickIntegration renders a real message (with region + color tags
// and a URL, exactly like getTextMessageString produces) onto a simulated
// screen, then verifies:
//  1. our line index predicts the URL position that tview actually renders;
//  2. clicking exactly on the URL opens it;
//  3. clicking elsewhere on the message does not open anything and keeps
//     the default region highlight.
func TestLinkClickIntegration(t *testing.T) {
	width, height := 60, 12
	tv, screen := buildTestView(width, height)
	defer screen.Fini()

	msg := `["mid1"](05-08-26 10:00:00) [green::b]Alice: [-::-]visita https://example.com/abc ahora[""]` + "\n" +
		`["mid2"](05-08-26 10:01:00) [cyan::b]Yo: [-::-]sin enlace[""]`
	tv.SetText(msg)
	linkLogAppendText(msg) // the app logs via NewScreen; the test must too
	tv.Draw(screen)
	screen.Show()

	// 1) The rendered screen must show the URL somewhere.
	row, col, ok := findText(screen, "https://example.com/abc")
	if !ok {
		_, w, h := screen.GetContents()
		for y := 0; y < h; y++ {
			t.Logf("row %02d: %q", y, screenRow(screen, y, w))
		}
		t.Fatalf("URL not found on rendered screen (width=%d)", width)
	}

	// 2) Our index must predict exactly the same position (validates that
	// stripLinkTags + wrapLinkLine replicate tview's rendering).
	_, _, w, _ := tv.GetInnerRect()
	rebuildLinkIndex(w)
	url := linkAt(row, col+3)
	if url != "https://example.com/abc" {
		t.Fatalf("linkAt(%d,%d) = %q, want the URL (index != rendered?)", row, col+3, url)
	}
	// The cell just before the URL must not be part of the link.
	if url := linkAt(row, col-1); url != "" {
		t.Fatalf("linkAt(%d,%d) = %q, want \"\" (off by one before URL)", row, col-1, url)
	}

	// 3) Clicking on the URL opens it.
	var opened []string
	linkOpen = func(u string) { opened = append(opened, u) }
	handler := tv.MouseHandler()
	setFocus := func(p tview.Primitive) {}
	handler(tview.MouseLeftClick, tcell.NewEventMouse(col+5, row, tcell.Button1, 0), setFocus)
	if len(opened) != 1 || opened[0] != "https://example.com/abc" {
		t.Fatalf("click on URL opened %v, want [https://example.com/abc]", opened)
	}
	// And the message region is still highlighted by the default handler.
	if hls := tv.GetHighlights(); len(hls) == 0 || hls[0] != "mid1" {
		t.Fatalf("GetHighlights after URL click = %v, want [mid1]", hls)
	}

	// 4) Clicking on the plain prefix of the message opens nothing.
	opened = nil
	handler(tview.MouseLeftClick, tcell.NewEventMouse(2, row, tcell.Button1, 0), setFocus)
	if len(opened) != 0 {
		t.Fatalf("click on prefix opened %v, want nothing", opened)
	}

	// 5) Clicking on the message without a link opens nothing.
	row2, _, _ := findText(screen, "sin enlace")
	handler(tview.MouseLeftClick, tcell.NewEventMouse(2, row2, tcell.Button1, 0), setFocus)
	if len(opened) != 0 {
		t.Fatalf("click on linkless message opened %v, want nothing", opened)
	}
}

func TestLinkClickCaptureIgnoresNonLeftClicks(t *testing.T) {
	width, height := 60, 10
	tv, screen := buildTestView(width, height)
	defer screen.Fini()
	tv.SetText(`["m1"]hola https://e.io[""]`)
	linkLogAppendText(`["m1"]hola https://e.io[""]`)
	tv.Draw(screen)
	screen.Show()

	row, col, ok := findText(screen, "https://e.io")
	if !ok {
		t.Fatalf("URL not rendered")
	}
	var opened []string
	linkOpen = func(u string) { opened = append(opened, u) }
	handler := tv.MouseHandler()
	setFocus := func(p tview.Primitive) {}
	// A right click on the URL must not open it.
	handler(tview.MouseRightClick, tcell.NewEventMouse(col+3, row, tcell.Button2, 0), setFocus)
	if len(opened) != 0 {
		t.Fatalf("right click opened %v, want nothing", opened)
	}
}
