// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"strings"
	"sync/atomic"
	"time"
)

// OCRConfig controls OCR preprocessing behavior during extraction.
type OCRConfig struct {
	Enabled            bool
	ImageEnabled       bool
	PDFFallbackEnabled bool
	Language           string
	Binary             string
	PDFRenderBinary    string
	Timeout            time.Duration
	MinTextChars       int
	MaxPDFPages        int
}

// ExtractionConfig controls extraction-level behavior shared by all providers.
type ExtractionConfig struct {
	OCR OCRConfig
}

var extractionConfig atomic.Value

func init() {
	extractionConfig.Store(defaultExtractionConfig())
}

func defaultExtractionConfig() ExtractionConfig {
	return ExtractionConfig{
		OCR: OCRConfig{
			Enabled:            false,
			ImageEnabled:       true,
			PDFFallbackEnabled: true,
			Language:           "eng",
			Binary:             "tesseract",
			PDFRenderBinary:    "pdftoppm",
			Timeout:            2 * time.Minute,
			MinTextChars:       40,
			MaxPDFPages:        20,
		},
	}
}

// SetExtractionConfig updates process-wide extraction settings.
func SetExtractionConfig(cfg ExtractionConfig) {
	extractionConfig.Store(normalizeExtractionConfig(cfg))
}

// GetExtractionConfig returns the current extraction settings.
func GetExtractionConfig() ExtractionConfig {
	cfg, ok := extractionConfig.Load().(ExtractionConfig)
	if !ok {
		return defaultExtractionConfig()
	}
	return cfg
}

func normalizeExtractionConfig(cfg ExtractionConfig) ExtractionConfig {
	def := defaultExtractionConfig()
	ocr := cfg.OCR
	if strings.TrimSpace(ocr.Language) == "" {
		ocr.Language = def.OCR.Language
	}
	if strings.TrimSpace(ocr.Binary) == "" {
		ocr.Binary = def.OCR.Binary
	}
	if strings.TrimSpace(ocr.PDFRenderBinary) == "" {
		ocr.PDFRenderBinary = def.OCR.PDFRenderBinary
	}
	if ocr.Timeout <= 0 {
		ocr.Timeout = def.OCR.Timeout
	}
	if ocr.MinTextChars <= 0 {
		ocr.MinTextChars = def.OCR.MinTextChars
	}
	if ocr.MaxPDFPages <= 0 {
		ocr.MaxPDFPages = def.OCR.MaxPDFPages
	}
	return ExtractionConfig{OCR: ocr}
}
