package usecase

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Budi Santoso":        "budi-santoso",
		"Budi & Rekan, S.H.":  "budi-rekan-s-h",
		"  Firma   Hukum  ":   "firma-hukum",
		"ACME":                "acme",
		"---":                 "",
		"Law123":              "law123",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, mau %q", in, got, want)
		}
	}
}

// U7b: 2 firma nama sama -> slug beda (suffix).
func TestUniqueSlugCollision(t *testing.T) {
	taken := map[string]bool{"budi-santoso": true}
	got, err := uniqueSlug("Budi Santoso", func(s string) bool { return taken[s] })
	if err != nil {
		t.Fatal(err)
	}
	if got == "budi-santoso" || !hasPrefix(got, "budi-santoso-") {
		t.Errorf("slug bentrok harus dapat suffix, dapat %q", got)
	}
}

func TestUniqueSlugEmptyName(t *testing.T) {
	got, err := uniqueSlug("!!!", func(string) bool { return false })
	if err != nil || got != "org" {
		t.Errorf("nama full-simbol -> fallback 'org', dapat %q err=%v", got, err)
	}
}

func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
