package handlers

import (
	"strings"
	"testing"
)

func TestGetAssistantCategoryMeta(t *testing.T) {
	cases := []struct {
		raw          string
		wantNorm     string
		wantPackage  string
		wantRolePart string
	}{
		{"", "decoration", "PAKET DEKORASI TERPOPULER", "vendor dekorasi"},
		{"decoration", "decoration", "PAKET DEKORASI TERPOPULER", "vendor dekorasi"},
		{"mua", "mua", "PAKET RIAS & MUA TERPOPULER", "Make Up Artist (MUA)"},
		{"wo", "wo", "PAKET LAYANAN WO TERPOPULER", "Wedding Organizer (WO)"},
		{"photo_video", "photo_video", "PAKET FOTO & VIDEO TERPOPULER", "fotografi & videografi"},
		{"catering", "catering", "PAKET MENU KATERING TERPOPULER", "katering"},
		{"unknown_cat", "other", "PAKET LAYANAN TERPOPULER", "vendor jasa acara"},
	}

	for _, tc := range cases {
		norm, meta := getAssistantCategoryMeta(tc.raw)
		if norm != tc.wantNorm {
			t.Errorf("raw=%q: expected norm %q, got %q", tc.raw, tc.wantNorm, norm)
		}
		if meta.PackageLabel != tc.wantPackage {
			t.Errorf("raw=%q: expected package label %q, got %q", tc.raw, tc.wantPackage, meta.PackageLabel)
		}
		if !strings.Contains(meta.VendorRole, tc.wantRolePart) {
			t.Errorf("raw=%q: expected role containing %q, got %q", tc.raw, tc.wantRolePart, meta.VendorRole)
		}
	}
}

func TestBuildAssistantSystemPrompt_NonDecoration(t *testing.T) {
	prompt := buildAssistantSystemPrompt("mua", "Kategori Usaha: MUA & Rias Pengantin (mua)")
	if !strings.Contains(prompt, "Make Up Artist (MUA)") {
		t.Fatalf("expected MUA persona in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Jangan menyebut properti dekorasi atau pelaminan") {
		t.Fatalf("expected non-decoration guard in prompt, got: %s", prompt)
	}
}

func TestParseSuggestionsFromReply(t *testing.T) {
	raw := `Omset bulan ini adalah Rp 45.000.000 dengan laba bersih Rp 35.800.000.

[SUGGESTIONS: Cek jadwal bentrok | Klien belum lunas | Rincian pengeluaran]`

	cleanText, suggestions := parseSuggestionsFromReply(raw)

	expectedText := "Omset bulan ini adalah Rp 45.000.000 dengan laba bersih Rp 35.800.000."
	if cleanText != expectedText {
		t.Errorf("expected clean text '%s', got '%s'", expectedText, cleanText)
	}

	if len(suggestions) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
	}

	if suggestions[0] != "Cek jadwal bentrok" {
		t.Errorf("unexpected suggestion[0]: %s", suggestions[0])
	}
	if suggestions[1] != "Klien belum lunas" {
		t.Errorf("unexpected suggestion[1]: %s", suggestions[1])
	}
	if suggestions[2] != "Rincian pengeluaran" {
		t.Errorf("unexpected suggestion[2]: %s", suggestions[2])
	}
}
