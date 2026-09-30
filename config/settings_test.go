package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbbreviateHome(t *testing.T) {
	home := GetHomeDir()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"config file", home + ".config/wash/wash.config", "~/.config/wash/wash.config"},
		{"home itself", home, "~/"},
		{"nested download", home + "Downloads/foto.png", "~/Downloads/foto.png"},
	}
	for _, c := range cases {
		if got := AbbreviateHome(c.in); got != c.want {
			t.Errorf("AbbreviateHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	// A path outside the home directory is returned untouched: there is no
	// meaningful ~ for it, and mangling it would misreport where a file lives.
	for _, in := range []string{
		"/etc/passwd",
		"/tmp/otro-usuario/foto.png",
		"",
	} {
		if got := AbbreviateHome(in); got != in {
			t.Errorf("AbbreviateHome(%q) = %q, want it unchanged", in, got)
		}
	}

	// The display form must never contain the account name.
	if user := strings.TrimSuffix(home, string(os.PathSeparator)); user != "" {
		if strings.Contains(AbbreviateHome(home+".config/wash/wash.config"), user) {
			t.Errorf("AbbreviateHome leaked the account name %q", user)
		}
	}
}

// TestAbbreviateHomeRoundTrip checks the abbreviated form still points at the
// same file, so what the UI shows remains copy-pasteable.
func TestAbbreviateHomeRoundTrip(t *testing.T) {
	abs := filepath.Join(GetHomeDir(), ".config", "wash", "wash.config")
	home := strings.TrimSuffix(GetHomeDir(), string(os.PathSeparator))
	expanded := strings.Replace(AbbreviateHome(abs), "~", home, 1)
	if expanded != abs {
		t.Errorf("round trip = %q, want %q", expanded, abs)
	}
}

// TestMigrateLegacySession checks the pre-v2.2.0 session DB (session.db) is
// moved to the app-named path once, without clobbering an existing file.
//
// It drives migrateLegacySession with explicit temp paths instead of
// touching the real config dir: adrg/xdg caches XDG_CONFIG_HOME at package
// init, so t.Setenv can't redirect it, and a test writing to the user's
// real session storage would destroy their WhatsApp link.
func TestMigrateLegacySession(t *testing.T) {
	newBase := filepath.Join(t.TempDir(), "wash.session.db")
	if _, err := os.Stat(newBase); err == nil {
		t.Fatal("new session should not exist yet")
	}

	if err := os.WriteFile(filepath.Join(filepath.Dir(newBase), "session.db"), []byte("legacy-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(newBase); err == nil {
		t.Fatal("new session should not exist yet")
	}

	migrateLegacySession(newBase)

	data, err := os.ReadFile(newBase)
	if err != nil {
		t.Fatalf("migrated session missing: %v", err)
	}
	if string(data) != "legacy-data" {
		t.Fatalf("migrated content = %q, want legacy-data", data)
	}

	if err := os.WriteFile(newBase, []byte("newer"), 0o644); err != nil {
		t.Fatal(err)
	}
	migrateLegacySession(newBase)
	data, err = os.ReadFile(newBase)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "newer" {
		t.Fatalf("second run clobbered session: %q", data)
	}
}
