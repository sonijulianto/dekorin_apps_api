package handlers

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dekorin_apps_api/internal/models"
)

func TestAgendaHasPackageAndStandaloneModel(t *testing.T) {
	standalone := models.Agenda{
		ID:                   "agd_1",
		PackageID:            "",
		PackageName:          "Belum Dipilih",
		PackagePrice:         0,
		ModelID:              "mdl_01",
		ModelName:            "Rustic Arch 3m",
		ModelAdditionalPrice: 2000000,
		TransportFee:         150000,
	}

	if standalone.HasPackage() {
		t.Error("expected standalone model agenda to have HasPackage() == false")
	}
	if !standalone.IsStandaloneModel() {
		t.Error("expected standalone model agenda to have IsStandaloneModel() == true")
	}
	if got := standalone.CalculateTotalBiaya(); got != 2150000 {
		t.Errorf("expected standalone total 2150000, got %.0f", got)
	}

	// Standalone model with ClientForm addons only (package_price 0, addons 300rb)
	standalone.ClientForm = &models.ClientForm{
		AddonsPrice: 300000,
		TotalPrice:  300000,
	}
	if got := standalone.CalculateTotalBiaya(); got != 2450000 {
		t.Errorf("expected standalone + addons + transport total 2450000, got %.0f", got)
	}

	// With package selected
	withPackage := models.Agenda{
		ID:                   "agd_2",
		PackageID:            "pkg_01",
		PackageName:          "Paket Silver",
		PackagePrice:         4000000,
		ModelID:              "mdl_01",
		ModelName:            "Rustic Arch 3m",
		ModelAdditionalPrice: 250000,
		TransportFee:         100000,
	}
	if !withPackage.HasPackage() {
		t.Error("expected agenda with PackageID to have HasPackage() == true")
	}
	if withPackage.IsStandaloneModel() {
		t.Error("expected agenda with PackageID to have IsStandaloneModel() == false")
	}
	if got := withPackage.CalculateTotalBiaya(); got != 4350000 {
		t.Errorf("expected package total 4350000, got %.0f", got)
	}
}

func TestPickContextualModelPrice(t *testing.T) {
	if got := pickContextualModelPrice(false, 250000, 2000000); got != 2000000 {
		t.Errorf("without package, expected model base price 2000000, got %.0f", got)
	}
	if got := pickContextualModelPrice(true, 250000, 2000000); got != 250000 {
		t.Errorf("with package, expected package additional_price 250000, got %.0f", got)
	}
	if got := pickContextualModelPrice(true, 0, 2000000); got != 0 {
		t.Errorf("with package and 0 additional_price, expected 0, got %.0f", got)
	}
}

func TestClientFormTemplateStandaloneModelRendering(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %v", err)
	}
	tmplPath := filepath.Join(wd, "..", "..", "web", "templates", "client_form.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		t.Fatalf("Failed to parse client_form.html: %v", err)
	}

	data := BookingDetailViewData{
		CompanyName:            "Dekorin Studio",
		ClientName:             "Rian & Nisa",
		EventDateFormatted:     "Sabtu, 20 November 2026",
		EventTimeFormatted:     "10:00 WIB",
		PackageName:            "Belum Dipilih",
		PackagePrice:           0,
		PackagePriceFormatted:  "Rp 0",
		HasPackage:             false,
		ModelName:              "Backdrop Arch Rustic",
		ModelWidth:             "3 Meter",
		ModelAddPrice:          2000000,
		ModelAddPriceFormatted: "Rp 2.000.000",
		TotalCost:              2000000,
		TotalCostFormatted:     "Rp 2.000.000",
		DpAmountFormatted:      "Rp 0",
		RemainingFormatted:     "Rp 2.000.000",
		FormToken:              "tok123",
		IsLocked:               true,
		ShowPrice:              true,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to execute client_form.html: %v", err)
	}
	htmlOut := buf.String()

	if strings.Contains(htmlOut, "Paket Dekorasi yang Dipilih") {
		t.Error("expected Paket Dekorasi card to be hidden when HasPackage is false")
	}
	if !strings.Contains(htmlOut, "Model Dekorasi (Backdrop Arch Rustic)") {
		t.Error("expected Model Dekorasi fee row in summary")
	}
	if strings.Contains(htmlOut, "Upgrade Model") {
		t.Error("did not expect 'Upgrade Model' label when HasPackage is false")
	}
}

func TestSuperadminTemplateExistsAndHasStandaloneModelHooks(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(wd, "..", "..", "web", "templates", "superadmin.html"))
	if err != nil {
		t.Fatalf("Failed to read superadmin.html: %v", err)
	}
	src := string(content)
	if !strings.Contains(src, "labelAgendaModelPrice") {
		t.Error("expected superadmin.html to contain labelAgendaModelPrice")
	}
	if !strings.Contains(src, "Harga Model Dekorasi (Rp)") {
		t.Error("expected superadmin.html to support 'Harga Model Dekorasi (Rp)' label when no package is chosen")
	}
}

func TestBuildAgendaPaymentStatusUpdatesPreservesSchedule(t *testing.T) {
	eventTime := time.Date(2026, 12, 20, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	now := time.Date(2026, 11, 10, 18, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	agenda := models.Agenda{
		ID:            "agd_pay_1",
		EventDateTime: eventTime,
		PackagePrice:  3000000,
		TransportFee:  250000,
		DpAmount:      1000000,
		PaymentStatus: "dp",
	}

	updates := buildAgendaPaymentStatusUpdates(agenda, "paid", now)
	if _, touchesSchedule := updates["event_date_time"]; touchesSchedule {
		t.Fatal("buildAgendaPaymentStatusUpdates must never touch event_date_time")
	}
	if updates["payment_status"] != "paid" {
		t.Errorf("expected payment_status 'paid', got %v", updates["payment_status"])
	}
	if updates["dp_amount"] != 3250000.0 {
		t.Errorf("expected dp_amount 3250000, got %v", updates["dp_amount"])
	}
	if updates["updated_at"] != now {
		t.Errorf("expected updated_at %v, got %v", now, updates["updated_at"])
	}
}

func TestBuildAgendaDpUpdatesPreservesSchedule(t *testing.T) {
	eventTime := time.Date(2026, 12, 20, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	now := time.Date(2026, 11, 10, 18, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	agenda := models.Agenda{
		ID:            "agd_dp_1",
		EventDateTime: eventTime,
		PackagePrice:  3000000,
		DpAmount:      0,
		PaymentStatus: "unpaid",
	}

	updates := buildAgendaDpUpdates(agenda, 3000000, now)
	if _, touchesSchedule := updates["event_date_time"]; touchesSchedule {
		t.Fatal("buildAgendaDpUpdates must never touch event_date_time")
	}
	if updates["payment_status"] != "paid" {
		t.Errorf("expected payment_status 'paid' when dp >= total, got %v", updates["payment_status"])
	}
	if updates["dp_amount"] != 3000000.0 {
		t.Errorf("expected dp_amount 3000000, got %v", updates["dp_amount"])
	}
}

func TestSuperadminHasMarkAgendaPaidAction(t *testing.T) {
	path := filepath.Join("..", "..", "web", "templates", "superadmin.html")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading superadmin.html: %v", err)
	}
	src := string(content)

	for _, marker := range []string{
		"btnMarkAgendaPaidInModal",
		"markCurrentAgendaAsPaid",
		"markAgendaAsPaid",
		"/payment-status",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("expected superadmin.html to contain %q", marker)
		}
	}
}
