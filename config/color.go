package config

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// colorAliases maps friendly color names onto the canonical ones tcell knows.
// tcell.ColorNames has no "cyan" or "magenta" entry, and a miss is silent: it
// returns ColorDefault, so a group node ends up drawn with the default
// background and its selection shading disappears.
var colorAliases = map[string]string{
	"cyan":    "aqua",
	"magenta": "fuchsia",
}

// NormalizeColorName maps a configured color name onto the canonical name
// tcell and tview understand, lowercasing it and resolving aliases. An
// unrecognized name is returned untouched so callers decide the fallback.
func NormalizeColorName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if alias, ok := colorAliases[name]; ok {
		return alias
	}
	return name
}

// Color resolves a configured color name to a tcell color. Unknown names fall
// back to white rather than silently becoming the terminal default, which is
// what makes misconfigured colors invisible instead of merely wrong.
func Color(name string) tcell.Color {
	name = NormalizeColorName(name)
	if name == "default" || name == "" {
		return tcell.ColorDefault
	}
	if color, ok := tcell.ColorNames[name]; ok {
		return color
	}
	if color := tcell.GetColor(name); color != tcell.ColorDefault {
		return color
	}
	return tcell.ColorWhite
}
