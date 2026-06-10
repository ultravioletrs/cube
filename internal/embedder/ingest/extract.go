// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
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
	ImageMode        ImageIngestMode
	OCRText          string
	OCRTextCharCount int
}

// ExtractText normalizes file content into plain text.
func ExtractText(f FileMeta, content []byte) (ExtractedDocument, error) {
	mime := strings.ToLower(strings.TrimSpace(f.MimeType))

	switch {
	case strings.HasPrefix(mime, "application/vnd.google-apps."):
		return ExtractedDocument{Text: string(content)}, nil
	case mime == "application/pdf":
		return extractPDF(content)
	case mime == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return extractDOCX(content)
	case mime == "image/svg+xml":
		return ExtractedDocument{Text: string(content)}, nil
	case strings.HasPrefix(mime, "image/"):
		return extractImageText(f, content), nil
	case isPlainTextLike(f.Name, mime):
		return ExtractedDocument{Text: string(content)}, nil
	default:
		return ExtractedDocument{}, fmt.Errorf("unsupported MIME type for extraction: %q", f.MimeType)
	}
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
