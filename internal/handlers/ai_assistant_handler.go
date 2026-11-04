package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
)

// AIAssistantMessage pesan dalam riwayat percakapan
type AIAssistantMessage struct {
	Role string `json:"role"` // "user" atau "model"
	Text string `json:"text"`
}

// AIAssistantRequest request payload dari mobile app
type AIAssistantRequest struct {
	Message string               `json:"message"`
	History []AIAssistantMessage `json:"history,omitempty"`
}

// AIAssistantResponse respons ke mobile app
type AIAssistantResponse struct {
	Reply       string   `json:"reply"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// HandleAIAssistant handles POST /api/ai/assistant
// Chatbot data & insight bisnis Dekorin untuk Owner & Admin
func HandleAIAssistant(c *fiber.Ctx) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "GEMINI_API_KEY belum dikonfigurasi di server backend.",
		})
	}

	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)
	userName, _ := c.Locals("user_name").(string)

	if strings.EqualFold(userRole, "superadmin") {
		if qComp := strings.TrimSpace(c.Query("company_id")); qComp != "" && qComp != "all" {
			companyID = qComp
		}
	}
	if companyID == "" {
		companyID = "cmp_default"
	}

	var req AIAssistantRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format request tidak valid",
		})
	}

	userMsg := strings.TrimSpace(req.Message)
	if userMsg == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Pesan pertanyaan tidak boleh kosong",
		})
	}

	// 1. Kumpulkan data bisnis real-time vendor ini (Read-Only)
	businessContext := compileBusinessContext(companyID, userName)

	// 2. Panggil Gemini dengan context grounding
	reply, suggestions, err := callGeminiBusinessAssistant(apiKey, businessContext, req.History, userMsg)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mendapatkan analisis AI: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data": AIAssistantResponse{
			Reply:       reply,
			Suggestions: suggestions,
		},
	})
}

// compileBusinessContext mengumpulkan data analitik aktual dari database secara read-only
func compileBusinessContext(companyID, userName string) string {
	now := time.Now()
	loc := now.Location()

	// Nama Perusahaan
	var comp models.Company
	compName := "Dekorin Vendor"
	if err := config.DB.Where("id = ?", companyID).First(&comp).Error; err == nil && comp.Name != "" {
		compName = comp.Name
	}

	// Tanggal awal dan akhir bulan ini
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// 1. Data Agenda Bulan Ini
	var monthAgendas []models.Agenda
	config.DB.Preload("ClientForm").
		Where("company_id = ? AND event_date_time >= ? AND event_date_time <= ? AND status != ?",
			companyID, startOfMonth, endOfMonth, "cancelled").
		Find(&monthAgendas)

	var totalOmsetMonth float64 = 0
	var totalKasMasukMonth float64 = 0
	for _, a := range monthAgendas {
		biaya := a.CalculateTotalBiaya()
		totalOmsetMonth += biaya
		if a.PaymentStatus == "paid" {
			totalKasMasukMonth += biaya
		} else if a.PaymentStatus == "dp" {
			totalKasMasukMonth += a.DpAmount
		}
	}
	piutangMonth := totalOmsetMonth - totalKasMasukMonth
	if piutangMonth < 0 {
		piutangMonth = 0
	}

	// 2. Data Pengeluaran Bulan Ini
	var monthExpenses []models.Expense
	config.DB.Where("company_id = ? AND date >= ? AND date <= ?", companyID, startOfMonth, endOfMonth).
		Find(&monthExpenses)

	var totalExpenseMonth float64 = 0
	expenseCategories := make(map[string]float64)
	for _, exp := range monthExpenses {
		totalExpenseMonth += exp.Amount
		cat := exp.Category
		if cat == "" {
			cat = "Lainnya"
		}
		expenseCategories[cat] += exp.Amount
	}

	labaBersihEstimasi := totalOmsetMonth - totalExpenseMonth
	labaKasRiil := totalKasMasukMonth - totalExpenseMonth

	var expBreakdown strings.Builder
	for cat, amt := range expenseCategories {
		expBreakdown.WriteString(fmt.Sprintf("    - %s: Rp %.0f\n", cat, amt))
	}
	if expBreakdown.Len() == 0 {
		expBreakdown.WriteString("    - Belum ada pengeluaran tercatat bulan ini.\n")
	}

	// 3. Agenda Mendatang (30 Hari ke Depan) & Deteksi Jadwal Bertabrakan (Bentrok)
	startToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endNext60 := startToday.AddDate(0, 2, 0)

	var upcomingAgendas []models.Agenda
	config.DB.Preload("ClientForm").
		Where("company_id = ? AND event_date_time >= ? AND event_date_time <= ? AND status != ?",
			companyID, startToday, endNext60, "cancelled").
		Order("event_date_time ASC").
		Find(&upcomingAgendas)

	// Deteksi Bentrok: kumpulkan per tanggal (format YYYY-MM-DD)
	dateAgendasMap := make(map[string][]models.Agenda)
	for _, a := range upcomingAgendas {
		dateKey := a.EventDateTime.Format("2006-01-02")
		dateAgendasMap[dateKey] = append(dateAgendasMap[dateKey], a)
	}

	var conflictInfo strings.Builder
	conflictCount := 0
	for dateKey, agList := range dateAgendasMap {
		if len(agList) > 1 {
			conflictCount++
			conflictInfo.WriteString(fmt.Sprintf("  * Tanggal %s (%d agenda bertabrakan di hari yang sama):\n", dateKey, len(agList)))
			for _, item := range agList {
				conflictInfo.WriteString(fmt.Sprintf("    - Klien: %s, Jam: %s, Lokasi: %s, Paket: %s\n",
					item.ClientName, item.EventDateTime.Format("15:04"), item.EventAddress, item.PackageName))
			}
		}
	}
	if conflictCount == 0 {
		conflictInfo.WriteString("  * Tidak ada jadwal yang bertabrakan di tanggal yang sama dalam 60 hari ke depan (Jadwal aman).\n")
	}

	// 4. Daftar Klien Belum Lunas (Piutang)
	var unpaidClients strings.Builder
	unpaidCount := 0
	for _, a := range upcomingAgendas {
		if a.PaymentStatus != "paid" {
			unpaidCount++
			totalBiaya := a.CalculateTotalBiaya()
			sisa := totalBiaya - a.DpAmount
			unpaidClients.WriteString(fmt.Sprintf("  * Klien: %s (HP: %s)\n    Tanggal Acara: %s, Total: Rp %.0f, DP Masuk: Rp %.0f, Sisa Tagihan: Rp %.0f (Status: %s)\n",
				a.ClientName, a.ClientPhone, a.EventDateTime.Format("02 Jan 2006"), totalBiaya, a.DpAmount, sisa, a.PaymentStatus))
		}
	}
	if unpaidCount == 0 {
		unpaidClients.WriteString("  * Semua agenda mendatang sudah lunas atau belum ada agenda dengan status DP/unpaid.\n")
	}

	// 5. Paket & Properti Terpopuler
	type PopularPackage struct {
		PackageName string
		Total       int
	}
	var popPackages []PopularPackage
	config.DB.Model(&models.Agenda{}).
		Select("package_name, count(*) as total").
		Where("company_id = ? AND package_name != '' AND status != ?", companyID, "cancelled").
		Group("package_name").
		Order("total DESC").
		Limit(5).
		Scan(&popPackages)

	var packageRank strings.Builder
	for i, p := range popPackages {
		packageRank.WriteString(fmt.Sprintf("  %d. %s (%d kali dipesan)\n", i+1, p.PackageName, p.Total))
	}
	if packageRank.Len() == 0 {
		packageRank.WriteString("  - Belum ada data paket yang dipesan.\n")
	}

	// Susun seluruh ringkasan menjadi dokumen konteks
	return fmt.Sprintf(`[DATA RESMI REAL-TIME BISNIS DEKORIN]
Nama Vendor: %s
Nama User / Penanya: %s
Waktu Saat Ini: %s (%s)

--- 1. FINANSIAL BULAN INI (%s %d) ---
- Total Agenda Aktif Bulan Ini: %d agenda
- Total Omset Kotor (Gross Revenue): Rp %.0f
- Uang Kas Masuk Riil (DP + Lunas): Rp %.0f
- Sisa Piutang Klien Bulan Ini: Rp %.0f
- Total Pengeluaran Operasional: Rp %.0f
- Rincian Pengeluaran per Kategori:
%s
- Estimasi Laba Bersih Keseluruhan (Omset - Pengeluaran): Rp %.0f
- Laba Kas Masuk Riil (Kas Masuk - Pengeluaran): Rp %.0f

--- 2. DETEKSI JADWAL BERTABRAKAN / BENTROK (60 HARI KE DEPAN) ---
%s

--- 3. DAFTAR TAGIHAN / KLIEN BELUM LUNAS ---
%s

--- 4. PAKET DEKORASI TERPOPULER ---
%s
`,
		compName, userName, now.Format("02 January 2006 15:04 WIB"), now.Weekday().String(),
		now.Month().String(), now.Year(),
		len(monthAgendas), totalOmsetMonth, totalKasMasukMonth, piutangMonth, totalExpenseMonth,
		expBreakdown.String(), labaBersihEstimasi, labaKasRiil,
		conflictInfo.String(),
		unpaidClients.String(),
		packageRank.String(),
	)
}

// callGeminiBusinessAssistant memanggil Gemini API dengan system prompt dan business context
func callGeminiBusinessAssistant(apiKey, businessContext string, history []AIAssistantMessage, userMsg string) (string, []string, error) {
	systemPrompt := `Kamu adalah "Asisten Dekorin AI", partner dan penasihat bisnis cerdas khusus untuk Owner dan Admin vendor dekorasi acara/wedding.
Tugasmu adalah menjawab pertanyaan pengguna secara akurat, ramah, profesional, dan memberikan wawasan/insight bisnis yang bernilai tambah berdasarkan data resmi yang disediakan.

ATURAN PENTING:
1. SELALU rujuk angka, nama klien, jadwal, dan status dari [DATA RESMI REAL-TIME BISNIS DEKORIN] yang diberikan di bawah. Jangan mengarang angka atau fakta yang bertentangan dengan data resmi.
2. Format angka uang ke format Rupiah yang mudah dibaca (misal: "Rp 35.000.000" atau "Rp 35 Juta").
3. Berikan jawaban yang terstruktur dengan poin-poin (bullet points atau bolding) yang rapi dan nyaman dibaca di layar HP.
4. Jika ada potensi masalah bisnis (seperti jadwal bentrok di hari yang sama, atau piutang yang belum lunas menjelang hari H), berikan saran proaktif (tips operasional atau pengingat tim).
5. Di akhir jawaban, berikan 2 hingga 3 rekomendasi pertanyaan lanjutan singkat yang relevan untuk diklik pengguna. Pisahkan pertanyaan rekomendasi tersebut dalam format tag khusus di baris paling akhir:
[SUGGESTIONS: Pertanyaan 1 | Pertanyaan 2 | Pertanyaan 3]

Konteks data bisnis vendor saat ini:
` + businessContext

	type geminiPartReq struct {
		Text string `json:"text"`
	}
	type geminiContentReq struct {
		Role  string          `json:"role,omitempty"`
		Parts []geminiPartReq `json:"parts"`
	}
	type geminiAssistantPayload struct {
		Contents []geminiContentReq `json:"contents"`
	}

	contents := []geminiContentReq{
		{
			Role: "user",
			Parts: []geminiPartReq{
				{Text: systemPrompt},
			},
		},
		{
			Role: "model",
			Parts: []geminiPartReq{
				{Text: "Siap! Saya adalah Asisten Dekorin AI. Saya siap membantu menyajikan analisis keuangan, pengecekan jadwal bentrok, monitoring piutang klien, dan insight operasional vendor Anda dengan akurat."},
			},
		},
	}

	// Masukkan riwayat pesan jika ada
	for _, h := range history {
		role := h.Role
		if role != "model" {
			role = "user"
		}
		trimmed := strings.TrimSpace(h.Text)
		if trimmed != "" {
			contents = append(contents, geminiContentReq{
				Role:  role,
				Parts: []geminiPartReq{{Text: trimmed}},
			})
		}
	}

	// Masukkan pesan pertanyaan terbaru
	contents = append(contents, geminiContentReq{
		Role:  "user",
		Parts: []geminiPartReq{{Text: userMsg}},
	})

	payload := geminiAssistantPayload{Contents: contents}
	reqJSON, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("marshal request error: %w", err)
	}

	primaryModel := os.Getenv("GEMINI_MODEL")
	if primaryModel == "" {
		primaryModel = "gemini-3.5-flash-lite"
	}

	modelsToTry := []string{primaryModel}
	if primaryModel != "gemini-3.5-flash-lite" {
		modelsToTry = append(modelsToTry, "gemini-3.5-flash-lite")
	}
	modelsToTry = append(modelsToTry, "gemini-3.5-flash", "gemini-3.8-flash")

	client := &http.Client{Timeout: 35 * time.Second}
	var lastErr error

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

		if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
			lastErr = fmt.Errorf("gemini tidak menghasilkan kandidat jawaban")
			continue
		}

		rawReply := gemResp.Candidates[0].Content.Parts[0].Text

		// Ekstrak [SUGGESTIONS: ... | ...] jika ada
		replyText, suggestions := parseSuggestionsFromReply(rawReply)
		return replyText, suggestions, nil
	}

	return "", nil, fmt.Errorf("semua model AI gagal merespons: %w", lastErr)
}

// parseSuggestionsFromReply mengekstrak tag [SUGGESTIONS: A | B | C] dari teks balasan
func parseSuggestionsFromReply(text string) (string, []string) {
	suggestions := []string{}
	idx := strings.Index(text, "[SUGGESTIONS:")
	if idx == -1 {
		// Default suggestions jika AI tidak menyertakan tag
		return strings.TrimSpace(text), []string{
			"Berapa total omset dan laba bulan ini?",
			"Cek jadwal bentrok di 30 hari ke depan",
			"Siapa saja klien yang belum lunas?",
		}
	}

	cleanReply := strings.TrimSpace(text[:idx])
	rem := text[idx+len("[SUGGESTIONS:"):]
	endIdx := strings.Index(rem, "]")
	if endIdx != -1 {
		sugPart := rem[:endIdx]
		items := strings.Split(sugPart, "|")
		for _, item := range items {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				suggestions = append(suggestions, trimmed)
			}
		}
	}

	if len(suggestions) == 0 {
		suggestions = []string{
			"Berapa total omset dan laba bulan ini?",
			"Cek jadwal bentrok di 30 hari ke depan",
			"Siapa saja klien yang belum lunas?",
		}
	}

	return cleanReply, suggestions
}
