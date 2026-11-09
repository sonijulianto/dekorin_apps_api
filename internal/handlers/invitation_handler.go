package handlers

import (
	"fmt"
	"html/template"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"
	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetInvitations menangani GET /api/invitations (daftar undangan untuk admin dengan pagination & filter)
func GetInvitations(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	role, _ := c.Locals("user_role").(string)
	if role == "" {
		role, _ = c.Locals("role").(string)
	}

	query := config.DB.Model(&models.Invitation{})

	// 1. Filter Scope Perusahaan / Hak Akses
	if strings.EqualFold(role, "superadmin") {
		if qComp := c.Query("company_id"); qComp != "" && qComp != "all" {
			query = query.Where("company_id = ?", qComp)
		}
	} else if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// 2. Filter Pencarian Teks (Judul, Slug, Nama Mempelai / Anak)
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(title) LIKE ? OR LOWER(slug) LIKE ? OR LOWER(groom_name) LIKE ? OR LOWER(bride_name) LIKE ? OR LOWER(child_name) LIKE ?",
			pattern, pattern, pattern, pattern, pattern,
		)
	}

	// 3. Filter Status (active / inactive)
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if status == "active" || status == "true" {
			query = query.Where("is_active = ?", true)
		} else if status == "inactive" || status == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	// 4. Hitung Total Data (untuk pagination metadata)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghitung total undangan: " + err.Error(),
		})
	}

	// 5. Parameter Paginasi
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	page := 1
	limit := 50 // default batas wajar agar aman dari memory leak
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100 // Batas maksimal per halaman
			}
		}
	}

	// 6. Optimasi Query:
	// - Omit 'guest_list_json' yang berukuran besar dan hanya dipakai di portal mandiri
	// - Preload hanya kolom penting dari Wishes (hindari load teks pesan panjang berulang)
	fetchQuery := query.
		Omit("guest_list_json").
		Preload("Wishes", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, invitation_id, attendance, guest_count, created_at")
		}).
		Order("created_at DESC")

	// Terapkan batas paginasi jika ada request page/limit atau data banyak
	if pageStr != "" || limitStr != "" || total > 50 {
		fetchQuery = fetchQuery.Limit(limit).Offset((page - 1) * limit)
	}

	var invitations []models.Invitation
	if err := fetchQuery.Find(&invitations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil daftar undangan: " + err.Error(),
		})
	}

	for i := range invitations {
		ensureInvitationAccessCredentials(&invitations[i])
		invitations[i].WishesCount = len(invitations[i].Wishes)
	}

	totalPages := 1
	if limit > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data undangan",
		"data":    invitations,
		"meta": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetInvitationByID menangani GET /api/invitations/:id
func GetInvitationByID(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	role, _ := c.Locals("user_role").(string)

	query := config.DB.Preload("Wishes").Where("id = ?", id)
	if !strings.EqualFold(role, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	var invitation models.Invitation
	if err := query.First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan atau Anda tidak memiliki akses",
		})
	}

	ensureInvitationAccessCredentials(&invitation)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil detail undangan",
		"data":    invitation,
	})
}

// CreateInvitation menangani POST /api/invitations
func CreateInvitation(c *fiber.Ctx) error {
	var req models.CreateOrUpdateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if req.EventType == "khitan" {
		if strings.TrimSpace(req.ChildName) == "" && strings.TrimSpace(req.GroomName) == "" {
			req.ChildName = "Rayyan Althaf Pratama"
			req.GroomName = req.ChildName
		}
		if strings.TrimSpace(req.GroomName) == "" {
			req.GroomName = req.ChildName
		}
		if strings.TrimSpace(req.ChildNickname) == "" {
			req.ChildNickname = "Rayyan"
		}
		if strings.TrimSpace(req.ChildOrder) == "" {
			req.ChildOrder = "Putra pertama dari"
		}
	} else {
		if strings.TrimSpace(req.GroomName) == "" {
			req.GroomName = "Bagas Pratama, S.Kom."
		}
		if strings.TrimSpace(req.BrideName) == "" {
			req.BrideName = "Zulia Maharani, S.Pd."
		}
		if strings.TrimSpace(req.GroomNickname) == "" {
			req.GroomNickname = "Bagas"
		}
		if strings.TrimSpace(req.BrideNickname) == "" {
			req.BrideNickname = "Zulia"
		}
		if strings.TrimSpace(req.GroomChildOrder) == "" {
			req.GroomChildOrder = "Putra pertama dari"
		}
		if strings.TrimSpace(req.BrideChildOrder) == "" {
			req.BrideChildOrder = "Putri kedua dari"
		}
		if strings.TrimSpace(req.GroomParents) == "" {
			req.GroomParents = "Bpk. Bambang & Ibu Siti"
		}
		if strings.TrimSpace(req.BrideParents) == "" {
			req.BrideParents = "Bpk. Agus & Ibu Dewi"
		}
	}

	if strings.TrimSpace(req.AkadDate) == "" {
		req.AkadDate = "Sabtu, 20 Desember 2026"
	}
	if strings.TrimSpace(req.AkadTime) == "" {
		req.AkadTime = "08:00 - 10:00 WIB"
	}
	if strings.TrimSpace(req.AkadVenue) == "" {
		req.AkadVenue = "Masjid Agung Al-Barkah"
	}
	if strings.TrimSpace(req.AkadAddress) == "" {
		req.AkadAddress = "Jl. Veteran No. 46, RT.004/RW.001, Marga Jaya, Bekasi Selatan"
	}
	if strings.TrimSpace(req.AkadMapsURL) == "" {
		req.AkadMapsURL = "https://maps.app.goo.gl/9bN52xXn"
	}

	if strings.TrimSpace(req.ResepsiDate) == "" {
		req.ResepsiDate = "Sabtu, 20 Desember 2026"
	}
	if strings.TrimSpace(req.ResepsiTime) == "" {
		req.ResepsiTime = "11:00 - 14:00 WIB"
	}
	if strings.TrimSpace(req.ResepsiVenue) == "" {
		req.ResepsiVenue = "Grand Ballroom Hotel Santika"
	}
	if strings.TrimSpace(req.ResepsiAddress) == "" {
		req.ResepsiAddress = "Jl. Jend. Ahmad Yani No. 1, RT.001/RW.005, Margajaya, Bekasi Selatan"
	}
	if strings.TrimSpace(req.ResepsiMapsURL) == "" {
		req.ResepsiMapsURL = "https://maps.app.goo.gl/9bN52xXn"
	}

	if strings.TrimSpace(req.Quotes) == "" {
		req.Quotes = `"Dan di antara tanda-tanda kekuasaan-Nya ialah Dia menciptakan untukmu isteri-isteri dari jenismu sendiri, supaya kamu cenderung dan merasa tenteram kepadanya, dan dijadikan-Nya di antaramu rasa kasih dan sayang." (QS. Ar-Rum: 21)`
	}
	req.BankName1 = strings.TrimSpace(req.BankName1)
	req.BankAccount1 = strings.TrimSpace(req.BankAccount1)
	req.BankHolder1 = strings.TrimSpace(req.BankHolder1)
	req.BankName2 = strings.TrimSpace(req.BankName2)
	req.BankAccount2 = strings.TrimSpace(req.BankAccount2)
	req.BankHolder2 = strings.TrimSpace(req.BankHolder2)
	req.GiftAddress = strings.TrimSpace(req.GiftAddress)
	if strings.TrimSpace(req.WaConfirmation) == "" {
		req.WaConfirmation = "081234567890"
	}
	if strings.TrimSpace(req.TurutMengundang) == "" {
		req.TurutMengundang = "Keluarga Besar Bpk. Bambang & Ibu Siti\nKeluarga Besar Bpk. Agus & Ibu Dewi"
	}
	if strings.TrimSpace(req.LoveStory) == "" {
		req.LoveStory = `[{"date":"1 September 2021","title":"Pertama Kali Bertemu","description":"Kami pertama kali bertemu di sebuah seminar kampus dan saling bertukar kontak."},{"date":"14 Februari 2023","title":"Komitmen Bersama","description":"Setelah melewati berbagai suka dan duka bersama, kami memutuskan untuk berkomitmen melangkah ke jenjang yang lebih serius."},{"date":"20 Oktober 2025","title":"Lamaran","description":"Dengan restu dari kedua keluarga besar, kami melangsungkan prosesi lamaran dan menyatukan tekad untuk menikah."}]`
	}

	isIllustration := req.Template == "ta3" || req.Template == "ta17" || req.Template == "ocean-ta17" || req.Template == "blue-ta17"
	isJawa3 := req.Template == "jawa3" || req.Template == "art-jawa-coklat-3" || req.Template == "jawa-coklat-3"
	if isJawa3 {
		if strings.TrimSpace(req.CoverImage) == "" || req.CoverImage == "/assets/templates/pn4/photos/LeniArdi2.jpeg" {
			req.CoverImage = "/assets/templates/jawa3/bg-cover.jpg"
		}
		if strings.TrimSpace(req.Sec1Photo) == "" || req.Sec1Photo == "/assets/templates/pn4/photos/LeniArdi15.jpeg" {
			req.Sec1Photo = "/assets/templates/jawa3/photos/gallery3.jpg"
		}
		if strings.TrimSpace(req.GroomPhoto) == "" || req.GroomPhoto == "/assets/templates/pn4/photos/LeniArdi15.jpeg" {
			req.GroomPhoto = "/assets/templates/jawa3/photos/groom-default.jpg"
		}
		if strings.TrimSpace(req.BridePhoto) == "" || req.BridePhoto == "/assets/templates/pn4/photos/LeniArdi16.jpeg" {
			req.BridePhoto = "/assets/templates/jawa3/photos/bride-default.jpg"
		}
		if strings.TrimSpace(req.AkadPhoto) == "" || req.AkadPhoto == "/assets/templates/pn4/photos/LeniArdi15.jpeg" {
			req.AkadPhoto = "/assets/templates/jawa3/photos/gallery4.jpg"
		}
		if strings.TrimSpace(req.ResepsiPhoto) == "" || req.ResepsiPhoto == "/assets/templates/pn4/photos/LeniArdi16.jpeg" {
			req.ResepsiPhoto = "/assets/templates/jawa3/photos/gallery4.jpg"
		}
		if strings.TrimSpace(req.EventPhoto) == "" || req.EventPhoto == "/assets/templates/pn4/photos/LeniArdi16.jpeg" {
			req.EventPhoto = "/assets/templates/jawa3/photos/gallery4.jpg"
		}
		if strings.TrimSpace(req.ClosingPhoto) == "" || req.ClosingPhoto == "/assets/templates/pn4/photos/LeniArdi2.jpeg" {
			req.ClosingPhoto = "/assets/templates/jawa3/photos/gallery5.jpg"
		}
		if strings.TrimSpace(req.GalleryImages) == "" || strings.HasPrefix(req.GalleryImages, "/assets/templates/pn4/photos/LeniArdi2.jpeg") {
			req.GalleryImages = "/assets/templates/jawa3/photos/gallery1.jpg,/assets/templates/jawa3/photos/gallery2.jpg,/assets/templates/jawa3/photos/gallery3.jpg,/assets/templates/jawa3/photos/gallery4.jpg,/assets/templates/jawa3/photos/gallery5.jpg,/assets/templates/jawa3/photos/gallery6.jpg,/assets/templates/jawa3/photos/gallery7.jpg,/assets/templates/jawa3/photos/gallery8.jpg"
		}
	} else if !isIllustration {
		if strings.TrimSpace(req.CoverImage) == "" {
			req.CoverImage = "/assets/templates/pn4/photos/LeniArdi2.jpeg"
		}
		if strings.TrimSpace(req.Sec1Photo) == "" {
			req.Sec1Photo = "/assets/templates/pn4/photos/LeniArdi15.jpeg"
		}
		if strings.TrimSpace(req.GroomPhoto) == "" {
			req.GroomPhoto = "/assets/templates/pn4/photos/LeniArdi15.jpeg"
		}
		if strings.TrimSpace(req.BridePhoto) == "" {
			req.BridePhoto = "/assets/templates/pn4/photos/LeniArdi16.jpeg"
		}
		if strings.TrimSpace(req.AkadPhoto) == "" {
			req.AkadPhoto = "/assets/templates/pn4/photos/LeniArdi15.jpeg"
		}
		if strings.TrimSpace(req.ResepsiPhoto) == "" {
			req.ResepsiPhoto = "/assets/templates/pn4/photos/LeniArdi16.jpeg"
		}
		if strings.TrimSpace(req.EventPhoto) == "" {
			req.EventPhoto = "/assets/templates/pn4/photos/LeniArdi16.jpeg"
		}
		if strings.TrimSpace(req.ClosingPhoto) == "" {
			req.ClosingPhoto = "/assets/templates/pn4/photos/LeniArdi2.jpeg"
		}
		if strings.TrimSpace(req.GalleryImages) == "" {
			req.GalleryImages = "/assets/templates/pn4/photos/LeniArdi2.jpeg,/assets/templates/pn4/photos/LeniArdi15.jpeg,/assets/templates/pn4/photos/LeniArdi16.jpeg"
		}
	}

	// Buat slug jika kosong
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		if req.EventType == "khitan" {
			childNick := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.ChildNickname), " ", "-"))
			if childNick == "" {
				childNick = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.ChildName), " ", "-"))
			}
			if childNick != "" {
				slug = fmt.Sprintf("khitan-%s", childNick)
			} else {
				slug = fmt.Sprintf("khitan-%s", uuid.New().String()[:6])
			}
		} else {
			groomNick := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.GroomNickname), " ", "-"))
			brideNick := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.BrideNickname), " ", "-"))
			if groomNick != "" && brideNick != "" {
				slug = fmt.Sprintf("%s-%s", groomNick, brideNick)
			} else {
				slug = fmt.Sprintf("undangan-%s", uuid.New().String()[:6])
			}
		}
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		if req.EventType == "khitan" {
			title = fmt.Sprintf("Tasyakuran Khitanan %s", req.ChildName)
		} else {
			title = fmt.Sprintf("The Wedding of %s & %s", req.GroomNickname, req.BrideNickname)
		}
	}

	invID := fmt.Sprintf("inv_%s", uuid.New().String()[:8])
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	coverImage := strings.TrimSpace(req.CoverImage)
	if coverImage == "" {
		coverImage = "/assets/owlink/b9c1e420-cd8d-11f0-9fb2-69fd61d4e768.jpg"
	}

	companyID := middleware.GetCompanyID(c)
	if qComp := c.Query("company_id"); qComp != "" && qComp != "all" {
		companyID = qComp
	}
	if companyID == "" {
		companyID = "cmp_default"
	}

	templateName := strings.TrimSpace(req.Template)
	if templateName != "pn4" && templateName != "ta3" && templateName != "ta17" && templateName != "jawa3" {
		templateName = "ta17"
	}

	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		eventType = "wedding"
	}

	invitation := models.Invitation{
		ID:              invID,
		CompanyID:       companyID,
		Slug:            slug,
		Title:           title,
		Template:        templateName,
		EventType:       eventType,
		CoverImage:      coverImage,
		Quotes:          req.Quotes,
		GroomName:       req.GroomName,
		GroomNickname:   req.GroomNickname,
		GroomChildOrder: req.GroomChildOrder,
		GroomParents:    req.GroomParents,
		GroomInstagram:  strings.TrimPrefix(req.GroomInstagram, "@"),
		GroomPhoto:      req.GroomPhoto,
		BrideName:       req.BrideName,
		BrideNickname:   req.BrideNickname,
		BrideChildOrder: req.BrideChildOrder,
		BrideParents:    req.BrideParents,
		BrideInstagram:  strings.TrimPrefix(req.BrideInstagram, "@"),
		BridePhoto:      req.BridePhoto,
		ChildName:       req.ChildName,
		ChildNickname:   req.ChildNickname,
		ChildOrder:      req.ChildOrder,
		AkadDate:        req.AkadDate,
		AkadTime:        req.AkadTime,
		AkadVenue:       req.AkadVenue,
		AkadAddress:     req.AkadAddress,
		AkadMapsURL:     req.AkadMapsURL,
		ResepsiDate:     req.ResepsiDate,
		ResepsiTime:     req.ResepsiTime,
		ResepsiVenue:    req.ResepsiVenue,
		ResepsiAddress:  req.ResepsiAddress,
		ResepsiMapsURL:  req.ResepsiMapsURL,
		BankName1:       req.BankName1,
		BankAccount1:    req.BankAccount1,
		BankHolder1:     req.BankHolder1,
		BankName2:       req.BankName2,
		BankAccount2:    req.BankAccount2,
		BankHolder2:     req.BankHolder2,
		GiftAddress:     req.GiftAddress,
		WaConfirmation:  req.WaConfirmation,
		AudioURL:        req.AudioURL,
		GalleryImages:   req.GalleryImages,
		YoutubeURL:      req.YoutubeURL,
		LoveStory:       req.LoveStory,
		TurutMengundang: req.TurutMengundang,
		Sec1Photo:       req.Sec1Photo,
		AkadPhoto:       req.AkadPhoto,
		ResepsiPhoto:    req.ResepsiPhoto,
		EventPhoto:      req.EventPhoto,
		ClosingPhoto:    req.ClosingPhoto,
		EditToken:       generateEditToken(),
		EditPin:         generateEditPin(),
		IsActive:        isActive,
	}

	if err := config.DB.Create(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat undangan: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Undangan digital berhasil dibuat",
		"data":    invitation,
	})
}

// UpdateInvitation menangani PUT /api/invitations/:id
func UpdateInvitation(c *fiber.Ctx) error {
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

	var req models.CreateOrUpdateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if strings.TrimSpace(req.Template) != "" {
		tmpl := strings.TrimSpace(req.Template)
		if tmpl != "pn4" && tmpl != "ta3" && tmpl != "ta17" && tmpl != "jawa3" {
			tmpl = "ta17"
		}
		invitation.Template = tmpl
	}
	if strings.TrimSpace(req.EventType) != "" {
		invitation.EventType = strings.TrimSpace(req.EventType)
	}
	if strings.TrimSpace(req.Slug) != "" {
		invitation.Slug = strings.TrimSpace(req.Slug)
	}
	if strings.TrimSpace(req.Title) != "" {
		invitation.Title = strings.TrimSpace(req.Title)
	}
	invitation.CoverImage = req.CoverImage
	invitation.Quotes = req.Quotes
	invitation.GroomName = req.GroomName
	invitation.GroomNickname = req.GroomNickname
	invitation.GroomChildOrder = req.GroomChildOrder
	invitation.GroomParents = req.GroomParents
	invitation.GroomInstagram = strings.TrimPrefix(req.GroomInstagram, "@")
	invitation.GroomPhoto = req.GroomPhoto
	invitation.BrideName = req.BrideName
	invitation.BrideNickname = req.BrideNickname
	invitation.BrideChildOrder = req.BrideChildOrder
	invitation.BrideParents = req.BrideParents
	invitation.BrideInstagram = strings.TrimPrefix(req.BrideInstagram, "@")
	invitation.BridePhoto = req.BridePhoto
	invitation.ChildName = req.ChildName
	invitation.ChildNickname = req.ChildNickname
	invitation.ChildOrder = req.ChildOrder
	invitation.AkadDate = req.AkadDate
	invitation.AkadTime = req.AkadTime
	invitation.AkadVenue = req.AkadVenue
	invitation.AkadAddress = req.AkadAddress
	invitation.AkadMapsURL = req.AkadMapsURL
	invitation.ResepsiDate = req.ResepsiDate
	invitation.ResepsiTime = req.ResepsiTime
	invitation.ResepsiVenue = req.ResepsiVenue
	invitation.ResepsiAddress = req.ResepsiAddress
	invitation.ResepsiMapsURL = req.ResepsiMapsURL
	invitation.BankName1 = req.BankName1
	invitation.BankAccount1 = req.BankAccount1
	invitation.BankHolder1 = req.BankHolder1
	invitation.BankName2 = req.BankName2
	invitation.BankAccount2 = req.BankAccount2
	invitation.BankHolder2 = req.BankHolder2
	invitation.GiftAddress = req.GiftAddress
	invitation.WaConfirmation = req.WaConfirmation
	invitation.AudioURL = req.AudioURL
	invitation.GalleryImages = req.GalleryImages
	invitation.YoutubeURL = req.YoutubeURL
	invitation.LoveStory = req.LoveStory
	invitation.TurutMengundang = req.TurutMengundang
	invitation.Sec1Photo = req.Sec1Photo
	invitation.AkadPhoto = req.AkadPhoto
	invitation.ResepsiPhoto = req.ResepsiPhoto
	invitation.EventPhoto = req.EventPhoto
	invitation.ClosingPhoto = req.ClosingPhoto
	if req.IsActive != nil {
		invitation.IsActive = *req.IsActive
	}
	if strings.TrimSpace(invitation.EditToken) == "" {
		invitation.EditToken = generateEditToken()
	}
	if strings.TrimSpace(invitation.EditPin) == "" {
		invitation.EditPin = generateEditPin()
	}

	if err := config.DB.Save(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui undangan: " + err.Error(),
		})
	}

	// Invalidate in-memory cache undangan
	utils.GlobalInvitationCache.Invalidate(invitation.Slug)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Undangan digital berhasil diperbarui",
		"data":    invitation,
	})
}

// DeleteInvitation menangani DELETE /api/invitations/:id
func DeleteInvitation(c *fiber.Ctx) error {
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

	// Hapus ucapan terkait terlebih dahulu
	config.DB.Where("invitation_id = ?", id).Delete(&models.InvitationWish{})

	if err := config.DB.Delete(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus undangan",
		})
	}

	// Invalidate in-memory cache undangan
	utils.GlobalInvitationCache.Invalidate(invitation.Slug)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Undangan digital berhasil dihapus",
	})
}

// GetInvitationWishes menangani GET /api/invitations/:id/wishes (dengan pagination & counter)
func GetInvitationWishes(c *fiber.Ctx) error {
	id := c.Params("id")

	query := config.DB.Model(&models.InvitationWish{}).Where("invitation_id = ?", id)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghitung ucapan: " + err.Error(),
		})
	}

	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	page := 1
	limit := 50
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 200 {
				limit = 200
			}
		}
	}

	var wishes []models.InvitationWish
	fetchQuery := query.Order("created_at DESC")
	if pageStr != "" || limitStr != "" {
		fetchQuery = fetchQuery.Limit(limit).Offset((page - 1) * limit)
	} else if total > 100 {
		fetchQuery = fetchQuery.Limit(100)
	}

	if err := fetchQuery.Find(&wishes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data ucapan: " + err.Error(),
		})
	}

	totalPages := 1
	if limit > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil ucapan tamu",
		"data":    wishes,
		"meta": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// SubmitInvitationWish menangani POST /api/invitations/:slug/wishes
func SubmitInvitationWish(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var invitation models.Invitation
	if err := config.DB.Where("slug = ?", slug).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan",
		})
	}

	var req models.SubmitWishRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if strings.TrimSpace(req.GuestName) == "" || strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama dan ucapan wajib diisi",
		})
	}

	guestCount := req.GuestCount
	if guestCount < 1 {
		guestCount = 1
	}

	attendance := strings.TrimSpace(req.Attendance)
	if attendance == "" {
		attendance = "hadir"
	}

	wish := models.InvitationWish{
		ID:           fmt.Sprintf("wsh_%s", uuid.New().String()[:8]),
		InvitationID: invitation.ID,
		GuestName:    strings.TrimSpace(req.GuestName),
		Attendance:   attendance,
		GuestCount:   guestCount,
		Message:      strings.TrimSpace(req.Message),
		CreatedAt:    time.Now(),
	}

	if err := config.DB.Create(&wish).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan ucapan: " + err.Error(),
		})
	}

	// Invalidate in-memory cache undangan agar ucapan baru langsung muncul saat reload
	utils.GlobalInvitationCache.Invalidate(slug)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Ucapan berhasil dikirim",
		"data":    wish,
	})
}

// convertYoutubeToEmbed mengubah URL youtube biasa menjadi format embed iframe
func convertYoutubeToEmbed(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if strings.Contains(rawURL, "youtube.com/embed/") {
		return rawURL
	}
	if strings.Contains(rawURL, "youtu.be/") {
		parts := strings.Split(rawURL, "youtu.be/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "?")[0]
			return fmt.Sprintf("https://www.youtube.com/embed/%s", id)
		}
	}
	if strings.Contains(rawURL, "watch?v=") {
		parts := strings.Split(rawURL, "watch?v=")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "&")[0]
			return fmt.Sprintf("https://www.youtube.com/embed/%s", id)
		}
	}
	return rawURL
}

// In-memory cache untuk parsed HTML template agar tidak re-parse dari disk pada setiap HTTP request
var (
	tmplCacheMu sync.RWMutex
	tmplCache   = make(map[string]*template.Template)
)

func getParsedTemplate(templateFile string) (*template.Template, error) {
	tmplCacheMu.RLock()
	tmpl, exists := tmplCache[templateFile]
	tmplCacheMu.RUnlock()
	if exists {
		return tmpl, nil
	}

	tmplCacheMu.Lock()
	defer tmplCacheMu.Unlock()

	// Double check setelah write lock
	if tmpl, exists = tmplCache[templateFile]; exists {
		return tmpl, nil
	}

	templatePath := filepath.Join("web", "templates", templateFile)
	parsedTmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		fallbackPath := filepath.Join("web", "templates", "template_ta17.html")
		var fallbackErr error
		parsedTmpl, fallbackErr = template.ParseFiles(fallbackPath)
		if fallbackErr != nil {
			return nil, fmt.Errorf("gagal memuat template: %w (fallback: %v)", err, fallbackErr)
		}
	}

	tmplCache[templateFile] = parsedTmpl
	return parsedTmpl, nil
}

// ShowPublicInvitation menangani GET /invitation/:slug (Render Halaman HTML Web Undangan dengan In-Memory Caching)
func ShowPublicInvitation(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var invitation models.Invitation
	var wishes []models.InvitationWish

	// Cek In-Memory Cache terlebih dahulu untuk kecepatan render instan (<1ms) & hemat koneksi PostgreSQL
	if cachedInv, cachedWishes, found := utils.GlobalInvitationCache.Get(slug); found && c.Query("preview") != "1" {
		invitation = *cachedInv
		wishes = cachedWishes
	} else {
		if err := config.DB.Where("slug = ?", slug).First(&invitation).Error; err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Undangan tidak ditemukan atau tautan salah.")
		}

		// Ambil daftar ucapan (dibatasi 50 ucapan terbaru pada initial render agar HTML tetap ringan)
		config.DB.Where("invitation_id = ?", invitation.ID).Order("created_at DESC").Limit(50).Find(&wishes)

		// Simpan ke memory cache jika bukan mode preview
		if c.Query("preview") != "1" {
			utils.GlobalInvitationCache.Set(slug, invitation, wishes)
		}
	}

	// Jika undangan dinonaktifkan sementara (Draft) dan bukan mode preview portal
	if !invitation.IsActive && c.Query("preview") != "1" {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Status(fiber.StatusOK).SendString(`<!DOCTYPE html><html lang="id"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Undangan Sedang Disiapkan</title><link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;600;700&display=swap" rel="stylesheet"></head><body style="margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#FAF7F2;font-family:'Plus Jakarta Sans',sans-serif;padding:24px;"><div style="max-width:420px;background:#fff;border:1px solid rgba(212,165,116,0.3);border-radius:24px;padding:32px;text-align:center;box-shadow:0 10px 30px rgba(0,0,0,0.04);"><div style="font-size:40px;margin-bottom:12px;">✨</div><h1 style="margin:0 0 8px;font-size:20px;color:#2C2C2C;">Undangan Sedang Disiapkan</h1><p style="margin:0;font-size:14px;color:#64748B;line-height:1.6;">Tautan undangan ini sedang dalam tahap penyempurnaan oleh tuan rumah. Silakan kunjungi kembali nanti.</p></div></body></html>`)
	}

	if strings.TrimSpace(invitation.Template) == "" {
		invitation.Template = "ta17"
	}
	if strings.TrimSpace(invitation.EventType) == "" {
		invitation.EventType = "wedding"
	}

	// Ambil nama tamu dari query parameter ?to=...
	rawGuestTo := strings.TrimSpace(c.Query("to"))
	hasGuestTo := rawGuestTo != ""
	guestTo := rawGuestTo
	if !hasGuestTo {
		guestTo = "Tamu Undangan"
	} else {
		// Rapikan format nama tamu jika ada tanda _ atau -
		guestTo = strings.ReplaceAll(guestTo, "_", " ")
		guestTo = strings.ReplaceAll(guestTo, "-", " & ")
	}

	// Pisahkan gallery images
	var galleryList []string
	if strings.TrimSpace(invitation.GalleryImages) != "" {
		rawImages := strings.Split(invitation.GalleryImages, ",")
		for _, img := range rawImages {
			trimmed := strings.TrimSpace(img)
			if trimmed != "" {
				galleryList = append(galleryList, trimmed)
			}
		}
	}

	// Pisahkan turut mengundang
	var turutList []string
	if strings.TrimSpace(invitation.TurutMengundang) != "" {
		rawTurut := strings.Split(strings.ReplaceAll(invitation.TurutMengundang, "\r\n", "\n"), "\n")
		for _, item := range rawTurut {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				turutList = append(turutList, trimmed)
			}
		}
	}

	// Tentukan file template HTML berdasarkan pilihan template undangan
	templateName := strings.TrimSpace(invitation.Template)
	// Memungkinkan preview template via query param: ?template=luxury-grey
	if overrideTemplate := strings.TrimSpace(c.Query("template")); overrideTemplate != "" {
		templateName = overrideTemplate
	}

	// Format Open Graph metadata untuk WhatsApp / Social Sharing
	baseURL := "https://" + c.Hostname()
	if strings.Contains(c.Hostname(), "localhost") {
		baseURL = "https://undangan.dekorin.my.id"
	}

	// 1. OG Title (The Wedding of Bride & Groom)
	ogTitle := strings.TrimSpace(invitation.Title)
	if ogTitle == "" {
		if invitation.EventType == "khitan" {
			child := strings.TrimSpace(invitation.ChildNickname)
			if child == "" {
				child = strings.TrimSpace(invitation.ChildName)
			}
			if child == "" {
				child = "Putra Kami"
			}
			ogTitle = "Tasyakuran Khitan " + child
		} else {
			bride := strings.TrimSpace(invitation.BrideNickname)
			if bride == "" {
				bride = strings.TrimSpace(invitation.BrideName)
			}
			if bride == "" {
				bride = "Mempelai Wanita"
			}
			groom := strings.TrimSpace(invitation.GroomNickname)
			if groom == "" {
				groom = strings.TrimSpace(invitation.GroomName)
			}
			if groom == "" {
				groom = "Mempelai Pria"
			}
			ogTitle = "The Wedding of " + bride + " & " + groom
		}
	}

	// 2. OG Description (Tanggal Acara Utama)
	ogDate := strings.TrimSpace(invitation.AkadDate)
	resepsiDate := strings.TrimSpace(invitation.ResepsiDate)
	if ogDate == "" {
		ogDate = resepsiDate
	} else if resepsiDate != "" && resepsiDate != ogDate {
		ogDate = ogDate + " & " + resepsiDate
	}
	if ogDate == "" {
		if invitation.EventType == "khitan" {
			ogDate = "Tasyakuran Khitan"
		} else {
			ogDate = "Acara Pernikahan"
		}
	}

	// 3. OG Image (Gambar resolusi tinggi & absolute URL untuk WhatsApp)
	ogImage := ""
	if invitation.CoverImage != "" && !strings.Contains(invitation.CoverImage, "b9c1e420-cd8d-11f0-9fb2-69fd61d4e768.jpg") {
		ogImage = invitation.CoverImage
	} else if len(galleryList) > 0 && galleryList[0] != "" {
		ogImage = galleryList[0]
	} else if invitation.Sec1Photo != "" {
		ogImage = invitation.Sec1Photo
	} else if invitation.BridePhoto != "" {
		ogImage = invitation.BridePhoto
	} else if invitation.GroomPhoto != "" {
		ogImage = invitation.GroomPhoto
	}

	if ogImage == "" {
		switch templateName {
		case "ta3", "sakura-ta3":
			ogImage = "/assets/templates/ta3/couple-welcome.png"
		case "pn4", "monochrome-pn4":
			ogImage = "/assets/templates/pn4/photos/LeniArdi2.jpeg"
		case "jawa3", "art-jawa-coklat-3", "jawa-coklat-3":
			ogImage = "/assets/templates/jawa3/bg-cover.jpg"
		default:
			ogImage = "/assets/templates/ta17/couple-welcome.png"
		}
	}

	// Pastikan ogImage selalu URL absolut (dimulai dengan https://)
	if strings.HasPrefix(ogImage, "/") {
		ogImage = baseURL + ogImage
	} else if !strings.HasPrefix(ogImage, "http://") && !strings.HasPrefix(ogImage, "https://") {
		ogImage = baseURL + "/" + ogImage
	}

	// 4. OG URL (Link utama undangan)
	ogURL := baseURL + "/" + invitation.Slug

	data := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation:          invitation,
		GuestTo:             guestTo,
		HasGuestTo:          hasGuestTo,
		GalleryList:         galleryList,
		Wishes:              wishes,
		YoutubeEmbedURL:     convertYoutubeToEmbed(invitation.YoutubeURL),
		TurutMengundangList: turutList,
		OGTitle:             ogTitle,
		OGDescription:       ogDate,
		OGImage:             ogImage,
		OGURL:               ogURL,
	}

	var templateFile string
	switch templateName {
	case "pn4", "monochrome-pn4":
		templateFile = "template_pn4.html"
	case "ta3", "sakura-ta3":
		templateFile = "template_ta3.html"
	case "ta17", "ocean-ta17", "blue-ta17":
		templateFile = "template_ta17.html"
	case "jawa3", "art-jawa-coklat-3", "jawa-coklat-3":
		templateFile = "template_jawa3.html"
	default:
		// Fallback otomatis ke Ocean Blossom (TA17) untuk template lama / selain yang disisakan
		templateFile = "template_ta17.html"
	}

	tmpl, err := getParsedTemplate(templateFile)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Gagal memuat template undangan: " + err.Error())
	}

	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	c.Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Response().BodyWriter(), data)
}
