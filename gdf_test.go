package main

import (
	"bytes"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
)

func TestParseArgsRequiresSearchTerms(t *testing.T) {
	_, _, err := parseArgs(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseArgsRejectsInvalidPageSize(t *testing.T) {
	_, _, err := parseArgs([]string{"-page-size", "1001", "quarterly"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEscapeDriveQueryString(t *testing.T) {
	got := escapeDriveQueryString(`Rick's \ Report`)
	want := `Rick\'s \\ Report`
	if got != want {
		t.Fatalf("escapeDriveQueryString() = %q, want %q", got, want)
	}
}

func TestPrintFiles(t *testing.T) {
	var out bytes.Buffer
	printFiles(&out, []*drive.File{
		{
			Name:         "Quarterly Report.pdf",
			MimeType:     "application/pdf",
			Size:         1048576,
			ModifiedTime: "2026-06-05T12:00:00Z",
		},
		{
			Name:         "Reports",
			MimeType:     "application/vnd.google-apps.folder",
			ModifiedTime: "2026-06-05T12:00:00Z",
		},
	})

	got := out.String()
	for _, want := range []string{"Quarterly Report.pdf", "Reports", "folder", "2 files found."} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q does not contain %q", got, want)
		}
	}
}
