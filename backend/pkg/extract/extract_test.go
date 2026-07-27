package extract

import (
	"strings"
	"testing"
)

func TestSplitPages(t *testing.T) {
	// \f penutup halaman terakhir tidak jadi halaman hantu
	pages := splitPages("hal1\fhal2\f")
	if len(pages) != 2 || pages[0] != "hal1" || pages[1] != "hal2" {
		t.Fatalf("got %#v", pages)
	}
	// halaman scan (kosong) di tengah dipertahankan, posisinya = nomor halaman
	pages = splitPages("hal1\f \fhal3\f")
	if len(pages) != 3 || strings.TrimSpace(pages[1]) != "" || pages[2] != "hal3" {
		t.Fatalf("halaman kosong tengah hilang: %#v", pages)
	}
}

func TestNeedsOCR(t *testing.T) {
	if !needsOCR("") || !needsOCR("  12 \n") {
		t.Fatal("halaman nyaris kosong harus kena OCR")
	}
	if needsOCR("Pasal 222 ayat (1) Undang-Undang ini mengatur permohonan PKPU.") {
		t.Fatal("halaman ber-teks jangan kena OCR")
	}
}

func TestTrimTrailingEmpty(t *testing.T) {
	pages := trimTrailingEmpty([]string{"isi", " ", ""})
	if len(pages) != 1 || pages[0] != "isi" {
		t.Fatalf("got %#v", pages)
	}
	if got := trimTrailingEmpty([]string{" ", ""}); len(got) != 0 {
		t.Fatalf("semua kosong harus habis: %#v", got)
	}
}
