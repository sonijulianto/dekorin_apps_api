package middleware

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ApiLogger merekam setiap HTTP request dan response API ke database secara asynchronous
func ApiLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Jangan log query log itu sendiri atau file statis untuk mencegah looping/overhead
		if strings.HasPrefix(path, "/api/apilog") ||
			strings.HasPrefix(path, "/uploads") ||
			strings.HasPrefix(path, "/assets") ||
			path == "/" ||
			path == "/superadmin" {
			return c.Next()
		}

		// Tangkap data request sebelum handler berjalan
		method := c.Method()
		url := c.OriginalURL()
		ipAddr := c.IP()
		if fwd := c.Get("X-Forwarded-For"); fwd != "" {
			ipAddr = strings.Split(fwd, ",")[0]
		}

		reqBody := string(c.Body())
		if len(reqBody) > 100000 { // Batasi 100KB agar database tetap optimal
			reqBody = reqBody[:100000] + "... (truncated)"
		}

		// Jalankan handler berikutnya
		err := c.Next()

		// Ambil data setelah request selesai diproses
		statusCode := c.Response().StatusCode()
		respBody := string(c.Response().Body())
		if len(respBody) > 100000 {
			respBody = respBody[:100000] + "... (truncated)"
		}

		status := "SUCCESS"
		if statusCode >= 400 {
			status = "FAILED"
		}

		// Cari username dari session / token JWT
		username := ""
		if uPhone, ok := c.Locals("user_phone").(string); ok && uPhone != "" {
			username = uPhone
		} else if uEmail, ok := c.Locals("user_email").(string); ok && uEmail != "" {
			username = uEmail
		} else if uRole, ok := c.Locals("user_role").(string); ok && uRole != "" {
			username = uRole
		}

		// Jika endpoint login, coba ekstrak username dari reqBody
		if username == "" && strings.Contains(path, "/login") && reqBody != "" {
			var loginData map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(reqBody), &loginData); jsonErr == nil {
				if p, ok := loginData["phone"].(string); ok && p != "" {
					username = p
				} else if e, ok := loginData["email"].(string); ok && e != "" {
					username = e
				}
			}
		}

		companyID := GetCompanyID(c)
		if companyID == "" {
			companyID = "cmp_default"
		}

		// Simpan log ke database secara async agar tidak memperlambat response HTTP
		go func(cid, m, u, ip, uname, req, resp, st string, code int, t time.Time) {
			if config.DB == nil {
				return
			}
			logEntry := models.ApiLog{
				ID:           fmt.Sprintf("log_%s", uuid.New().String()[:12]),
				CompanyID:    cid,
				Method:       m,
				Status:       st,
				StatusCode:   code,
				URL:          u,
				IPAddr:       ip,
				Username:     uname,
				LogTime:      t,
				ReqData:      req,
				ResponseData: resp,
				CreatedAt:    t,
			}
			_ = config.DB.Create(&logEntry).Error
		}(companyID, method, url, ipAddr, username, reqBody, respBody, status, statusCode, time.Now())

		return err
	}
}
