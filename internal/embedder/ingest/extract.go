// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	stdmime "mime"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"rsc.io/pdf"
)

// FileMeta is the source-agnostic file description needed to extract text.
// All providers (Drive, S3, Microsoft, local) construct it instead of leaking
// a provider-specific type into the shared extraction pipeline.
type FileMeta struct {
	ID       string
	Name     string
	MimeType string
}

// ImageIngestMode describes which signals should be indexed for an image.
type ImageIngestMode string

const (
	ImageIngestModeNone   ImageIngestMode = ""
	ImageIngestModeOCR    ImageIngestMode = "ocr_only"
	ImageIngestModeImage  ImageIngestMode = "image_only"
	ImageIngestModeHybrid ImageIngestMode = "hybrid"
)

// ExtractedDocument is normalized text plus metadata captured during extraction.
type ExtractedDocument struct {
	Text             string
	PageCount        *int
	MimeType         string
	ImageMode        ImageIngestMode
	OCRText          string
	OCRTextCharCount int
}

// ExtractText normalizes file content into plain text.
func ExtractText(f FileMeta, content []byte) (ExtractedDocument, error) {
	f.MimeType = normalizeFileMetaMIMEType(f, content)
	mime := strings.ToLower(strings.TrimSpace(f.MimeType))

	var doc ExtractedDocument
	switch {
	case strings.HasPrefix(mime, "application/vnd.google-apps."):
		doc = ExtractedDocument{Text: string(content)}
	case mime == "application/pdf":
		var err error
		doc, err = extractPDF(content)
		if err != nil {
			return ExtractedDocument{}, err
		}
	case mime == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		var err error
		doc, err = extractDOCX(content)
		if err != nil {
			return ExtractedDocument{}, err
		}
	case mime == "image/svg+xml":
		doc = ExtractedDocument{Text: string(content)}
	case strings.HasPrefix(mime, "image/"):
		doc = extractImageText(f, content)
	case isPlainTextLike(f.Name, mime):
		doc = ExtractedDocument{Text: string(content)}
	default:
		return ExtractedDocument{}, fmt.Errorf("unsupported MIME type for extraction: %q", f.MimeType)
	}
	doc.MimeType = f.MimeType
	return doc, nil
}

func normalizeFileMetaMIMEType(f FileMeta, content []byte) string {
	mimeType := NormalizeFileMIMEType(f.Name, f.MimeType)
	mimeType = NormalizeFileMIMEType(f.ID, mimeType)
	if inferred := mimeTypeFromContent(content); inferred != "" && (mimeType == "" || isGenericBinaryMIME(mimeType)) {
		return inferred
	}
	return mimeType
}

// NormalizeFileMIMEType returns a media type usable by the extraction pipeline.
// Object stores often omit content-type metadata, or return the generic
// application/octet-stream type, so fall back to stable extension mappings.
func NormalizeFileMIMEType(fileName, mimeType string) string {
	normalized := normalizeMediaType(mimeType)
	if normalized != "" && !isGenericBinaryMIME(normalized) {
		return normalized
	}
	if inferred := mimeTypeFromExtension(fileName); inferred != "" {
		return inferred
	}
	return normalized
}

func normalizeMediaType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	mediaType, _, err := stdmime.ParseMediaType(value)
	if err == nil {
		return strings.ToLower(strings.TrimSpace(mediaType))
	}
	return value
}

func isGenericBinaryMIME(mimeType string) bool {
	switch mimeType {
	case "application/octet-stream", "binary/octet-stream", "application/x-binary":
		return true
	default:
		return false
	}
}

func mimeTypeFromExtension(fileName string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(strings.TrimSpace(fileName))))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".md", ".markdown":
		return "text/markdown"
	case ".html", ".htm":
		return "text/html"
	case ".txt":
		return "text/plain"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".tif", ".tiff":
		return "image/tiff"
	}

	if ext == "" {
		return ""
	}
	if inferred := normalizeMediaType(stdmime.TypeByExtension(ext)); inferred != "" && !isGenericBinaryMIME(inferred) {
		return inferred
	}
	if isTextFileExt(ext) {
		return "text/plain"
	}
	return ""
}

func mimeTypeFromContent(content []byte) string {
	if len(content) == 0 {
		return ""
	}
	if isDOCXContent(content) {
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}

	sniff := content
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	detected := normalizeMediaType(http.DetectContentType(sniff))
	if detected == "" || isGenericBinaryMIME(detected) {
		return ""
	}
	return detected
}

func isDOCXContent(content []byte) bool {
	if len(content) < 4 || string(content[:2]) != "PK" {
		return false
	}

	reader := bytes.NewReader(content)
	archive, err := zip.NewReader(reader, int64(len(content)))
	if err != nil {
		return false
	}
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			return true
		}
	}
	return false
}

func extractImageText(f FileMeta, content []byte) ExtractedDocument {
	if text, ok := maybeImageOCR(f, content); ok {
		cfg := GetExtractionConfig().OCR
		charCount := len([]rune(condensedText(text)))
		mode := ImageIngestModeHybrid
		if charCount < cfg.MinTextChars {
			mode = ImageIngestModeImage
		}
		return ExtractedDocument{
			Text:             imageTextForMode(f, text, mode),
			ImageMode:        mode,
			OCRText:          text,
			OCRTextCharCount: charCount,
		}
	}

	return ExtractedDocument{Text: imageDescriptor(f, ""), ImageMode: ImageIngestModeImage}
}

func imageTextForMode(f FileMeta, ocrText string, mode ImageIngestMode) string {
	if mode == ImageIngestModeImage {
		return imageDescriptor(f, ocrText)
	}
	return ocrText
}

func imageDescriptor(f FileMeta, ocrText string) string {
	name := strings.TrimSpace(f.Name)
	if name == "" {
		name = "unnamed-image"
	}
	mime := strings.TrimSpace(f.MimeType)
	if mime == "" {
		mime = "image/unknown"
	}

	// Image-only records still need a small text chunk so the record can be
	// indexed in the existing text chunk table while the visual vector is stored
	// separately in image_embeddings.
	text := fmt.Sprintf("image file: %s; mime_type: %s", name, mime)
	if strings.TrimSpace(ocrText) != "" {
		text += "; detected_text: " + condensedText(ocrText)
	}
	return text
}

func extractPDF(content []byte) (ExtractedDocument, error) {
	text, err := pdfToText(content)
	if err != nil {
		return ExtractedDocument{}, fmt.Errorf("extract pdf text: %w", err)
	}
	if fallbackText, ok := maybePDFFallbackOCR(content, text); ok {
		text = fallbackText
	}
	return ExtractedDocument{
		Text:      text,
		PageCount: pdfPageCount(content),
	}, nil
}

// pdfToText uses pdftotext (poppler) for accurate Unicode text extraction.
func pdfToText(content []byte) (string, error) {
	cmd := exec.Command("pdftotext", "-", "-")
	cmd.Stdin = bytes.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return strings.ReplaceAll(string(out), "\x00", ""), nil
}

// pdfPageCount returns the page count using rsc.io/pdf, or nil on failure.
func pdfPageCount(content []byte) *int {
	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil
	}
	n := reader.NumPage()
	return &n
}

func extractDOCX(content []byte) (ExtractedDocument, error) {
	readerAt := bytes.NewReader(content)
	archive, err := zip.NewReader(readerAt, int64(len(content)))
	if err != nil {
		return ExtractedDocument{}, fmt.Errorf("open docx zip: %w", err)
	}

	var documentXML []byte
	for _, file := range archive.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return ExtractedDocument{}, fmt.Errorf("open docx xml: %w", err)
		}
		documentXML, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return ExtractedDocument{}, fmt.Errorf("read docx xml: %w", err)
		}
		break
	}
	if len(documentXML) == 0 {
		return ExtractedDocument{}, fmt.Errorf("word/document.xml not found in docx")
	}

	decoder := xml.NewDecoder(bytes.NewReader(documentXML))
	var (
		paragraphs []string
		builder    strings.Builder
		inText     bool
	)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ExtractedDocument{}, fmt.Errorf("decode docx xml: %w", err)
		}

		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "t":
				inText = true
			case "tab":
				builder.WriteByte('\t')
			case "br":
				builder.WriteByte('\n')
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "t":
				inText = false
			case "p":
				if paragraph := strings.TrimSpace(builder.String()); paragraph != "" {
					paragraphs = append(paragraphs, paragraph)
				}
				builder.Reset()
			}
		case xml.CharData:
			if inText {
				builder.Write(tok)
			}
		}
	}

	return ExtractedDocument{Text: strings.Join(paragraphs, "\n\n")}, nil
}

func isPlainTextLike(fileName, mime string) bool {
	return isTextMIME(mime) || isTextFileExt(filepath.Ext(strings.TrimSpace(fileName)))
}

func isTextMIME(mime string) bool {
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	switch mime {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/typescript",
		"application/x-sh",
		"application/sql",
		"application/x-sql",
		"application/yaml",
		"application/x-yaml",
		"application/toml",
		"application/x-toml":
		return true
	default:
		return false
	}
}

func isTextFileExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".txt", ".md", ".html", ".htm",
		".go", ".js", ".ts", ".tsx", ".jsx",
		".py", ".java", ".rs", ".c", ".cpp",
		".h", ".hpp", ".cs", ".php", ".rb",
		".kt", ".swift", ".sh", ".sql", ".yaml",
		".yml", ".json", ".toml", ".xml", ".css":
		return true
	default:
		return false
	}
}
