package handler

import "testing"

func TestArchivePreviewMIME(t *testing.T) {
	tests := map[string]string{
		".pdf":  "application/pdf",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}
	for ext, want := range tests {
		if got := archivePreviewMIME(ext); got != want {
			t.Fatalf("archivePreviewMIME(%q) = %q, want %q", ext, got, want)
		}
	}
	if got := archivePreviewMIME(".gif"); got != "application/octet-stream" {
		t.Fatalf("unsupported extension MIME = %q", got)
	}
}
