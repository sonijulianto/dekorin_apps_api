package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
)

func TestNormalizeVendorCategory(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", "decoration"},
		{"Dekorasi", "decoration"},
		{"wo", "wo"},
		{"Wedding_Organizer", "wo"},
		{"MUA", "mua"},
		{"photography", "photo_video"},
		{"katering", "catering"},
		{"Busana", "attire"},
		{"MC", "entertainment"},
		{"gedung", "venue"},
		{"souvenir", "other"},
	}

	for _, tc := range cases {
		actual := models.NormalizeVendorCategory(tc.input)
		if actual != tc.expected {
			t.Errorf("NormalizeVendorCategory(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestExtractLastPathToken(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"tok_abc123", "tok_abc123"},
		{"https://dekorin.my.id/form/tok_xyz789", "tok_xyz789"},
		{"https://undangan.dekorin.my.id/kelola/inv_sec_123456789012?src=wa", "inv_sec_123456789012"},
		{"/form/tok_clean/", "tok_clean"},
	}

	for _, tc := range cases {
		actual := extractLastPathToken(tc.input)
		if actual != tc.expected {
			t.Errorf("extractLastPathToken(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestGenerateWeddingJoinCode(t *testing.T) {
	code := generateWeddingJoinCode()
	if len(code) != 8 {
		t.Fatalf("expected 8-char join code, got %d (%q)", len(code), code)
	}
}

func TestCoupleRoleMiddlewareIsolation(t *testing.T) {
	app := fiber.New()
	injectRole := func(role string) fiber.Handler {
		return func(c *fiber.Ctx) error {
			c.Locals("user_role", role)
			return c.Next()
		}
	}

	app.Get("/vendor-only-as-couple", injectRole("couple"), middleware.RequireVendor(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/vendor-only-as-owner", injectRole("owner"), middleware.RequireVendor(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/couple-only-as-couple", injectRole("couple"), middleware.RequireCouple(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/couple-only-as-crew", injectRole("crew"), middleware.RequireCouple(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/vendor-only-as-couple", http.StatusForbidden},
		{"/vendor-only-as-owner", http.StatusOK},
		{"/couple-only-as-couple", http.StatusOK},
		{"/couple-only-as-crew", http.StatusForbidden},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %s failed: %v", tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.wantStatus {
			t.Errorf("GET %s returned status %d; want %d", tc.path, resp.StatusCode, tc.wantStatus)
		}
	}
}

func TestCoupleDefaultPermissionsEmpty(t *testing.T) {
	perms := GetDefaultPermissionsForRole("couple")
	if perms == nil || len(perms) != 0 {
		t.Fatalf("expected empty non-nil permissions slice for couple role, got %#v", perms)
	}
}
