package handlers

import (
	"strconv"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
)

// GetApiLogs menangani GET /api/apilog/get dan GET /api/apilogs
func GetApiLogs(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	query := config.DB.Model(&models.ApiLog{})

	// Filter company
	qCompany := c.Query("company_id")
	if strings.EqualFold(userRole, "superadmin") {
		if qCompany != "" && qCompany != "all" {
			query = query.Where("company_id = ?", qCompany)
		}
	} else if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// Filter tanggal log
	dateStr := strings.TrimSpace(c.Query("date"))
	if dateStr != "" {
		var parsedDate time.Time
		var parseErr error

		dateFormats := []string{"2006-01-02", "02/01/2006", "02-01-2006", "2006/01/02", "02/Jan/2006"}
		for _, format := range dateFormats {
			parsedDate, parseErr = time.Parse(format, dateStr)
			if parseErr == nil {
				break
			}
		}

		if parseErr == nil {
			startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, time.Local)
			endOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 23, 59, 59, 999999999, time.Local)
			query = query.Where("log_time >= ? AND log_time <= ?", startOfDay, endOfDay)
		}
	}

	// Filter Search umum (URL, Username, Method, Remark)
	searchStr := strings.TrimSpace(c.Query("search"))
	if searchStr != "" {
		searchPattern := "%" + searchStr + "%"
		query = query.Where("url ILIKE ? OR username ILIKE ? OR method ILIKE ? OR remark ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Filter Method
	if m := strings.TrimSpace(c.Query("method")); m != "" && m != "all" {
		query = query.Where("method = ?", strings.ToUpper(m))
	}

	// Filter Status (SUCCESS, FAILED)
	if s := strings.TrimSpace(c.Query("status")); s != "" && s != "all" {
		query = query.Where("status = ?", strings.ToUpper(s))
	}

	// Total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status_code": 500,
			"success":     false,
			"message":     "Gagal menghitung jumlah log: " + err.Error(),
		})
	}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	var logs []models.ApiLog
	if err := query.Order("log_time DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status_code": 500,
			"success":     false,
			"message":     "Gagal mengambil data log: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status_code": 200,
		"success":     true,
		"message":     "Successfully",
		"data":        logs,
		"total":       total,
		"page":        page,
		"limit":       limit,
	})
}

// GetApiLogByID menangani GET /api/apilog/:id
func GetApiLogByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var logItem models.ApiLog

	if err := config.DB.Where("id = ?", id).First(&logItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status_code": 404,
			"success":     false,
			"message":     "Data log tidak ditemukan",
		})
	}

	statusCodeStr := strconv.Itoa(logItem.StatusCode)
	if logItem.StatusCode == 0 {
		statusCodeStr = "200"
	}

	dataMap := fiber.Map{
		"id":            logItem.ID,
		"url":           logItem.URL,
		"method":        logItem.Method,
		"ip_addr":       logItem.IPAddr,
		"username":      logItem.Username,
		"log_time":      logItem.LogTime.Format("2006-01-02 15:04:05"),
		"req_data":      logItem.ReqData,
		"response_data": logItem.ResponseData,
		"status":        logItem.Status,
		"status_code":   statusCodeStr,
		"remark":        logItem.Remark,
		"apk_version":   logItem.ApkVersion,
		"device_model":  logItem.DeviceModel,
		"build_date":    logItem.BuildDate,
		"name":          logItem.Name,
		"branch_code":   logItem.BranchCode,
		"branch_name":   logItem.BranchName,
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status_code": 200,
		"success":     true,
		"message":     "Successfully",
		"data":        dataMap,
	})
}
