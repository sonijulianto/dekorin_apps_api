package handlers

import (
	"testing"

	"dekorin_apps_api/internal/models"
)

func TestBuildLandingPartners(t *testing.T) {
	companies := []models.Company{
		{
			ID:        "comp-1",
			Name:      "Zebra Dekorasi",
			Category:  "dekorasi",
			City:      "Bandung",
			LogoURL:   "/uploads/zebra.png",
			Instagram: "@zebradekor",
		},
		{
			ID:        "comp-2",
			Name:      "Alpha Organizer",
			Category:  "wo",
			City:      "Jakarta",
			LogoURL:   "",
			Instagram: "https://instagram.com/alphawo",
		},
		{
			ID:   "comp-empty",
			Name: "   ",
		},
	}

	brands := []models.CompanyBrand{
		{
			ID:        "brand-1",
			CompanyID: "comp-1",
			Name:      "Aurel Rustic",
			LogoURL:   "/uploads/aurel.png",
			Instagram: "@aurelrustic",
		},
		{
			ID:        "brand-2",
			CompanyID: "comp-2",
			Name:      "Beta Planner",
			LogoURL:   "",
			Instagram: "@betaplanner",
		},
	}

	got := BuildLandingPartners(companies, brands)
	if len(got) != 4 {
		t.Fatalf("expected 4 partners, got %d", len(got))
	}

	// Expected order: logo-bearing items first (Aurel Rustic, Zebra Dekorasi),
	// then no-logo items alphabetically (Alpha Organizer, Beta Planner).
	wantOrder := []struct {
		name       string
		isSubBrand bool
		category   string
		city       string
		hasLogo    bool
	}{
		{"Aurel Rustic", true, "decoration", "Bandung", true},
		{"Zebra Dekorasi", false, "decoration", "Bandung", true},
		{"Alpha Organizer", false, "wo", "Jakarta", false},
		{"Beta Planner", true, "wo", "Jakarta", false},
	}

	for i, want := range wantOrder {
		item := got[i]
		if item.Name != want.name {
			t.Errorf("index %d: got name %q, want %q", i, item.Name, want.name)
		}
		if item.IsSubBrand != want.isSubBrand {
			t.Errorf("index %d (%s): got IsSubBrand=%v, want %v", i, item.Name, item.IsSubBrand, want.isSubBrand)
		}
		if item.Category != want.category {
			t.Errorf("index %d (%s): got Category=%q, want %q", i, item.Name, item.Category, want.category)
		}
		if item.City != want.city {
			t.Errorf("index %d (%s): got City=%q, want %q", i, item.Name, item.City, want.city)
		}
		if (item.LogoURL != "") != want.hasLogo {
			t.Errorf("index %d (%s): got LogoURL=%q, want hasLogo=%v", i, item.Name, item.LogoURL, want.hasLogo)
		}
	}
}
