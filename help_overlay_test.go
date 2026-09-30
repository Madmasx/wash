package main

import (
	"strings"
	"testing"
)

func TestHelpOverlayNoDuplication(t *testing.T) {
	buildTestView(120, 40)
	renderHelpOverlay(PrintHelp)

	nav, qr := applyQRText(textView.GetText(false), "", "[::b][#1][::-] QRBLOCK")
	lastQRText = qr
	textView.SetText(nav)

	renderHelpOverlay(PrintHelp, PrintCommands)
	got := textView.GetText(false)
	if !strings.Contains(got, "QRBLOCK") {
		t.Fatal("QR perdido tras renderizar ayuda")
	}
	if strings.Count(got, "Teclas:") != 1 {
		t.Fatalf("help duplicado: %d 'Teclas:' ->\n%s", strings.Count(got, "Teclas:"), got)
	}
	if strings.Count(got, "Comandos:") != 1 {
		t.Fatalf("commands duplicado: %d 'Comandos:'", strings.Count(got, "Comandos:"))
	}
	if strings.Count(got, "QRBLOCK") != 1 {
		t.Fatal("QR duplicado/perdido")
	}

	renderHelpOverlay(PrintHelp, PrintCommands)
	got = textView.GetText(false)
	if strings.Count(got, "Teclas:") != 1 || strings.Count(got, "Comandos:") != 1 {
		t.Fatalf("repetir Ctrl+p acumula: Teclas=%d Comandos=%d", strings.Count(got, "Teclas:"), strings.Count(got, "Comandos:"))
	}
	if strings.Count(got, "QRBLOCK") != 1 {
		t.Fatal("QR perdido al repetir ayuda")
	}

	renderHelpOverlay(PrintHelp)
	got = textView.GetText(false)
	if strings.Count(got, "Teclas:") != 1 || strings.Count(got, "Comandos:") != 0 {
		t.Fatalf("/help tras Ctrl+p: Teclas=%d Comandos=%d", strings.Count(got, "Teclas:"), strings.Count(got, "Comandos:"))
	}
	if strings.Count(got, "QRBLOCK") != 1 {
		t.Fatal("QR perdido tras /help")
	}
}
