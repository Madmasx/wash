package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
	"github.com/rivo/uniseg"
	"github.com/skratchdot/open-golang/open"

	"wash/config"
)

// linkURLPattern finds URLs inside message text. It mirrors the detection used
// by the /url command (messages/session_manager.go) so clicking a link behaves
// exactly like pressing the url key on the selected message.
var linkURLPattern = regexp.MustCompile(`https?://[^\s]+`)

// The following regexes are copied verbatim from rivo/tview's util.go for the
// tview version pinned in go.mod, so the text we measure wraps and strips to
// exactly what tview renders. If tview is ever upgraded, these must be
// re-synced (or replaced with an exported tview API).
var (
	linkColorPattern    = regexp.MustCompile(`\[([a-zA-Z]+|#[0-9a-zA-Z]{6}|\-)?(:([a-zA-Z]+|#[0-9a-zA-Z]{6}|\-)?(:([lbdru]+|\-)?)?)?\]`)
	linkRegionPattern   = regexp.MustCompile(`\["([a-zA-Z0-9_,;: \-\.]*)"\]`)
	linkEscapePattern   = regexp.MustCompile(`\[([a-zA-Z0-9_,;: \-\."#]+)\[(\[*)\]`)
	linkBoundaryPattern = regexp.MustCompile(`(([,\-\.:;!\?&#+]|\n)[ \t\f\r]*|([ \t\f\r]+))`)
	linkSpacePattern    = regexp.MustCompile(`\s+`)
)

// linkLine maps one rendered display line of the messages view to the URL
// fragment shown on it. An empty url means the line holds no link.
type linkLine struct {
	url            string
	fromCol, toCol int // display columns of the URL fragment on this line
}

var (
	// linkLog records every raw buffer line written to the messages view, in
	// order. It mirrors the internal buffer of the tview TextView so click
	// coordinates can be mapped back to the text under the cursor.
	linkLog []string

	// linkIndex is the cached per-display-line URL map for the current width.
	linkIndex    []linkLine
	linkIndexFor = -1 // content width the cache was built for (-1 = dirty)
)

// linkLogAppend records one raw buffer line exactly as written to textView.
func linkLogAppend(raw string) {
	linkLog = append(linkLog, raw)
	linkIndexFor = -1
}

// linkLogAppendText records a multi-line block the way tview's TextView.Write
// splits it (see textview.go: newLineRegex.Split + tab expansion).
func linkLogAppendText(text string) {
	linkLog = linkLog[:0]
	linkIndexFor = -1
	if len(text) == 0 {
		return
	}
	text = strings.ReplaceAll(text, "\t", strings.Repeat(" ", 4))
	linkLog = append(linkLog, strings.Split(text, "\n")...)
}

// linkLogReset drops the recorded content (after textView.Clear()).
func linkLogReset() {
	linkLog = linkLog[:0]
	linkIndexFor = -1
}

// tviewLine writes one line to the messages view and records it in the link
// log. It is the single write path for all Fprintln-style output, so the log
// stays in sync with what tview actually renders.
func tviewLine(parts ...interface{}) {
	fmt.Fprintln(textView, parts...)
	content := strings.TrimSuffix(fmt.Sprintln(parts...), "\n")
	for _, seg := range strings.Split(content, "\n") {
		linkLogAppend(seg)
	}
}

// stripLinkTags removes tview color/region/escape tags the same way tview's
// decomposeString does, so byte offsets and widths match the rendered output.
func stripLinkTags(text string) string {
	colorIndices := linkColorPattern.FindAllStringIndex(text, -1)
	regionIndices := linkRegionPattern.FindAllStringIndex(text, -1)
	escapeIndices := linkEscapePattern.FindAllStringIndex(text, -1)

	type tag struct{ start, end, kind int } // kind: 0 color, 1 region, 2 escape
	tags := make([]tag, 0, len(colorIndices)+len(regionIndices)+len(escapeIndices))
	for _, idx := range colorIndices {
		if idx[1]-idx[0] == 2 {
			continue // tview filters empty color tags
		}
		tags = append(tags, tag{idx[0], idx[1], 0})
	}
	for _, idx := range regionIndices {
		tags = append(tags, tag{idx[0], idx[1], 1})
	}
	for _, idx := range escapeIndices {
		tags = append(tags, tag{idx[0], idx[1], 2})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].start < tags[j].start })

	var b strings.Builder
	from := 0
	for _, t := range tags {
		if t.kind == 2 {
			// Escape sequences keep the text up to the trailing bracket and a
			// single ']' (decomposeString semantics: "[X[]" renders as "[X]").
			b.WriteString(text[from : t.end-2])
			b.WriteByte(']')
		} else {
			b.WriteString(text[from:t.start])
		}
		from = t.end
	}
	b.WriteString(text[from:])
	return b.String()
}

// wrapLinkLine replicates tview's TextView.reindexBuffer wrap loop so wrapped
// line boundaries match the rendered output cell by cell.
func wrapLinkLine(str string, width int, wordWrap bool) []string {
	var splitLines []string
	if len(str) > 0 {
		for len(str) > 0 {
			extract := runewidth.Truncate(str, width, "")
			if len(extract) == 0 {
				// We'll extract at least one grapheme cluster.
				gr := uniseg.NewGraphemes(str)
				gr.Next()
				_, to := gr.Positions()
				extract = str[:to]
			}
			if wordWrap && len(extract) < len(str) {
				// Add any spaces from the next line.
				if spaces := linkSpacePattern.FindStringIndex(str[len(extract):]); spaces != nil && spaces[0] == 0 {
					extract = str[:len(extract)+spaces[1]]
				}
				// Can we split before the mandatory end?
				if matches := linkBoundaryPattern.FindAllStringIndex(extract, -1); len(matches) > 0 {
					extract = extract[:matches[len(matches)-1][1]]
				}
			}
			splitLines = append(splitLines, extract)
			str = str[len(extract):]
		}
	} else {
		splitLines = []string{str}
	}
	return splitLines
}

// rebuildLinkIndex maps every display line of the messages view to the URL
// fragment (if any) rendered on it, for the given content width.
func rebuildLinkIndex(width int) {
	linkIndex = linkIndex[:0]
	for _, raw := range linkLog {
		stripped := stripLinkTags(raw)
		lines := wrapLinkLine(stripped, width, true)
		// Full-text URL spans, so links split across wrap boundaries still
		// map to their visible fragments.
		urls := linkURLPattern.FindAllStringIndex(stripped, -1)
		pos := 0
		for _, line := range lines {
			lineEnd := pos + len(line)
			url, fromCol, toCol := "", 0, 0
			for _, u := range urls {
				if u[1] <= pos || u[0] >= lineEnd {
					continue
				}
				fragStart, fragEnd := u[0], u[1]
				if fragStart < pos {
					fragStart = pos
				}
				if fragEnd > lineEnd {
					fragEnd = lineEnd
				}
				fromCol = runewidth.StringWidth(stripped[pos:fragStart])
				toCol = fromCol + runewidth.StringWidth(stripped[fragStart:fragEnd])
				url = stripped[u[0]:u[1]]
				break
			}
			linkIndex = append(linkIndex, linkLine{url: url, fromCol: fromCol, toCol: toCol})
			pos = lineEnd
		}
	}
	linkIndexFor = width
}

// linkAt returns the URL shown at the given content row and column of the
// messages view, or "" when there is no link at that position.
func linkAt(row, col int) string {
	_, _, width, _ := textView.GetInnerRect()
	if width != linkIndexFor {
		rebuildLinkIndex(width)
	}
	if row < 0 || row >= len(linkIndex) {
		return ""
	}
	line := linkIndex[row]
	if line.url == "" || col < line.fromCol || col >= line.toCol {
		return ""
	}
	return line.url
}

// linkClickCapture is installed on the messages TextView. A left click exactly
// on a link opens it in the default browser; every other click keeps the
// default tview behavior (highlight the message region, scrolling, focus).
func linkClickCapture(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
	if action != tview.MouseLeftClick {
		return action, event
	}
	x, y := event.Position()
	if !textView.InRect(x, y) {
		return action, event
	}
	rectX, rectY, _, _ := textView.GetRect()
	scrollRow, scrollCol := textView.GetScrollOffset()
	row := y - rectY + scrollRow
	col := x - rectX + scrollCol
	if row < 0 {
		return action, event
	}
	if url := linkAt(row, col); url != "" {
		linkOpen(url)
	}
	// Return the event unchanged: the default handler still highlights the
	// message region, so selection keeps working after opening a link.
	return action, event
}

// linkOpen opens a URL in the default browser. It is a variable so tests can
// intercept it.
var linkOpen = func(url string) {
	open.Run(url)
	tviewLine("[::d]" + config.T("ui.link_open") + " " + url + "[::-]")
}

// colorizeLinks wraps every URL in the (already escaped) message text with the
// configured link color, making links visually distinct from the rest of the
// message.
func colorizeLinks(text string) string {
	linkColor := config.NormalizeColorName(config.Config.Colors.LinkColor)
	if linkColor == "" {
		return text
	}
	matches := linkURLPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, m := range matches {
		b.WriteString(text[last:m[0]])
		b.WriteString("[" + linkColor + "]")
		b.WriteString(text[m[0]:m[1]])
		b.WriteString("[-]")
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}
