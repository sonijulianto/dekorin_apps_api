package utils

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
)

func createTestFileHeader(filename string, data []byte) (*multipart.FileHeader, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="file"; filename="` + filename + `"`},
	})
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(body.Len()))
	if err != nil {
		return nil, err
	}
	return form.File["file"][0], nil
}

func TestProcessAndSaveImage_LargeJPEG(t *testing.T) {
	// Buat gambar buatan ukuran 2000 x 1000 pixel
	img := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 2000; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 150, B: 100, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("Gagal encode dummy JPEG: %v", err)
	}

	header, err := createTestFileHeader("large_test.jpg", buf.Bytes())
	if err != nil {
		t.Fatalf("Gagal buat header multipart: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "test_upload_*")
	if err != nil {
		t.Fatalf("Gagal buat temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	savedName, err := ProcessAndSaveImage(header, tempDir, "test")
	if err != nil {
		t.Fatalf("ProcessAndSaveImage gagal: %v", err)
	}

	savedPath := filepath.Join(tempDir, savedName)
	savedFile, err := os.Open(savedPath)
	if err != nil {
		t.Fatalf("Gagal buka file tersimpan: %v", err)
	}
	defer savedFile.Close()

	decodedImg, _, err := image.Decode(savedFile)
	if err != nil {
		t.Fatalf("Gagal decode file tersimpan: %v", err)
	}

	bounds := decodedImg.Bounds()
	if bounds.Dx() > 1600 {
		t.Errorf("Lebar gambar melebihi 1600px: dapat %d", bounds.Dx())
	}
	if bounds.Dx() != 1600 {
		t.Errorf("Harusnya lebar tepat 1600px: dapat %d", bounds.Dx())
	}
	if bounds.Dy() != 800 {
		t.Errorf("Harusnya tinggi tepat 800px: dapat %d", bounds.Dy())
	}
}

func TestProcessAndSaveImage_TransparentPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	img.Set(50, 50, color.NRGBA{R: 255, G: 0, B: 0, A: 128}) // Transparan

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("Gagal encode dummy PNG: %v", err)
	}

	header, err := createTestFileHeader("icon_test.png", buf.Bytes())
	if err != nil {
		t.Fatalf("Gagal buat header multipart: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "test_upload_png_*")
	if err != nil {
		t.Fatalf("Gagal buat temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	savedName, err := ProcessAndSaveImage(header, tempDir, "icon")
	if err != nil {
		t.Fatalf("ProcessAndSaveImage gagal: %v", err)
	}

	if filepath.Ext(savedName) != ".png" {
		t.Errorf("Harusnya mempertahankan ekstensi .png untuk gambar transparan, dapat %s", filepath.Ext(savedName))
	}
}
