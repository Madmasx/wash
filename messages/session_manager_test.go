package messages

import (
	"path/filepath"
	"testing"

	"wash/config"
)

func TestMediaDirPathNestsInsideAppFolder(t *testing.T) {
	origDl := config.Config.General.DownloadPath
	origPrev := config.Config.General.PreviewPath
	defer func() {
		config.Config.General.DownloadPath = origDl
		config.Config.General.PreviewPath = origPrev
	}()
	config.Config.General.DownloadPath = "/media/DL"
	config.Config.General.PreviewPath = "/media/PV"

	want := filepath.Join("/media/DL", config.AppFolder)
	if got := mediaDirPath(false); got != want {
		t.Fatalf("mediaDirPath(false) = %q, want %q", got, want)
	}
	want = filepath.Join("/media/PV", config.AppFolder)
	if got := mediaDirPath(true); got != want {
		t.Fatalf("mediaDirPath(true) = %q, want %q", got, want)
	}
}

func TestDownloadFileNameSanitizesPathTraversal(t *testing.T) {
	msg := Message{
		Id:       "msg-1",
		FileName: "../../.ssh/authorized_keys",
	}

	got := downloadFileName(msg)
	if got != "authorized_keys" {
		t.Fatalf("expected sanitized basename, got %q", got)
	}
}

func TestDownloadFileNameSanitizesWindowsPathTraversal(t *testing.T) {
	msg := Message{
		Id:       "msg-2",
		FileName: `..\..\AppData\Roaming\startup.bat`,
	}

	got := downloadFileName(msg)
	if got != "startup.bat" {
		t.Fatalf("expected sanitized basename, got %q", got)
	}
}

func TestDownloadFileNameFallsBackForInvalidName(t *testing.T) {
	msg := Message{
		Id:       "msg-3",
		FileName: "..",
		MimeType: "image/png",
	}

	got := downloadFileName(msg)
	if got != "msg-3.png" {
		t.Fatalf("expected fallback filename, got %q", got)
	}
}
