package handlers

import (
	"math"
	"strconv"
	"strings"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
)

// GetPublicVendors handles GET /api/vendors/directory
// Menampilkan daftar vendor terkurasi yang mengaktifkan listing publik untuk dicari oleh Calon Pengantin.
func GetPublicVendors(c *fiber.Ctx) error {
	query := config.DB.Model(&models.Company{}).Where("is_public_listed = ?", true)

	if category := strings.TrimSpace(c.Query("category")); category != "" && !strings.EqualFold(category, "all") {
		query = query.Where("LOWER(category) = ?", models.NormalizeVendorCategory(category))
	}

	if city := strings.TrimSpace(c.Query("city")); city != "" {
		query = query.Where("LOWER(city) LIKE ?", "%"+strings.ToLower(city)+"%")
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(city) LIKE ? OR LOWER(description) LIKE ? OR LOWER(address) LIKE ?",
			pattern, pattern, pattern, pattern,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat direktori vendor"})
	}

	page := 1
	limit := 30
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 && l <= 60 {
		limit = l
	}

	var companies []models.Company
	if err := query.Order("updated_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&companies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data direktori vendor"})
	}

	companyIDs := make([]string, 0, len(companies))
	for _, comp := range companies {
		companyIDs = append(companyIDs, comp.ID)
	}

	pkgByComp := make(map[string][]models.Package)
	modelByComp := make(map[string][]models.DecorationModel)
	if len(companyIDs) > 0 {
		var allPackages []models.Package
		config.DB.Where("company_id IN ?", companyIDs).Order("sort_order ASC, base_price ASC").Find(&allPackages)
		for _, p := range allPackages {
			pkgByComp[p.CompanyID] = append(pkgByComp[p.CompanyID], p)
		}

		var allModels []models.DecorationModel
		config.DB.Where("company_id IN ?", companyIDs).Order("sort_order ASC, created_at DESC").Find(&allModels)
		for _, m := range allModels {
			modelByComp[m.CompanyID] = append(modelByComp[m.CompanyID], m)
		}
	}

	results := make([]models.PublicVendorSummary, 0, len(companies))
	for _, comp := range companies {
		pkgs := pkgByComp[comp.ID]
		mods := modelByComp[comp.ID]

		minPrice := 0.0
		sampleImages := make([]string, 0, 4)
		seenImg := make(map[string]bool)

		for _, p := range pkgs {
			if p.BasePrice > 0 && (minPrice == 0 || p.BasePrice < minPrice) {
				minPrice = p.BasePrice
			}
			img := strings.TrimSpace(p.ImageURL)
			if img != "" && !seenImg[img] && len(sampleImages) < 4 {
				seenImg[img] = true
				sampleImages = append(sampleImages, img)
			}
		}

		for _, m := range mods {
			if m.Price > 0 && (minPrice == 0 || m.Price < minPrice) {
				minPrice = m.Price
			}
			img := strings.TrimSpace(m.ImageURL)
			if img != "" && !seenImg[img] && len(sampleImages) < 4 {
				seenImg[img] = true
				sampleImages = append(sampleImages, img)
			}
		}

		results = append(results, models.PublicVendorSummary{
			ID:           comp.ID,
			Name:         comp.Name,
			Category:     models.NormalizeVendorCategory(comp.Category),
			City:         comp.City,
			Address:      comp.Address,
			Phone:        comp.Phone,
			Instagram:    comp.Instagram,
			LogoURL:      comp.LogoURL,
			Description:  comp.Description,
			PackageCount: len(pkgs),
			ModelCount:   len(mods),
			MinPrice:     minPrice,
			SampleImages: sampleImages,
		})
	}

	totalPages := 1
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat direktori vendor",
		"data":    results,
		"meta": fiber.Map{
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// GetPublicVendorDetail handles GET /api/vendors/directory/:id
// Mengembalikan detail profil publik vendor beserta daftar paket & portofolio model (tanpa data finansial/bank internal).
func GetPublicVendorDetail(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	var comp models.Company
	if err := config.DB.Where("id = ? AND is_public_listed = ?", id, true).First(&comp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Profil vendor tidak ditemukan atau tidak dipublikasikan"})
	}

	var pkgs []models.Package
	config.DB.Where("company_id = ?", comp.ID).Order("sort_order ASC, base_price ASC").Find(&pkgs)

	var mods []models.DecorationModel
	config.DB.Where("company_id = ?", comp.ID).Order("sort_order ASC, created_at DESC").Find(&mods)

	minPrice := 0.0
	sampleImages := make([]string, 0, 6)
	seenImg := make(map[string]bool)

	pubPkgs := make([]models.PublicVendorPackageItem, 0, len(pkgs))
	for _, p := range pkgs {
		if p.BasePrice > 0 && (minPrice == 0 || p.BasePrice < minPrice) {
			minPrice = p.BasePrice
		}
		if img := strings.TrimSpace(p.ImageURL); img != "" && !seenImg[img] && len(sampleImages) < 6 {
			seenImg[img] = true
			sampleImages = append(sampleImages, img)
		}
		pubPkgs = append(pubPkgs, models.PublicVendorPackageItem{
			ID:          p.ID,
			BrandName:   p.BrandName,
			Name:        p.Name,
			ImageURL:    p.ImageURL,
			Description: p.Description,
			BasePrice:   p.BasePrice,
		})
	}

	pubMods := make([]models.PublicVendorModelItem, 0, len(mods))
	for _, m := range mods {
		if m.Price > 0 && (minPrice == 0 || m.Price < minPrice) {
			minPrice = m.Price
		}
		if img := strings.TrimSpace(m.ImageURL); img != "" && !seenImg[img] && len(sampleImages) < 6 {
			seenImg[img] = true
			sampleImages = append(sampleImages, img)
		}
		pubMods = append(pubMods, models.PublicVendorModelItem{
			ID:          m.ID,
			Name:        m.Name,
			ImageURL:    m.ImageURL,
			Width:       m.Width,
			Price:       m.Price,
			Description: m.Description,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat detail vendor",
		"data": models.PublicVendorDetailResponse{
			PublicVendorSummary: models.PublicVendorSummary{
				ID:           comp.ID,
				Name:         comp.Name,
				Category:     models.NormalizeVendorCategory(comp.Category),
				City:         comp.City,
				Address:      comp.Address,
				Phone:        comp.Phone,
				Instagram:    comp.Instagram,
				LogoURL:      comp.LogoURL,
				Description:  comp.Description,
				PackageCount: len(pubPkgs),
				ModelCount:   len(pubMods),
				MinPrice:     minPrice,
				SampleImages: sampleImages,
			},
			Email:    comp.Email,
			Packages: pubPkgs,
			Models:   pubMods,
		},
	})
}
