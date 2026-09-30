package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"wash/config"
	"wash/messages"
)

func TestMediaCommandForPath(t *testing.T) {
	dir := t.TempDir()

	img := filepath.Join(dir, "foto.png")
	vid := filepath.Join(dir, "clip.MP4")
	aud := filepath.Join(dir, "audio.mp3")
	doc := filepath.Join(dir, "informe.pdf")
	for _, f := range []string{img, vid, aud, doc} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		text string
		cmd  string
	}{
		{img, "sendimage"},
		{vid, "sendvideo"},
		{aud, "sendaudio"},
		{doc, "upload"},
		{`"` + img + `"`, "sendimage"}, // quoted path (some file managers)
		{`'` + aud + `'`, "sendaudio"},
	}
	for _, c := range cases {
		cmd, path, ok := mediaCommandForPath(c.text)
		if !ok || cmd != c.cmd {
			t.Errorf("mediaCommandForPath(%q) = %q,%q,%v; want cmd %q", c.text, cmd, path, ok, c.cmd)
		}
	}

	// Non-file inputs must not be treated as attachments.
	for _, text := range []string{
		"no existe esta ruta",
		"/help",
		"/commands",
		"hola que tal",
		"",
	} {
		if cmd, _, ok := mediaCommandForPath(text); ok {
			t.Errorf("mediaCommandForPath(%q) = cmd %q, want not-ok", text, cmd)
		}
	}
	// Directories are not attachable.
	if cmd, _, ok := mediaCommandForPath(dir); ok {
		t.Errorf("mediaCommandForPath(dir) = cmd %q, want not-ok", cmd)
	}
}

func TestContactMentions(t *testing.T) {
	contactList = []messages.Contact{
		{Id: "5215512345678@s.whatsapp.net", Name: "Juan Pérez", Short: "Juan"},
		{Id: "5215522222222@s.whatsapp.net", Name: "Ana María", Short: "Ana"},
		{Id: "5215533333333@s.whatsapp.net", Name: "Julieta", Short: ""},
	}
	t.Cleanup(func() { contactList = nil })

	// --- contactSuggestions ---
	cases := []struct {
		text string
		want []string
	}{
		{"@ju", []string{"@Juan Pérez", "@Julieta"}},
		{"hola @ju", []string{"hola @Juan Pérez", "hola @Julieta"}},
		{"@ana", []string{"@Ana María"}},
		{"@", nil},          // empty term: no suggestions yet
		{"@ju ", nil},       // trailing space no longer matches any name
		{"sin arroba", nil}, // no @ at all
		{"@zzz", nil},       // no match
		// Multi-word names keep the drop-down open (needed for arrow-key
		// navigation, since tview fills the field with the full name).
		{"@Juan Pérez", []string{"@Juan Pérez"}},
		{"@ana maría", []string{"@Ana María"}},
		{"hola @Juan Pérez qué tal", nil}, // text after the name closes it
	}
	for _, c := range cases {
		got := contactSuggestions(c.text)
		if len(got) != len(c.want) {
			t.Errorf("contactSuggestions(%q) = %v, want %v", c.text, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("contactSuggestions(%q)[%d] = %q, want %q", c.text, i, got[i], c.want[i])
			}
		}
	}

	// --- chatForMention ---
	if chat, ok := chatForMention("juan pérez"); !ok || chat.Id != "5215512345678@s.whatsapp.net" {
		t.Errorf("chatForMention(juan pérez) = %+v,%v; want JID match", chat, ok)
	}
	if chat, ok := chatForMention("Juan Pérez"); !ok || chat.Id != "5215512345678@s.whatsapp.net" {
		t.Errorf("chatForMention(Juan Pérez) = %+v,%v; want multi-word match", chat, ok)
	}
	if chat, ok := chatForMention("ANA MARÍA"); !ok || chat.Id != "5215522222222@s.whatsapp.net" {
		t.Errorf("chatForMention(ANA MARÍA) = %+v,%v; want case-insensitive match", chat, ok)
	}
	if _, ok := chatForMention("nadie"); ok {
		t.Errorf("chatForMention(nadie) ok = true, want false")
	}

	// --- expandMentions ---
	expand := []struct{ in, want string }{
		{"hola @Juan Pérez", "hola @5215512345678"},
		{"Hola @ana maría, ¿todo bien?", "Hola @5215522222222, ¿todo bien?"},
		{"sin menciones aquí", "sin menciones aquí"},
		{"@NoExiste", "@NoExiste"},
		{"@Juan Pérez y @julieta", "@5215512345678 y @5215533333333"},
	}
	for _, c := range expand {
		if got := expandMentions(c.in); got != c.want {
			t.Errorf("expandMentions(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCommandSuggestions(t *testing.T) {
	t.Cleanup(func() { config.Config.General.CmdPrefix = "/" })

	// "/" alone offers every supported command, prefixed.
	all := commandSuggestions("/")
	if len(all) != len(messages.AvailableCommands()) {
		t.Errorf("commandSuggestions(/) = %d entries, want %d", len(all), len(messages.AvailableCommands()))
	}
	for _, cmd := range messages.AvailableCommands() {
		if !contains(all, "/"+cmd) {
			t.Errorf("commandSuggestions(/) missing /%s", cmd)
		}
	}

	cases := []struct {
		text string
		want []string
	}{
		{"/send", []string{"/send", "/sendimage", "/sendvideo", "/sendaudio"}},
		{"/SEND", []string{"/send", "/sendimage", "/sendvideo", "/sendaudio"}}, // case-insensitive
		{"/back", []string{"/backlog"}},
		{"/send ", nil}, // space means arguments follow: no suggestions
		{"/xyz", nil},   // unknown command
		{"/lang", []string{"/lang"}},
	}
	for _, c := range cases {
		got := commandSuggestions(c.text)
		if len(got) != len(c.want) {
			t.Errorf("commandSuggestions(%q) = %v, want %v", c.text, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("commandSuggestions(%q)[%d] = %q, want %q", c.text, i, got[i], c.want[i])
			}
		}
	}

	// A custom command prefix is honored.
	config.Config.General.CmdPrefix = ":"
	if got := commandSuggestions(":hel"); len(got) != 1 || got[0] != ":help" {
		t.Errorf("commandSuggestions(:hel) = %v, want [:help]", got)
	}
	config.Config.General.CmdPrefix = "/"

	// inputSuggestions dispatches: commands for prefix, contacts for @, else nil.
	contactList = []messages.Contact{{Id: "1@s.whatsapp.net", Name: "Juan", Short: ""}}
	t.Cleanup(func() { contactList = nil })
	if got := inputSuggestions("/send"); len(got) == 0 {
		t.Errorf("inputSuggestions(/send) = %v, want command suggestions", got)
	}
	if got := inputSuggestions("@ju"); len(got) == 0 {
		t.Errorf("inputSuggestions(@ju) = %v, want contact suggestions", got)
	}
	if got := inputSuggestions("hola"); got != nil {
		t.Errorf("inputSuggestions(hola) = %v, want nil", got)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestClipboardImageHelpers(t *testing.T) {
	// pickClipboardImageType picks the first image/* target.
	got := pickClipboardImageType([]string{"TIMESTAMP", "TARGETS", "image/png", "text/plain;charset=utf-8"})
	if got != "image/png" {
		t.Errorf("pickClipboardImageType = %q, want image/png", got)
	}
	got = pickClipboardImageType([]string{"TARGETS", "text/plain;charset=utf-8"})
	if got != "" {
		t.Errorf("pickClipboardImageType(text-only) = %q, want empty", got)
	}
	got = pickClipboardImageType([]string{"image/jpeg", "image/png"})
	if got != "image/jpeg" {
		t.Errorf("pickClipboardImageType order = %q, want image/jpeg", got)
	}

	// writeTempImage writes the bytes with the right extension and cleans up.
	path, err := writeTempImage([]byte{0x89, 'P', 'N', 'G', 0x0d}, "png")
	if err != nil {
		t.Fatalf("writeTempImage(png): %v", err)
	}
	if !strings.HasSuffix(path, ".png") {
		t.Errorf("writeTempImage png path = %q, want .png suffix", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) != 5 || data[1] != 'P' {
		t.Errorf("writeTempImage content mismatch: %v %v", data, err)
	}
	os.Remove(path)

	path, err = writeTempImage([]byte("xx"), "jpeg")
	if err != nil {
		t.Fatalf("writeTempImage(jpeg): %v", err)
	}
	if !strings.HasSuffix(path, ".jpg") {
		t.Errorf("writeTempImage jpeg path = %q, want .jpg suffix", path)
	}
	os.Remove(path)

	if _, err := writeTempImage(nil, "png"); err == nil {
		t.Errorf("writeTempImage(empty) = nil error, want error")
	}
}

// TestApplyQRText verifies the pairing QR code is appended below the existing
// text, replaced in place on refresh, and removed once pairing finishes.
func TestApplyQRText(t *testing.T) {
	chat := "lin1\nlin2"

	// first QR goes below the text
	body, last := applyQRText(chat, "", "QR1")
	if body != chat+"\nQR1" || last != "QR1" {
		t.Fatalf("primer QR: body=%q last=%q", body, last)
	}

	// rotation replaces QR1 in place, text above untouched
	body, last = applyQRText(body, last, "QR2")
	if body != chat+"\nQR2" || last != "QR2" {
		t.Fatalf("rotacion: body=%q last=%q", body, last)
	}

	// more chat text arrives, QR stays as the tail block
	body = body + "\nlin3"

	// pairing finished: block removed, text intact
	body, last = applyQRText(body, last, "")
	if body != chat+"\nlin3" || last != "" {
		t.Fatalf("fin: body=%q last=%q", body, last)
	}

	// nothing before and no QR: empty body
	body, last = applyQRText("", "", "")
	if body != "" || last != "" {
		t.Fatalf("vacio: body=%q last=%q", body, last)
	}
}

// TestMediaNumbering verifies /show N resolves the per-chat attachment
// numbering: media messages get 1-based numbers that restart per chat and per
// screen rebuild.
func TestMediaNumbering(t *testing.T) {
	mediaIndexByChat = make(map[string][]string)
	resetChatMediaIndex("chatA")

	if n := assignMediaNumber("chatA", "id1"); n != 1 {
		t.Fatalf("primer numero = %d, want 1", n)
	}
	if n := assignMediaNumber("chatA", "id2"); n != 2 {
		t.Fatalf("segundo numero = %d, want 2", n)
	}
	if n := assignMediaNumber("chatA", "id3"); n != 3 {
		t.Fatalf("tercer numero = %d, want 3", n)
	}

	if id, ok := resolveMediaNumber("chatA", 2); !ok || id != "id2" {
		t.Fatalf("resolve 2 = %q, %v; want id2,true", id, ok)
	}
	if _, ok := resolveMediaNumber("chatA", 0); ok {
		t.Fatal("resolve 0 debe fallar")
	}
	if _, ok := resolveMediaNumber("chatA", 4); ok {
		t.Fatal("resolve 4 debe fallar (solo hay 3)")
	}

	// numbering resets per chat
	if n := assignMediaNumber("chatB", "oid"); n != 1 {
		t.Fatalf("otro chat empieza en 1, got %d", n)
	}

	// screen rebuild restarts the chat numbering
	resetChatMediaIndex("chatA")
	if n := assignMediaNumber("chatA", "id1"); n != 1 {
		t.Fatalf("tras rebuild = %d, want 1", n)
	}
}

// TestMediaTagFor verifies only media messages get the [#N] marker.
func TestMediaTagFor(t *testing.T) {
	mediaIndexByChat = make(map[string][]string)
	resetChatMediaIndex("chatA")

	img := messages.Message{Id: "i1", ChatId: "chatA", Kind: messages.MessageKindImage}
	if tag := mediaTagFor(&img); tag != "[::b][#1][::-] " {
		t.Fatalf("tag imagen = %q", tag)
	}
	for i, kind := range []messages.MessageKind{
		messages.MessageKindVideo,
		messages.MessageKindAudio,
		messages.MessageKindDocument,
	} {
		want := "[::b][#" + strconv.Itoa(i+2) + "][::-] "
		vid := messages.Message{Id: "m" + string(kind), ChatId: "chatA", Kind: kind}
		if tag := mediaTagFor(&vid); tag != want {
			t.Fatalf("tag %s = %q, want %q", kind, tag, want)
		}
	}
	txt := messages.Message{Id: "t1", ChatId: "chatA", Kind: messages.MessageKindText}
	if tag := mediaTagFor(&txt); tag != "" {
		t.Fatalf("tag texto = %q, want ''", tag)
	}
}
