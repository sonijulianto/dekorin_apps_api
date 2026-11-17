package handlers

import (
	"fmt"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"
	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func notifyCompanyAdminsAndPush(companyID, notifType, title, body, referenceID string, extraData map[string]string) {
	if strings.TrimSpace(companyID) == "" {
		return
	}

	var users []models.User
	config.DB.Select("id").Where("company_id = ? AND role IN ?", companyID, []string{"owner", "admin", "superadmin"}).Find(&users)
	if len(users) == 0 {
		return
	}

	userIDs := make([]string, 0, len(users))
	notifs := make([]models.AppNotification, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
		notifs = append(notifs, models.AppNotification{
			ID:          fmt.Sprintf("ntf_%s", uuid.New().String()[:8]),
			UserID:      u.ID,
			CompanyID:   companyID,
			Type:        notifType,
			Title:       title,
			Body:        body,
			ReferenceID: referenceID,
			IsRead:      false,
		})
	}
	_ = config.DB.Create(&notifs).Error

	var deviceTokens []models.UserDeviceToken
	config.DB.Where("user_id IN ?", userIDs).Find(&deviceTokens)
	if len(deviceTokens) == 0 {
		return
	}

	tokens := make([]string, 0, len(deviceTokens))
	for _, dt := range deviceTokens {
		if strings.TrimSpace(dt.Token) != "" {
			tokens = append(tokens, dt.Token)
		}
	}
	if len(tokens) == 0 {
		return
	}

	payloadData := map[string]string{
		"type":         notifType,
		"reference_id": referenceID,
	}
	for k, v := range extraData {
		payloadData[k] = v
	}

	go func(toks []string, t, b string, d map[string]string) {
		invalid := utils.SendPushToTokens(toks, t, b, d)
		if len(invalid) > 0 {
			config.DB.Where("token IN ?", invalid).Delete(&models.UserDeviceToken{})
		}
	}(tokens, title, body, payloadData)
}

// EnrichAgendasWithPartnerCosts mengisi accepted_partner_cost dan pending_partner_cost untuk slice Agenda.
func EnrichAgendasWithPartnerCosts(agendas []models.Agenda) {
	if len(agendas) == 0 {
		return
	}
	ids := make([]string, 0, len(agendas))
	for _, a := range agendas {
		ids = append(ids, a.ID)
	}
	var collabs []models.AgendaCollaboration
	if err := config.DB.Where("lead_agenda_id IN ? AND status IN ?", ids, []string{"accepted", "pending"}).Find(&collabs).Error; err != nil {
		return
	}

	acceptedMap := make(map[string]float64)
	pendingMap := make(map[string]float64)
	for _, c := range collabs {
		cost := c.TotalCost()
		if c.Status == "accepted" {
			acceptedMap[c.LeadAgendaID] += cost
		} else if c.Status == "pending" {
			pendingMap[c.LeadAgendaID] += cost
		}
	}

	for i := range agendas {
		agendas[i].AcceptedPartnerCost = acceptedMap[agendas[i].ID]
		agendas[i].PendingPartnerCost = pendingMap[agendas[i].ID]
	}
}

// SyncCollaborationPartnerAgendas menyinkronkan perubahan tanggal/lokasi/pembatalan dari Lead Agenda ke Partner Agenda.
func SyncCollaborationPartnerAgendas(leadAgenda models.Agenda) {
	var collabs []models.AgendaCollaboration
	if err := config.DB.Where("lead_agenda_id = ? AND status = ? AND partner_agenda_id <> ''", leadAgenda.ID, "accepted").Find(&collabs).Error; err != nil {
		return
	}
	for _, col := range collabs {
		updates := map[string]interface{}{
			"event_date_time": leadAgenda.EventDateTime,
			"event_address":   leadAgenda.EventAddress,
			"maps_url":        leadAgenda.MapsURL,
			"client_name":     leadAgenda.ClientName,
			"client_phone":    leadAgenda.ClientPhone,
			"updated_at":      time.Now(),
		}
		if strings.EqualFold(leadAgenda.Status, "cancelled") {
			updates["status"] = "cancelled"
		}
		config.DB.Model(&models.Agenda{}).Where("id = ?", col.PartnerAgendaID).Updates(updates)

		msg := fmt.Sprintf("Jadwal atau lokasi acara %s diperbarui oleh %s.", leadAgenda.ClientName, col.LeadCompanyName)
		if strings.EqualFold(leadAgenda.Status, "cancelled") {
			msg = fmt.Sprintf("Acara %s dibatalkan oleh %s.", leadAgenda.ClientName, col.LeadCompanyName)
		}
		notifyCompanyAdminsAndPush(
			col.PartnerCompanyID,
			"collaboration_updated",
			"Pembaruan Agenda Kolaborasi",
			msg,
			col.ID,
			map[string]string{"agenda_id": col.PartnerAgendaID},
		)
	}
}

// RegisterDeviceToken handles POST /api/devices/token
func RegisterDeviceToken(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	companyID := middleware.GetCompanyID(c)
	if strings.TrimSpace(userID) == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req models.RegisterDeviceTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	token := strings.TrimSpace(req.Token)
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token perangkat wajib diisi"})
	}
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform != "web" && platform != "ios" {
		platform = "android"
	}

	var existing models.UserDeviceToken
	err := config.DB.Where("token = ?", token).First(&existing).Error
	if err == nil {
		existing.UserID = userID
		existing.CompanyID = companyID
		existing.Platform = platform
		if saveErr := config.DB.Save(&existing).Error; saveErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui token perangkat"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
	}

	rec := models.UserDeviceToken{
		ID:        fmt.Sprintf("dtok_%s", uuid.New().String()[:8]),
		UserID:    userID,
		CompanyID: companyID,
		Platform:  platform,
		Token:     token,
	}
	if err := config.DB.Create(&rec).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan token perangkat"})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

// UnregisterDeviceToken handles DELETE /api/devices/token
func UnregisterDeviceToken(c *fiber.Ctx) error {
	var req models.RegisterDeviceTokenRequest
	_ = c.BodyParser(&req)
	token := strings.TrimSpace(req.Token)
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token != "" {
		config.DB.Where("token = ?", token).Delete(&models.UserDeviceToken{})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

// SearchCollaborationVendors handles GET /api/collaborations/vendors?category=...&q=...
func SearchCollaborationVendors(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	category := strings.TrimSpace(c.Query("category"))
	queryStr := strings.TrimSpace(c.Query("q"))

	dbQuery := config.DB.Model(&models.Company{}).Where("id <> ?", companyID)
	if category != "" && category != "all" {
		dbQuery = dbQuery.Where("category = ?", models.NormalizeVendorCategory(category))
	}
	if queryStr != "" {
		like := "%" + strings.ToLower(queryStr) + "%"
		dbQuery = dbQuery.Where("LOWER(name) LIKE ? OR LOWER(city) LIKE ? OR LOWER(phone) LIKE ?", like, like, like)
	}

	var companies []models.Company
	if err := dbQuery.Order("name asc").Limit(40).Find(&companies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mencari vendor partner"})
	}

	results := make([]models.CollaborationVendorOption, 0, len(companies))
	for _, comp := range companies {
		results = append(results, models.CollaborationVendorOption{
			ID:       comp.ID,
			Name:     comp.Name,
			Category: models.NormalizeVendorCategory(comp.Category),
			City:     comp.City,
			Phone:    comp.Phone,
			LogoURL:  comp.LogoURL,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   results,
	})
}

// GetAgendaCollaborations handles GET /api/agendas/:id/collaborations
func GetAgendaCollaborations(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	agendaID := c.Params("id")

	var agenda models.Agenda
	if err := config.DB.Where("id = ? AND company_id = ?", agendaID, companyID).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Agenda tidak ditemukan"})
	}

	var collabs []models.AgendaCollaboration
	if err := config.DB.Where("lead_agenda_id = ?", agendaID).Order("created_at desc").Find(&collabs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat data kolaborasi"})
	}

	var acceptedCost, pendingCost float64
	for _, col := range collabs {
		if col.Status == "accepted" {
			acceptedCost += col.TotalCost()
		} else if col.Status == "pending" {
			pendingCost += col.TotalCost()
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":                "success",
		"data":                  collabs,
		"accepted_partner_cost": acceptedCost,
		"pending_partner_cost":  pendingCost,
	})
}

// CreateAgendaCollaboration handles POST /api/agendas/:id/collaborations
func CreateAgendaCollaboration(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	userID, _ := c.Locals("user_id").(string)
	agendaID := c.Params("id")

	var agenda models.Agenda
	if err := config.DB.Where("id = ? AND company_id = ?", agendaID, companyID).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Agenda tidak ditemukan"})
	}

	var req models.CreateCollaborationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	partnerCompanyID := strings.TrimSpace(req.PartnerCompanyID)
	if partnerCompanyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Vendor partner wajib dipilih"})
	}
	if partnerCompanyID == companyID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tidak dapat mengundang usaha Anda sendiri"})
	}
	if req.PartnerPrice < 0 || req.PartnerTransportFee < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nominal harga dan transport tidak boleh negatif"})
	}

	var partnerComp models.Company
	if err := config.DB.Where("id = ?", partnerCompanyID).First(&partnerComp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Vendor partner tidak ditemukan"})
	}
	var leadComp models.Company
	_ = config.DB.Where("id = ?", companyID).First(&leadComp).Error

	var existingCount int64
	config.DB.Model(&models.AgendaCollaboration{}).
		Where("lead_agenda_id = ? AND partner_company_id = ? AND status IN ?", agendaID, partnerCompanyID, []string{"pending", "accepted"}).
		Count(&existingCount)
	if existingCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Vendor ini sudah diundang pada agenda ini"})
	}

	collab := models.AgendaCollaboration{
		ID:                  fmt.Sprintf("col_%s", uuid.New().String()[:8]),
		LeadAgendaID:        agenda.ID,
		LeadCompanyID:       companyID,
		LeadCompanyName:     leadComp.Name,
		PartnerCompanyID:    partnerComp.ID,
		PartnerCompanyName:  partnerComp.Name,
		PartnerCategory:     models.NormalizeVendorCategory(partnerComp.Category),
		PartnerPrice:        req.PartnerPrice,
		PartnerTransportFee: req.PartnerTransportFee,
		Notes:               strings.TrimSpace(req.Notes),
		Status:              "pending",
		CreatedByUserID:     userID,
	}

	if err := config.DB.Create(&collab).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat undangan kolaborasi"})
	}

	totalOffer := collab.TotalCost()
	leadName := leadComp.Name
	if leadName == "" {
		leadName = "Vendor Partner"
	}
	dateStr := agenda.EventDateTime.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan 2006")
	notifTitle := "Undangan Kolaborasi Baru"
	notifBody := fmt.Sprintf("%s mengundang Anda untuk acara %s (%s) — Nilai Rp %.0f", leadName, agenda.ClientName, dateStr, totalOffer)
	notifyCompanyAdminsAndPush(
		partnerComp.ID,
		"collaboration_invite",
		notifTitle,
		notifBody,
		collab.ID,
		map[string]string{"lead_agenda_id": agenda.ID},
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Undangan kolaborasi berhasil dikirim",
		"data":    collab,
	})
}

// CancelAgendaCollaboration handles DELETE /api/collaborations/:id
func CancelAgendaCollaboration(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	collabID := c.Params("id")

	var collab models.AgendaCollaboration
	if err := config.DB.Where("id = ? AND lead_company_id = ?", collabID, companyID).First(&collab).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data kolaborasi tidak ditemukan"})
	}

	if collab.Status == "cancelled" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&collab).Updates(map[string]interface{}{
			"status":     "cancelled",
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
		if collab.PartnerAgendaID != "" {
			_ = tx.Model(&models.Agenda{}).Where("id = ?", collab.PartnerAgendaID).Update("status", "cancelled").Error
		}
		return nil
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membatalkan kolaborasi"})
	}

	notifyCompanyAdminsAndPush(
		collab.PartnerCompanyID,
		"collaboration_updated",
		"Kolaborasi Dibatalkan",
		fmt.Sprintf("%s membatalkan undangan kolaborasi.", collab.LeadCompanyName),
		collab.ID,
		nil,
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Kolaborasi berhasil dibatalkan",
	})
}

// GetIncomingCollaborations handles GET /api/collaborations/incoming
func GetIncomingCollaborations(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)

	var collabs []models.AgendaCollaboration
	if err := config.DB.Where("partner_company_id = ?", companyID).Order("created_at desc").Find(&collabs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil undangan kolaborasi"})
	}

	if len(collabs) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":        "success",
			"data":          []models.IncomingCollaborationItem{},
			"pending_count": 0,
		})
	}

	leadAgendaIDs := make([]string, 0, len(collabs))
	for _, col := range collabs {
		leadAgendaIDs = append(leadAgendaIDs, col.LeadAgendaID)
	}

	var leadAgendas []models.Agenda
	config.DB.Where("id IN ?", leadAgendaIDs).Find(&leadAgendas)
	agendaMap := make(map[string]models.Agenda, len(leadAgendas))
	for _, a := range leadAgendas {
		agendaMap[a.ID] = a
	}

	// Ambil daftar tanggal agenda aktif milik partner untuk mendeteksi bentrok jadwal
	var partnerAgendas []models.Agenda
	config.DB.Select("event_date_time").Where("company_id = ? AND status <> ?", companyID, "cancelled").Find(&partnerAgendas)
	busyDates := make(map[string]bool)
	wib := time.FixedZone("WIB", 7*3600)
	for _, pa := range partnerAgendas {
		busyDates[pa.EventDateTime.In(wib).Format("2006-01-02")] = true
	}

	items := make([]models.IncomingCollaborationItem, 0, len(collabs))
	pendingCount := 0
	for _, col := range collabs {
		if col.Status == "pending" {
			pendingCount++
		}
		la := agendaMap[col.LeadAgendaID]
		dateKey := la.EventDateTime.In(wib).Format("2006-01-02")
		hasConflict := col.Status == "pending" && !la.EventDateTime.IsZero() && busyDates[dateKey]

		items = append(items, models.IncomingCollaborationItem{
			AgendaCollaboration: col,
			ClientName:          la.ClientName,
			ClientPhone:         la.ClientPhone,
			EventType:           la.EventType,
			EventLabel:          la.EventLabel,
			EventDateTime:       la.EventDateTime.Format(time.RFC3339),
			EventAddress:        la.EventAddress,
			MapsURL:             la.MapsURL,
			HasScheduleConflict: hasConflict,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":        "success",
		"data":          items,
		"pending_count": pendingCount,
	})
}

// RespondCollaboration handles POST /api/collaborations/:id/respond
func RespondCollaboration(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	userID, _ := c.Locals("user_id").(string)
	collabID := c.Params("id")

	var collab models.AgendaCollaboration
	if err := config.DB.Where("id = ? AND partner_company_id = ?", collabID, companyID).First(&collab).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Undangan kolaborasi tidak ditemukan"})
	}
	if collab.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Undangan ini sudah diproses sebelumnya"})
	}

	var req models.RespondCollaborationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "accept" && action != "reject" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Action harus 'accept' atau 'reject'"})
	}

	now := time.Now()
	if action == "reject" {
		collab.Status = "rejected"
		collab.RejectReason = strings.TrimSpace(req.RejectReason)
		collab.RespondedByUserID = userID
		collab.RespondedAt = &now
		if err := config.DB.Save(&collab).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menolak undangan"})
		}

		reasonText := ""
		if collab.RejectReason != "" {
			reasonText = " (" + collab.RejectReason + ")"
		}
		notifyCompanyAdminsAndPush(
			collab.LeadCompanyID,
			"collaboration_rejected",
			"Undangan Kolaborasi Ditolak",
			fmt.Sprintf("%s menolak undangan kolaborasi%s.", collab.PartnerCompanyName, reasonText),
			collab.ID,
			map[string]string{"lead_agenda_id": collab.LeadAgendaID},
		)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "success",
			"message": "Undangan kolaborasi ditolak",
			"data":    collab,
		})
	}

	// Accept: buat Agenda baru di akun Partner secara atomik dalam transaksi
	var leadAgenda models.Agenda
	if err := config.DB.Where("id = ?", collab.LeadAgendaID).First(&leadAgenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Agenda utama tidak ditemukan"})
	}

	newAgendaID := fmt.Sprintf("agd_%s", uuid.New().String()[:8])
	formToken := strings.ReplaceAll(uuid.New().String(), "-", "")
	partnerNotes := strings.TrimSpace(collab.Notes)
	if collab.LeadCompanyName != "" {
		prefix := fmt.Sprintf("[Kolaborasi dari %s]", collab.LeadCompanyName)
		if partnerNotes != "" {
			partnerNotes = prefix + "\n" + partnerNotes
		} else {
			partnerNotes = prefix
		}
	}

	newAgenda := models.Agenda{
		ID:            newAgendaID,
		CompanyID:     companyID,
		EventType:     leadAgenda.EventType,
		EventLabel:    leadAgenda.EventLabel,
		ClientName:    leadAgenda.ClientName,
		ClientPhone:   leadAgenda.ClientPhone,
		EventDateTime: leadAgenda.EventDateTime,
		EventAddress:  leadAgenda.EventAddress,
		MapsURL:       leadAgenda.MapsURL,
		PackageName:   fmt.Sprintf("Kolaborasi: %s", collab.LeadCompanyName),
		PackagePrice:  collab.PartnerPrice,
		TransportFee:  collab.PartnerTransportFee,
		Status:        "upcoming",
		PaymentStatus: "unpaid",
		FormToken:     formToken,
		FormStatus:    "pending",
		Notes:         partnerNotes,
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newAgenda).Error; err != nil {
			return err
		}
		return tx.Model(&collab).Updates(map[string]interface{}{
			"status":               "accepted",
			"partner_agenda_id":    newAgendaID,
			"responded_by_user_id": userID,
			"responded_at":         now,
			"updated_at":           now,
		}).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menerima undangan kolaborasi"})
	}

	collab.Status = "accepted"
	collab.PartnerAgendaID = newAgendaID
	collab.RespondedByUserID = userID
	collab.RespondedAt = &now

	notifyCompanyAdminsAndPush(
		collab.LeadCompanyID,
		"collaboration_accepted",
		"Undangan Kolaborasi Diterima",
		fmt.Sprintf("%s menerima undangan kolaborasi untuk acara %s.", collab.PartnerCompanyName, leadAgenda.ClientName),
		collab.ID,
		map[string]string{
			"lead_agenda_id":    collab.LeadAgendaID,
			"partner_agenda_id": newAgendaID,
		},
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Undangan diterima & agenda otomatis ditambahkan",
		"data":    collab,
		"agenda":  newAgenda,
	})
}

// GetNotifications handles GET /api/notifications
func GetNotifications(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	var notifs []models.AppNotification
	if err := config.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(50).Find(&notifs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat notifikasi"})
	}
	var unreadCount int64
	config.DB.Model(&models.AppNotification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&unreadCount)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "success",
		"data":         notifs,
		"unread_count": unreadCount,
	})
}

// MarkNotificationRead handles POST /api/notifications/:id/read
func MarkNotificationRead(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	notifID := c.Params("id")
	config.DB.Model(&models.AppNotification{}).Where("id = ? AND user_id = ?", notifID, userID).Update("is_read", true)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

// MarkAllNotificationsRead handles POST /api/notifications/read-all
func MarkAllNotificationsRead(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	config.DB.Model(&models.AppNotification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}
