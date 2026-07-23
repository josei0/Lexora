package extract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// text extractor: pdf (pdftotext), docx (unzip xml), txt
type Extractor struct{}

func New() *Extractor { return &Extractor{} }

// returns per-page text (1-based). non-pdf = single "page"
func (e *Extractor) Extract(path, mimeType string) ([]string, error) {
	switch {
	case strings.Contains(mimeType, "pdf"):
		return extractPDF(path)
	case strings.Contains(mimeType, "word") || strings.HasSuffix(path, ".docx"):
		txt, err := extractDOCX(path)
		if err != nil {
			return nil, err
		}
		return []string{txt}, nil
	default: // txt
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return []string{string(b)}, nil
	}
}

// pdftotext -layout, split pages on form feed (\f)
func extractPDF(path string) ([]string, error) {
	// -q quiet, - = stdout
	out, err := exec.Command("pdftotext", "-q", path, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext: %w", err)
	}
	pages := strings.Split(string(out), "\f")
	// trim trailing empty page
	for len(pages) > 0 && strings.TrimSpace(pages[len(pages)-1]) == "" {
		pages = pages[:len(pages)-1]
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdf kosong (mungkin hasil scan, butuh OCR)")
	}
	return pages, nil
}

var wordText = regexp.MustCompile(`<w:t[^>]*>([^<]*)</w:t>`)
var paraBreak = regexp.MustCompile(`</w:p>`)

// docx = zip; text in word/document.xml inside <w:t> tags
func extractDOCX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			rc.Close()
			return "", err
		}
		rc.Close()
		xml := paraBreak.ReplaceAllString(buf.String(), "\n")
		var sb strings.Builder
		for _, m := range wordText.FindAllStringSubmatch(xml, -1) {
			sb.WriteString(m[1])
		}
		return sb.String(), nil
	}
	return "", fmt.Errorf("word/document.xml tidak ditemukan")
}
