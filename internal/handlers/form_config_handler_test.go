package handlers

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"dekorin_apps_api/internal/models"
)

func TestFormConfigDefaultsAndLockingRule(t *testing.T) {
	defaults := GetDefaultFormFields()
	if len(defaults) == 0 {
		t.Fatal("GetDefaultFormFields() should not be empty")
	}

	cfg := models.CompanyFormConfig{
		CompanyID: "cmp_test",
		BrandID:   "",
		ShowPrice: true,
		Fields:    defaults,
	}

	// 1. Draft agenda pending -> NOT locked
	draft := models.Agenda{
		ID:          "agd_draft",
		ClientName:  "Menunggu Isian Client",
		FormStatus:  "pending",
		IsFormDraft: true,
	}
	locked, missing := IsClientFormComplete(draft, cfg)
	if locked {
		t.Error("Expected draft agenda with pending status to be unlocked")
	}
	if len(missing) == 0 {
		t.Error("Expected missing required fields on empty draft agenda")
	}

	// 2. Filled status, but still missing required event_address -> NOT locked
	partialFilled := models.Agenda{
		ID:            "agd_partial",
		ClientName:    "Rian & Nisa",
		ClientPhone:   "08123456789",
		EventDateTime: time.Now().Add(48 * time.Hour),
		EventAddress:  "",
		FormStatus:    "filled",
		IsFormDraft:   false,
	}
	locked, missing = IsClientFormComplete(partialFilled, cfg)
	if locked {
		t.Error("Expected partially filled agenda (missing required address) to remain unlocked")
	}
	if len(missing) != 1 || !strings.Contains(missing[0], "Alamat") {
		t.Errorf("Expected missing event_address, got: %v", missing)
	}

	// 3. All required fields filled & status filled -> LOCKED (Ringkasan Konfirmasi)
	completeAgenda := models.Agenda{
		ID:            "agd_complete",
		ClientName:    "Rian & Nisa",
		ClientPhone:   "08123456789",
		EventDateTime: time.Now().Add(48 * time.Hour),
		EventAddress:  "Gedung Harmoni Jakarta",
		FormStatus:    "filled",
		IsFormDraft:   false,
	}
	locked, missing = IsClientFormComplete(completeAgenda, cfg)
	if !locked {
		t.Errorf("Expected complete agenda to be locked, missing: %v", missing)
	}

	// 4. Custom required field added by vendor -> UNLOCKED until custom field is answered
	cfgWithCustom := models.CompanyFormConfig{
		CompanyID: "cmp_test",
		ShowPrice: true,
		Fields: append(defaults, models.FormFieldDefinition{
			Key:        "custom_jam_loading",
			Label:      "Jam Akses Loading",
			Type:       "text",
			IsSystem:   false,
			IsActive:   true,
			IsRequired: true,
			SortOrder:  len(defaults) + 1,
		}),
	}
	locked, missing = IsClientFormComplete(completeAgenda, cfgWithCustom)
	if locked {
		t.Error("Expected agenda missing newly required custom field to be unlocked")
	}
	if len(missing) != 1 || missing[0] != "Jam Akses Loading" {
		t.Errorf("Expected missing custom field 'Jam Akses Loading', got %v", missing)
	}

	// Once custom answer is present -> LOCKED again
	completeAgenda.ClientForm = &models.ClientForm{
		CustomAnswers: []models.CustomFormAnswer{
			{Key: "custom_jam_loading", Label: "Jam Akses Loading", Type: "text", Value: "22:00 WIB"},
		},
	}
	locked, missing = IsClientFormComplete(completeAgenda, cfgWithCustom)
	if !locked {
		t.Errorf("Expected agenda with custom answer to be locked, missing: %v", missing)
	}
}

func TestClientFormAndSuperadminTemplatesContainDynamicFormFeatures(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	clientFormPath := filepath.Join(wd, "..", "..", "web", "templates", "client_form.html")
	cfBytes, err := os.ReadFile(clientFormPath)
	if err != nil {
		t.Fatalf("client_form.html missing: %v", err)
	}
	cfHTML := string(cfBytes)
	for _, sub := range []string{
		"dynamicClientForm",
		"handleSubmitClientForm",
		"uploadClientImage",
		"Ajukan Perubahan / Tambahan ke Admin (WA)",
		"Data Terkonfirmasi & Terkunci",
	} {
		if !strings.Contains(cfHTML, sub) {
			t.Errorf("client_form.html missing expected substring %q", sub)
		}
	}

	superadminPath := filepath.Join(wd, "..", "..", "web", "templates", "superadmin.html")
	saBytes, err := os.ReadFile(superadminPath)
	if err != nil {
		t.Fatalf("superadmin.html missing: %v", err)
	}
	saHTML := string(saBytes)
	for _, sub := range []string{
		"nav-form-config",
		"viewFormConfig",
		"modalFormField",
		"modalPendingLinks",
		"generateQuickClientLink()",
		"loadFormConfigView()",
		"saveFormConfig()",
		"unlockCurrentAgendaForm()",
	} {
		if !strings.Contains(saHTML, sub) {
			t.Errorf("superadmin.html missing expected substring %q", sub)
		}
	}
}

func TestWebTemplatesJavaScriptSyntax(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed, skipping JS syntax check")
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	scriptRe := regexp.MustCompile(`(?s)<script>(.*?)</script>`)
	tmplExprRe := regexp.MustCompile(`\{\{\.[^}]+\}\}`)

	for _, name := range []string{"superadmin.html", "client_form.html"} {
		p := filepath.Join(wd, "..", "..", "web", "templates", name)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", name, err)
		}
		matches := scriptRe.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			t.Errorf("%s expected to have inline <script> block", name)
		}
		for idx, m := range matches {
			cleaned := tmplExprRe.ReplaceAllString(m[1], "null")
			tmpFile, err := os.CreateTemp("", "check_*.js")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			tmpPath := tmpFile.Name()
			_, _ = tmpFile.WriteString(cleaned)
			_ = tmpFile.Close()

			out, err := exec.Command("node", "--check", tmpPath).CombinedOutput()
			_ = os.Remove(tmpPath)
			if err != nil {
				t.Errorf("%s inline script #%d has JS syntax error: %v\n%s", name, idx, err, string(out))
			}
		}
	}
}

func TestEventTypeNormalizationAndPresets(t *testing.T) {
	if got := NormalizeEventType("Ulang Tahun"); got != "birthday" {
		t.Errorf("Expected 'Ulang Tahun' -> 'birthday', got %q", got)
	}
	if got := NormalizeEventType("Engagement"); got != "lamaran" {
		t.Errorf("Expected 'Engagement' -> 'lamaran', got %q", got)
	}
	if got := NormalizeEventType("Umum"); got != "" {
		t.Errorf("Expected 'Umum' -> '', got %q", got)
	}
	if got := SlugifyCustomEventType("Khitanan Anak"); got != "custom_khitanan_anak" {
		t.Errorf("Expected 'custom_khitanan_anak', got %q", got)
	}
	if got := GetEventTypeLabel("custom_khitanan_anak", ""); got != "Khitanan Anak" {
		t.Errorf("Expected 'Khitanan Anak', got %q", got)
	}

	for _, et := range []string{"", "wedding", "lamaran", "birthday", "aqiqah"} {
		fields := GetPresetFormFields(et)
		if len(fields) == 0 {
			t.Errorf("GetPresetFormFields(%q) returned empty", et)
		}
		hasBirthdayAge := false
		hasLamaranSeat := false
		for _, f := range fields {
			if f.Key == "bday_age" {
				hasBirthdayAge = true
			}
			if f.Key == "lamaran_seating" {
				hasLamaranSeat = true
			}
		}
		if et == "birthday" && !hasBirthdayAge {
			t.Errorf("Expected birthday preset to include bday_age")
		}
		if et == "lamaran" && !hasLamaranSeat {
			t.Errorf("Expected lamaran preset to include lamaran_seating")
		}
	}
}
