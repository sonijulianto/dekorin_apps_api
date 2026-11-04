package handlers

import (
	"testing"
)

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
