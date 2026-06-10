// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"strings"
	"testing"
	"time"
)

func TestExtractTextImageUsesOCRWhenEnabled(t *testing.T) {
	restoreCfg := setTestExtractionConfig(ExtractionConfig{
		OCR: OCRConfig{
			Enabled:                  true,
			ImageEnabled:             true,
			PDFFallbackEnabled:       true,
			Language:                 "eng",
			Binary:                   "tesseract",
			PDFRenderBinary:          "pdftoppm",
			Timeout:                  time.Second,
			MinTextChars:             5,
			ImageOCROnlyMinTextChars: 1200,
			MaxPDFPages:              5,
		},
	})
	defer restoreCfg()

	restoreOCR := setTestOCRRunners(
		func(_ []byte, _ OCRConfig) (string, error) { return "OCR image text", nil },
		runPDFOCRFunc,
	)
	defer restoreOCR()

	doc, err := ExtractText(FileMeta{
		Name:     "scan.png",
		MimeType: "image/png",
	}, []byte{0x89, 0x50, 0x4e, 0x47})
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}
	if got := strings.TrimSpace(doc.Text); got != "OCR image text" {
		t.Fatalf("expected OCR image text, got %q", doc.Text)
	}
	if doc.ImageMode != ImageIngestModeHybrid {
		t.Fatalf("expected hybrid mode, got %q", doc.ImageMode)
	}
}

func TestExtractTextImageUsesHybridForLargeText(t *testing.T) {
	restoreCfg := setTestExtractionConfig(ExtractionConfig{
		OCR: OCRConfig{
			Enabled:                  true,
			ImageEnabled:             true,
			PDFFallbackEnabled:       true,
			Language:                 "eng",
			Binary:                   "tesseract",
			PDFRenderBinary:          "pdftoppm",
			Timeout:                  time.Second,
			MinTextChars:             40,
			ImageOCROnlyMinTextChars: 80,
			MaxPDFPages:              5,
		},
	})
	defer restoreCfg()

	longText := strings.Repeat("invoice line item total amount ", 5)
	restoreOCR := setTestOCRRunners(
		func(_ []byte, _ OCRConfig) (string, error) { return longText, nil },
		runPDFOCRFunc,
	)
	defer restoreOCR()

	doc, err := ExtractText(FileMeta{
		Name:     "scan.png",
		MimeType: "image/png",
	}, []byte{0x89, 0x50, 0x4e, 0x47})
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}
	if doc.ImageMode != ImageIngestModeHybrid {
		t.Fatalf("expected hybrid mode, got %q", doc.ImageMode)
	}
	if got := strings.TrimSpace(doc.Text); got != strings.TrimSpace(longText) {
		t.Fatalf("expected OCR text, got %q", got)
	}
}

func TestExtractTextImageKeepsShortOCRAsDescriptor(t *testing.T) {
	restoreCfg := setTestExtractionConfig(ExtractionConfig{
		OCR: OCRConfig{
			Enabled:                  true,
			ImageEnabled:             true,
			PDFFallbackEnabled:       true,
			Language:                 "eng",
			Binary:                   "tesseract",
			PDFRenderBinary:          "pdftoppm",
			Timeout:                  time.Second,
			MinTextChars:             40,
			ImageOCROnlyMinTextChars: 1200,
			MaxPDFPages:              5,
		},
	})
	defer restoreCfg()

	restoreOCR := setTestOCRRunners(
		func(_ []byte, _ OCRConfig) (string, error) { return "SALE 50%", nil },
		runPDFOCRFunc,
	)
	defer restoreOCR()

	doc, err := ExtractText(FileMeta{
		Name:     "poster.png",
		MimeType: "image/png",
	}, []byte{0x89, 0x50, 0x4e, 0x47})
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}
	if doc.ImageMode != ImageIngestModeImage {
		t.Fatalf("expected image-only mode, got %q", doc.ImageMode)
	}
	if !strings.Contains(doc.Text, "detected_text: SALE 50%") {
		t.Fatalf("expected descriptor to include short OCR text, got %q", doc.Text)
	}
}

func TestPDFFallbackOCRTriggeredOnSmallText(t *testing.T) {
	restoreCfg := setTestExtractionConfig(ExtractionConfig{
		OCR: OCRConfig{
			Enabled:            true,
			ImageEnabled:       true,
			PDFFallbackEnabled: true,
			Language:           "eng",
			Binary:             "tesseract",
			PDFRenderBinary:    "pdftoppm",
			Timeout:            time.Second,
			MinTextChars:       40,
			MaxPDFPages:        5,
		},
	})
	defer restoreCfg()

	restoreOCR := setTestOCRRunners(
		runImageOCRFunc,
		func(_ []byte, _ OCRConfig) (string, error) { return "OCR PDF text", nil },
	)
	defer restoreOCR()

	text, ok := maybePDFFallbackOCR([]byte("fake-pdf"), "tiny")
	if !ok {
		t.Fatalf("expected OCR fallback to trigger")
	}
	if text != "OCR PDF text" {
		t.Fatalf("expected OCR PDF text, got %q", text)
	}
}

func setTestExtractionConfig(cfg ExtractionConfig) func() {
	prev := GetExtractionConfig()
	SetExtractionConfig(cfg)
	return func() {
		SetExtractionConfig(prev)
	}
}

func setTestOCRRunners(
	image func([]byte, OCRConfig) (string, error),
	pdf func([]byte, OCRConfig) (string, error),
) func() {
	prevImage := runImageOCRFunc
	prevPDF := runPDFOCRFunc
	runImageOCRFunc = image
	runPDFOCRFunc = pdf
	return func() {
		runImageOCRFunc = prevImage
		runPDFOCRFunc = prevPDF
	}
}
