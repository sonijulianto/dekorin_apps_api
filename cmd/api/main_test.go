package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSuperadminPWARoutes(t *testing.T) {
	webDir := filepath.Join("..", "..", "web")
	app := fiber.New()
	registerSuperadminWebRoutes(app, webDir)

	t.Run("superadmin html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/superadmin", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
			t.Fatalf("expected no-store Cache-Control, got %q", cc)
		}
	})

	t.Run("superadmin manifest json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/superadmin/manifest.json", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/manifest+json") {
			t.Fatalf("expected application/manifest+json Content-Type, got %q", ct)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read manifest body: %v", err)
		}

		var manifest struct {
			StartURL string `json:"start_url"`
			Scope    string `json:"scope"`
			Display  string `json:"display"`
			Icons    []struct {
				Src     string `json:"src"`
				Sizes   string `json:"sizes"`
				Purpose string `json:"purpose"`
			} `json:"icons"`
		}
		if err := json.Unmarshal(body, &manifest); err != nil {
			t.Fatalf("invalid manifest JSON: %v", err)
		}
		if manifest.StartURL != "/superadmin" || manifest.Scope != "/superadmin" || manifest.Display != "standalone" {
			t.Fatalf("unexpected manifest config: %+v", manifest)
		}
		if len(manifest.Icons) < 2 {
			t.Fatalf("expected at least 2 icons in manifest, got %d", len(manifest.Icons))
		}
		for _, icon := range manifest.Icons {
			cleanPath := strings.Split(icon.Src, "?")[0]
			cleanPath = strings.TrimPrefix(cleanPath, "/assets/")
			diskPath := filepath.Join(webDir, "assets", filepath.FromSlash(cleanPath))
			if _, err := os.Stat(diskPath); err != nil {
				t.Fatalf("manifest icon %q missing on disk (%s): %v", icon.Src, diskPath, err)
			}
		}
	})

	t.Run("superadmin service worker", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/superadmin/sw.js", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/javascript") {
			t.Fatalf("expected application/javascript Content-Type, got %q", ct)
		}
		if swAllowed := resp.Header.Get("Service-Worker-Allowed"); swAllowed != "/superadmin" {
			t.Fatalf("expected Service-Worker-Allowed=/superadmin, got %q", swAllowed)
		}
		if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
			t.Fatalf("expected no-store Cache-Control, got %q", cc)
		}
	})
}
