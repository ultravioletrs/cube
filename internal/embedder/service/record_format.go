package service

import (
	"path/filepath"
	"strings"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
)

var codeFileExtensions = map[string]struct{}{
	".go": {}, ".js": {}, ".ts": {}, ".tsx": {}, ".jsx": {},
	".py": {}, ".java": {}, ".rs": {}, ".c": {}, ".cpp": {},
	".h": {}, ".hpp": {}, ".cs": {}, ".php": {}, ".rb": {},
	".kt": {}, ".swift": {}, ".sh": {}, ".sql": {}, ".yaml": {},
	".yml": {}, ".json": {}, ".toml": {}, ".xml": {}, ".css": {},
}

var textFileFormats = map[string]domain.RecordFormat{
	".md":   domain.RecordFormatMD,
	".txt":  domain.RecordFormatText,
	".pdf":  domain.RecordFormatPDF,
	".docx": domain.RecordFormatDOCX,
	".html": domain.RecordFormatText,
	".htm":  domain.RecordFormatText,
}

var imageFileExtensions = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".bmp": {}, ".tif": {}, ".tiff": {}, ".svg": {},
}

// DetectRecordFormat maps filename/content-type into a logical record format.
func DetectRecordFormat(fileName, mimeType string) domain.RecordFormat {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	if _, ok := codeFileExtensions[ext]; ok {
		return domain.RecordFormatCode
	}
	if _, ok := imageFileExtensions[ext]; ok {
		return domain.RecordFormatImage
	}
	if format, ok := textFileFormats[ext]; ok {
		return format
	}

	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "application/pdf":
		return domain.RecordFormatPDF
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return domain.RecordFormatDOCX
	case "text/markdown":
		return domain.RecordFormatMD
	case "text/plain", "application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet", "application/vnd.google-apps.presentation":
		return domain.RecordFormatText
	case "text/html":
		return domain.RecordFormatText
	default:
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/") {
			return domain.RecordFormatImage
		}
		return domain.RecordFormatLink
	}
}
