package transport

import "testing"

func TestParseDriveFolderID(t *testing.T) {
	tests := []struct {
		name       string
		folderID   string
		folderLink string
		want       string
	}{
		{
			name:       "explicit folder id",
			folderID:   "abc123",
			folderLink: "https://drive.google.com/drive/folders/xyz",
			want:       "abc123",
		},
		{
			name:       "folder id from link",
			folderLink: "https://drive.google.com/drive/folders/xyz987?usp=sharing",
			want:       "xyz987",
		},
		{
			name:       "raw input falls back",
			folderLink: "root",
			want:       "root",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDriveFolderID(tc.folderID, tc.folderLink)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
