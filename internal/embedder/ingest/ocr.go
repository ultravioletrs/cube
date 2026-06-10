// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var (
	runImageOCRFunc = runImageOCR
	runPDFOCRFunc   = runPDFOCR
)

func maybeImageOCR(file FileMeta, content []byte) (string, bool) {
	cfg := GetExtractionConfig().OCR
	if !cfg.Enabled || !cfg.ImageEnabled {
		return "", false
	}
	if len(content) == 0 {
		return "", false
	}

	text, err := runImageOCRFunc(content, cfg)
	if err != nil {
		return "", false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}

	_ = file // keep signature consistent if later we add file-aware OCR prompts/metadata.
	return text, true
}

func maybePDFFallbackOCR(content []byte, extractedText string) (string, bool) {
	cfg := GetExtractionConfig().OCR
	if !cfg.Enabled || !cfg.PDFFallbackEnabled {
		return extractedText, false
	}
	if len(condensedText(extractedText)) >= cfg.MinTextChars {
		return extractedText, false
	}

	text, err := runPDFOCRFunc(content, cfg)
	if err != nil {
		return extractedText, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return extractedText, false
	}
	return text, true
}

func runImageOCR(content []byte, cfg OCRConfig) (string, error) {
	runCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, cfg.Binary, "stdin", "stdout", "-l", cfg.Language, "quiet")
	cmd.Stdin = bytes.NewReader(content)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("image ocr: %w", err)
	}
	return string(out), nil
}

func runPDFOCR(content []byte, cfg OCRConfig) (string, error) {
	workDir, err := os.MkdirTemp("", "cube-ocr-pdf-*")
	if err != nil {
		return "", fmt.Errorf("pdf ocr make temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	pdfPath := filepath.Join(workDir, "input.pdf")
	if err := os.WriteFile(pdfPath, content, 0o600); err != nil {
		return "", fmt.Errorf("pdf ocr write temp pdf: %w", err)
	}

	// Convert selected PDF pages into PNG files for OCR.
	renderCtx, renderCancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer renderCancel()
	prefix := filepath.Join(workDir, "page")
	renderCmd := exec.CommandContext(
		renderCtx,
		cfg.PDFRenderBinary,
		"-png",
		"-f",
		"1",
		"-l",
		fmt.Sprintf("%d", cfg.MaxPDFPages),
		pdfPath,
		prefix,
	)
	if out, err := renderCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdf ocr render pages: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	matches, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return "", fmt.Errorf("pdf ocr list rendered pages: %w", err)
	}
	if len(matches) == 0 {
		return "", nil
	}
	sort.Strings(matches)

	var all strings.Builder
	for i, img := range matches {
		runCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		cmd := exec.CommandContext(runCtx, cfg.Binary, img, "stdout", "-l", cfg.Language, "quiet")
		out, err := cmd.Output()
		cancel()
		if err != nil {
			continue
		}
		part := strings.TrimSpace(string(out))
		if part == "" {
			continue
		}
		if i > 0 && all.Len() > 0 {
			all.WriteString("\n\n")
		}
		all.WriteString(part)
	}
	return all.String(), nil
}

func condensedText(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}
