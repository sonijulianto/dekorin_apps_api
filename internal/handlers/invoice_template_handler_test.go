package handlers

import (
	"testing"

	"dekorin_apps_api/internal/models"
)

func TestNormalizeAndEffectiveInvoiceTemplate(t *testing.T) {
	if tmpl, ok := NormalizeInvoiceTemplate("template_modern_navy"); !ok || tmpl != InvoiceTemplateModernNavy {
		t.Errorf("expected template_modern_navy valid, got %q (%v)", tmpl, ok)
	}
	if tmpl, ok := NormalizeInvoiceTemplate("  template_minimalist_grey "); !ok || tmpl != InvoiceTemplateMinimalistGrey {
		t.Errorf("expected template_minimalist_grey valid after trim, got %q (%v)", tmpl, ok)
	}
	if tmpl, ok := NormalizeInvoiceTemplate("invalid_template"); ok || tmpl != DefaultInvoiceTemplate {
		t.Errorf("expected invalid template to fallback to default %q, got %q (%v)", DefaultInvoiceTemplate, tmpl, ok)
	}
	if eff := EffectiveInvoiceTemplate(""); eff != DefaultInvoiceTemplate {
		t.Errorf("expected empty effective template to return %q, got %q", DefaultInvoiceTemplate, eff)
	}
}

func TestResolveBrandInvoiceTemplateAndSummary(t *testing.T) {
	brands := []models.CompanyBrand{
		{ID: "brd_1", Name: "Brand Inherited", InvoiceTemplate: ""},
		{ID: "brd_2", Name: "Brand Custom Navy", InvoiceTemplate: InvoiceTemplateModernNavy},
		{ID: "brd_3", Name: "Brand Custom Grey", InvoiceTemplate: InvoiceTemplateMinimalistGrey},
	}

	// Case 1: Main brand uses minimalist grey
	summary, inherited := buildBrandTemplatesSummary(InvoiceTemplateMinimalistGrey, brands)

	if summary[""] != InvoiceTemplateMinimalistGrey {
		t.Errorf("expected main brand template %q, got %q", InvoiceTemplateMinimalistGrey, summary[""])
	}
	if summary["brd_1"] != InvoiceTemplateMinimalistGrey {
		t.Errorf("expected brd_1 to inherit %q, got %q", InvoiceTemplateMinimalistGrey, summary["brd_1"])
	}
	if summary["brd_2"] != InvoiceTemplateModernNavy {
		t.Errorf("expected brd_2 explicit override %q, got %q", InvoiceTemplateModernNavy, summary["brd_2"])
	}
	if summary["brd_3"] != InvoiceTemplateMinimalistGrey {
		t.Errorf("expected brd_3 explicit %q, got %q", InvoiceTemplateMinimalistGrey, summary["brd_3"])
	}
	if len(inherited) != 1 || inherited[0] != "brd_1" {
		t.Errorf("expected inherited list [brd_1], got %v", inherited)
	}

	// Case 2: Main brand empty falls back to default modern navy
	eff, isInherited := resolveBrandInvoiceTemplate("", "")
	if eff != DefaultInvoiceTemplate || !isInherited {
		t.Errorf("expected fallback default inherited=true, got %q (%v)", eff, isInherited)
	}
}
