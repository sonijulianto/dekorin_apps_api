package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"
	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// chatHub mengelola koneksi WebSocket aktif per room diskusi kolaborasi.
type chatHubManager struct {
	mu    sync.RWMutex
	rooms map[string]map[*websocket.Conn]string // roomID -> conn -> userID
}

var chatHub = &chatHubManager{
	rooms: make(map[string]map[*websocket.Conn]string),
}

func (h *chatHubManager) Register(roomID string, conn *websocket.Conn, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*websocket.Conn]string)
	}
	h.rooms[roomID][conn] = userID
}

func (h *chatHubManager) Unregister(roomID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.rooms[roomID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *chatHubManager) Broadcast(roomID string, eventType string, payload interface{}) {
	h.BroadcastExcept(roomID, nil, eventType, payload)
}

func (h *chatHubManager) BroadcastExcept(roomID string, excludeConn *websocket.Conn, eventType string, payload interface{}) {
	data, err := json.Marshal(fiber.Map{
		"event": eventType,
		"data":  payload,
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	connsMap := h.rooms[roomID]
	conns := make([]*websocket.Conn, 0, len(connsMap))
	for conn := range connsMap {
		if excludeConn != nil && conn == excludeConn {
			continue
		}
		conns = append(conns, conn)
	}
	h.mu.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			h.Unregister(roomID, conn)
			_ = conn.Close()
		}
	}
}

func (h *chatHubManager) ActiveUserIDs(roomID string) map[string]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	active := make(map[string]bool)
	for _, uid := range h.rooms[roomID] {
		if uid != "" {
			active[uid] = true
		}
	}
	return active
}

type chatAccessContext struct {
	Room            models.AgendaChatRoom
	LeadAgenda      models.Agenda
	AcceptedCollabs []models.AgendaCollaboration
	CompanyName     string
	CategoryLabel   string
	UserName        string
	UserRole        string
}

// resolveChatAccess memvalidasi apakah user berhak mengakses room diskusi agenda kolaborasi.
func resolveChatAccess(userID, companyID, role, agendaID string) (*chatAccessContext, int, string) {
	normRole := strings.ToLower(strings.TrimSpace(role))
	if userID == "" || agendaID == "" {
		return nil, fiber.StatusBadRequest, "Parameter agenda tidak valid"
	}

	var agenda models.Agenda
	if err := config.DB.Where("id = ?", agendaID).First(&agenda).Error; err != nil {
		return nil, fiber.StatusNotFound, "Agenda tidak ditemukan"
	}

	if normRole != "superadmin" && agenda.CompanyID != companyID {
		return nil, fiber.StatusForbidden, "Anda tidak memiliki akses ke agenda ini"
	}

	// Tentukan LeadAgendaID: apakah agendaID adalah LeadAgenda atau PartnerAgenda
	leadAgendaID := agenda.ID
	var leadAgenda models.Agenda
	var acceptedCollabs []models.AgendaCollaboration

	// Cek apakah agenda ini adalah PartnerAgenda dari kolaborasi yang accepted
	var asPartner models.AgendaCollaboration
	if err := config.DB.Where("partner_agenda_id = ? AND status = ?", agenda.ID, "accepted").First(&asPartner).Error; err == nil {
		leadAgendaID = asPartner.LeadAgendaID
		if err := config.DB.Where("id = ?", leadAgendaID).First(&leadAgenda).Error; err != nil {
			return nil, fiber.StatusNotFound, "Agenda utama kolaborasi tidak ditemukan"
		}
	} else {
		leadAgenda = agenda
	}

	config.DB.Where("lead_agenda_id = ? AND status = ?", leadAgendaID, "accepted").Order("created_at ASC").Find(&acceptedCollabs)
	if len(acceptedCollabs) == 0 {
		return nil, fiber.StatusForbidden, "Diskusi kolaborasi hanya aktif setelah minimal 1 partner menerima undangan"
	}

	// Pastikan room sudah ada di DB
	var room models.AgendaChatRoom
	err := config.DB.Where("lead_agenda_id = ?", leadAgendaID).First(&room).Error
	if err == gorm.ErrRecordNotFound {
		room = models.AgendaChatRoom{
			ID:           fmt.Sprintf("chat_%s", uuid.New().String()[:8]),
			LeadAgendaID: leadAgendaID,
		}
		if createErr := config.DB.Create(&room).Error; createErr != nil {
			// Handle race condition jika dibuat bersamaan
			if retryErr := config.DB.Where("lead_agenda_id = ?", leadAgendaID).First(&room).Error; retryErr != nil {
				return nil, fiber.StatusInternalServerError, "Gagal menyiapkan ruang diskusi"
			}
		}
	} else if err != nil {
		return nil, fiber.StatusInternalServerError, "Gagal memuat ruang diskusi"
	}

	// Validasi hak akses role: owner/admin/superadmin otomatis masuk, crew/staff harus di-assign
	if normRole != "superadmin" && normRole != "owner" && normRole != "admin" {
		var assignCount int64
		config.DB.Model(&models.AgendaChatMemberAssignment{}).
			Where("room_id = ? AND user_id = ?", room.ID, userID).
			Count(&assignCount)
		if assignCount == 0 {
			return nil, fiber.StatusForbidden, "Anda belum di-assign oleh Owner/Admin ke diskusi agenda ini"
		}
	}

	// Ambil profil user & perusahaan untuk identitas pengirim
	var user models.User
	userName := "Tim Vendor"
	if err := config.DB.Select("id, full_name, role").Where("id = ?", userID).First(&user).Error; err == nil {
		if strings.TrimSpace(user.FullName) != "" {
			userName = user.FullName
		}
	}

	var company models.Company
	companyName := ""
	categoryLabel := ""
	if companyID != "" {
		if err := config.DB.Select("id, name, category").Where("id = ?", companyID).First(&company).Error; err == nil {
			companyName = company.Name
			categoryLabel = formatVendorCategoryLabel(company.Category)
		}
	}
	if leadAgenda.CompanyID == companyID {
		if categoryLabel == "" {
			categoryLabel = "Lead Vendor"
		} else {
			categoryLabel = "Lead · " + categoryLabel
		}
	} else {
		for _, col := range acceptedCollabs {
			if col.PartnerCompanyID == companyID && col.PartnerCategory != "" {
				categoryLabel = formatVendorCategoryLabel(col.PartnerCategory)
				break
			}
		}
	}

	return &chatAccessContext{
		Room:            room,
		LeadAgenda:      leadAgenda,
		AcceptedCollabs: acceptedCollabs,
		CompanyName:     companyName,
		CategoryLabel:   categoryLabel,
		UserName:        userName,
		UserRole:        normRole,
	}, fiber.StatusOK, ""
}

func formatVendorCategoryLabel(cat string) string {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "dekorasi", "decor", "decoration":
		return "Dekorasi"
	case "mua":
		return "MUA"
	case "attire", "busana":
		return "Busana"
	case "fotografer", "foto", "photo", "photography":
		return "Fotografer"
	case "mc":
		return "MC"
	case "catering":
		return "Catering"
	case "entertainment", "musik":
		return "Entertainment"
	case "tenda":
		return "Tenda"
	case "wo":
		return "WO"
	case "":
		return ""
	default:
		return strings.ToUpper(cat[:1]) + cat[1:]
	}
}

func markRoomReadForUser(roomID, userID string) {
	if roomID == "" || userID == "" {
		return
	}
	now := time.Now()
	var existing models.AgendaChatRead
	err := config.DB.Where("room_id = ? AND user_id = ?", roomID, userID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		_ = config.DB.Create(&models.AgendaChatRead{
			ID:         fmt.Sprintf("crd_%s", uuid.New().String()[:8]),
			RoomID:     roomID,
			UserID:     userID,
			LastReadAt: now,
		}).Error
		return
	}
	if err == nil {
		_ = config.DB.Model(&existing).Updates(map[string]interface{}{
			"last_read_at": now,
			"updated_at":   now,
		}).Error
	}
}

// EnsureAndPostChatSystemMessage mencatat pesan sistem otomatis ke room diskusi & broadcast via WebSocket.
func EnsureAndPostChatSystemMessage(leadAgendaID, content string) {
	if strings.TrimSpace(leadAgendaID) == "" || strings.TrimSpace(content) == "" {
		return
	}

	var room models.AgendaChatRoom
	err := config.DB.Where("lead_agenda_id = ?", leadAgendaID).First(&room).Error
	if err == gorm.ErrRecordNotFound {
		room = models.AgendaChatRoom{
			ID:           fmt.Sprintf("chat_%s", uuid.New().String()[:8]),
			LeadAgendaID: leadAgendaID,
		}
		if createErr := config.DB.Create(&room).Error; createErr != nil {
			if retryErr := config.DB.Where("lead_agenda_id = ?", leadAgendaID).First(&room).Error; retryErr != nil {
				return
			}
		}
	} else if err != nil {
		return
	}

	now := time.Now()
	msg := models.AgendaChatMessage{
		ID:          fmt.Sprintf("msg_%s", uuid.New().String()[:8]),
		RoomID:      room.ID,
		MessageType: "system",
		Content:     content,
		CreatedAt:   now,
	}
	if err := config.DB.Create(&msg).Error; err != nil {
		return
	}

	preview := content
	if len(preview) > 200 {
		preview = preview[:200]
	}
	_ = config.DB.Model(&room).Updates(map[string]interface{}{
		"last_message_at":      now,
		"last_message_preview": preview,
		"last_sender_name":     "Sistem",
		"updated_at":           now,
	}).Error

	chatHub.Broadcast(room.ID, "new_message", msg)
}

func notifyChatParticipantsPush(ctx *chatAccessContext, msg models.AgendaChatMessage) {
	// Kumpulkan companyIDs yang berhak: LeadCompanyID + semua PartnerCompanyID accepted
	companyIDSet := map[string]bool{
		ctx.LeadAgenda.CompanyID: true,
	}
	for _, c := range ctx.AcceptedCollabs {
		if c.PartnerCompanyID != "" {
			companyIDSet[c.PartnerCompanyID] = true
		}
	}

	companyIDs := make([]string, 0, len(companyIDSet))
	for cid := range companyIDSet {
		companyIDs = append(companyIDs, cid)
	}

	// Ambil semua owner & admin dari perusahaan terkait
	var adminUsers []models.User
	config.DB.Select("id").Where("company_id IN ? AND role IN ?", companyIDs, []string{"owner", "admin", "superadmin"}).Find(&adminUsers)

	// Ambil crew yang di-assign ke room ini
	var crewAssigns []models.AgendaChatMemberAssignment
	config.DB.Select("user_id").Where("room_id = ?", ctx.Room.ID).Find(&crewAssigns)

	activeWS := chatHub.ActiveUserIDs(ctx.Room.ID)
	targetUserSet := make(map[string]bool)
	for _, u := range adminUsers {
		if u.ID != msg.SenderUserID && !activeWS[u.ID] {
			targetUserSet[u.ID] = true
		}
	}
	for _, a := range crewAssigns {
		if a.UserID != msg.SenderUserID && !activeWS[a.UserID] {
			targetUserSet[a.UserID] = true
		}
	}

	if len(targetUserSet) == 0 {
		return
	}

	targetUserIDs := make([]string, 0, len(targetUserSet))
	for uid := range targetUserSet {
		targetUserIDs = append(targetUserIDs, uid)
	}

	var deviceTokens []models.UserDeviceToken
	config.DB.Where("user_id IN ?", targetUserIDs).Find(&deviceTokens)
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

	title := fmt.Sprintf("Diskusi · %s", ctx.LeadAgenda.ClientName)
	if strings.TrimSpace(msg.MentionedCompanyIDs) != "" {
		title = fmt.Sprintf("Anda disebut · %s", ctx.LeadAgenda.ClientName)
	}
	bodyText := msg.Content
	if msg.MessageType == "image" && strings.TrimSpace(bodyText) == "" {
		bodyText = "Mengirim foto"
	} else if msg.MessageType == "file" {
		if strings.TrimSpace(msg.AttachmentName) != "" {
			bodyText = fmt.Sprintf("Mengirim PDF: %s", msg.AttachmentName)
		} else {
			bodyText = "Mengirim dokumen PDF"
		}
	}
	senderLabel := msg.SenderName
	if msg.SenderCompanyName != "" {
		senderLabel = fmt.Sprintf("%s (%s)", msg.SenderName, msg.SenderCompanyName)
	}
	body := fmt.Sprintf("%s: %s", senderLabel, bodyText)

	data := map[string]string{
		"type":           "agenda_chat",
		"lead_agenda_id": ctx.LeadAgenda.ID,
		"room_id":        ctx.Room.ID,
		"reference_id":   ctx.LeadAgenda.ID,
	}

	go func(toks []string, t, b string, d map[string]string) {
		invalid := utils.SendPushToTokens(toks, t, b, d)
		if len(invalid) > 0 {
			config.DB.Where("token IN ?", invalid).Delete(&models.UserDeviceToken{})
		}
	}(tokens, title, body, data)
}

// GetAgendaChatMessages handles GET /api/agendas/:id/chat/messages
func GetAgendaChatMessages(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	role := middleware.GetUserRole(c)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	limit := c.QueryInt("limit", 50)
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := config.DB.Where("room_id = ?", ctx.Room.ID)
	if before := strings.TrimSpace(c.Query("before")); before != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, before); err == nil {
			query = query.Where("created_at < ?", parsed)
		}
	}

	var messages []models.AgendaChatMessage
	if err := query.Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat pesan diskusi"})
	}

	// Balikkan urutan jadi kronologis (ASC) agar mudah dirender
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	markRoomReadForUser(ctx.Room.ID, userID)

	// Bangun daftar peserta vendor
	var leadCompany models.Company
	leadCompanyName := "Lead Vendor"
	if err := config.DB.Select("id, name").Where("id = ?", ctx.LeadAgenda.CompanyID).First(&leadCompany).Error; err == nil && leadCompany.Name != "" {
		leadCompanyName = leadCompany.Name
	}

	participants := []fiber.Map{
		{
			"company_id":   ctx.LeadAgenda.CompanyID,
			"company_name": leadCompanyName,
			"category":     "Lead Vendor",
			"is_lead":      true,
		},
	}
	for _, col := range ctx.AcceptedCollabs {
		participants = append(participants, fiber.Map{
			"company_id":   col.PartnerCompanyID,
			"company_name": col.PartnerCompanyName,
			"category":     formatVendorCategoryLabel(col.PartnerCategory),
			"is_lead":      false,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"room": fiber.Map{
			"id":              ctx.Room.ID,
			"lead_agenda_id":  ctx.LeadAgenda.ID,
			"client_name":     ctx.LeadAgenda.ClientName,
			"event_label":     ctx.LeadAgenda.EventLabel,
			"event_type":      ctx.LeadAgenda.EventType,
			"event_date_time": ctx.LeadAgenda.EventDateTime,
			"event_address":   ctx.LeadAgenda.EventAddress,
			"maps_url":        ctx.LeadAgenda.MapsURL,
			"participants":    participants,
			"can_manage_crew": ctx.UserRole == "owner" || ctx.UserRole == "admin" || ctx.UserRole == "superadmin",
		},
		"messages":      messages,
		"read_receipts": buildRoomReadReceipts(ctx.Room.ID),
	})
}

func buildRoomReadReceipts(roomID string) []fiber.Map {
	var reads []models.AgendaChatRead
	config.DB.Where("room_id = ?", roomID).Find(&reads)
	if len(reads) == 0 {
		return []fiber.Map{}
	}

	userIDs := make([]string, 0, len(reads))
	for _, r := range reads {
		userIDs = append(userIDs, r.UserID)
	}

	var users []models.User
	config.DB.Select("id, full_name, company_id").Where("id IN ?", userIDs).Find(&users)
	userByID := make(map[string]models.User)
	companyIDs := make([]string, 0, len(users))
	for _, u := range users {
		userByID[u.ID] = u
		if u.CompanyID != "" {
			companyIDs = append(companyIDs, u.CompanyID)
		}
	}

	companyNameByID := make(map[string]string)
	if len(companyIDs) > 0 {
		var companies []models.Company
		config.DB.Select("id, name").Where("id IN ?", companyIDs).Find(&companies)
		for _, comp := range companies {
			companyNameByID[comp.ID] = comp.Name
		}
	}

	result := make([]fiber.Map, 0, len(reads))
	for _, r := range reads {
		u := userByID[r.UserID]
		result = append(result, fiber.Map{
			"user_id":      r.UserID,
			"user_name":    u.FullName,
			"company_id":   u.CompanyID,
			"company_name": companyNameByID[u.CompanyID],
			"last_read_at": r.LastReadAt,
		})
	}
	return result
}

// SendAgendaChatMessage handles POST /api/agendas/:id/chat/messages
func SendAgendaChatMessage(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	role := middleware.GetUserRole(c)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	var req models.SendChatMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload pesan tidak valid"})
	}

	msgType := strings.ToLower(strings.TrimSpace(req.MessageType))
	if msgType == "" {
		msgType = "text"
	}
	if msgType != "text" && msgType != "image" && msgType != "file" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Tipe pesan harus 'text', 'image', atau 'file'"})
	}

	content := strings.TrimSpace(req.Content)
	attachmentURL := strings.TrimSpace(req.AttachmentURL)
	attachmentName := strings.TrimSpace(req.AttachmentName)
	if msgType == "text" && content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Pesan tidak boleh kosong"})
	}
	if (msgType == "image" || msgType == "file") && attachmentURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "URL lampiran tidak boleh kosong"})
	}

	replyToID := strings.TrimSpace(req.ReplyToID)
	replySenderName := ""
	replyPreview := ""
	if replyToID != "" {
		var replyMsg models.AgendaChatMessage
		if err := config.DB.Where("id = ? AND room_id = ?", replyToID, ctx.Room.ID).First(&replyMsg).Error; err == nil {
			replySenderName = replyMsg.SenderName
			if replySenderName == "" {
				replySenderName = "Sistem"
			}
			replyPreview = strings.TrimSpace(replyMsg.Content)
			if replyPreview == "" {
				if replyMsg.MessageType == "image" {
					replyPreview = "[Foto]"
				} else if replyMsg.MessageType == "file" {
					replyPreview = "[PDF] " + replyMsg.AttachmentName
				}
			}
			if len(replyPreview) > 150 {
				replyPreview = replyPreview[:150]
			}
		} else {
			replyToID = ""
		}
	}

	mentionedClean := make([]string, 0, len(req.MentionedCompanyIDs))
	for _, id := range req.MentionedCompanyIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			mentionedClean = append(mentionedClean, trimmed)
		}
	}

	now := time.Now()
	msg := models.AgendaChatMessage{
		ID:                  fmt.Sprintf("msg_%s", uuid.New().String()[:8]),
		RoomID:              ctx.Room.ID,
		SenderUserID:        userID,
		SenderCompanyID:     companyID,
		SenderCompanyName:   ctx.CompanyName,
		SenderCategory:      ctx.CategoryLabel,
		SenderName:          ctx.UserName,
		SenderRole:          ctx.UserRole,
		MessageType:         msgType,
		Content:             content,
		AttachmentURL:       attachmentURL,
		AttachmentName:      attachmentName,
		AttachmentSize:      req.AttachmentSize,
		ReplyToID:           replyToID,
		ReplySenderName:     replySenderName,
		ReplyPreview:        replyPreview,
		MentionedCompanyIDs: strings.Join(mentionedClean, ","),
		CreatedAt:           now,
	}

	if err := config.DB.Create(&msg).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengirim pesan"})
	}

	preview := content
	if msgType == "image" && preview == "" {
		preview = "[Foto]"
	} else if msgType == "file" {
		if attachmentName != "" {
			preview = "[PDF] " + attachmentName
		} else {
			preview = "[Dokumen PDF]"
		}
	}
	if len(preview) > 200 {
		preview = preview[:200]
	}
	_ = config.DB.Model(&ctx.Room).Updates(map[string]interface{}{
		"last_message_at":      now,
		"last_message_preview": preview,
		"last_sender_name":     ctx.UserName,
		"updated_at":           now,
	}).Error

	markRoomReadForUser(ctx.Room.ID, userID)

	// Broadcast real-time via WebSocket & kirim push notification ke peserta yang offline
	chatHub.Broadcast(ctx.Room.ID, "new_message", msg)
	notifyChatParticipantsPush(ctx, msg)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   msg,
	})
}

// UploadAgendaChatImage handles POST /api/agendas/:id/chat/upload
func UploadAgendaChatImage(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	role := middleware.GetUserRole(c)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File wajib dilampirkan"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	isPDF := ext == ".pdf"
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		if file.Size > 5*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran gambar maksimal 5MB"})
		}
	case ".pdf":
		if file.Size > 10*1024*1024 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Ukuran file PDF maksimal 10MB"})
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format file harus JPG, PNG, WEBP, atau PDF"})
	}

	uploadDir := filepath.Join("./uploads/chat", ctx.Room.ID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyiapkan direktori upload"})
	}

	filename := fmt.Sprintf("chat_%d_%s%s", time.Now().UnixMilli(), uuid.New().String()[:6], ext)
	savePath := filepath.Join(uploadDir, filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan file"})
	}

	url := fmt.Sprintf("%s://%s/uploads/chat/%s/%s", c.Protocol(), c.Hostname(), ctx.Room.ID, filename)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":   "success",
		"url":      url,
		"filename": file.Filename,
		"size":     file.Size,
		"is_pdf":   isPDF,
	})
}

// MarkAgendaChatRead handles POST /api/agendas/:id/chat/read
func MarkAgendaChatRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	role := middleware.GetUserRole(c)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	markRoomReadForUser(ctx.Room.ID, userID)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "success"})
}

// GetAgendaChatCrew handles GET /api/agendas/:id/chat/crew (Owner/Admin melihat daftar crew & status assignment-nya)
func GetAgendaChatCrew(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	role := middleware.GetUserRole(c)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	if ctx.UserRole != "owner" && ctx.UserRole != "admin" && ctx.UserRole != "superadmin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Hanya Owner/Admin yang dapat mengatur anggota tim diskusi"})
	}

	var crewUsers []models.User
	config.DB.Select("id, full_name, phone, role").
		Where("company_id = ? AND role IN ?", companyID, []string{"crew", "staff"}).
		Order("full_name ASC").
		Find(&crewUsers)

	var assigns []models.AgendaChatMemberAssignment
	config.DB.Where("room_id = ? AND company_id = ?", ctx.Room.ID, companyID).Find(&assigns)
	assignedMap := make(map[string]bool)
	for _, a := range assigns {
		assignedMap[a.UserID] = true
	}

	items := make([]fiber.Map, 0, len(crewUsers))
	for _, u := range crewUsers {
		items = append(items, fiber.Map{
			"user_id":     u.ID,
			"name":        u.FullName,
			"phone":       u.Phone,
			"role":        u.Role,
			"is_assigned": assignedMap[u.ID],
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   items,
	})
}

// UpdateAgendaChatCrew handles POST /api/agendas/:id/chat/crew
func UpdateAgendaChatCrew(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	role := middleware.GetUserRole(c)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	if ctx.UserRole != "owner" && ctx.UserRole != "admin" && ctx.UserRole != "superadmin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Hanya Owner/Admin yang dapat mengatur anggota tim diskusi"})
	}

	var req models.AssignChatCrewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Payload tidak valid"})
	}

	// Validasi userIDs memang milik companyID
	validIDs := make([]string, 0, len(req.UserIDs))
	if len(req.UserIDs) > 0 {
		var validUsers []models.User
		config.DB.Select("id").Where("company_id = ? AND id IN ? AND role IN ?", companyID, req.UserIDs, []string{"crew", "staff"}).Find(&validUsers)
		for _, u := range validUsers {
			validIDs = append(validIDs, u.ID)
		}
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("room_id = ? AND company_id = ?", ctx.Room.ID, companyID).Delete(&models.AgendaChatMemberAssignment{}).Error; err != nil {
			return err
		}
		if len(validIDs) == 0 {
			return nil
		}
		newAssigns := make([]models.AgendaChatMemberAssignment, 0, len(validIDs))
		for _, uid := range validIDs {
			newAssigns = append(newAssigns, models.AgendaChatMemberAssignment{
				ID:               fmt.Sprintf("cas_%s", uuid.New().String()[:8]),
				RoomID:           ctx.Room.ID,
				CompanyID:        companyID,
				UserID:           uid,
				AssignedByUserID: userID,
			})
		}
		return tx.Create(&newAssigns).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui anggota tim diskusi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Anggota tim diskusi berhasil diperbarui",
	})
}

// GetChatUnreadCounts handles GET /api/chat/unread-counts
// Mengembalikan map[agenda_id]unread_count serta set agenda_id yang punya room diskusi aktif.
func GetChatUnreadCounts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	normRole := strings.ToLower(strings.TrimSpace(middleware.GetUserRole(c)))

	// Cari semua kolaborasi accepted yang melibatkan companyID ini
	var collabs []models.AgendaCollaboration
	config.DB.Where("status = ? AND (lead_company_id = ? OR partner_company_id = ?)", "accepted", companyID, companyID).Find(&collabs)
	if len(collabs) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":        "success",
			"unread_counts": fiber.Map{},
			"active_agendas": []string{},
		})
	}

	leadToLocalAgendas := make(map[string][]string)
	leadIDsMap := make(map[string]bool)
	for _, col := range collabs {
		leadIDsMap[col.LeadAgendaID] = true
		if col.LeadCompanyID == companyID {
			leadToLocalAgendas[col.LeadAgendaID] = append(leadToLocalAgendas[col.LeadAgendaID], col.LeadAgendaID)
		}
		if col.PartnerCompanyID == companyID && col.PartnerAgendaID != "" {
			leadToLocalAgendas[col.LeadAgendaID] = append(leadToLocalAgendas[col.LeadAgendaID], col.PartnerAgendaID)
		}
	}

	leadIDs := make([]string, 0, len(leadIDsMap))
	for id := range leadIDsMap {
		leadIDs = append(leadIDs, id)
	}

	var rooms []models.AgendaChatRoom
	config.DB.Where("lead_agenda_id IN ?", leadIDs).Find(&rooms)
	roomByLeadID := make(map[string]models.AgendaChatRoom)
	roomIDs := make([]string, 0, len(rooms))
	for _, r := range rooms {
		roomByLeadID[r.LeadAgendaID] = r
		roomIDs = append(roomIDs, r.ID)
	}

	// Jika crew/staff, filter hanya room yang sudah di-assign ke userID ini
	assignedRoomSet := make(map[string]bool)
	isAdminOrOwner := normRole == "superadmin" || normRole == "owner" || normRole == "admin"
	if !isAdminOrOwner && len(roomIDs) > 0 {
		var assigns []models.AgendaChatMemberAssignment
		config.DB.Select("room_id").Where("room_id IN ? AND user_id = ?", roomIDs, userID).Find(&assigns)
		for _, a := range assigns {
			assignedRoomSet[a.RoomID] = true
		}
	}

	// Ambil last_read_at per room
	readMap := make(map[string]time.Time)
	if len(roomIDs) > 0 {
		var reads []models.AgendaChatRead
		config.DB.Where("room_id IN ? AND user_id = ?", roomIDs, userID).Find(&reads)
		for _, r := range reads {
			readMap[r.RoomID] = r.LastReadAt
		}
	}

	unreadCounts := make(map[string]int64)
	activeAgendas := make([]string, 0)

	for _, leadID := range leadIDs {
		localIDs := leadToLocalAgendas[leadID]
		room, hasRoom := roomByLeadID[leadID]

		if !isAdminOrOwner {
			if !hasRoom || !assignedRoomSet[room.ID] {
				continue
			}
		}

		for _, localID := range localIDs {
			activeAgendas = append(activeAgendas, localID)
		}

		if !hasRoom {
			continue
		}

		var count int64
		q := config.DB.Model(&models.AgendaChatMessage{}).
			Where("room_id = ? AND sender_user_id <> ? AND message_type <> ?", room.ID, userID, "system")
		if lastRead, ok := readMap[room.ID]; ok {
			q = q.Where("created_at > ?", lastRead)
		}
		q.Count(&count)

		if count > 0 {
			for _, localID := range localIDs {
				unreadCounts[localID] = count
			}
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":         "success",
		"unread_counts":  unreadCounts,
		"active_agendas": activeAgendas,
	})
}

// AgendaChatWebSocketUpgrade memvalidasi JWT token dari query param `?token=` atau header `Authorization` sebelum upgrade ke WebSocket.
func AgendaChatWebSocketUpgrade(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	tokenStr := strings.TrimSpace(c.Query("token"))
	if tokenStr == "" {
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
	}
	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token WebSocket tidak ditemukan"})
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return middleware.JwtSecret, nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token WebSocket tidak valid"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Klaim token tidak valid"})
	}

	userID, _ := claims["id"].(string)
	companyID, _ := claims["company_id"].(string)
	role, _ := claims["role"].(string)
	agendaID := c.Params("id")

	ctx, status, errMsg := resolveChatAccess(userID, companyID, role, agendaID)
	if status != fiber.StatusOK {
		return c.Status(status).JSON(fiber.Map{"error": errMsg})
	}

	c.Locals("ws_room_id", ctx.Room.ID)
	c.Locals("ws_user_id", userID)
	c.Locals("ws_user_name", ctx.UserName)
	c.Locals("ws_company_id", companyID)
	c.Locals("ws_company_name", ctx.CompanyName)
	c.Locals("ws_category", ctx.CategoryLabel)
	return c.Next()
}

// HandleAgendaChatWebSocket menangani koneksi WebSocket aktif untuk room diskusi agenda.
func HandleAgendaChatWebSocket(conn *websocket.Conn) {
	roomID, _ := conn.Locals("ws_room_id").(string)
	userID, _ := conn.Locals("ws_user_id").(string)
	userName, _ := conn.Locals("ws_user_name").(string)
	companyID, _ := conn.Locals("ws_company_id").(string)
	companyName, _ := conn.Locals("ws_company_name").(string)
	category, _ := conn.Locals("ws_category").(string)
	if roomID == "" || userID == "" {
		_ = conn.Close()
		return
	}

	chatHub.Register(roomID, conn, userID)
	// Broadcast bahwa user ini baru saja membaca room saat terhubung
	markRoomReadForUser(roomID, userID)
	chatHub.BroadcastExcept(roomID, conn, "read_updated", fiber.Map{
		"user_id":      userID,
		"user_name":    userName,
		"company_id":   companyID,
		"company_name": companyName,
		"last_read_at": time.Now(),
	})

	defer func() {
		// Pastikan indikator typing berhenti saat disconnect
		chatHub.BroadcastExcept(roomID, conn, "typing", fiber.Map{
			"user_id":      userID,
			"user_name":    userName,
			"company_name": companyName,
			"category":     category,
			"is_typing":    false,
		})
		chatHub.Unregister(roomID, conn)
		_ = conn.Close()
	}()

	// Kirim event konfirmasi koneksi
	if ack, err := json.Marshal(fiber.Map{"event": "connected", "room_id": roomID}); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, ack)
	}

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var incoming struct {
			Event    string `json:"event"`
			IsTyping bool   `json:"is_typing"`
		}
		if json.Unmarshal(msgBytes, &incoming) != nil {
			continue
		}
		switch incoming.Event {
		case "ping":
			if pong, err := json.Marshal(fiber.Map{"event": "pong"}); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, pong)
			}
		case "typing":
			chatHub.BroadcastExcept(roomID, conn, "typing", fiber.Map{
				"user_id":      userID,
				"user_name":    userName,
				"company_name": companyName,
				"category":     category,
				"is_typing":    incoming.IsTyping,
			})
		case "mark_read":
			markRoomReadForUser(roomID, userID)
			chatHub.BroadcastExcept(roomID, conn, "read_updated", fiber.Map{
				"user_id":      userID,
				"user_name":    userName,
				"company_id":   companyID,
				"company_name": companyName,
				"last_read_at": time.Now(),
			})
		}
	}
}

// GetChatRooms handles GET /api/chat/rooms — daftar semua room diskusi kolaborasi aktif untuk tab Diskusi (Inbox).
func GetChatRooms(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	companyID := middleware.GetCompanyID(c)
	normRole := strings.ToLower(strings.TrimSpace(middleware.GetUserRole(c)))

	var collabs []models.AgendaCollaboration
	config.DB.Where("status = ? AND (lead_company_id = ? OR partner_company_id = ?)", "accepted", companyID, companyID).Find(&collabs)
	if len(collabs) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"data":   []fiber.Map{},
		})
	}

	leadIDsMap := make(map[string]bool)
	localAgendaByLead := make(map[string]string)
	partnerNamesByLead := make(map[string][]string)

	for _, col := range collabs {
		leadIDsMap[col.LeadAgendaID] = true
		if col.LeadCompanyID == companyID {
			localAgendaByLead[col.LeadAgendaID] = col.LeadAgendaID
		} else if col.PartnerCompanyID == companyID && col.PartnerAgendaID != "" {
			localAgendaByLead[col.LeadAgendaID] = col.PartnerAgendaID
		}
		if col.PartnerCompanyName != "" {
			partnerNamesByLead[col.LeadAgendaID] = append(partnerNamesByLead[col.LeadAgendaID], col.PartnerCompanyName)
		}
	}

	leadIDs := make([]string, 0, len(leadIDsMap))
	for id := range leadIDsMap {
		leadIDs = append(leadIDs, id)
	}

	var agendas []models.Agenda
	config.DB.Where("id IN ?", leadIDs).Find(&agendas)
	agendaByID := make(map[string]models.Agenda)
	for _, a := range agendas {
		agendaByID[a.ID] = a
	}

	var rooms []models.AgendaChatRoom
	config.DB.Where("lead_agenda_id IN ?", leadIDs).Find(&rooms)
	roomByLeadID := make(map[string]models.AgendaChatRoom)
	roomIDs := make([]string, 0, len(rooms))
	for _, r := range rooms {
		roomByLeadID[r.LeadAgendaID] = r
		roomIDs = append(roomIDs, r.ID)
	}

	isAdminOrOwner := normRole == "superadmin" || normRole == "owner" || normRole == "admin"
	assignedRoomSet := make(map[string]bool)
	if !isAdminOrOwner && len(roomIDs) > 0 {
		var assigns []models.AgendaChatMemberAssignment
		config.DB.Select("room_id").Where("room_id IN ? AND user_id = ?", roomIDs, userID).Find(&assigns)
		for _, a := range assigns {
			assignedRoomSet[a.RoomID] = true
		}
	}

	readMap := make(map[string]time.Time)
	if len(roomIDs) > 0 {
		var reads []models.AgendaChatRead
		config.DB.Where("room_id IN ? AND user_id = ?", roomIDs, userID).Find(&reads)
		for _, r := range reads {
			readMap[r.RoomID] = r.LastReadAt
		}
	}

	type roomRow struct {
		sortTime time.Time
		payload  fiber.Map
	}
	rows := make([]roomRow, 0, len(leadIDs))

	for _, leadID := range leadIDs {
		agenda, ok := agendaByID[leadID]
		if !ok {
			continue
		}
		room, hasRoom := roomByLeadID[leadID]
		if !isAdminOrOwner && (!hasRoom || !assignedRoomSet[room.ID]) {
			continue
		}

		localAgendaID := localAgendaByLead[leadID]
		if localAgendaID == "" {
			localAgendaID = leadID
		}

		var unreadCount int64
		if hasRoom {
			q := config.DB.Model(&models.AgendaChatMessage{}).
				Where("room_id = ? AND sender_user_id <> ? AND message_type <> ?", room.ID, userID, "system")
			if lastRead, hasRead := readMap[room.ID]; hasRead {
				q = q.Where("created_at > ?", lastRead)
			}
			q.Count(&unreadCount)
		}

		sortAt := agenda.CreatedAt
		if hasRoom && room.LastMessageAt != nil {
			sortAt = *room.LastMessageAt
		}

		rows = append(rows, roomRow{
			sortTime: sortAt,
			payload: fiber.Map{
				"room_id":              room.ID,
				"agenda_id":            localAgendaID,
				"lead_agenda_id":       leadID,
				"client_name":          agenda.ClientName,
				"event_label":          agenda.EventLabel,
				"event_date_time":      agenda.EventDateTime,
				"last_message_at":      room.LastMessageAt,
				"last_message_preview": room.LastMessagePreview,
				"last_sender_name":     room.LastSenderName,
				"unread_count":         unreadCount,
				"partner_names":        partnerNamesByLead[leadID],
			},
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].sortTime.After(rows[j].sortTime)
	})

	result := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		result = append(result, r.payload)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   result,
	})
}
