package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // Daftarkan webp decoder
	xdraw "golang.org/x/image/draw"
)

const (
	MaxImageDimension = 1600 // Lebar/tinggi maksimal foto undangan dalam pixel
	JPEGQuality       = 82   // Kualitas kompresi optimal (visual tajam, ukuran file sangat kecil ~150-250KB)
)

// ProcessAndSaveImage mengoptimasi (resize proporsional & kompresi) file gambar secara otomatis
// Mengembalikan: nama file tersimpan, error (jika gagal)
func ProcessAndSaveImage(fileHeader *multipart.FileHeader, targetDir string, prefix string) (string, error) {
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("gagal membuat direktori upload: %w", err)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("gagal membuka file upload: %w", err)
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp"

	if prefix == "" {
		prefix = "img"
	}

	// Jika bukan format gambar umum (misal audio mp3), simpan langsung apa adanya
	if !isImage {
		filename := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().Unix(), uuid.New().String()[:8], ext)
		savePath := filepath.Join(targetDir, filename)
		out, err := os.Create(savePath)
		if err != nil {
			return "", err
		}
		defer out.Close()
		if _, err := io.Copy(out, src); err != nil {
			return "", err
		}
		return filename, nil
	}

	// Baca seluruh data ke memori buffer untuk decode
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, src); err != nil {
		return "", fmt.Errorf("gagal membaca buffer gambar: %w", err)
	}

	// Decode gambar (mendukung JPEG, PNG, dan WebP)
	img, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		// Fallback aman: jika decode gagal (misal format non-standar), simpan file asli
		filename := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().Unix(), uuid.New().String()[:8], ext)
		savePath := filepath.Join(targetDir, filename)
		if err := os.WriteFile(savePath, buf.Bytes(), 0644); err != nil {
			return "", err
		}
		return filename, nil
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	targetW := origW
	targetH := origH

	// Hitung resize proporsional jika resolusi melebihi batas maksimal (1600px)
	if origW > MaxImageDimension || origH > MaxImageDimension {
		if origW >= origH {
			targetW = MaxImageDimension
			targetH = int(float64(origH) * float64(MaxImageDimension) / float64(origW))
		} else {
			targetH = MaxImageDimension
			targetW = int(float64(origW) * float64(MaxImageDimension) / float64(origH))
		}
		if targetW < 1 {
			targetW = 1
		}
		if targetH < 1 {
			targetH = 1
		}
	}

	// Lakukan scaling jika ukuran berubah
	var processedImg image.Image = img
	if targetW != origW || targetH != origH {
		// Gunakan NRGBA jika format PNG transparan
		if format == "png" && hasTransparency(img) {
			dst := image.NewNRGBA(image.Rect(0, 0, targetW, targetH))
			xdraw.BiLinear.Scale(dst, dst.Rect, img, bounds, draw.Over, nil)
			processedImg = dst
		} else {
			dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
			xdraw.BiLinear.Scale(dst, dst.Rect, img, bounds, draw.Over, nil)
			processedImg = dst
		}
	}

	// Simpan gambar hasil optimasi
	// - Jika PNG transparan: simpan sebagai .png
	// - Selain itu (JPEG, WebP, PNG tanpa transparansi): simpan sebagai .jpg dengan quality 82 (hemat 90%+ ukuran file)
	var finalExt string
	var outBuf bytes.Buffer

	if format == "png" && hasTransparency(img) {
		finalExt = ".png"
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&outBuf, processedImg); err != nil {
			return "", fmt.Errorf("gagal encode PNG: %w", err)
		}
	} else {
		finalExt = ".jpg"
		opts := &jpeg.Options{Quality: JPEGQuality}
		if err := jpeg.Encode(&outBuf, processedImg, opts); err != nil {
			return "", fmt.Errorf("gagal encode JPEG: %w", err)
		}
	}

	// Jika karena alasan tertentu hasil optimasi lebih besar dari aslinya, gunakan file asli
	if outBuf.Len() > buf.Len() && ext != "" {
		filename := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().Unix(), uuid.New().String()[:8], ext)
		savePath := filepath.Join(targetDir, filename)
		if err := os.WriteFile(savePath, buf.Bytes(), 0644); err != nil {
			return "", err
		}
		return filename, nil
	}

	filename := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().Unix(), uuid.New().String()[:8], finalExt)
	savePath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(savePath, outBuf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("gagal menyimpan file optimasi ke disk: %w", err)
	}

	return filename, nil
}

// SaveOptimizedImageBytes mengoptimasi dan menyimpan byte gambar mentah (misal hasil generate Gemini AI) ke disk
func SaveOptimizedImageBytes(rawBytes []byte, targetDir string, prefix string) (string, error) {
	if len(rawBytes) == 0 {
		return "", fmt.Errorf("data byte gambar kosong")
	}
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("gagal membuat direktori upload: %w", err)
	}
	if prefix == "" {
		prefix = "sim"
	}

	img, format, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		// Fallback aman: simpan apa adanya sebagai .jpg/.png sesuai deteksi sederhana
		ext := ".jpg"
		if len(rawBytes) > 8 && bytes.Equal(rawBytes[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
			ext = ".png"
		}
		filename := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().Unix(), uuid.New().String()[:8], ext)
		savePath := filepath.Join(targetDir, filename)
		if err := os.WriteFile(savePath, rawBytes, 0644); err != nil {
			return "", err
		}
		return filename, nil
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()

	targetW := origW
	targetH := origH

	if origW > MaxImageDimension || origH > MaxImageDimension {
		if origW >= origH {
			targetW = MaxImageDimension
			targetH = int(float64(origH) * float64(MaxImageDimension) / float64(origW))
		} else {
			targetH = MaxImageDimension
			targetW = int(float64(origW) * float64(MaxImageDimension) / float64(origH))
		}
		if targetW < 1 {
			targetW = 1
		}
		if targetH < 1 {
			targetH = 1
		}
	}

	var processedImg image.Image = img
	if targetW != origW || targetH != origH {
		if format == "png" && hasTransparency(img) {
			dst := image.NewNRGBA(image.Rect(0, 0, targetW, targetH))
			xdraw.BiLinear.Scale(dst, dst.Rect, img, bounds, draw.Over, nil)
			processedImg = dst
		} else {
			dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
			xdraw.BiLinear.Scale(dst, dst.Rect, img, bounds, draw.Over, nil)
			processedImg = dst
		}
	}

	var finalExt string
	var outBuf bytes.Buffer
	if format == "png" && hasTransparency(img) {
		finalExt = ".png"
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&outBuf, processedImg); err != nil {
			return "", fmt.Errorf("gagal encode PNG hasil AI: %w", err)
		}
	} else {
		finalExt = ".jpg"
		opts := &jpeg.Options{Quality: 86}
		if err := jpeg.Encode(&outBuf, processedImg, opts); err != nil {
			return "", fmt.Errorf("gagal encode JPEG hasil AI: %w", err)
		}
	}

	filename := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().Unix(), uuid.New().String()[:8], finalExt)
	savePath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(savePath, outBuf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("gagal menyimpan gambar hasil AI ke disk: %w", err)
	}

	return filename, nil
}

// hasTransparency mengecek apakah gambar PNG memiliki pixel transparan (alpha < 255)
func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()
	// Sampling 20 baris secara acak/berkala untuk efisiensi CPU
	stepY := bounds.Dy() / 20
	if stepY < 1 {
		stepY = 1
	}
	stepX := bounds.Dx() / 20
	if stepX < 1 {
		stepX = 1
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return true
			}
		}
	}
	return false
}
