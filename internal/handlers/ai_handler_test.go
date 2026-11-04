package handlers

import (
	"encoding/json"
	"testing"
)

func TestScanReceiptResultUnmarshal(t *testing.T) {
	rawJSON := `{
		"title": "Beli Mawar & Melati (Pasar Bunga Rawa Belong)",
		"amount": 250000,
		"category": "Bunga",
		"date": "2026-11-04",
		"notes": "10 ikat mawar merah, 2 kantong melati"
	}`

	var result ScanReceiptResult
	err := json.Unmarshal([]byte(rawJSON), &result)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Title != "Beli Mawar & Melati (Pasar Bunga Rawa Belong)" {
		t.Errorf("unexpected title: %s", result.Title)
	}
	if result.Amount != 250000 {
		t.Errorf("unexpected amount: %f", result.Amount)
	}
	if result.Category != "Bunga" {
		t.Errorf("unexpected category: %s", result.Category)
	}
	if result.Date != "2026-11-04" {
		t.Errorf("unexpected date: %s", result.Date)
	}
}
