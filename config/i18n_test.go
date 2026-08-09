package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestT(t *testing.T) {
	Config.General.Language = "es"
	if got := T("ui.contacts"); got != "Contactos" {
		t.Errorf("es ui.contacts = %q, want Contactos", got)
	}
	if got := T("ui.groups"); got != "Grupos" {
		t.Errorf("es ui.groups = %q, want Grupos", got)
	}
	if got := T("help.keys"); got != "Teclas:" {
		t.Errorf("es help.keys = %q, want Teclas:", got)
	}
	if got := T("cmds.lang"); got != "Cambiar el idioma de la interfaz (es | en)" {
		t.Errorf("es cmds.lang = %q, want the /lang help text", got)
	}
	if got := T("help.lang"); got != "Cambiar el idioma de la interfaz (es | en)" {
		t.Errorf("es help.lang = %q, want the /lang help text", got)
	}
	Config.General.Language = "en"
	if got := T("cmds.lang"); got != "Change the UI language (es | en)" {
		t.Errorf("en cmds.lang = %q, want the /lang help text", got)
	}
	if got := T("help.lang"); got != "Change the UI language (es | en)" {
		t.Errorf("en help.lang = %q, want the /lang help text", got)
	}
	if got := T("ui.contacts"); got != "Contacts" {
		t.Errorf("en ui.contacts = %q, want Contacts", got)
	}
	if got := T("ui.groups"); got != "Groups" {
		t.Errorf("en ui.groups = %q, want Groups", got)
	}
	// Unknown language falls back to English, then to the raw key.
	Config.General.Language = "fr"
	if got := T("ui.contacts"); got != "Contacts" {
		t.Errorf("fallback ui.contacts = %q, want Contacts", got)
	}
	if got := T("key.that.does.not.exist"); got != "key.that.does.not.exist" {
		t.Errorf("missing key = %q, want raw key", got)
	}
}

func TestSavePersistsLanguage(t *testing.T) {
	tmp := t.TempDir()
	// adrg/xdg caches base paths on first use, so XDG_CONFIG_HOME cannot be
	// changed after import; assign the internal path directly instead.
	path := filepath.Join(tmp, "wash", "wash.config")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configFilePath = path

	Config.General.Language = "en"
	if err := Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	languageRe := regexp.MustCompile(`(?m)^language\s+=\s+en$`)
	if !languageRe.Match(content) {
		t.Errorf("config file does not contain 'language = en':\n%s", content)
	}
}
