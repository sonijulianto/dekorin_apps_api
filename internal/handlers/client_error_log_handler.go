package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createClientErrorLogRequest struct {
	Level        string          `json:"level"`
	Platform     string          `json:"platform"`
	AppVersion   string          `json:"app_version"`
	Route        string          `json:"route"`
	ErrorType    string          `json:"error_type"`
	ErrorMessage string          `json:"error_message"`
	StackTrace   string          `json:"stack_trace"`
	Breadcrumbs  json.RawMessage `json:"breadcrumbs"`
	ApiContext   json.RawMessage `json:"api_context"`
	DeviceInfo   string          `json:"device_info"`
}

// CreateClientErrorLog menangani POST /api/client-logs (optional-auth).
// Mengelompokkan error identik menggunakan fingerprint agar tidak membanjiri tabel.
func CreateClientErrorLog(c *fiber.Ctx) error {
	var req createClientErrorLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format payload log tidak valid",
		})
	}

	errMsg := sanitizeTruncate(req.ErrorMessage, 4000)
	if errMsg == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "error_message wajib diisi",
		})
	}

	level := normalizeLogLevel(req.Level)
	platform := normalizePlatform(req.Platform)
	appVersion := sanitizeTruncate(req.AppVersion, 50)
	route := sanitizeTruncate(req.Route, 150)
	errorType := sanitizeTruncate(req.ErrorType, 120)
	stackTrace := sanitizeTruncate(req.StackTrace, 16000)
	deviceInfo := sanitizeTruncate(req.DeviceInfo, 255)
	breadcrumbs := rawJSONToTruncatedString(req.Breadcrumbs, 8000, "[]")
	apiContext := rawJSONToTruncatedString(req.ApiContext, 4000, "{}")

	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)
	userRole = sanitizeTruncate(userRole, 50)
	ipAddr := sanitizeTruncate(c.IP(), 50)

	fingerprint := computeClientErrorFingerprint(platform, errorType, route, stackTrace, errMsg)
	now := time.Now()

	var existing models.ClientErrorLog
	err := config.DB.Where("fingerprint = ?", fingerprint).First(&existing).Error
	if err == nil {
		updates := map[string]interface{}{
			"occurrence_count": gorm.Expr("occurrence_count + 1"),
			"last_seen_at":     now,
			"level":            level,
			"error_message":    errMsg,
			"breadcrumbs":      breadcrumbs,
			"api_context":      apiContext,
			"ip_addr":          ipAddr,
		}
		if appVersion != "" {
			updates["app_version"] = appVersion
		}
		if stackTrace != "" {
			updates["stack_trace"] = stackTrace
		}
		if deviceInfo != "" {
			updates["device_info"] = deviceInfo
		}
		if userID != "" {
			updates["user_id"] = userID
		}
		if companyID != "" {
			updates["company_id"] = companyID
		}
		if userRole != "" {
			updates["user_role"] = userRole
		}
		if existing.Status == "resolved" {
			updates["status"] = "open"
			updates["resolved_at"] = nil
		}

		if updateErr := config.DB.Model(&existing).Updates(updates).Error; updateErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal memperbarui log error",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":      "ok",
			"id":          existing.ID,
			"fingerprint": fingerprint,
			"grouped":     true,
		})
	}

	if err != gorm.ErrRecordNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memeriksa log error",
		})
	}

	newLog := models.ClientErrorLog{
		ID:              "cel_" + uuid.New().String()[:12],
		Fingerprint:     fingerprint,
		Level:           level,
		Platform:        platform,
		AppVersion:      appVersion,
		Route:           route,
		ErrorType:       errorType,
		ErrorMessage:    errMsg,
		StackTrace:      stackTrace,
		Breadcrumbs:     breadcrumbs,
		ApiContext:      apiContext,
		DeviceInfo:      deviceInfo,
		UserID:          userID,
		CompanyID:       companyID,
		UserRole:        userRole,
		IPAddr:          ipAddr,
		Status:          "open",
		OccurrenceCount: 1,
		FirstSeenAt:     now,
		LastSeenAt:      now,
	}

	if createErr := config.DB.Create(&newLog).Error; createErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan log error",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "ok",
		"id":          newLog.ID,
		"fingerprint": fingerprint,
		"grouped":     false,
	})
}

// GetClientErrorLogs menangani GET /api/superadmin/client-logs.
func GetClientErrorLogs(c *fiber.Ctx) error {
	query := config.DB.Model(&models.ClientErrorLog{})

	if platform := strings.TrimSpace(c.Query("platform")); platform != "" && platform != "all" {
		query = query.Where("platform = ?", strings.ToLower(platform))
	}
	if level := strings.TrimSpace(c.Query("level")); level != "" && level != "all" {
		query = query.Where("level = ?", strings.ToLower(level))
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		query = query.Where("status = ?", strings.ToLower(status))
	}
	if appVersion := strings.TrimSpace(c.Query("app_version")); appVersion != "" && appVersion != "all" {
		query = query.Where("app_version = ?", appVersion)
	}
	if companyID := strings.TrimSpace(c.Query("company_id")); companyID != "" && companyID != "all" {
		query = query.Where("company_id = ?", companyID)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		pattern := "%" + search + "%"
		query = query.Where(
			"error_message ILIKE ? OR error_type ILIKE ? OR route ILIKE ? OR stack_trace ILIKE ? OR user_id ILIKE ?",
			pattern, pattern, pattern, pattern, pattern,
		)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit < 1 || limit > 100 {
		limit = 25
	}
	offset := (page - 1) * limit

	var total int64
	query.Count(&total)

	var logs []models.ClientErrorLog
	if err := query.Order("last_seen_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memuat log error FE",
		})
	}

	var openCount int64
	config.DB.Model(&models.ClientErrorLog{}).Where("status = ?", "open").Count(&openCount)
	var fatalOpenCount int64
	config.DB.Model(&models.ClientErrorLog{}).Where("status = ? AND level = ?", "open", "fatal").Count(&fatalOpenCount)

	return c.JSON(fiber.Map{
		"data": logs,
		"meta": fiber.Map{
			"page":             page,
			"limit":            limit,
			"total":            total,
			"open_count":       openCount,
			"fatal_open_count": fatalOpenCount,
		},
	})
}

// UpdateClientErrorLogStatus menangani PATCH /api/superadmin/client-logs/:id/status.
func UpdateClientErrorLogStatus(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID log tidak valid",
		})
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format payload tidak valid",
		})
	}

	status := strings.ToLower(strings.TrimSpace(body.Status))
	if status != "open" && status != "resolved" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status harus 'open' atau 'resolved'",
		})
	}

	var existing models.ClientErrorLog
	if err := config.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Log error tidak ditemukan",
		})
	}

	updates := map[string]interface{}{
		"status": status,
	}
	if status == "resolved" {
		now := time.Now()
		updates["resolved_at"] = &now
	} else {
		updates["resolved_at"] = nil
	}

	if err := config.DB.Model(&existing).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui status log error",
		})
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"id":     id,
		"state":  status,
	})
}

func sanitizeTruncate(val string, maxLen int) string {
	trimmed := strings.TrimSpace(val)
	if maxLen <= 0 || len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen]
}

func rawJSONToTruncatedString(raw json.RawMessage, maxLen int, fallback string) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return fallback
	}
	return sanitizeTruncate(trimmed, maxLen)
}

func normalizeLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "fatal", "warning", "error":
		return strings.ToLower(strings.TrimSpace(level))
	default:
		return "error"
	}
}

func normalizePlatform(platform string) string {
	p := strings.ToLower(strings.TrimSpace(platform))
	if p == "" {
		return "unknown"
	}
	if len(p) > 20 {
		return p[:20]
	}
	return p
}

// extractTopAppFrame mencari baris stack trace pertama dari package:dekorin_apps/
// agar grouping error tetap konsisten meskipun pesan error mengandung ID dinamis.
func extractTopAppFrame(stackTrace string) string {
	if strings.TrimSpace(stackTrace) == "" {
		return ""
	}
	lines := strings.Split(stackTrace, "\n")
	firstNonEmpty := ""
	for _, line := range lines {
		clean := strings.TrimSpace(line)
		if clean == "" {
			continue
		}
		if firstNonEmpty == "" {
			firstNonEmpty = clean
		}
		if strings.Contains(clean, "package:dekorin_apps/") {
			return clean
		}
	}
	return firstNonEmpty
}

func computeClientErrorFingerprint(platform, errorType, route, stackTrace, errMsg string) string {
	topFrame := extractTopAppFrame(stackTrace)
	if topFrame == "" {
		// Fallback ke potongan awal error message jika stack trace kosong
		topFrame = sanitizeTruncate(errMsg, 120)
	}
	raw := strings.Join([]string{
		strings.ToLower(platform),
		strings.ToLower(errorType),
		strings.ToLower(route),
		topFrame,
	}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:32]
}
