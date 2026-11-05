package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// SimulateDecorRequest struktur input JSON/Form untuk simulasi warna dekorasi AI
type SimulateDecorRequest struct {
	ImageURL    string `json:"image_url"`
	ModelName   string `json:"model_name"`
	FabricColor string `json:"fabric_color"`
	FlowerColor string `json:"flower_color"`
	Nuance      string `json:"nuance"`
	Notes       string `json:"notes"`
}

// geminiImageGenConfig konfigurasi khusus multimodal output gambar Gemini API
type geminiImageGenConfig struct {
	ResponseModalities []string `json:"responseModalities"`
}

type geminiImageRequest struct {
	Contents         []geminiContent      `json:"contents"`
	GenerationConfig geminiImageGenConfig `json:"generationConfig"`
}

type geminiImageResponsePart struct {
	Text            string            `json:"text,omitempty"`
	InlineData      *geminiInlineData `json:"inlineData,omitempty"`
	InlineDataSnake *struct {
		MimeType string `json:"mime_type"`
		Data     string `json:"data"`
	} `json:"inline_data,omitempty"`
}

type geminiImageResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiImageResponsePart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason,omitempty"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// SimulateDecorAI handles POST /api/ai/simulate-decor
// Menerima foto referensi (upload multipart atau image_url katalog) beserta catatan custom warna kain & bunga,
// lalu menghasilkan gambar simulasi dekorasi menggunakan Gemini Image Editing AI.
func SimulateDecorAI(c *fiber.Ctx) error {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if apiKey == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "GEMINI_API_KEY belum dikonfigurasi di server backend.",
		})
	}

	var reqInput SimulateDecorRequest

	// Dukung pembacaan baik dari multipart/form-data maupun application/json
	contentType := strings.ToLower(c.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		_ = c.BodyParser(&reqInput)
	} else {
		reqInput.ImageURL = strings.TrimSpace(c.FormValue("image_url"))
		reqInput.ModelName = strings.TrimSpace(c.FormValue("model_name"))
		reqInput.FabricColor = strings.TrimSpace(c.FormValue("fabric_color"))
		reqInput.FlowerColor = strings.TrimSpace(c.FormValue("flower_color"))
		reqInput.Nuance = strings.TrimSpace(c.FormValue("nuance"))
		reqInput.Notes = strings.TrimSpace(c.FormValue("notes"))
	}

	reqInput.ImageURL = strings.TrimSpace(reqInput.ImageURL)
	reqInput.ModelName = strings.TrimSpace(reqInput.ModelName)
	reqInput.FabricColor = strings.TrimSpace(reqInput.FabricColor)
	reqInput.FlowerColor = strings.TrimSpace(reqInput.FlowerColor)
	reqInput.Nuance = strings.TrimSpace(reqInput.Nuance)
	reqInput.Notes = strings.TrimSpace(reqInput.Notes)

	if reqInput.FabricColor == "" && reqInput.FlowerColor == "" && reqInput.Nuance == "" && reqInput.Notes == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Silakan isi minimal satu kustomisasi (warna kain, warna bunga, nuansa, atau catatan khusus).",
		})
	}

	uploadsDir := "./uploads"
	proto := strings.TrimSpace(c.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = c.Protocol()
	}
	baseHostURL := fmt.Sprintf("%s://%s", proto, c.Hostname())

	var sourceBytes []byte
	var sourceMime string
	var originalPublicURL string

	// 1. Cek apakah ada file gambar yang diunggah langsung
	fileHeader, err := c.FormFile("image")
	if err != nil || fileHeader == nil {
		fileHeader, _ = c.FormFile("file")
	}

	if fileHeader != nil {
		if fileHeader.Size > 15*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Ukuran foto referensi terlalu besar (maksimal 15 MB).",
			})
		}

		// Simpan foto referensi yang sudah dioptimasi ke ./uploads agar memiliki URL permanen
		savedRefName, saveErr := utils.ProcessAndSaveImage(fileHeader, uploadsDir, "ref")
		if saveErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menyimpan foto referensi: " + saveErr.Error(),
			})
		}

		savedPath := filepath.Join(uploadsDir, savedRefName)
		sourceBytes, err = os.ReadFile(savedPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal membaca foto referensi yang diunggah.",
			})
		}
		sourceMime = detectImageMime(savedRefName, sourceBytes)
		originalPublicURL = fmt.Sprintf("%s/uploads/%s", baseHostURL, savedRefName)
	} else if reqInput.ImageURL != "" {
		// 2. Gunakan foto dari URL katalog yang sudah ada
		loadedBytes, loadedMime, loadErr := loadReferenceImageFromURL(reqInput.ImageURL, uploadsDir)
		if loadErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal memuat foto referensi dari katalog: " + loadErr.Error(),
			})
		}
		sourceBytes = loadedBytes
		sourceMime = loadedMime
		originalPublicURL = reqInput.ImageURL
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Silakan pilih atau unggah foto dekorasi referensi terlebih dahulu.",
		})
	}

	// Susun prompt kustomisasi dekorasi profesional
	prompt := buildDecorSimulationPrompt(reqInput)

	// Eksekusi Gemini Image Editing
	generatedBytes, aiNote, genErr := callGeminiDecorImageEdit(apiKey, sourceBytes, sourceMime, prompt)
	if genErr != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": genErr.Error(),
		})
	}

	// Optimasi & simpan gambar hasil simulasi ke ./uploads
	simFilename, saveSimErr := utils.SaveOptimizedImageBytes(generatedBytes, uploadsDir, "sim")
	if saveSimErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan gambar hasil simulasi AI: " + saveSimErr.Error(),
		})
	}

	resultPublicURL := fmt.Sprintf("%s/uploads/%s", baseHostURL, simFilename)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Simulasi warna dekorasi berhasil dibuat dengan AI",
		"data": fiber.Map{
			"id":           uuid.New().String(),
			"original_url": originalPublicURL,
			"result_url":   resultPublicURL,
			"model_name":   reqInput.ModelName,
			"fabric_color": reqInput.FabricColor,
			"flower_color": reqInput.FlowerColor,
			"nuance":       reqInput.Nuance,
			"notes":        reqInput.Notes,
			"ai_note":      aiNote,
			"created_at":   time.Now().Format(time.RFC3339),
		},
	})
}

// buildDecorSimulationPrompt menyusun instruksi multimodal yang menjaga struktur asli panggung/dekorasi
// sekaligus menyesuaikan warna kain, bunga, dan catatan klien secara fotorealistis.
func buildDecorSimulationPrompt(req SimulateDecorRequest) string {
	var b strings.Builder
	b.WriteString("You are an expert wedding & event decoration visual designer.\n")
	b.WriteString("Edit the provided real wedding/event decoration photograph strictly according to the client's customization requests below:\n\n")

	b.WriteString("CRITICAL STRUCTURE PRESERVATION RULES:\n")
	b.WriteString("1. Keep the EXACT stage layout, camera angle, framing, chairs/sofa (kursi pelaminan), backdrop panels/boards/arches, standing structures, lighting bulbs/chandeliers, signage/lettering text, and floor/carpet intact unless explicitly asked to change in Additional Notes.\n")
	b.WriteString("2. Do NOT distort furniture proportions, do NOT add people, and do NOT turn the image into a cartoon or 3D render.\n")
	b.WriteString("3. Maintain realistic photography lighting, natural shadows, fabric drape folds/pleats, and lifelike flower petal textures.\n\n")

	b.WriteString("CLIENT CUSTOMIZATION REQUESTS TO APPLY:\n")
	if req.FabricColor != "" {
		b.WriteString(fmt.Sprintf("- Backdrop Curtain / Fabric Drape Color (Warna Kain Backdrop): Change the backdrop curtain/drape fabric color palette to: %s. Keep realistic fabric folds, sheer/pleated texture, and natural stage lighting.\n", req.FabricColor))
	} else {
		b.WriteString("- Backdrop Curtain / Fabric Drape: Keep the existing backdrop curtain color harmonious with the overall scene.\n")
	}

	if req.FlowerColor != "" {
		b.WriteString(fmt.Sprintf("- Floral Arrangements (Warna & Jenis Bunga): Update the flower arrangements on the backdrop, arches, and floor to feature: %s. Replace conflicting flower colors from the original photo so the entire floral palette matches this request naturally.\n", req.FlowerColor))
	}

	if req.Nuance != "" {
		b.WriteString(fmt.Sprintf("- Overall Ambience / Theme (Nuansa Dekorasi): %s.\n", req.Nuance))
	}

	if req.Notes != "" {
		b.WriteString(fmt.Sprintf("- Specific Client Instructions / Additional Notes: %s.\n", req.Notes))
	}

	b.WriteString("\nOutput the edited photorealistic wedding/event decoration photograph.")
	return b.String()
}

// loadReferenceImageFromURL membaca foto referensi dari ./uploads lokal (jika URL mengarah ke /uploads/)
// atau mengunduh dari URL HTTPS publik secara aman.
func loadReferenceImageFromURL(rawURL string, uploadsDir string) ([]byte, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("format URL gambar tidak valid")
	}

	// Cek apakah path merujuk ke /uploads/<filename> di server ini
	cleanPath := parsed.Path
	if cleanPath == "" {
		cleanPath = rawURL
	}
	if idx := strings.Index(cleanPath, "/uploads/"); idx != -1 {
		baseName := filepath.Base(cleanPath[idx+len("/uploads/"):])
		if baseName != "." && baseName != "/" && baseName != "" {
			localCandidate := filepath.Join(uploadsDir, baseName)
			if data, readErr := os.ReadFile(localCandidate); readErr == nil && len(data) > 0 {
				return data, detectImageMime(baseName, data), nil
			}
		}
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, "", fmt.Errorf("skema URL gambar harus http atau https")
	}

	// Proteksi SSRF: tolak host loopback/private/link-local jika mengambil dari jaringan eksternal
	hostName := parsed.Hostname()
	if ips, lookupErr := net.LookupIP(hostName); lookupErr == nil {
		for _, ip := range ips {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				return nil, "", fmt.Errorf("akses URL jaringan internal tidak diizinkan")
			}
		}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("gagal mengunduh gambar referensi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("gambar referensi mengembalikan status HTTP %d", resp.StatusCode)
	}

	const maxDownloadBytes = 15 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("gagal membaca stream gambar referensi: %w", err)
	}
	if len(data) > maxDownloadBytes {
		return nil, "", fmt.Errorf("ukuran gambar referensi melebihi 15 MB")
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("data gambar referensi kosong")
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = detectImageMime(parsed.Path, data)
	}
	return data, mimeType, nil
}

func detectImageMime(filename string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	}
	detected := http.DetectContentType(data)
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "image/jpeg"
}

// callGeminiDecorImageEdit memanggil model image generation/editing Gemini API dengan rantai fallback
func callGeminiDecorImageEdit(apiKey string, imageBytes []byte, mimeType string, prompt string) ([]byte, string, error) {
	base64Img := base64.StdEncoding.EncodeToString(imageBytes)

	reqBody := geminiImageRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
					{
						InlineData: &geminiInlineData{
							MimeType: mimeType,
							Data:     base64Img,
						},
					},
				},
			},
		},
		GenerationConfig: geminiImageGenConfig{
			ResponseModalities: []string{"TEXT", "IMAGE"},
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("gagal menyiapkan request AI: %w", err)
	}

	// Urutan model pembuat/pengedit gambar Gemini
	modelsToTry := []string{}
	if customModel := strings.TrimSpace(os.Getenv("GEMINI_IMAGE_MODEL")); customModel != "" {
		modelsToTry = append(modelsToTry, customModel)
	}
	for _, m := range []string{
		"gemini-3.1-flash-image",
		"gemini-3.1-flash-lite-image",
		"gemini-2.5-flash-image",
		"gemini-3-pro-image",
	} {
		exists := false
		for _, existing := range modelsToTry {
			if existing == m {
				exists = true
				break
			}
		}
		if !exists {
			modelsToTry = append(modelsToTry, m)
		}
	}

	client := &http.Client{Timeout: 90 * time.Second}
	var lastErr error
	sawZeroFreeQuota := false
	sawRateLimit := false

	for _, model := range modelsToTry {
		endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		httpReq, reqErr := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqJSON))
		if reqErr != nil {
			lastErr = reqErr
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, doErr := client.Do(httpReq)
		if doErr != nil {
			lastErr = doErr
			continue
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode != http.StatusOK {
			bodyStr := string(bodyBytes)
			if resp.StatusCode == http.StatusTooManyRequests {
				sawRateLimit = true
				if strings.Contains(bodyStr, "limit: 0") {
					sawZeroFreeQuota = true
				}
				lastErr = fmt.Errorf("model %s terkena batas kuota (HTTP 429)", model)
				continue
			}
			if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusNotFound {
				lastErr = fmt.Errorf("model %s tidak tersedia (HTTP %d)", model, resp.StatusCode)
				continue
			}
			lastErr = fmt.Errorf("Gemini API (%s) mengembalikan HTTP %d: %s", model, resp.StatusCode, truncateString(bodyStr, 240))
			continue
		}

		var gemResp geminiImageResponse
		if unmarshalErr := json.Unmarshal(bodyBytes, &gemResp); unmarshalErr != nil {
			lastErr = fmt.Errorf("gagal membaca respons Gemini: %w", unmarshalErr)
			continue
		}

		if gemResp.Error != nil {
			lastErr = fmt.Errorf("error Gemini: %s", gemResp.Error.Message)
			continue
		}

		if len(gemResp.Candidates) == 0 {
			lastErr = fmt.Errorf("AI tidak mengembalikan kandidat gambar")
			continue
		}

		var outImageBytes []byte
		var outNotes []string

		for _, part := range gemResp.Candidates[0].Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				outNotes = append(outNotes, strings.TrimSpace(part.Text))
			}
			if part.InlineData != nil && part.InlineData.Data != "" {
				decoded, decErr := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if decErr == nil && len(decoded) > 0 {
					outImageBytes = decoded
				}
			} else if part.InlineDataSnake != nil && part.InlineDataSnake.Data != "" {
				decoded, decErr := base64.StdEncoding.DecodeString(part.InlineDataSnake.Data)
				if decErr == nil && len(decoded) > 0 {
					outImageBytes = decoded
				}
			}
		}

		if len(outImageBytes) > 0 {
			return outImageBytes, strings.Join(outNotes, " "), nil
		}

		finishReason := gemResp.Candidates[0].FinishReason
		if finishReason != "" {
			lastErr = fmt.Errorf("AI belum berhasil membuat gambar (alasan: %s). Coba perjelas deskripsi warna kain/bunga", finishReason)
		} else {
			lastErr = fmt.Errorf("respons AI hanya berisi teks tanpa gambar")
		}
	}

	if sawZeroFreeQuota {
		return nil, "", fmt.Errorf("Kuota pembuatan gambar (Image Generation) pada API Key Gemini ini masih 0 (Free Tier). Silakan aktifkan Billing / Tier 1 (atau klaim Free Credit $300 Google Cloud) di Google AI Studio agar model gambar dapat digunakan.")
	}
	if sawRateLimit {
		return nil, "", fmt.Errorf("Batas pemakaian AI per menit sedang penuh (429). Silakan tunggu 30-60 detik lalu coba kembali.")
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("Gagal membuat simulasi gambar AI: %v", lastErr)
	}
	return nil, "", fmt.Errorf("Gagal membuat simulasi gambar AI.")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
