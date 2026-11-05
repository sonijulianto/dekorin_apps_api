package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDecorSimulationPrompt(t *testing.T) {
	req := SimulateDecorRequest{
		FabricColor: "Sage Green & Putih Tulang",
		FlowerColor: "Mawar Putih, Peach & Daun Eucalyptus",
		Nuance:      "Elegant Rustic",
		Notes:       "Hilangkan bunga biru pada lengkungan kiri",
	}

	prompt := buildDecorSimulationPrompt(req)
	if !strings.Contains(prompt, "Sage Green & Putih Tulang") {
		t.Errorf("expected fabric color in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Mawar Putih, Peach & Daun Eucalyptus") {
		t.Errorf("expected flower color in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Elegant Rustic") {
		t.Errorf("expected nuance in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "Hilangkan bunga biru pada lengkungan kiri") {
		t.Errorf("expected notes in prompt, got: %s", prompt)
	}
}

func TestLoadReferenceImageFromLocalUploads(t *testing.T) {
	tmpDir := t.TempDir()
	dummyFile := filepath.Join(tmpDir, "sample_decor.jpg")
	dummyBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	if err := os.WriteFile(dummyFile, dummyBytes, 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	loaded, mime, err := loadReferenceImageFromURL("https://dekorin.my.id/uploads/sample_decor.jpg", tmpDir)
	if err != nil {
		t.Fatalf("expected local upload load to succeed, got error: %v", err)
	}
	if len(loaded) != len(dummyBytes) {
		t.Errorf("expected %d bytes, got %d", len(dummyBytes), len(loaded))
	}
	if mime != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", mime)
	}
}

func TestLoadReferenceImageBlocksLoopbackSSRF(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := loadReferenceImageFromURL("http://127.0.0.1:8080/internal-secret.jpg", tmpDir)
	if err == nil {
		t.Fatalf("expected loopback URL to be blocked")
	}
}
