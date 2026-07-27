package extract

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// text extractor: pdf (pdftotext, fallback OCR), docx (unzip xml), txt
type Extractor struct {
	ocr bool // OCR halaman scan; hanya jalur ingestion (lambat, jangan di jalur chat sinkron)
}

func New(ocr bool) *Extractor { return &Extractor{ocr: ocr} }

const (
	extractTimeout = 10 * time.Minute
	minPageChars   = 20 // di bawah ini dianggap halaman scan
)

// returns per-page text (1-based). non-pdf = single "page"
func (e *Extractor) Extract(path, mimeType string) ([]string, error) {
	switch {
	case strings.Contains(mimeType, "pdf"):
		return e.extractPDF(path)
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

func (e *Extractor) extractPDF(path string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), extractTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "pdftotext", "-q", path, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext: %w", err)
	}
	pages := splitPages(string(out))

	if e.ocr {
		if err := e.ocrScannedPages(ctx, path, pages); err != nil {
			return nil, err
		}
	}

	pages = trimTrailingEmpty(pages)
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdf kosong (mungkin hasil scan, butuh OCR)")
	}
	return pages, nil
}

// pdftotext pisahkan halaman dengan \f; \f penutup menyisakan elemen kosong di ekor
func splitPages(out string) []string {
	pages := strings.Split(out, "\f")
	if n := len(pages); n > 0 && strings.TrimSpace(pages[n-1]) == "" {
		pages = pages[:n-1]
	}
	return pages
}

func trimTrailingEmpty(pages []string) []string {
	for len(pages) > 0 && strings.TrimSpace(pages[len(pages)-1]) == "" {
		pages = pages[:len(pages)-1]
	}
	return pages
}

func needsOCR(page string) bool { return len(strings.TrimSpace(page)) < minPageChars }

// halaman nyaris kosong dirender 300dpi lalu dibaca tesseract; halaman ber-teks dibiarkan.
// gagal = error keras -> dokumen failed dengan alasan, jangan diam-diam kosong
func (e *Extractor) ocrScannedPages(ctx context.Context, path string, pages []string) error {
	dir, err := os.MkdirTemp("", "ocr-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for i := range pages {
		if !needsOCR(pages[i]) {
			continue
		}
		txt, err := ocrPage(ctx, path, i+1, dir)
		if err != nil {
			return fmt.Errorf("ocr hal %d: %w", i+1, err)
		}
		pages[i] = txt
	}
	return nil
}

func ocrPage(ctx context.Context, path string, pageNo int, dir string) (string, error) {
	no := strconv.Itoa(pageNo)
	prefix := filepath.Join(dir, "p"+no)
	if err := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "300", "-f", no, "-l", no, path, prefix).Run(); err != nil {
		return "", fmt.Errorf("pdftoppm: %w", err)
	}
	imgs, _ := filepath.Glob(prefix + "*.png")
	if len(imgs) == 0 {
		return "", fmt.Errorf("pdftoppm: tidak ada output")
	}
	out, err := exec.CommandContext(ctx, tesseractBin(), imgs[0], "-", "-l", "ind").Output()
	if err != nil {
		return "", fmt.Errorf("tesseract: %w", err)
	}
	return string(out), nil
}

// PATH dulu, fallback lokasi install default Windows (biar tak perlu ubah PATH user)
func tesseractBin() string {
	if _, err := exec.LookPath("tesseract"); err == nil {
		return "tesseract"
	}
	if p := `C:\Program Files\Tesseract-OCR\tesseract.exe`; fileExists(p) {
		return p
	}
	return "tesseract"
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
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
