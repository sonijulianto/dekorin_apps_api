package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/url"
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

const joinCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// generateWeddingJoinCode menghasilkan kode 8 karakter yang mudah dibagikan ke pasangan.
func generateWeddingJoinCode() string {
	b := make([]byte, 8)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(joinCodeAlphabet))))
		if err != nil {
			b[i] = joinCodeAlphabet[(time.Now().UnixNano()+int64(i))%int64(len(joinCodeAlphabet))]
		} else {
			b[i] = joinCodeAlphabet[idx.Int64()]
		}
	}
	return string(b)
}

func generateUniqueWeddingJoinCode(tx *gorm.DB) string {
	for i := 0; i < 8; i++ {
		code := generateWeddingJoinCode()
		var count int64
		tx.Model(&models.WeddingPlan{}).Where("UPPER(join_code) = ?", code).Count(&count)
		if count == 0 {
			return code
		}
	}
	return strings.ToUpper(strings.ReplaceAll(uuid.New().String(), "-", "")[:8])
}

// extractLastPathToken mengekstrak segmen terakhir jika input berupa URL penuh, atau mengembalikan string bersihnya.
func extractLastPathToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			path := strings.Trim(parsed.Path, "/")
			if path != "" {
				parts := strings.Split(path, "/")
				return strings.TrimSpace(parts[len(parts)-1])
			}
		}
	}
	trimmed = strings.Trim(trimmed, "/")
	if idx := strings.LastIndex(trimmed, "/"); idx != -1 {
		trimmed = trimmed[idx+1:]
	}
	if idx := strings.Index(trimmed, "?"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(trimmed)
}

// seedStarterWeddingTasks mengisi daftar checklist awal persiapan pernikahan Indonesia.
func seedStarterWeddingTasks(tx *gorm.DB, planID string) error {
	now := time.Now()
	starter := []struct {
		Title    string
		Category string
		Phase    string
	}{
		{"Tentukan tanggal acara & perkiraan jumlah tamu", "Administrasi & KUA", "H-6 Bulan"},
		{"Survei & booking gedung / venue pernikahan", "Venue & Dekorasi", "H-6 Bulan"},
		{"Booking vendor Dekorasi & pelaminan", "Venue & Dekorasi", "H-6 Bulan"},
		{"Booking Wedding Organizer (WO) & MC", "Administrasi & KUA", "H-3 Bulan"},
		{"Booking MUA, busana pengantin & keluarga", "Busana & MUA", "H-3 Bulan"},
		{"Booking vendor Fotografi & Videografi", "Dokumentasi", "H-3 Bulan"},
		{"Test food & tentukan menu Katering", "Katering & Suvenir", "H-3 Bulan"},
		{"Lengkapi berkas pengantar nikah & daftar KUA", "Administrasi & KUA", "H-1 Bulan"},
		{"Siapkan daftar tamu & sebar Undangan Digital", "Undangan & Tamu", "H-1 Bulan"},
		{"Technical meeting vendor & pelunasan H-7", "Administrasi & KUA", "H-7"},
	}

	tasks := make([]models.WeddingTask, 0, len(starter))
	for i, item := range starter {
		tasks = append(tasks, models.WeddingTask{
			ID:            "wt_" + uuid.New().String()[:12],
			WeddingPlanID: planID,
			Title:         item.Title,
			Category:      item.Category,
			Phase:         item.Phase,
			IsCompleted:   false,
			SortOrder:     i + 1,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	return tx.Create(&tasks).Error
}

// CreateOrJoinWeddingPlanForCouple membuat WeddingPlan baru atau menggabungkan user ke rencana yang sudah ada via JoinCode.
func CreateOrJoinWeddingPlanForCouple(
	tx *gorm.DB,
	user *models.User,
	partnerName string,
	weddingDate string,
	city string,
	venueName string,
	budgetTarget float64,
	joinCode string,
) (*models.WeddingPlan, error) {
	now := time.Now()
	cleanJoin := strings.ToUpper(strings.TrimSpace(joinCode))

	if cleanJoin != "" {
		var plan models.WeddingPlan
		if err := tx.Where("UPPER(join_code) = ?", cleanJoin).First(&plan).Error; err != nil {
			return nil, errors.New("Kode gabung pasangan tidak ditemukan. Periksa kembali kode yang Anda masukkan.")
		}

		var existingMember models.WeddingPlanMember
		if err := tx.Where("wedding_plan_id = ? AND user_id = ?", plan.ID, user.ID).First(&existingMember).Error; err == nil {
			return &plan, nil
		}

		member := models.WeddingPlanMember{
			ID:            "wpm_" + uuid.New().String()[:12],
			WeddingPlanID: plan.ID,
			UserID:        user.ID,
			MemberRole:    "partner",
			CreatedAt:     now,
		}
		if err := tx.Create(&member).Error; err != nil {
			return nil, err
		}

		if strings.TrimSpace(plan.BrideName) == "" {
			plan.BrideName = user.FullName
			if strings.TrimSpace(plan.GroomName) != "" {
				plan.Title = fmt.Sprintf("Pernikahan %s & %s", plan.GroomName, plan.BrideName)
			}
			plan.UpdatedAt = now
			_ = tx.Save(&plan).Error
		}
		return &plan, nil
	}

	groomName := strings.TrimSpace(user.FullName)
	brideName := strings.TrimSpace(partnerName)
	title := "Rencana Pernikahan " + groomName
	if brideName != "" {
		title = fmt.Sprintf("Pernikahan %s & %s", groomName, brideName)
	}
	if budgetTarget < 0 {
		budgetTarget = 0
	}

	plan := models.WeddingPlan{
		ID:              "wp_" + uuid.New().String()[:12],
		Title:           title,
		GroomName:       groomName,
		BrideName:       brideName,
		WeddingDate:     strings.TrimSpace(weddingDate),
		City:            strings.TrimSpace(city),
		VenueName:       strings.TrimSpace(venueName),
		BudgetTarget:    budgetTarget,
		JoinCode:        generateUniqueWeddingJoinCode(tx),
		CreatedByUserID: user.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := tx.Create(&plan).Error; err != nil {
		return nil, err
	}

	member := models.WeddingPlanMember{
		ID:            "wpm_" + uuid.New().String()[:12],
		WeddingPlanID: plan.ID,
		UserID:        user.ID,
		MemberRole:    "creator",
		CreatedAt:     now,
	}
	if err := tx.Create(&member).Error; err != nil {
		return nil, err
	}

	if err := seedStarterWeddingTasks(tx, plan.ID); err != nil {
		return nil, err
	}

	return &plan, nil
}

// resolveCoupleWeddingPlan mengambil WeddingPlan aktif milik akun couple yang login (auto-inisialisasi jika belum ada).
func resolveCoupleWeddingPlan(c *fiber.Ctx) (*models.WeddingPlan, string, error) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return nil, "", errors.New("sesi login tidak valid")
	}

	var member models.WeddingPlanMember
	err := config.DB.Where("user_id = ?", userID).Order("created_at ASC").First(&member).Error
	if err == nil {
		var plan models.WeddingPlan
		if err := config.DB.Where("id = ?", member.WeddingPlanID).First(&plan).Error; err == nil {
			return &plan, userID, nil
		}
	}

	// Fallback aman jika user couple dibuat manual tanpa WeddingPlan
	var user models.User
	if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, userID, errors.New("akun pengguna tidak ditemukan")
	}

	var createdPlan *models.WeddingPlan
	txErr := config.DB.Transaction(func(tx *gorm.DB) error {
		p, err := CreateOrJoinWeddingPlanForCouple(tx, &user, "", "", "", "", 0, "")
		if err != nil {
			return err
		}
		createdPlan = p
		return nil
	})
	if txErr != nil {
		return nil, userID, txErr
	}
	return createdPlan, userID, nil
}

// buildEnrichedWeddingVendors menggabungkan catatan vendor pengantin dengan data live Agenda & Company Dekorin tanpa N+1 query.
func buildEnrichedWeddingVendors(planID string) ([]models.WeddingVendorViewResponse, error) {
	var vendors []models.WeddingPlanVendor
	if err := config.DB.Where("wedding_plan_id = ?", planID).Order("created_at DESC").Find(&vendors).Error; err != nil {
		return nil, err
	}

	agendaIDs := make([]string, 0)
	companyIDs := make([]string, 0)
	for _, v := range vendors {
		if strings.TrimSpace(v.AgendaID) != "" {
			agendaIDs = append(agendaIDs, v.AgendaID)
		}
		if strings.TrimSpace(v.CompanyID) != "" {
			companyIDs = append(companyIDs, v.CompanyID)
		}
	}

	agendaMap := make(map[string]models.Agenda)
	if len(agendaIDs) > 0 {
		var agendas []models.Agenda
		if err := config.DB.Preload("ClientForm").Where("id IN ?", agendaIDs).Find(&agendas).Error; err == nil {
			for _, a := range agendas {
				agendaMap[a.ID] = a
			}
		}
	}

	companyMap := make(map[string]models.Company)
	if len(companyIDs) > 0 {
		var companies []models.Company
		if err := config.DB.Where("id IN ?", companyIDs).Find(&companies).Error; err == nil {
			for _, comp := range companies {
				companyMap[comp.ID] = comp
			}
		}
	}

	results := make([]models.WeddingVendorViewResponse, 0, len(vendors))
	for _, v := range vendors {
		item := models.WeddingVendorViewResponse{
			WeddingPlanVendor: v,
		}

		if comp, ok := companyMap[v.CompanyID]; ok {
			item.CompanyLogo = comp.LogoURL
			if strings.TrimSpace(item.VendorPhone) == "" {
				item.VendorPhone = comp.Phone
			}
			if strings.TrimSpace(item.VendorInstagram) == "" {
				item.VendorInstagram = comp.Instagram
			}
		}

		if agenda, ok := agendaMap[v.AgendaID]; ok {
			subtotalDecor := agenda.CalculateDecorSubtotal()
			discountAmount := agenda.CalculateDiscountAmount()
			totalBiaya := agenda.CalculateTotalBiaya()

			item.IsSyncedAgenda = true
			item.AgreedPrice = totalBiaya
			item.DpPaid = agenda.DpAmount
			item.PaymentStatus = agenda.PaymentStatus
			item.AgendaStatus = agenda.Status
			item.FormToken = agenda.FormToken
			item.FormStatus = agenda.FormStatus
			item.ModelName = agenda.ModelName
			item.SubtotalDecor = subtotalDecor
			item.DiscountAmount = discountAmount
			item.TransportFee = agenda.TransportFee
			if strings.TrimSpace(agenda.PackageName) != "" {
				item.PackageName = agenda.PackageName
			}
			if strings.TrimSpace(agenda.BrandLogo) != "" {
				item.CompanyLogo = agenda.BrandLogo
			}
			if strings.TrimSpace(agenda.BrandPhone) != "" {
				item.VendorPhone = agenda.BrandPhone
			}
			if !agenda.EventDateTime.IsZero() {
				item.EventDateTime = agenda.EventDateTime.Format("2006-01-02 15:04")
			}
		}

		rem := item.AgreedPrice - item.DpPaid
		if rem < 0 {
			rem = 0
		}
		item.RemainingBalance = rem
		results = append(results, item)
	}

	return results, nil
}

// GetMyWeddingPlan handles GET /api/couple/plan
func GetMyWeddingPlan(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	// Members
	var members []models.WeddingPlanMember
	config.DB.Preload("User").Where("wedding_plan_id = ?", plan.ID).Order("created_at ASC").Find(&members)
	memberSummaries := make([]models.WeddingPlanMemberSummary, 0, len(members))
	for _, m := range members {
		name := ""
		phone := ""
		if m.User != nil {
			name = m.User.FullName
			phone = m.User.Phone
		}
		memberSummaries = append(memberSummaries, models.WeddingPlanMemberSummary{
			UserID:     m.UserID,
			FullName:   name,
			Phone:      phone,
			MemberRole: m.MemberRole,
		})
	}

	// Checklist Stats
	var totalTasks, completedTasks int64
	config.DB.Model(&models.WeddingTask{}).Where("wedding_plan_id = ?", plan.ID).Count(&totalTasks)
	config.DB.Model(&models.WeddingTask{}).Where("wedding_plan_id = ? AND is_completed = ?", plan.ID, true).Count(&completedTasks)

	// Vendor & Budget Stats (menggunakan data live Agenda bila tertaut)
	enrichedVendors, err := buildEnrichedWeddingVendors(plan.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat ringkasan vendor"})
	}
	var totalAgreedCost, totalDpPaid, remainingBalance float64
	syncedCount := 0
	for _, v := range enrichedVendors {
		if v.IsSyncedAgenda {
			syncedCount++
		}
		totalAgreedCost += v.AgreedPrice
		totalDpPaid += v.DpPaid
		remainingBalance += v.RemainingBalance
	}

	// Undangan & RSVP Stats
	var invitations []models.Invitation
	config.DB.Preload("Wishes").Where("wedding_plan_id = ?", plan.ID).Find(&invitations)
	totalRsvpHadir := 0
	totalPaxHadir := 0
	for _, inv := range invitations {
		for _, w := range inv.Wishes {
			if strings.EqualFold(strings.TrimSpace(w.Attendance), "hadir") {
				totalRsvpHadir++
				pax := w.GuestCount
				if pax < 1 {
					pax = 1
				}
				totalPaxHadir += pax
			}
		}
	}

	// Countdown H-Hari
	var daysRemaining *int
	if strings.TrimSpace(plan.WeddingDate) != "" {
		if t, parseErr := time.Parse("2006-01-02", strings.TrimSpace(plan.WeddingDate)); parseErr == nil {
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			target := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
			diff := int(math.Round(target.Sub(today).Hours() / 24.0))
			daysRemaining = &diff
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat rencana pernikahan",
		"data": models.WeddingPlanDetailResponse{
			Plan:    *plan,
			Members: memberSummaries,
			Stats: models.WeddingPlanSummaryStats{
				DaysRemaining:          daysRemaining,
				TotalTasks:             int(totalTasks),
				CompletedTasks:         int(completedTasks),
				TotalVendors:           len(enrichedVendors),
				SyncedVendors:          syncedCount,
				TotalAgreedCost:        totalAgreedCost,
				TotalDpPaid:            totalDpPaid,
				RemainingVendorBalance: remainingBalance,
				LinkedInvitationsCount: len(invitations),
				TotalRsvpHadir:         totalRsvpHadir,
				TotalPaxHadir:          totalPaxHadir,
			},
		},
	})
}

// UpdateMyWeddingPlan handles PUT /api/couple/plan
func UpdateMyWeddingPlan(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.UpdateWeddingPlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	if strings.TrimSpace(req.GroomName) != "" {
		plan.GroomName = strings.TrimSpace(req.GroomName)
	}
	if strings.TrimSpace(req.BrideName) != "" {
		plan.BrideName = strings.TrimSpace(req.BrideName)
	} else if req.BrideName == "" && strings.TrimSpace(req.Title) != "" {
		// Biarkan opsional bila hanya mengubah title
	}
	if strings.TrimSpace(req.Title) != "" {
		plan.Title = strings.TrimSpace(req.Title)
	} else if plan.GroomName != "" && plan.BrideName != "" {
		plan.Title = fmt.Sprintf("Pernikahan %s & %s", plan.GroomName, plan.BrideName)
	}

	plan.WeddingDate = strings.TrimSpace(req.WeddingDate)
	plan.AkadTime = strings.TrimSpace(req.AkadTime)
	plan.ResepsiTime = strings.TrimSpace(req.ResepsiTime)
	plan.City = strings.TrimSpace(req.City)
	plan.VenueName = strings.TrimSpace(req.VenueName)
	plan.VenueAddress = strings.TrimSpace(req.VenueAddress)
	plan.MapsURL = strings.TrimSpace(req.MapsURL)
	if req.BudgetTarget >= 0 {
		plan.BudgetTarget = req.BudgetTarget
	}
	plan.Notes = strings.TrimSpace(req.Notes)
	plan.UpdatedAt = time.Now()

	if err := config.DB.Save(plan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui rencana pernikahan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Rencana pernikahan berhasil diperbarui",
		"data":    plan,
	})
}

// JoinWeddingPlanByCode handles POST /api/couple/plan/join untuk bergabung ke WeddingPlan pasangan
func JoinWeddingPlanByCode(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Sesi tidak valid"})
	}

	var req models.JoinWeddingPlanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	code := strings.ToUpper(strings.TrimSpace(req.JoinCode))
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kode gabung pasangan wajib diisi"})
	}

	var targetPlan models.WeddingPlan
	if err := config.DB.Where("UPPER(join_code) = ?", code).First(&targetPlan).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kode gabung tidak ditemukan. Periksa kembali kode pasangan Anda."})
	}

	var alreadyMember models.WeddingPlanMember
	if err := config.DB.Where("wedding_plan_id = ? AND user_id = ?", targetPlan.ID, userID).First(&alreadyMember).Error; err == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Anda sudah tergabung dalam rencana pernikahan ini",
			"data":    targetPlan,
		})
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// Pindahkan keanggotaan user dari plan solo lama (jika ada) ke plan pasangan
		var oldMembers []models.WeddingPlanMember
		tx.Where("user_id = ?", userID).Find(&oldMembers)
		if err := tx.Where("user_id = ?", userID).Delete(&models.WeddingPlanMember{}).Error; err != nil {
			return err
		}

		for _, om := range oldMembers {
			var remainingMembers, vendorCount, invCount int64
			tx.Model(&models.WeddingPlanMember{}).Where("wedding_plan_id = ?", om.WeddingPlanID).Count(&remainingMembers)
			tx.Model(&models.WeddingPlanVendor{}).Where("wedding_plan_id = ?", om.WeddingPlanID).Count(&vendorCount)
			tx.Model(&models.Invitation{}).Where("wedding_plan_id = ?", om.WeddingPlanID).Count(&invCount)
			if remainingMembers == 0 && vendorCount == 0 && invCount == 0 {
				tx.Where("wedding_plan_id = ?", om.WeddingPlanID).Delete(&models.WeddingTask{})
				tx.Where("id = ?", om.WeddingPlanID).Delete(&models.WeddingPlan{})
			}
		}

		newMember := models.WeddingPlanMember{
			ID:            "wpm_" + uuid.New().String()[:12],
			WeddingPlanID: targetPlan.ID,
			UserID:        userID,
			MemberRole:    "partner",
			CreatedAt:     time.Now(),
		}
		return tx.Create(&newMember).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal bergabung ke rencana pernikahan pasangan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil bergabung ke Wedding Plan pasangan",
		"data":    targetPlan,
	})
}

// ── Vendor & Budget Handlers ───────────────────────────────────────────────────────────────

// GetWeddingVendors handles GET /api/couple/vendors
func GetWeddingVendors(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	vendors, err := buildEnrichedWeddingVendors(plan.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil daftar vendor"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat daftar vendor pernikahan",
		"data":    vendors,
	})
}

// AddManualWeddingVendor handles POST /api/couple/vendors
func AddManualWeddingVendor(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.CreateOrUpdateWeddingVendorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	name := strings.TrimSpace(req.VendorName)
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama vendor wajib diisi"})
	}
	if req.AgreedPrice < 0 || req.DpPaid < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nominal harga atau DP tidak boleh negatif"})
	}

	status := strings.ToLower(strings.TrimSpace(req.PaymentStatus))
	if status == "" {
		if req.AgreedPrice > 0 && req.DpPaid >= req.AgreedPrice {
			status = "paid"
		} else if req.DpPaid > 0 {
			status = "dp"
		} else {
			status = "unpaid"
		}
	}

	now := time.Now()
	vendor := models.WeddingPlanVendor{
		ID:              "wpv_" + uuid.New().String()[:12],
		WeddingPlanID:   plan.ID,
		VendorCategory:  models.NormalizeVendorCategory(req.VendorCategory),
		VendorName:      name,
		VendorPhone:     utils.NormalizePhone(req.VendorPhone),
		VendorInstagram: strings.TrimPrefix(strings.TrimSpace(req.VendorInstagram), "@"),
		PackageName:     strings.TrimSpace(req.PackageName),
		AgreedPrice:     req.AgreedPrice,
		DpPaid:          req.DpPaid,
		PaymentStatus:   status,
		DueDate:         strings.TrimSpace(req.DueDate),
		Notes:           strings.TrimSpace(req.Notes),
		SourceType:      "manual",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := config.DB.Create(&vendor).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambahkan vendor"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Vendor berhasil ditambahkan",
		"data":    vendor,
	})
}

// UpdateWeddingVendor handles PUT /api/couple/vendors/:id
func UpdateWeddingVendor(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var vendor models.WeddingPlanVendor
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&vendor).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data vendor tidak ditemukan"})
	}

	var req models.CreateOrUpdateWeddingVendorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	// Untuk vendor yang tersinkron Agenda Dekorin, nominal biaya & status bayar tetap mengikuti data resmi vendor
	if vendor.AgendaID == "" {
		if strings.TrimSpace(req.VendorName) != "" {
			vendor.VendorName = strings.TrimSpace(req.VendorName)
		}
		if strings.TrimSpace(req.VendorCategory) != "" {
			vendor.VendorCategory = models.NormalizeVendorCategory(req.VendorCategory)
		}
		if req.VendorPhone != "" {
			vendor.VendorPhone = utils.NormalizePhone(req.VendorPhone)
		}
		vendor.VendorInstagram = strings.TrimPrefix(strings.TrimSpace(req.VendorInstagram), "@")
		vendor.PackageName = strings.TrimSpace(req.PackageName)
		if req.AgreedPrice >= 0 {
			vendor.AgreedPrice = req.AgreedPrice
		}
		if req.DpPaid >= 0 {
			vendor.DpPaid = req.DpPaid
		}
		if strings.TrimSpace(req.PaymentStatus) != "" {
			vendor.PaymentStatus = strings.ToLower(strings.TrimSpace(req.PaymentStatus))
		}
	}

	vendor.DueDate = strings.TrimSpace(req.DueDate)
	vendor.Notes = strings.TrimSpace(req.Notes)
	vendor.UpdatedAt = time.Now()

	if err := config.DB.Save(&vendor).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui vendor"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data vendor berhasil diperbarui",
		"data":    vendor,
	})
}

// DeleteWeddingVendor handles DELETE /api/couple/vendors/:id
func DeleteWeddingVendor(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var vendor models.WeddingPlanVendor
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&vendor).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Vendor tidak ditemukan"})
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if vendor.AgendaID != "" {
			if err := tx.Model(&models.Agenda{}).
				Where("id = ? AND wedding_plan_id = ?", vendor.AgendaID, plan.ID).
				Update("wedding_plan_id", "").Error; err != nil {
				return err
			}
		}
		return tx.Delete(&vendor).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus vendor dari daftar"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Vendor berhasil dihapus dari rencana pernikahan"})
}

// LinkDekorinAgenda handles POST /api/couple/vendors/link-agenda untuk menautkan booking/form vendor Dekorin
func LinkDekorinAgenda(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.LinkDekorinAgendaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	token := extractLastPathToken(req.TokenOrUrl)
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kode form atau link booking vendor wajib diisi"})
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda / link form vendor tidak ditemukan. Pastikan kode atau tautan dari vendor sudah benar.",
		})
	}

	if agenda.WeddingPlanID != "" && agenda.WeddingPlanID != plan.ID {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Booking vendor ini sudah tertaut ke Wedding Plan lain.",
		})
	}

	var existingCount int64
	config.DB.Model(&models.WeddingPlanVendor{}).
		Where("wedding_plan_id = ? AND agenda_id = ?", plan.ID, agenda.ID).
		Count(&existingCount)
	if existingCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Agenda vendor ini sudah terhubung di rencana pernikahan Anda.",
		})
	}

	var comp models.Company
	vendorName := strings.TrimSpace(agenda.BrandName)
	vendorCategory := "decoration"
	vendorPhone := strings.TrimSpace(agenda.BrandPhone)
	vendorIG := strings.TrimSpace(agenda.BrandInstagram)
	if agenda.CompanyID != "" {
		if err := config.DB.Where("id = ?", agenda.CompanyID).First(&comp).Error; err == nil {
			if vendorName == "" {
				vendorName = comp.Name
			}
			vendorCategory = models.NormalizeVendorCategory(comp.Category)
			if vendorPhone == "" {
				vendorPhone = comp.Phone
			}
			if vendorIG == "" {
				vendorIG = comp.Instagram
			}
		}
	}
	if vendorName == "" {
		vendorName = "Vendor Dekorin"
	}

	now := time.Now()
	newVendor := models.WeddingPlanVendor{
		ID:              "wpv_" + uuid.New().String()[:12],
		WeddingPlanID:   plan.ID,
		VendorCategory:  vendorCategory,
		VendorName:      vendorName,
		VendorPhone:     vendorPhone,
		VendorInstagram: vendorIG,
		CompanyID:       agenda.CompanyID,
		AgendaID:        agenda.ID,
		PackageName:     agenda.PackageName,
		AgreedPrice:     agenda.CalculateTotalBiaya(),
		DpPaid:          agenda.DpAmount,
		PaymentStatus:   agenda.PaymentStatus,
		SourceType:      "dekorin_agenda",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Agenda{}).
			Where("id = ?", agenda.ID).
			Update("wedding_plan_id", plan.ID).Error; err != nil {
			return err
		}
		return tx.Create(&newVendor).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menautkan booking vendor"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Booking vendor Dekorin berhasil disinkronkan!",
		"data":    newVendor,
	})
}

// AddPublicVendorToPlan handles POST /api/couple/vendors/from-directory
func AddPublicVendorToPlan(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.AddPublicVendorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	cid := strings.TrimSpace(req.CompanyID)
	if cid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID vendor wajib diisi"})
	}

	var comp models.Company
	if err := config.DB.Where("id = ? AND is_public_listed = ?", cid, true).First(&comp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Vendor tidak ditemukan di direktori publik"})
	}

	var existingCount int64
	config.DB.Model(&models.WeddingPlanVendor{}).
		Where("wedding_plan_id = ? AND company_id = ?", plan.ID, comp.ID).
		Count(&existingCount)
	if existingCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Vendor ini sudah masuk ke daftar Wedding Plan Anda"})
	}

	now := time.Now()
	vendor := models.WeddingPlanVendor{
		ID:              "wpv_" + uuid.New().String()[:12],
		WeddingPlanID:   plan.ID,
		VendorCategory:  models.NormalizeVendorCategory(comp.Category),
		VendorName:      comp.Name,
		VendorPhone:     comp.Phone,
		VendorInstagram: comp.Instagram,
		CompanyID:       comp.ID,
		PackageName:     strings.TrimSpace(req.PackageName),
		AgreedPrice:     math.Max(0, req.AgreedPrice),
		DpPaid:          0,
		PaymentStatus:   "inquiry",
		Notes:           strings.TrimSpace(req.Notes),
		SourceType:      "dekorin_directory",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := config.DB.Create(&vendor).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambahkan vendor ke Wedding Plan"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Vendor berhasil ditambahkan ke rencana pernikahan",
		"data":    vendor,
	})
}

// ── Checklist Tugas Pernikahan Handlers ────────────────────────────────────────────────────

// GetWeddingTasks handles GET /api/couple/tasks
func GetWeddingTasks(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var tasks []models.WeddingTask
	if err := config.DB.Where("wedding_plan_id = ?", plan.ID).
		Order("is_completed ASC, sort_order ASC, created_at ASC").
		Find(&tasks).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat checklist persiapan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat checklist persiapan",
		"data":    tasks,
	})
}

// CreateWeddingTask handles POST /api/couple/tasks
func CreateWeddingTask(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.CreateOrUpdateWeddingTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Judul tugas checklist wajib diisi"})
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Lainnya"
	}
	phase := strings.TrimSpace(req.Phase)
	if phase == "" {
		phase = "Persiapan"
	}

	var count int64
	config.DB.Model(&models.WeddingTask{}).Where("wedding_plan_id = ?", plan.ID).Count(&count)

	now := time.Now()
	task := models.WeddingTask{
		ID:            "wt_" + uuid.New().String()[:12],
		WeddingPlanID: plan.ID,
		Title:         title,
		Category:      category,
		Phase:         phase,
		DueDate:       strings.TrimSpace(req.DueDate),
		IsCompleted:   req.IsCompleted != nil && *req.IsCompleted,
		Notes:         strings.TrimSpace(req.Notes),
		SortOrder:     int(count) + 1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := config.DB.Create(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambah tugas checklist"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Tugas checklist berhasil ditambahkan",
		"data":    task,
	})
}

// UpdateWeddingTask handles PUT /api/couple/tasks/:id
func UpdateWeddingTask(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var task models.WeddingTask
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&task).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tugas checklist tidak ditemukan"})
	}

	var req models.CreateOrUpdateWeddingTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	if strings.TrimSpace(req.Title) != "" {
		task.Title = strings.TrimSpace(req.Title)
	}
	if strings.TrimSpace(req.Category) != "" {
		task.Category = strings.TrimSpace(req.Category)
	}
	if strings.TrimSpace(req.Phase) != "" {
		task.Phase = strings.TrimSpace(req.Phase)
	}
	task.DueDate = strings.TrimSpace(req.DueDate)
	task.Notes = strings.TrimSpace(req.Notes)
	if req.IsCompleted != nil {
		task.IsCompleted = *req.IsCompleted
	}
	task.UpdatedAt = time.Now()

	if err := config.DB.Save(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui tugas checklist"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Tugas checklist berhasil diperbarui",
		"data":    task,
	})
}

// ToggleWeddingTask handles PATCH /api/couple/tasks/:id/toggle
func ToggleWeddingTask(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var task models.WeddingTask
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&task).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tugas checklist tidak ditemukan"})
	}

	task.IsCompleted = !task.IsCompleted
	task.UpdatedAt = time.Now()
	if err := config.DB.Save(&task).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui status checklist"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Status checklist diperbarui",
		"data":    task,
	})
}

// DeleteWeddingTask handles DELETE /api/couple/tasks/:id
func DeleteWeddingTask(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	res := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).Delete(&models.WeddingTask{})
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus tugas checklist"})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Tugas checklist tidak ditemukan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Tugas checklist berhasil dihapus"})
}

// ── Klaim & Kelola Undangan Digital oleh Pengantin ─────────────────────────────────────────

// GetCoupleInvitations handles GET /api/couple/invitations
func GetCoupleInvitations(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var invitations []models.Invitation
	if err := config.DB.Preload("Wishes", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Where("wedding_plan_id = ?", plan.ID).Order("created_at DESC").Find(&invitations).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat undangan digital"})
	}

	for i := range invitations {
		ensureInvitationAccessCredentials(&invitations[i])
		invitations[i].WishesCount = len(invitations[i].Wishes)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat undangan tertaut",
		"data":    invitations,
	})
}

// ClaimCoupleInvitation handles POST /api/couple/invitations/claim
func ClaimCoupleInvitation(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	var req models.ClaimInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	var invitation models.Invitation
	tokenInput := extractLastPathToken(req.EditTokenOrUrl)
	slugInput := sanitizeInvitationSlug(req.Slug)
	pinInput := strings.TrimSpace(req.Pin)

	if tokenInput != "" && strings.HasPrefix(tokenInput, "inv_sec_") {
		if err := config.DB.Where("edit_token = ?", tokenInput).First(&invitation).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Tautan kelola undangan tidak valid atau tidak ditemukan.",
			})
		}
		ensureInvitationAccessCredentials(&invitation)
	} else {
		if slugInput == "" || pinInput == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Masukkan Slug Undangan + PIN 6 digit, atau tempel Link Kelola Undangan dari vendor Anda.",
			})
		}
		if err := config.DB.Where("LOWER(slug) = ?", slugInput).First(&invitation).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Undangan tidak ditemukan atau PIN akses salah.",
			})
		}
		ensureInvitationAccessCredentials(&invitation)
		expectedPin := strings.TrimSpace(invitation.EditPin)
		if expectedPin == "" || subtle.ConstantTimeCompare([]byte(expectedPin), []byte(pinInput)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Undangan tidak ditemukan atau PIN akses salah.",
			})
		}
	}

	if invitation.WeddingPlanID != "" && invitation.WeddingPlanID != plan.ID {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Undangan digital ini sudah diklaim oleh Wedding Plan lain.",
		})
	}

	if err := config.DB.Model(&models.Invitation{}).
		Where("id = ?", invitation.ID).
		Update("wedding_plan_id", plan.ID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menautkan undangan ke Wedding Plan"})
	}

	invitation.WeddingPlanID = plan.ID
	config.DB.Where("invitation_id = ?", invitation.ID).Order("created_at DESC").Find(&invitation.Wishes)
	invitation.WishesCount = len(invitation.Wishes)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Undangan digital berhasil ditautkan ke Wedding Plan!",
		"data":    invitation,
	})
}

// UnclaimCoupleInvitation handles DELETE /api/couple/invitations/:id/claim
func UnclaimCoupleInvitation(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	res := config.DB.Model(&models.Invitation{}).
		Where("id = ? AND wedding_plan_id = ?", id, plan.ID).
		Update("wedding_plan_id", "")
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal melepas tautan undangan"})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Undangan tidak ditemukan pada Wedding Plan Anda"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Tautan undangan berhasil dilepas"})
}

// GetCoupleInvitationDetail handles GET /api/couple/invitations/:id
func GetCoupleInvitationDetail(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var invitation models.Invitation
	if err := config.DB.Preload("Wishes", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Undangan tidak ditemukan atau bukan milik Wedding Plan Anda"})
	}

	ensureInvitationAccessCredentials(&invitation)
	invitation.WishesCount = len(invitation.Wishes)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil detail undangan",
		"data":    invitation,
	})
}

// UpdateCoupleInvitation handles PUT /api/couple/invitations/:id
func UpdateCoupleInvitation(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var invitation models.Invitation
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Undangan tidak ditemukan atau Anda tidak memiliki akses"})
	}

	var req models.CreateOrUpdateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tidak valid"})
	}

	if errStatus, errMsg := applyPortalInvitationUpdates(&invitation, &req); errStatus != 0 {
		return c.Status(errStatus).JSON(fiber.Map{"error": errMsg})
	}

	if err := config.DB.Save(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui undangan: " + err.Error()})
	}

	utils.GlobalInvitationCache.Invalidate(invitation.Slug)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Perubahan undangan berhasil disimpan",
		"data":    invitation,
	})
}

// SaveCoupleInvitationGuests handles PUT /api/couple/invitations/:id/guests
func SaveCoupleInvitationGuests(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var invitation models.Invitation
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Undangan tidak ditemukan"})
	}

	var req models.SavePortalGuestsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format data tamu tidak valid"})
	}

	invitation.GuestListJSON = req.GuestListJSON
	if err := config.DB.Model(&models.Invitation{}).
		Where("id = ?", invitation.ID).
		Update("guest_list_json", invitation.GuestListJSON).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan daftar tamu"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Daftar tamu undangan berhasil disimpan",
		"data":    invitation,
	})
}

// SyncWeddingPlanToInvitation handles POST /api/couple/invitations/:id/sync-plan
// untuk menyalin data tanggal, jam, lokasi, Maps, dan nama mempelai dari WeddingPlan ke Undangan Digital.
func SyncWeddingPlanToInvitation(c *fiber.Ctx) error {
	plan, _, err := resolveCoupleWeddingPlan(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	id := c.Params("id")
	var invitation models.Invitation
	if err := config.DB.Where("id = ? AND wedding_plan_id = ?", id, plan.ID).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Undangan tidak ditemukan"})
	}

	if strings.TrimSpace(plan.GroomName) != "" {
		invitation.GroomName = plan.GroomName
	}
	if strings.TrimSpace(plan.BrideName) != "" {
		invitation.BrideName = plan.BrideName
	}
	if strings.TrimSpace(plan.WeddingDate) != "" {
		invitation.AkadDate = plan.WeddingDate
		invitation.ResepsiDate = plan.WeddingDate
	}
	if strings.TrimSpace(plan.AkadTime) != "" {
		invitation.AkadTime = plan.AkadTime
	}
	if strings.TrimSpace(plan.ResepsiTime) != "" {
		invitation.ResepsiTime = plan.ResepsiTime
	}
	if strings.TrimSpace(plan.VenueName) != "" {
		invitation.AkadVenue = plan.VenueName
		invitation.ResepsiVenue = plan.VenueName
	}
	if strings.TrimSpace(plan.VenueAddress) != "" {
		invitation.AkadAddress = plan.VenueAddress
		invitation.ResepsiAddress = plan.VenueAddress
	}
	if strings.TrimSpace(plan.MapsURL) != "" {
		invitation.AkadMapsURL = plan.MapsURL
		invitation.ResepsiMapsURL = plan.MapsURL
	}

	if err := config.DB.Save(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyinkronkan data Wedding Plan ke undangan"})
	}

	utils.GlobalInvitationCache.Invalidate(invitation.Slug)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data jadwal, lokasi, & mempelai berhasil disinkronkan ke undangan digital",
		"data":    invitation,
	})
}
