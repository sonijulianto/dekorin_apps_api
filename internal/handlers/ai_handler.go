package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ScanReceiptResult struktur data hasil ekstraksi nota dari Gemini AI
type ScanReceiptResult struct {
	Title    string  `json:"title"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Date     string  `json:"date"`
	Notes    string  `json:"notes"`
}

// GeminiPart bagian dari konten request Gemini API
type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents         []geminiContent       `json:"contents"`
	GenerationConfig geminiGenConfig       `json:"generationConfig"`
}

type geminiGenConfig struct {
	ResponseMimeType string        `json:"responseMimeType"`
	ResponseSchema   *geminiSchema `json:"responseSchema,omitempty"`
}

type geminiSchema struct {
	Type       string                   `json:"type"`
	Properties map[string]*geminiSchema `json:"properties,omitempty"`
	Enum       []string                 `json:"enum,omitempty"`
	Required   []string                 `json:"required,omitempty"`
}

type geminiCandidatePart struct {
	Text string `json:"text"`
}

type geminiCandidateContent struct {
	Parts []geminiCandidatePart `json:"parts"`
}

type geminiCandidate struct {
	Content geminiCandidateContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// ScanReceiptAI handles POST /api/ai/scan-receipt
// Menerima file gambar nota/struk (multipart/form-data) dan mengekstrak rincian pengeluaran menggunakan Gemini Vision AI
func ScanReceiptAI(c *fiber.Ctx) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "GEMINI_API_KEY belum dikonfigurasi di server backend.",
		})
	}

	// Ambil file dari form-data (bisa key 'receipt', 'file', atau 'image')
	fileHeader, err := c.FormFile("receipt")
	if err != nil {
		fileHeader, err = c.FormFile("file")
		if err != nil {
			fileHeader, err = c.FormFile("image")
		}
	}

	if err != nil || fileHeader == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Silakan unggah foto nota atau struk pengeluaran (key: receipt)",
		})
	}

	// Batasi ukuran file (maksimal 15 MB)
	if fileHeader.Size > 15*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Ukuran file foto terlalu besar (maksimal 15 MB)",
		})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca file foto yang diunggah",
		})
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memproses data file foto",
		})
	}

	// Deteksi MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".webp":
			mimeType = "image/webp"
		case ".heic":
			mimeType = "image/heic"
		default:
			mimeType = http.DetectContentType(fileBytes)
		}
	}

	// Panggil Gemini Vision API
	result, err := callGeminiReceiptOCR(apiKey, fileBytes, mimeType)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menganalisis nota dengan AI: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Nota berhasil dianalisis dengan AI",
		"data":    result,
	})
}

// callGeminiReceiptOCR mengeksekusi request multimodal ke Gemini API dengan fallback model
func callGeminiReceiptOCR(apiKey string, imageBytes []byte, mimeType string) (*ScanReceiptResult, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)

	promptText := `Kamu adalah asisten OCR cerdas untuk Dekorin (aplikasi manajemen vendor dekorasi acara/wedding/event).
Tugasmu adalah membaca dan mengekstrak informasi pengeluaran dari foto nota/struk belanja/kuitansi/bukti transaksi yang diberikan.

Ketentuan ekstraksi:
1. title: Judul singkat dan jelas mengenai apa yang dibeli atau dibayar, sertakan nama toko/tempat jika terbaca. Contoh: "Bunga Mawar & Baby Breath (Toko Berkah)" atau "Sewa Mobil Pick-up".
2. amount: Total nominal uang akhir yang harus dibayarkan (angka murni tanpa titik/koma/simbol Rp). Jika ada diskon, gunakan total netto/akhir yang dibayar.
3. category: WAJIB pilih salah satu dari:
   - "Bunga": Pembelian bunga mawar, melati, baby breath, daun ruskus, bunga segar, dsb.
   - "Properti Dekor": Pembelian/sewa kain backdrop, lampu hias, karpet, styrofoam, paku, lem tembak, papan, bunga artifisial, dsb.
   - "Operasional": Konsumsi/makan minum kru, lakban, plastik sampah, print dokumen, kabel, baterai, ATK, perlengkapan umum lapangan.
   - "Transport": Pembelian bensin, tol, tiket parkir, sewa armada truk/pick-up, ongkir pengiriman properti.
   - "Honor Crew": Pembayaran fee/upah freelance, kru pasang/bongkar, tukang bunga, supir, dsb.
   - "Lainnya": Pengeluaran lain yang tidak termasuk kategori di atas.
4. date: Tanggal transaksi dalam format "YYYY-MM-DD". Jika tahun tidak tertera, asumsikan tahun saat ini (` + time.Now().Format("2006") + `). Jika tanggal sama sekali tidak ada atau tidak terbaca, gunakan tanggal hari ini (` + time.Now().Format("2006-01-02") + `).
5. notes: Rincian singkat daftar barang/jasa yang dibeli beserta kuantitas/harga satuan jika terlihat.`

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: promptText},
					{
						InlineData: &geminiInlineData{
							MimeType: mimeType,
							Data:     base64Image,
						},
					},
				},
			},
		},
		GenerationConfig: geminiGenConfig{
			ResponseMimeType: "application/json",
			ResponseSchema: &geminiSchema{
				Type: "OBJECT",
				Properties: map[string]*geminiSchema{
					"title":    {Type: "STRING"},
					"amount":   {Type: "NUMBER"},
					"category": {Type: "STRING", Enum: []string{"Bunga", "Properti Dekor", "Operasional", "Transport", "Honor Crew", "Lainnya"}},
					"date":     {Type: "STRING"},
					"notes":    {Type: "STRING"},
				},
				Required: []string{"title", "amount", "category", "date"},
			},
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request error: %w", err)
	}

	// Model prioritas: gunakan GEMINI_MODEL dari .env atau fallback ke gemini-3.5-flash-lite, lalu gemini-3.5-flash, lalu gemini-3.8-flash
	primaryModel := os.Getenv("GEMINI_MODEL")
	if primaryModel == "" {
		primaryModel = "gemini-3.5-flash-lite"
	}

	modelsToTry := []string{primaryModel}
	if primaryModel != "gemini-3.5-flash-lite" {
		modelsToTry = append(modelsToTry, "gemini-3.5-flash-lite")
	}
	modelsToTry = append(modelsToTry, "gemini-3.5-flash", "gemini-3.8-flash")

	var lastErr error
	client := &http.Client{Timeout: 30 * time.Second}

	for _, model := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
		if err != nil {
			lastErr = err
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			// Jika 503 (high demand) atau 429 atau 404, coba model berikutnya
			lastErr = fmt.Errorf("model %s returned HTTP %d: %s", model, resp.StatusCode, string(bodyBytes))
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusNotFound {
				continue
			}
			break
		}

		var gemResp geminiResponse
		if err := json.Unmarshal(bodyBytes, &gemResp); err != nil {
			lastErr = fmt.Errorf("gagal parse response Gemini: %w", err)
			continue
		}

		if gemResp.Error != nil {
			lastErr = fmt.Errorf("gemini API error: %s", gemResp.Error.Message)
			continue
		}

		if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("gemini tidak menghasilkan kandidat jawaban")
			continue
		}

		rawText := gemResp.Candidates[0].Content.Parts[0].Text
		var result ScanReceiptResult
		if err := json.Unmarshal([]byte(rawText), &result); err != nil {
			lastErr = fmt.Errorf("gagal parse JSON hasil nota dari Gemini: %w (raw: %s)", err, rawText)
			continue
		}

		// Normalisasi fallback jika kategori kosong atau tidak dikenali
		if result.Category == "" {
			result.Category = "Lainnya"
		}
		if result.Date == "" {
			result.Date = time.Now().Format("2006-01-02")
		}

		return &result, nil
	}

	return nil, fmt.Errorf("semua model AI gagal: %w", lastErr)
}
