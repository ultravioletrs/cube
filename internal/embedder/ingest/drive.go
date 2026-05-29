// Copyright (c) Ultraviolet
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ultravioletrs/cube/internal/embedder/domain"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"rsc.io/pdf"
)

const (
	driveScope        = "https://www.googleapis.com/auth/drive.readonly"
	driveFileMaxBytes = 200 << 20
)

var (
	driveFilesURL     = "https://www.googleapis.com/drive/v3/files"
	driveExportURLFmt = "https://www.googleapis.com/drive/v3/files/%s/export"
	driveDownloadURL  = "https://www.googleapis.com/drive/v3/files/%s?alt=media"
)

// SetDriveAPIEndpoints overrides Drive API endpoints and returns a restore function.
// Intended for tests that need deterministic HTTP fixtures.
func SetDriveAPIEndpoints(filesURL, exportURLFmt, downloadURLFmt string) func() {
	prevFilesURL := driveFilesURL
	prevExportURLFmt := driveExportURLFmt
	prevDownloadURLFmt := driveDownloadURL

	driveFilesURL = filesURL
	driveExportURLFmt = exportURLFmt
	driveDownloadURL = downloadURLFmt

	return func() {
		driveFilesURL = prevFilesURL
		driveExportURLFmt = prevExportURLFmt
		driveDownloadURL = prevDownloadURLFmt
	}
}

// DriveFile is a single file entry from the Drive files.list API.
type DriveFile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	MimeType     string   `json:"mimeType"`
	Version      string   `json:"version"`
	ModifiedTime string   `json:"modifiedTime"`
	WebViewLink  string   `json:"webViewLink"`
	Parents      []string `json:"parents"`
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

// DriveReader lists and downloads files from Google Drive.
type DriveReader struct {
	httpClient *http.Client
}

// NewDriveReaderFromJSON creates a DriveReader authenticated with a service account JSON key.
func NewDriveReaderFromJSON(ctx context.Context, saJSON []byte) (*DriveReader, error) {
	creds, err := google.CredentialsFromJSON(ctx, saJSON, driveScope)
	if err != nil {
		return nil, fmt.Errorf("parse service account credentials: %w", err)
	}
	return &DriveReader{
		httpClient: oauth2.NewClient(ctx, creds.TokenSource),
	}, nil
}

// NewDriveReaderFromConfig creates a DriveReader from a Google Drive source config.
func NewDriveReaderFromConfig(ctx context.Context, cfg domain.GoogleDriveConfig) (*DriveReader, error) {
	if cfg.ServiceAccountJSON != "" {
		credJSON, err := decodeCredentialJSON(cfg.ServiceAccountJSON)
		if err != nil {
			return nil, err
		}
		return NewDriveReaderFromJSON(ctx, credJSON)
	}

	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("google drive config requires service_account_json or access_token")
	}

	token := &oauth2.Token{
		AccessToken:  cfg.AccessToken,
		RefreshToken: cfg.RefreshToken,
	}

	var tokenSource oauth2.TokenSource
	if cfg.RefreshToken != "" && cfg.ClientID != "" && cfg.ClientSecret != "" {
		// Prefer the provided access token first. Some Google apps can return
		// unauthorized_client for refresh while a fresh access token is valid.
		// We keep refresh credentials attached so refresh remains available when
		// token expiry metadata is present.
		oauthCfg := &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{driveScope},
		}
		tokenSource = oauthCfg.TokenSource(ctx, token)
	} else {
		tokenSource = oauth2.StaticTokenSource(token)
	}

	return &DriveReader{
		httpClient: oauth2.NewClient(ctx, tokenSource),
	}, nil
}

func decodeCredentialJSON(raw string) ([]byte, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty service account json")
	}
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return []byte(raw), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil {
		return decoded, nil
	}

	decoded, err = base64.RawStdEncoding.DecodeString(raw)
	if err == nil {
		return decoded, nil
	}

	return nil, fmt.Errorf("decode service account json: %w", err)
}

// ListFiles returns all Drive files accessible to the token, optionally
// restricted to a single folder. Binary files (PDFs, DOCX) and Google
// Workspace files are both included.
func (d *DriveReader) ListFiles(ctx context.Context, folderID string) ([]DriveFile, error) {
	q := "trashed=false and mimeType!='application/vnd.google-apps.folder'"
	if folderID != "" {
		q = fmt.Sprintf("'%s' in parents and (%s)", folderID, q)
	}

	var all []DriveFile
	pageToken := ""
	for {
		params := url.Values{
			"q":        {q},
			"fields":   {"nextPageToken,files(id,name,mimeType,version,modifiedTime,webViewLink,parents)"},
			"pageSize": {"1000"},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, driveFilesURL+"?"+params.Encode(), http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("drive list files request: %w", err)
		}
		resp, err := d.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("drive list files: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)

			_ = resp.Body.Close()

			return nil, fmt.Errorf("drive list files status %d: %s", resp.StatusCode, body)
		}

		var page struct {
			Files         []DriveFile `json:"files"`
			NextPageToken string      `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			_ = resp.Body.Close()

			return nil, fmt.Errorf("drive list files decode: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("drive list files close body: %w", err)
		}
		for _, file := range page.Files {
			if supportsDriveFile(file) {
				all = append(all, file)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return all, nil
}

// ListFilesRecursive returns supported files contained in folderID and all of
// its descendant folders.
func (d *DriveReader) ListFilesRecursive(ctx context.Context, folderID string) ([]DriveFile, error) {
	rootID := strings.TrimSpace(folderID)
	if rootID == "" {
		return d.ListFiles(ctx, "")
	}

	queue := []string{rootID}
	seenFolders := map[string]struct{}{rootID: {}}
	filesByID := make(map[string]DriveFile)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		folders, files, err := d.ListFolderContent(ctx, current)
		if err != nil {
			return nil, err
		}
		for _, folder := range folders {
			if _, ok := seenFolders[folder.ID]; ok {
				continue
			}
			seenFolders[folder.ID] = struct{}{}
			queue = append(queue, folder.ID)
		}
		for _, file := range files {
			filesByID[file.ID] = file
		}
	}

	out := make([]DriveFile, 0, len(filesByID))
	for _, file := range filesByID {
		out = append(out, file)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// ListFolderContent returns direct children of a folder split into folders and
// supported files. Empty folderID resolves to Drive root.
func (d *DriveReader) ListFolderContent(ctx context.Context, folderID string) ([]DriveFile, []DriveFile, error) {
	parentID := strings.TrimSpace(folderID)
	if parentID == "" {
		parentID = "root"
	}
	q := fmt.Sprintf("'%s' in parents and trashed=false", parentID)

	var folders []DriveFile
	var files []DriveFile
	pageToken := ""
	for {
		params := url.Values{
			"q":        {q},
			"fields":   {"nextPageToken,files(id,name,mimeType,version,modifiedTime,webViewLink,parents)"},
			"pageSize": {"1000"},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, driveFilesURL+"?"+params.Encode(), http.NoBody)
		if err != nil {
			return nil, nil, fmt.Errorf("drive list folder content request: %w", err)
		}
		resp, err := d.httpClient.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("drive list folder content: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)

			_ = resp.Body.Close()

			return nil, nil, fmt.Errorf("drive list folder content status %d: %s", resp.StatusCode, body)
		}

		var page struct {
			Files         []DriveFile `json:"files"`
			NextPageToken string      `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			_ = resp.Body.Close()

			return nil, nil, fmt.Errorf("drive list folder content decode: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			return nil, nil, fmt.Errorf("drive list folder content close body: %w", err)
		}
		for _, file := range page.Files {
			if strings.EqualFold(strings.TrimSpace(file.MimeType), "application/vnd.google-apps.folder") {
				folders = append(folders, file)
				continue
			}
			if supportsDriveFile(file) {
				files = append(files, file)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	sort.Slice(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	return folders, files, nil
}

// SearchFolders returns folders matching name query. If scopeFolderID is
// provided, search is restricted to that folder hierarchy root (direct and
// nested descendants are returned by Drive search semantics).
func (d *DriveReader) SearchFolders(ctx context.Context, scopeFolderID, nameQuery string) ([]DriveFile, error) {
	query := "mimeType='application/vnd.google-apps.folder' and trashed=false"
	scope := strings.TrimSpace(scopeFolderID)
	if scope != "" {
		query = fmt.Sprintf("'%s' in parents and (%s)", scope, query)
	}
	nameQuery = strings.TrimSpace(nameQuery)
	if nameQuery != "" {
		safe := strings.ReplaceAll(nameQuery, "'", "\\'")
		query = fmt.Sprintf("%s and name contains '%s'", query, safe)
	}

	var folders []DriveFile
	pageToken := ""
	for {
		params := url.Values{
			"q":        {query},
			"fields":   {"nextPageToken,files(id,name,mimeType,version,modifiedTime,webViewLink,parents)"},
			"pageSize": {"200"},
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, driveFilesURL+"?"+params.Encode(), http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("drive search folders request: %w", err)
		}
		resp, err := d.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("drive search folders: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)

			_ = resp.Body.Close()

			return nil, fmt.Errorf("drive search folders status %d: %s", resp.StatusCode, body)
		}

		var page struct {
			Files         []DriveFile `json:"files"`
			NextPageToken string      `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			_ = resp.Body.Close()

			return nil, fmt.Errorf("drive search folders decode: %w", err)
		}
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("drive search folders close body: %w", err)
		}
		for _, folder := range page.Files {
			if strings.EqualFold(strings.TrimSpace(folder.MimeType), "application/vnd.google-apps.folder") {
				folders = append(folders, folder)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	sort.Slice(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
	})
	return folders, nil
}

// DownloadFile downloads or exports a file as raw bytes.
func (d *DriveReader) DownloadFile(ctx context.Context, f DriveFile) ([]byte, error) {
	var reqURL string
	switch {
	case strings.HasPrefix(f.MimeType, "application/vnd.google-apps."):
		reqURL = fmt.Sprintf(driveExportURLFmt, f.ID) + "?mimeType=" + url.QueryEscape(googleAppsExportMIME(f.MimeType))
	default:
		reqURL = fmt.Sprintf(driveDownloadURL, f.ID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("drive download %s request: %w", f.ID, err)
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("drive download %s: %w", f.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("drive download %s status %d: %s", f.ID, resp.StatusCode, body)
	}

	lim := &io.LimitedReader{R: resp.Body, N: driveFileMaxBytes + 1}
	body, err := io.ReadAll(lim)
	if err != nil {
		return nil, fmt.Errorf("drive read %s: %w", f.ID, err)
	}
	if int64(len(body)) > driveFileMaxBytes {
		return nil, fmt.Errorf("drive file %s too large: %d bytes (max %d)", f.ID, len(body), driveFileMaxBytes)
	}
	return body, nil
}

// ExtractText normalizes Drive content into plain text.
func ExtractText(f DriveFile, content []byte) (ExtractedDocument, error) {
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

func extractImageText(f DriveFile, content []byte) ExtractedDocument {
	if text, ok := maybeImageOCR(f, content); ok {
		cfg := GetExtractionConfig().OCR
		charCount := len([]rune(condensedText(text)))
		mode := ImageIngestModeHybrid
		if charCount >= cfg.ImageOCROnlyMinTextChars {
			mode = ImageIngestModeOCR
		} else if charCount < cfg.MinTextChars {
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

func imageTextForMode(f DriveFile, ocrText string, mode ImageIngestMode) string {
	if mode == ImageIngestModeImage {
		return imageDescriptor(f, ocrText)
	}
	return ocrText
}

func imageDescriptor(f DriveFile, ocrText string) string {
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

func googleAppsExportMIME(mimeType string) string {
	switch mimeType {
	case "application/vnd.google-apps.spreadsheet":
		return "text/csv"
	default:
		return "text/plain"
	}
}

func supportsDriveFile(f DriveFile) bool {
	mime := strings.ToLower(strings.TrimSpace(f.MimeType))
	switch mime {
	case "application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet",
		"application/vnd.google-apps.presentation",
		"application/pdf",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return true
	case "image/svg+xml":
		return true
	}
	if strings.HasPrefix(mime, "image/") || isTextMIME(mime) {
		return true
	}
	return isTextFileExt(filepath.Ext(strings.TrimSpace(f.Name)))
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

// parseDriveModifiedTime parses the Drive modifiedTime RFC3339 string.
func parseDriveModifiedTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func intPtr(v int) *int {
	return &v
}
