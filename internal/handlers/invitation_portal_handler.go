package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/csv"
	"fmt"
	"math/big"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"
	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var nonSlugCharsRegex = regexp.MustCompile(`[^a-z0-9-]+`)
var multiDashRegex = regexp.MustCompile(`-+`)

var reservedInvitationSlugs = map[string]bool{
	"api":             true,
	"kelola":          true,
	"manage":          true,
	"login":           true,
	"portal":          true,
	"kelola-undangan": true,
	"uploads":         true,
	"assets":          true,
	"images":          true,
	"invitation":      true,
	"invitations":     true,
	"superadmin":      true,
	"preview":         true,
	"form":            true,
	"download":        true,
	"favicon.ico":     true,
	"robots.txt":      true,
	"sitemap.xml":     true,
}

// generateEditToken membuat token akses portal pengantin 40 karakter berentropi tinggi
func generateEditToken() string {
	return "inv_sec_" + strings.ReplaceAll(uuid.New().String(), "-", "")
}

// generateEditPin membuat 6 digit PIN numerik acak yang aman secara kriptografis
func generateEditPin() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%900000+100000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

// ensureInvitationAccessCredentials memastikan undangan lama maupun baru memiliki EditToken & EditPin
func ensureInvitationAccessCredentials(inv *models.Invitation) {
	if inv == nil || inv.ID == "" {
		return
	}
	updates := map[string]interface{}{}
	if strings.TrimSpace(inv.EditToken) == "" {
		inv.EditToken = generateEditToken()
		updates["edit_token"] = inv.EditToken
	}
	if strings.TrimSpace(inv.EditPin) == "" {
		inv.EditPin = generateEditPin()
		updates["edit_pin"] = inv.EditPin
	}
	if len(updates) > 0 && config.DB != nil {
		config.DB.Model(&models.Invitation{}).Where("id = ?", inv.ID).Updates(updates)
	}
}

// sanitizeInvitationSlug membersihkan input slug atau URL penuh menjadi slug bersih a-z0-9-
func sanitizeInvitationSlug(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	// Jika user menempelkan URL penuh (mis. https://undangan.dekorin.my.id/bagas-zulia?to=Budi)
	if strings.Contains(s, "://") {
		if parsed, err := url.Parse(s); err == nil && parsed.Path != "" {
			s = parsed.Path
		}
	}
	// Hilangkan query string jika ada
	if idx := strings.Index(s, "?"); idx != -1 {
		s = s[:idx]
	}
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "invitation/")
	s = strings.TrimSuffix(s, "/")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = nonSlugCharsRegex.ReplaceAllString(s, "")
	s = multiDashRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func isReservedInvitationSlug(slug string) bool {
	return reservedInvitationSlugs[strings.ToLower(strings.TrimSpace(slug))]
}

// IsInvitationSubdomain mendeteksi apakah request masuk melalui subdomain undangan.dekorin.my.id
func IsInvitationSubdomain(c *fiber.Ctx) bool {
	host := strings.ToLower(strings.TrimSpace(c.Get("X-Forwarded-Host")))
	if host == "" {
		host = strings.ToLower(strings.TrimSpace(c.Hostname()))
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return strings.HasPrefix(host, "undangan.")
}

// findInvitationByEditToken mencari undangan berdasarkan edit_token yang valid
func findInvitationByEditToken(token string) (*models.Invitation, error) {
	cleanToken := strings.TrimSpace(token)
	if cleanToken == "" || len(cleanToken) < 12 {
		return nil, fmt.Errorf("token tidak valid")
	}
	var invitation models.Invitation
	if err := config.DB.Where("edit_token = ?", cleanToken).First(&invitation).Error; err != nil {
		return nil, err
	}
	ensureInvitationAccessCredentials(&invitation)
	return &invitation, nil
}

// ShowInvitationPortalPage merender SPA Web Portal Kelola Undangan Pengantin
func ShowInvitationPortalPage(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	return c.SendFile("./web/templates/invitation_portal.html")
}

// ShowSubdomainPublicInvitation merender undangan publik pada path singkat /:slug khusus subdomain undangan.dekorin.my.id
func ShowSubdomainPublicInvitation(c *fiber.Ctx) error {
	if !IsInvitationSubdomain(c) {
		return c.Next()
	}
	slug := strings.TrimSpace(c.Params("slug"))
	if slug == "" || isReservedInvitationSlug(slug) || strings.Contains(slug, ".") {
		return c.Next()
	}
	return ShowPublicInvitation(c)
}

// PortalLogin menangani POST /api/invitation-portal/login (Login dengan Slug + PIN 6 Digit)
func PortalLogin(c *fiber.Ctx) error {
	var req models.PortalLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	slug := sanitizeInvitationSlug(req.Slug)
	pin := strings.TrimSpace(req.Pin)
	if slug == "" || pin == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Kode tautan (slug) undangan dan PIN akses wajib diisi",
		})
	}

	var invitation models.Invitation
	if err := config.DB.Where("LOWER(slug) = ?", slug).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan atau PIN akses salah",
		})
	}

	ensureInvitationAccessCredentials(&invitation)

	expectedPin := strings.TrimSpace(invitation.EditPin)
	if expectedPin == "" || subtle.ConstantTimeCompare([]byte(expectedPin), []byte(pin)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan atau PIN akses salah",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":    "Login portal undangan berhasil",
		"edit_token": invitation.EditToken,
		"slug":       invitation.Slug,
		"title":      invitation.Title,
	})
}

// GetPortalInvitation menangani GET /api/invitation-portal/:token
func GetPortalInvitation(c *fiber.Ctx) error {
	token := c.Params("token")
	invitation, err := findInvitationByEditToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Tautan akses kelola tidak valid atau telah diperbarui. Silakan hubungi vendor dekorasi Anda.",
		})
	}

	var wishes []models.InvitationWish
	config.DB.Where("invitation_id = ?", invitation.ID).Order("created_at DESC").Find(&wishes)
	invitation.Wishes = wishes

	// Hitung statistik RSVP kehadiran
	var hadirCount, tidakHadirCount, raguCount, totalPaxHadir int
	for _, w := range wishes {
		att := strings.ToLower(strings.TrimSpace(w.Attendance))
		switch att {
		case "hadir":
			hadirCount++
			pax := w.GuestCount
			if pax < 1 {
				pax = 1
			}
			totalPaxHadir += pax
		case "tidak_hadir", "tidak hadir":
			tidakHadirCount++
		default:
			raguCount++
		}
	}

	// Ambil profil vendor dekorasi untuk header portal
	companyName := "Dekorin"
	companyLogo := "/assets/logo.png"
	companyPhone := ""
	if invitation.CompanyID != "" {
		var comp models.Company
		if err := config.DB.Where("id = ?", invitation.CompanyID).First(&comp).Error; err == nil {
			if strings.TrimSpace(comp.Name) != "" {
				companyName = comp.Name
			}
			if strings.TrimSpace(comp.LogoURL) != "" {
				companyLogo = comp.LogoURL
			}
			companyPhone = comp.Phone
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat data portal undangan",
		"data":    invitation,
		"stats": fiber.Map{
			"total_wishes":      len(wishes),
			"hadir_count":       hadirCount,
			"tidak_hadir_count": tidakHadirCount,
			"ragu_count":        raguCount,
			"total_pax_hadir":   totalPaxHadir,
		},
		"company": fiber.Map{
			"name":  companyName,
			"logo":  companyLogo,
			"phone": companyPhone,
		},
	})
}

// applyPortalInvitationUpdates memvalidasi dan menerapkan perubahan field undangan untuk Portal Pengantin maupun Akun Couple.
// Mengembalikan (httpStatus, errorMsg); jika httpStatus == 0 berarti validasi & penerapan berhasil (tinggal Save).
func applyPortalInvitationUpdates(invitation *models.Invitation, req *models.CreateOrUpdateInvitationRequest) (int, string) {
	// Validasi & perbarui slug jika diubah oleh pengantin
	if strings.TrimSpace(req.Slug) != "" {
		cleanSlug := sanitizeInvitationSlug(req.Slug)
		if cleanSlug == "" || len(cleanSlug) < 3 {
			return fiber.StatusBadRequest, "Kode tautan (slug) minimal 3 huruf/angka"
		}
		if isReservedInvitationSlug(cleanSlug) {
			return fiber.StatusBadRequest, "Kode tautan tersebut tidak dapat digunakan, silakan pilih nama lain"
		}
		if cleanSlug != invitation.Slug {
			var existingCount int64
			config.DB.Model(&models.Invitation{}).
				Where("LOWER(slug) = ? AND id != ?", cleanSlug, invitation.ID).
				Count(&existingCount)
			if existingCount > 0 {
				return fiber.StatusConflict, "Tautan /" + cleanSlug + " sudah dipakai undangan lain. Silakan tambahkan angka atau nama panggilan lain."
			}
			invitation.Slug = cleanSlug
		}
	}

	if strings.TrimSpace(req.EventType) != "" {
		invitation.EventType = strings.TrimSpace(req.EventType)
	}
	if strings.TrimSpace(req.Template) != "" {
		tmpl := strings.TrimSpace(req.Template)
		if tmpl == "art-jawa-biru-2" || tmpl == "jawa-biru-2" {
			tmpl = "jawa2"
		} else if tmpl == "art-jawa-coklat-3" || tmpl == "jawa-coklat-3" {
			tmpl = "jawa3"
		} else if tmpl == "pita-bola-disko" || tmpl == "birthday-1" {
			tmpl = "ultah1"
		}
		if tmpl != "pn4" && tmpl != "ta3" && tmpl != "ta17" && tmpl != "jawa3" && tmpl != "jawa2" && tmpl != "ultah1" {
			tmpl = "ta17"
		}
		invitation.Template = tmpl
	}

	if invitation.EventType == "khitan" {
		if strings.TrimSpace(req.ChildName) == "" && strings.TrimSpace(req.GroomName) == "" {
			return fiber.StatusBadRequest, "Nama anak khitan wajib diisi"
		}
		if strings.TrimSpace(req.GroomName) == "" {
			req.GroomName = req.ChildName
		}
	} else if invitation.EventType == "birthday" {
		if strings.TrimSpace(req.ChildName) == "" && strings.TrimSpace(req.BrideName) == "" && strings.TrimSpace(req.GroomName) == "" {
			return fiber.StatusBadRequest, "Nama yang berulang tahun wajib diisi"
		}
		if strings.TrimSpace(req.ChildName) == "" {
			if strings.TrimSpace(req.BrideName) != "" {
				req.ChildName = req.BrideName
			} else {
				req.ChildName = req.GroomName
			}
		}
		if strings.TrimSpace(req.GroomName) == "" {
			req.GroomName = req.ChildName
		}
		if strings.TrimSpace(req.BrideName) == "" {
			req.BrideName = req.ChildName
		}
	} else {
		if strings.TrimSpace(req.GroomName) == "" || strings.TrimSpace(req.BrideName) == "" {
			return fiber.StatusBadRequest, "Nama lengkap mempelai pria dan wanita wajib diisi"
		}
	}

	if strings.TrimSpace(req.Title) != "" {
		invitation.Title = strings.TrimSpace(req.Title)
	}
	if invitation.Template == "jawa3" || invitation.Template == "art-jawa-coklat-3" || invitation.Template == "jawa-coklat-3" {
		invitation.CoverImage = "/assets/templates/jawa3/bg-cover.jpg"
	} else if strings.TrimSpace(req.CoverImage) != "" {
		invitation.CoverImage = strings.TrimSpace(req.CoverImage)
	}

	invitation.Quotes = req.Quotes
	invitation.GroomName = strings.TrimSpace(req.GroomName)
	invitation.GroomNickname = strings.TrimSpace(req.GroomNickname)
	invitation.GroomChildOrder = strings.TrimSpace(req.GroomChildOrder)
	invitation.GroomParents = strings.TrimSpace(req.GroomParents)
	invitation.GroomInstagram = strings.TrimPrefix(strings.TrimSpace(req.GroomInstagram), "@")
	invitation.GroomPhoto = strings.TrimSpace(req.GroomPhoto)

	invitation.BrideName = strings.TrimSpace(req.BrideName)
	invitation.BrideNickname = strings.TrimSpace(req.BrideNickname)
	invitation.BrideChildOrder = strings.TrimSpace(req.BrideChildOrder)
	invitation.BrideParents = strings.TrimSpace(req.BrideParents)
	invitation.BrideInstagram = strings.TrimPrefix(strings.TrimSpace(req.BrideInstagram), "@")
	invitation.BridePhoto = strings.TrimSpace(req.BridePhoto)

	invitation.ChildName = strings.TrimSpace(req.ChildName)
	invitation.ChildNickname = strings.TrimSpace(req.ChildNickname)
	invitation.ChildOrder = strings.TrimSpace(req.ChildOrder)

	invitation.AkadDate = strings.TrimSpace(req.AkadDate)
	invitation.AkadTime = strings.TrimSpace(req.AkadTime)
	invitation.AkadVenue = strings.TrimSpace(req.AkadVenue)
	invitation.AkadAddress = strings.TrimSpace(req.AkadAddress)
	invitation.AkadMapsURL = strings.TrimSpace(req.AkadMapsURL)

	invitation.ResepsiDate = strings.TrimSpace(req.ResepsiDate)
	invitation.ResepsiTime = strings.TrimSpace(req.ResepsiTime)
	invitation.ResepsiVenue = strings.TrimSpace(req.ResepsiVenue)
	invitation.ResepsiAddress = strings.TrimSpace(req.ResepsiAddress)
	invitation.ResepsiMapsURL = strings.TrimSpace(req.ResepsiMapsURL)

	invitation.BankName1 = strings.TrimSpace(req.BankName1)
	invitation.BankAccount1 = strings.TrimSpace(req.BankAccount1)
	invitation.BankHolder1 = strings.TrimSpace(req.BankHolder1)
	invitation.BankName2 = strings.TrimSpace(req.BankName2)
	invitation.BankAccount2 = strings.TrimSpace(req.BankAccount2)
	invitation.BankHolder2 = strings.TrimSpace(req.BankHolder2)
	invitation.GiftAddress = strings.TrimSpace(req.GiftAddress)
	invitation.WaConfirmation = strings.TrimSpace(req.WaConfirmation)

	invitation.AudioURL = strings.TrimSpace(req.AudioURL)
	invitation.GalleryImages = strings.TrimSpace(req.GalleryImages)
	invitation.YoutubeURL = strings.TrimSpace(req.YoutubeURL)
	invitation.LoveStory = req.LoveStory
	invitation.TurutMengundang = req.TurutMengundang

	invitation.Sec1Photo = strings.TrimSpace(req.Sec1Photo)
	invitation.AkadPhoto = strings.TrimSpace(req.AkadPhoto)
	invitation.ResepsiPhoto = strings.TrimSpace(req.ResepsiPhoto)
	invitation.EventPhoto = strings.TrimSpace(req.EventPhoto)
	invitation.ClosingPhoto = strings.TrimSpace(req.ClosingPhoto)

	if req.IsActive != nil {
		invitation.IsActive = *req.IsActive
	}

	return 0, ""
}

// UpdatePortalInvitation menangani PUT /api/invitation-portal/:token (Simpan perubahan mandiri pengantin)
func UpdatePortalInvitation(c *fiber.Ctx) error {
	token := c.Params("token")
	invitation, err := findInvitationByEditToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Tautan akses kelola tidak valid atau sudah kedaluwarsa",
		})
	}

	var req models.CreateOrUpdateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	if errStatus, errMsg := applyPortalInvitationUpdates(invitation, &req); errStatus != 0 {
		return c.Status(errStatus).JSON(fiber.Map{
			"error": errMsg,
		})
	}

	if err := config.DB.Save(invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan perubahan undangan: " + err.Error(),
		})
	}

	// Invalidate in-memory cache undangan
	utils.GlobalInvitationCache.Invalidate(invitation.Slug)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Perubahan data undangan berhasil disimpan!",
		"data":    invitation,
	})
}

// SavePortalGuestList menangani PUT /api/invitation-portal/:token/guests
func SavePortalGuestList(c *fiber.Ctx) error {
	token := c.Params("token")
	invitation, err := findInvitationByEditToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Tautan akses kelola tidak valid",
		})
	}

	var req models.SavePortalGuestsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format daftar tamu tidak valid",
		})
	}

	if len(req.GuestListJSON) > 1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ukuran daftar tamu melebihi batas maksimal (1 MB)",
		})
	}

	if err := config.DB.Model(&models.Invitation{}).
		Where("id = ?", invitation.ID).
		Update("guest_list_json", req.GuestListJSON).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan daftar tamu: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar tamu berhasil disinkronkan",
	})
}

// UploadPortalMedia menangani POST /api/invitation-portal/:token/upload (Upload foto / lagu latar oleh pengantin)
func UploadPortalMedia(c *fiber.Ctx) error {
	token := c.Params("token")
	if _, err := findInvitationByEditToken(token); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Tautan akses kelola tidak valid",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		file, err = c.FormFile("image")
	}
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File wajib dipilih untuk diunggah",
		})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	audioExts := map[string]bool{".mp3": true, ".m4a": true}

	if !imageExts[ext] && !audioExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format file tidak didukung. Gunakan foto (.jpg, .png, .webp) atau audio (.mp3)",
		})
	}

	if imageExts[ext] && file.Size > 8*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ukuran foto terlalu besar (maksimal 8 MB per foto)",
		})
	}
	if audioExts[ext] && file.Size > 15*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ukuran file lagu terlalu besar (maksimal 15 MB)",
		})
	}

	uploadsDir := "./uploads"
	prefix := "inv_img"
	if audioExts[ext] {
		prefix = "inv_audio"
	}

	// Kompresi & optimasi otomatis gambar (resize maksimal 1600px & JPEG quality 82)
	filename, err := utils.ProcessAndSaveImage(file, uploadsDir, prefix)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan file ke server: " + err.Error(),
		})
	}

	// Gunakan path relatif /uploads/... atau URL absolut HTTPS sesuai host pengakses
	// Path relatif /uploads/<filename> aman untuk dekorin.my.id maupun undangan.dekorin.my.id!
	proto := strings.TrimSpace(c.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = c.Protocol()
	}
	fileURL := fmt.Sprintf("%s://%s/uploads/%s", proto, c.Hostname(), filename)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "File berhasil diunggah",
		"filename": filename,
		"url":      fileURL,
	})
}

// ExportPortalWishesCSV menangani GET /api/invitation-portal/:token/export-csv
func ExportPortalWishesCSV(c *fiber.Ctx) error {
	token := c.Params("token")
	invitation, err := findInvitationByEditToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Akses tidak valid")
	}

	var wishes []models.InvitationWish
	config.DB.Where("invitation_id = ?", invitation.ID).Order("created_at DESC").Find(&wishes)

	var buf bytes.Buffer
	// UTF-8 BOM agar Microsoft Excel langsung mengenali karakter Indonesia & simbol dengan rapi
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"No", "Nama Tamu", "Status Kehadiran", "Jumlah Orang (Pax)", "Ucapan & Doa", "Waktu Mengisi"})

	loc, locErr := time.LoadLocation("Asia/Jakarta")
	if locErr != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	totalHadir := 0
	totalPaxHadir := 0

	for idx, w := range wishes {
		attLabel := "Masih Ragu"
		attRaw := strings.ToLower(strings.TrimSpace(w.Attendance))
		pax := w.GuestCount
		if pax < 1 {
			pax = 1
		}
		if attRaw == "hadir" {
			attLabel = "Hadir"
			totalHadir++
			totalPaxHadir += pax
		} else if attRaw == "tidak_hadir" || attRaw == "tidak hadir" {
			attLabel = "Tidak Hadir"
			pax = 0
		}

		timeStr := w.CreatedAt.In(loc).Format("02-01-2006 15:04 WIB")
		_ = writer.Write([]string{
			strconv.Itoa(idx + 1),
			w.GuestName,
			attLabel,
			strconv.Itoa(pax),
			strings.ReplaceAll(strings.ReplaceAll(w.Message, "\r\n", " "), "\n", " "),
			timeStr,
		})
	}

	_ = writer.Write([]string{})
	_ = writer.Write([]string{"RINGKASAN REKAP KEHADIRAN", invitation.Title})
	_ = writer.Write([]string{"Total Respon Masuk", strconv.Itoa(len(wishes))})
	_ = writer.Write([]string{"Total Konfirmasi Hadir", strconv.Itoa(totalHadir)})
	_ = writer.Write([]string{"Total Estimasi Tamu Hadir (Pax)", strconv.Itoa(totalPaxHadir)})
	writer.Flush()

	fileName := fmt.Sprintf("rekap-kehadiran-%s.csv", invitation.Slug)
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	return c.Send(buf.Bytes())
}

// ResetInvitationAccess menangani POST /api/invitations/:id/reset-access (Oleh Vendor / Admin)
func ResetInvitationAccess(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	role, _ := c.Locals("user_role").(string)

	query := config.DB.Where("id = ?", id)
	if !strings.EqualFold(role, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	var invitation models.Invitation
	if err := query.First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan atau Anda tidak memiliki akses",
		})
	}

	invitation.EditToken = generateEditToken()
	invitation.EditPin = generateEditPin()

	if err := config.DB.Model(&models.Invitation{}).
		Where("id = ?", invitation.ID).
		Updates(map[string]interface{}{
			"edit_token": invitation.EditToken,
			"edit_pin":   invitation.EditPin,
		}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui akses kelola undangan: " + err.Error(),
		})
	}

	// Invalidate in-memory cache undangan
	utils.GlobalInvitationCache.Invalidate(invitation.Slug)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Link akses & PIN pengantin berhasil diperbarui",
		"data":    invitation,
	})
}
