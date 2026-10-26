package handlers

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetInvitations menangani GET /api/invitations (daftar undangan untuk admin)
func GetInvitations(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	var invitations []models.Invitation
	query := config.DB.Model(&models.Invitation{})
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.Order("created_at DESC").Find(&invitations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil daftar undangan: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data undangan",
		"data":    invitations,
	})
}

// GetInvitationByID menangani GET /api/invitations/:id
func GetInvitationByID(c *fiber.Ctx) error {
	id := c.Params("id")
	var invitation models.Invitation
	if err := config.DB.Preload("Wishes").Where("id = ?", id).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan",
		})
	}

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

	if strings.TrimSpace(req.GroomName) == "" || strings.TrimSpace(req.BrideName) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama mempelai pria dan wanita wajib diisi",
		})
	}

	// Buat slug jika kosong
	slug := strings.TrimSpace(req.Slug)
	if slug == "" {
		groomNick := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.GroomNickname), " ", "-"))
		brideNick := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(req.BrideNickname), " ", "-"))
		if groomNick != "" && brideNick != "" {
			slug = fmt.Sprintf("%s-%s", groomNick, brideNick)
		} else {
			slug = fmt.Sprintf("undangan-%s", uuid.New().String()[:6])
		}
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = fmt.Sprintf("The Wedding of %s & %s", req.GroomNickname, req.BrideNickname)
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
	if companyID == "" {
		companyID = "cmp_default"
	}

	templateName := strings.TrimSpace(req.Template)
	if templateName == "" {
		templateName = "green-leaves"
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
	var invitation models.Invitation
	if err := config.DB.Where("id = ?", id).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Undangan tidak ditemukan",
		})
	}

	var req models.CreateOrUpdateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if strings.TrimSpace(req.Template) != "" {
		invitation.Template = strings.TrimSpace(req.Template)
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
	if req.IsActive != nil {
		invitation.IsActive = *req.IsActive
	}

	if err := config.DB.Save(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui undangan: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Undangan digital berhasil diperbarui",
		"data":    invitation,
	})
}

// DeleteInvitation menangani DELETE /api/invitations/:id
func DeleteInvitation(c *fiber.Ctx) error {
	id := c.Params("id")
	// Hapus ucapan terkait terlebih dahulu
	config.DB.Where("invitation_id = ?", id).Delete(&models.InvitationWish{})

	if err := config.DB.Where("id = ?", id).Delete(&models.Invitation{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus undangan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Undangan digital berhasil dihapus",
	})
}

// GetInvitationWishes menangani GET /api/invitations/:id/wishes
func GetInvitationWishes(c *fiber.Ctx) error {
	id := c.Params("id")
	var wishes []models.InvitationWish
	if err := config.DB.Where("invitation_id = ?", id).Order("created_at DESC").Find(&wishes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data ucapan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil ucapan tamu",
		"data":    wishes,
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

// ShowPublicInvitation menangani GET /invitation/:slug (Render Halaman HTML Web Undangan)
func ShowPublicInvitation(c *fiber.Ctx) error {
	slug := c.Params("slug")
	var invitation models.Invitation
	if err := config.DB.Where("slug = ?", slug).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Undangan tidak ditemukan atau tautan salah.")
	}

	if strings.TrimSpace(invitation.Template) == "" {
		invitation.Template = "green-leaves"
	}
	if strings.TrimSpace(invitation.EventType) == "" {
		invitation.EventType = "wedding"
	}

	// Ambil nama tamu dari query parameter ?to=...
	guestTo := c.Query("to")
	if strings.TrimSpace(guestTo) == "" {
		guestTo = "Tamu Undangan"
	} else {
		// Rapikan format nama tamu jika ada tanda _ atau -
		guestTo = strings.ReplaceAll(guestTo, "_", " ")
		guestTo = strings.ReplaceAll(guestTo, "-", " & ")
	}

	// Ambil daftar ucapan
	var wishes []models.InvitationWish
	config.DB.Where("invitation_id = ?", invitation.ID).Order("created_at DESC").Find(&wishes)

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

	data := struct {
		models.Invitation
		GuestTo           string
		GalleryList       []string
		Wishes            []models.InvitationWish
		YoutubeEmbedURL   string
		TurutMengundangList []string
	}{
		Invitation:          invitation,
		GuestTo:             guestTo,
		GalleryList:         galleryList,
		Wishes:              wishes,
		YoutubeEmbedURL:     convertYoutubeToEmbed(invitation.YoutubeURL),
		TurutMengundangList: turutList,
	}

	// Tentukan file template HTML berdasarkan pilihan template undangan
	templateName := strings.TrimSpace(invitation.Template)
	// Memungkinkan preview template via query param: ?template=luxury-grey
	if overrideTemplate := strings.TrimSpace(c.Query("template")); overrideTemplate != "" {
		templateName = overrideTemplate
	}

	var templateFile string
	switch templateName {
	case "floral":
		templateFile = "template_floral.html"
	case "luxury-grey":
		templateFile = "template_luxury_grey.html"
	case "khitan-green":
		templateFile = "template_khitan_green.html"
	case "creamy-latte":
		templateFile = "template_creamy_latte.html"
	case "green-leaves":
		templateFile = "template_green_leaves.html"
	default:
		templateFile = "template_green_leaves.html"
	}

	templatePath := filepath.Join("web", "templates", templateFile)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		// Fallback ke template default jika file spesifik tidak ditemukan
		fallbackPath := filepath.Join("web", "templates", "wedding_invitation.html")
		var fallbackErr error
		tmpl, fallbackErr = template.ParseFiles(fallbackPath)
		if fallbackErr != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Gagal memuat template undangan: " + err.Error())
		}
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Response().BodyWriter(), data)
}
