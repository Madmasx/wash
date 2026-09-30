package config

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// TestColorResolvesAliases guards the "invisible group selection" bug:
// tcell.ColorNames has no "cyan"/"magenta" entry and a map miss silently
// returns ColorDefault, which made selected nodes draw with the default
// background.
func TestColorResolvesAliases(t *testing.T) {
	cases := []struct {
		name string
		want tcell.Color
	}{
		{"cyan", tcell.ColorNames["aqua"]},
		{"CYAN", tcell.ColorNames["aqua"]},
		{" magenta ", tcell.ColorNames["fuchsia"]},
		{"green", tcell.ColorNames["green"]},
		{"yellow", tcell.ColorNames["yellow"]},
	}
	for _, tc := range cases {
		got := Color(tc.name)
		if got != tc.want {
			t.Errorf("Color(%q) = %d, want %d", tc.name, int(got), int(tc.want))
		}
		if got == tcell.ColorDefault {
			t.Errorf("Color(%q) resolved to ColorDefault, which hides the color", tc.name)
		}
	}
}

// TestColorDefaultAndFallback verifies the two deliberate fallbacks: an
// explicit "default" keeps terminal default semantics, and an unknown name
// becomes visible white instead of vanishing into the background.
func TestColorDefaultAndFallback(t *testing.T) {
	for _, name := range []string{"default", "DEFAULT", ""} {
		if got := Color(name); got != tcell.ColorDefault {
			t.Errorf("Color(%q) = %d, want ColorDefault", name, int(got))
		}
	}
	for _, name := range []string{"notacolor", "puce"} {
		if got := Color(name); got != tcell.ColorWhite {
			t.Errorf("Color(%q) = %d, want white", name, int(got))
		}
	}
}

// TestNormalizeColorName covers the tag path (tview resolves tags through
// tcell.GetColor, which has the same missing names).
func TestNormalizeColorName(t *testing.T) {
	if got := NormalizeColorName("Cyan"); got != "aqua" {
		t.Errorf("NormalizeColorName(Cyan) = %q, want aqua", got)
	}
	if got := NormalizeColorName("green"); got != "green" {
		t.Errorf("NormalizeColorName(green) = %q, want green", got)
	}
	if got := NormalizeColorName("notacolor"); got != "notacolor" {
		t.Errorf("unknown name should pass through, got %q", got)
	}
}

// TestDefaultConfigColorsAreResolvable ensures every color the app ships with
// actually resolves, so the defaults can't silently degrade to ColorDefault.
func TestDefaultConfigColorsAreResolvable(t *testing.T) {
	colors := Colors{
		Background: "default", Text: "white", ForwardedText: "purple",
		ListHeader: "yellow", ListContact: "green", ListGroup: "cyan",
		ChatContact: "green", ChatMe: "cyan", LinkColor: "lightblue",
		Borders: "purple", InputBackground: "default", InputText: "white",
		UnreadCount: "yellow", Positive: "green", Negative: "red",
	}
	// Background and InputBackground are intentionally terminal default.
	expectedDefault := map[string]bool{"Background": true, "InputBackground": true}

	for name, value := range map[string]string{
		"Background": colors.Background, "Text": colors.Text,
		"ForwardedText": colors.ForwardedText, "ListHeader": colors.ListHeader,
		"ListContact": colors.ListContact, "ListGroup": colors.ListGroup,
		"ChatContact": colors.ChatContact, "ChatMe": colors.ChatMe,
		"LinkColor": colors.LinkColor, "Borders": colors.Borders,
		"InputBackground": colors.InputBackground, "InputText": colors.InputText,
		"UnreadCount": colors.UnreadCount, "Positive": colors.Positive,
		"Negative": colors.Negative,
	} {
		got := Color(value)
		if expectedDefault[name] {
			if got != tcell.ColorDefault {
				t.Errorf("%s (%q) = %d, want ColorDefault", name, value, int(got))
			}
			continue
		}
		if got == tcell.ColorDefault {
			t.Errorf("%s (%q) resolved to ColorDefault", name, value)
		}
	}
}
