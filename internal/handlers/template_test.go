package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dekorin_apps_api/internal/models"
)

func TestParseAllInvitationTemplates(t *testing.T) {
	// Find base directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	templatesDir := filepath.Join(wd, "..", "..", "web", "templates")
	templateFiles := []string{
		"template_pn4.html",
		"template_ta3.html",
		"template_ta17.html",
		"template_jawa3.html",
	}

	testData := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation: models.Invitation{
			ID:              "inv_test",
			Title:           "The Wedding of Romeo & Juliet",
			Slug:            "romeo-juliet",
			Template:        "green-leaves",
			EventType:       "wedding",
			Quotes:          "Dan di antara tanda-tanda kekuasaan-Nya...",
			GroomName:       "Romeo Montague",
			GroomNickname:   "Romeo",
			GroomChildOrder: "Putra pertama",
			GroomParents:    "Bpk. Lord Montague & Ibu Lady Montague",
			GroomInstagram:  "romeo",
			GroomPhoto:      "/assets/templates/floral/avatar.png",
			BrideName:       "Juliet Capulet",
			BrideNickname:   "Juliet",
			BrideChildOrder: "Putri kedua",
			BrideParents:    "Bpk. Lord Capulet & Ibu Lady Capulet",
			BrideInstagram:  "juliet",
			BridePhoto:      "/assets/templates/floral/avatar.png",
			ChildName:       "Ahmad Fauzan",
			ChildNickname:   "Fauzan",
			ChildOrder:      "Putra pertama",
			AkadDate:        "2026-11-20",
			AkadTime:        "08:00 - 10:00 WIB",
			AkadVenue:       "Masjid Agung",
			AkadAddress:     "Jl. Merdeka No. 1",
			AkadMapsURL:     "https://maps.google.com",
			ResepsiDate:     "2026-11-20",
			ResepsiTime:     "11:00 - 14:00 WIB",
			ResepsiVenue:    "Grand Ballroom",
			ResepsiAddress:  "Jl. Merdeka No. 2",
			ResepsiMapsURL:  "https://maps.google.com",
			BankName1:       "BCA",
			BankAccount1:    "1234567890",
			BankHolder1:     "Romeo",
			GiftAddress:     "Jl. Merdeka No. 1",
			AudioURL:        "",
			YoutubeURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			LoveStory:       "2020: Pertama kali bertemu...",
			TurutMengundang: "Keluarga Besar Montague\nKeluarga Besar Capulet",
			IsActive:        true,
		},
		GuestTo:             "Bpk. Budi & Keluarga",
		GalleryList:         []string{"/assets/templates/floral/wedding.png"},
		Wishes: []models.InvitationWish{
			{
				ID:         "wsh_1",
				GuestName:  "Andi",
				Attendance: "hadir",
				GuestCount: 2,
				Message:    "Selamat menempuh hidup baru!",
				CreatedAt:  time.Now(),
			},
		},
		YoutubeEmbedURL:     "https://www.youtube.com/embed/dQw4w9WgXcQ",
		TurutMengundangList: []string{"Keluarga Besar Montague", "Keluarga Besar Capulet"},
		OGTitle:             "The Wedding of Romeo & Juliet",
		OGDescription:       "20 November 2026",
		OGImage:             "https://dekorin.my.id/assets/dekorin_logo_square.png",
		OGURL:               "https://dekorin.my.id/romeo-juliet",
	}

	for _, file := range templateFiles {
		filePath := filepath.Join(templatesDir, file)
		t.Run(file, func(t *testing.T) {
			tmpl, err := template.ParseFiles(filePath)
			if err != nil {
				t.Fatalf("Failed to parse template %s: %v", file, err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, testData); err != nil {
				t.Fatalf("Failed to execute template %s: %v", file, err)
			}

			if buf.Len() == 0 {
				t.Fatalf("Template %s executed to empty buffer", file)
			}
		})
	}
}

func TestLoveStoryHiddenWhenEmpty(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	templatesDir := filepath.Join(wd, "..", "..", "web", "templates")
	targets := []string{"template_pn4.html", "template_ta3.html", "template_ta17.html", "template_jawa3.html"}

	testDataWithoutStory := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation: models.Invitation{
			ID:        "inv_no_story",
			Title:     "Pernikahan Tanpa Love Story",
			Slug:      "tanpa-story",
			EventType: "wedding",
			LoveStory: "", // EMPTY LOVE STORY
			IsActive:  true,
		},
		OGTitle: "Pernikahan Tanpa Love Story",
	}

	for _, file := range targets {
		t.Run(file, func(t *testing.T) {
			filePath := filepath.Join(templatesDir, file)
			tmpl, err := template.ParseFiles(filePath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", file, err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, testDataWithoutStory); err != nil {
				t.Fatalf("Failed to execute %s: %v", file, err)
			}

			rendered := buf.String()
			if bytes.Contains([]byte(rendered), []byte("href=\"#cerita\"")) {
				t.Errorf("%s still contains href=\"#cerita\" link in navbar when LoveStory is empty!", file)
			}
			if bytes.Contains([]byte(rendered), []byte("id=\"cerita\"")) {
				t.Errorf("%s still contains id=\"cerita\" section when LoveStory is empty!", file)
			}
		})
	}
}

func TestGuestToLockedWhenProvided(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	templatesDir := filepath.Join(wd, "..", "..", "web", "templates")
	allTemplates := []string{
		"template_pn4.html",
		"template_ta3.html",
		"template_ta17.html",
		"template_jawa3.html",
	}

	// 1. With GuestTo = "Soni"
	dataWithGuest := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation: models.Invitation{
			ID:        "inv_guest",
			Title:     "Pernikahan Test",
			Slug:      "test-guest",
			EventType: "wedding",
			IsActive:  true,
		},
		GuestTo:    "Soni",
		HasGuestTo: true,
		OGTitle:    "Pernikahan Test",
	}

	for _, file := range allTemplates {
		t.Run(file+"/WithGuest", func(t *testing.T) {
			filePath := filepath.Join(templatesDir, file)
			tmpl, err := template.ParseFiles(filePath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", file, err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, dataWithGuest); err != nil {
				t.Fatalf("Failed to execute %s: %v", file, err)
			}

			rendered := buf.String()
			if !bytes.Contains([]byte(rendered), []byte("value=\"Soni\"")) {
				t.Errorf("%s does not contain value=\"Soni\"!", file)
			}
			if !bytes.Contains([]byte(rendered), []byte("readonly")) {
				t.Errorf("%s does not contain readonly for locked guest name!", file)
			}
		})
	}

	// 2. Without Guest (GuestTo = "Tamu Undangan")
	dataWithoutGuest := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation: models.Invitation{
			ID:        "inv_no_guest",
			Title:     "Pernikahan Test",
			Slug:      "test-no-guest",
			EventType: "wedding",
			IsActive:  true,
		},
		GuestTo:    "Tamu Undangan",
		HasGuestTo: false,
		OGTitle:    "Pernikahan Test",
	}

	for _, file := range allTemplates {
		t.Run(file+"/WithoutGuest", func(t *testing.T) {
			filePath := filepath.Join(templatesDir, file)
			tmpl, err := template.ParseFiles(filePath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", file, err)
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, dataWithoutGuest); err != nil {
				t.Fatalf("Failed to execute %s: %v", file, err)
			}

			rendered := buf.String()
			if !bytes.Contains([]byte(rendered), []byte("placeholder=\"Nama")) {
				t.Errorf("%s missing editable placeholder when guest is not locked!", file)
			}
		})
	}
}

func TestInvitationPortalHelpers(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	portalPath := filepath.Join(wd, "..", "..", "web", "templates", "invitation_portal.html")
	info, err := os.Stat(portalPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("invitation_portal.html missing or empty: %v", err)
	}

	tok := generateEditToken()
	if len(tok) < 32 {
		t.Fatalf("generateEditToken entropy too short: %s", tok)
	}

	pin := generateEditPin()
	if len(pin) != 6 {
		t.Fatalf("generateEditPin expected 6 digits, got: %s", pin)
	}

	cases := map[string]string{
		"Bagas Zulia":                                      "bagas-zulia",
		"https://undangan.dekorin.my.id/bagas-zulia?to=AB": "bagas-zulia",
		"https://dekorin.my.id/invitation/romeo_juliet/":   "romeo-juliet",
	}
	for input, expected := range cases {
		got := sanitizeInvitationSlug(input)
		if got != expected {
			t.Errorf("sanitizeInvitationSlug(%q) = %q, expected %q", input, got, expected)
		}
	}

	if !isReservedInvitationSlug("kelola") || !isReservedInvitationSlug("api") {
		t.Errorf("Expected reserved slugs to be blocked")
	}
}

func TestSakuraBlossomCoverReplacement(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	ta3Path := filepath.Join(wd, "..", "..", "web", "templates", "template_ta3.html")
	tmpl, err := template.ParseFiles(ta3Path)
	if err != nil {
		t.Fatalf("Failed to parse template_ta3.html: %v", err)
	}

	type TemplateData struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}

	// Case 1: Without CoverImage -> uses couple-hero.png
	dataNoCover := TemplateData{
		Invitation: models.Invitation{
			ID:         "inv_no_cover",
			Title:      "Umay & Irfan",
			Slug:       "umay-irfan",
			EventType:  "wedding",
			CoverImage: "",
			IsActive:   true,
		},
		GuestTo: "Soni",
	}

	var bufNoCover bytes.Buffer
	if err := tmpl.Execute(&bufNoCover, dataNoCover); err != nil {
		t.Fatalf("Failed to render without cover: %v", err)
	}
	renderedNoCover := bufNoCover.String()
	if !strings.Contains(renderedNoCover, "/assets/templates/ta3/couple-hero.png") {
		t.Errorf("Expected couple-hero.png when CoverImage is empty!")
	}

	// Case 2: With Custom CoverImage -> replaces couple-hero.png in Section 1
	dataWithCover := TemplateData{
		Invitation: models.Invitation{
			ID:         "inv_with_cover",
			Title:      "Umay & Irfan",
			Slug:       "umay-irfan",
			EventType:  "wedding",
			CoverImage: "https://dekorin.my.id/uploads/custom-couple-cover.jpg",
			IsActive:   true,
		},
		GuestTo: "Soni",
	}

	var bufWithCover bytes.Buffer
	if err := tmpl.Execute(&bufWithCover, dataWithCover); err != nil {
		t.Fatalf("Failed to render with cover: %v", err)
	}
	renderedWithCover := bufWithCover.String()
	if !strings.Contains(renderedWithCover, "https://dekorin.my.id/uploads/custom-couple-cover.jpg") {
		t.Errorf("Expected custom cover photo to be rendered!")
	}
	// Specifically check section #COVER
	coverSectionIdx := strings.Index(renderedWithCover, `id="COVER"`)
	if coverSectionIdx == -1 {
		t.Fatalf("Could not find section id=\"COVER\" in rendered template")
	}
	subSlice := renderedWithCover[coverSectionIdx : coverSectionIdx+1500]
	if !strings.Contains(subSlice, "https://dekorin.my.id/uploads/custom-couple-cover.jpg") {
		t.Errorf("Section #COVER should contain the custom cover photo!")
	}
	if strings.Contains(subSlice, "couple-hero.png") {
		t.Errorf("Section #COVER should NOT contain couple-hero.png when custom cover is provided!")
	}
}

func TestSuperadminDefaultValues(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	superadminPath := filepath.Join(wd, "..", "..", "web", "templates", "superadmin.html")
	content, err := os.ReadFile(superadminPath)
	if err != nil {
		t.Fatalf("superadmin.html missing or unreadable: %v", err)
	}
	html := string(content)

	// Ensure openCreateInvitationModal sets pre-filled defaults
	requiredSubstrings := []string{
		"The Wedding of Bagas & Zulia",
		"Bagas Pratama, S.Kom.",
		"Zulia Maharani, S.Pd.",
		"Masjid Agung Al-Barkah",
		"Grand Ballroom Hotel Santika",
		"LeniArdi15.jpeg",
		"LeniArdi16.jpeg",
		"1 September 2021",
	}

	for _, sub := range requiredSubstrings {
		if !strings.Contains(html, sub) {
			t.Errorf("superadmin.html expected to contain default value %q in openCreateInvitationModal()", sub)
		}
	}
}

func TestSuperadminRoleAccessAndTeamManagement(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	superadminPath := filepath.Join(wd, "..", "..", "web", "templates", "superadmin.html")
	content, err := os.ReadFile(superadminPath)
	if err != nil {
		t.Fatalf("superadmin.html missing or unreadable: %v", err)
	}
	html := string(content)

	roleFeatures := []string{
		"id=\"nav-role-permissions\"",
		"id=\"viewRolePermissions\"",
		"id=\"viewUsers\"",
		"id=\"viewAddons\"",
		"id=\"modalAddUser\"",
		"id=\"modalEditUser\"",
		"id=\"modalResetUserPassword\"",
		"id=\"modalAddon\"",
		"loadRolePermissions()",
		"saveRolePermissions()",
		"confirmResetRolePermissions()",
		"selectRoleForPermissions",
		"userSummaryTotal",
		"userSummaryOwner",
		"userSummaryAdmin",
		"userSummaryCrew",
		"can(permissionKey)",
	}

	for _, elem := range roleFeatures {
		if !strings.Contains(html, elem) {
			t.Errorf("superadmin.html expected to contain role access / team feature %q", elem)
		}
	}
}

func TestBankLogoBriResolution(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	targets := []string{"template_ta3.html", "template_ta17.html", "template_pn4.html", "template_jawa3.html"}
	for _, target := range targets {
		p := filepath.Join(wd, "..", "..", "web", "templates", target)
		content, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s missing or unreadable: %v", target, err)
		}
		html := string(content)

		if !strings.Contains(html, "logo_bri.png") {
			t.Errorf("%s should contain logo_bri.png in bank resolver", target)
		}
		if !strings.Contains(html, "bri") {
			t.Errorf("%s should check for bri bank name", target)
		}
	}
}

func TestGiftSectionHiddenWhenBankAccountsEmpty(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	templatesDir := filepath.Join(wd, "..", "..", "web", "templates")
	targets := []string{
		"template_pn4.html",
		"template_ta3.html",
		"template_ta17.html",
		"template_jawa3.html",
	}

	testDataWithoutGift := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation: models.Invitation{
			ID:           "inv_empty_bank",
			Title:        "Pernikahan Tanpa Rekening",
			Slug:         "tanpa-rekening",
			EventType:    "wedding",
			BankName1:    "",
			BankAccount1: "",
			BankHolder1:  "",
			BankName2:    "",
			BankAccount2: "",
			BankHolder2:  "",
			GiftAddress:  "",
			IsActive:     true,
		},
		OGTitle: "Pernikahan Tanpa Rekening",
	}

	testDataWithGift := testDataWithoutGift
	testDataWithGift.BankAccount1 = "987654321"
	testDataWithGift.BankName1 = "BCA"
	testDataWithGift.BankHolder1 = "Romeo"

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			filePath := filepath.Join(templatesDir, target)
			tmpl, err := template.ParseFiles(filePath)
			if err != nil {
				t.Fatalf("Failed to parse template %s: %v", target, err)
			}

			// 1. Without gift: should NOT contain bank numbers or gift sections
			var bufWithout bytes.Buffer
			if err := tmpl.Execute(&bufWithout, testDataWithoutGift); err != nil {
				t.Fatalf("Failed to execute template %s without gift: %v", target, err)
			}
			renderedWithout := bufWithout.String()

			// Check that demo accounts are never shown
			if strings.Contains(renderedWithout, "8630245908") {
				t.Errorf("%s leaked demo BNI account 8630245908 when bank accounts are empty", target)
			}
			if strings.Contains(renderedWithout, "1234XXXX") {
				t.Errorf("%s leaked demo account 1234XXXX when bank accounts are empty", target)
			}
			if strings.Contains(renderedWithout, "Shashita Vinskalery") {
				t.Errorf("%s leaked demo name Shashita Vinskalery when bank accounts are empty", target)
			}

			// Specific section & navbar checks per template
			switch target {
			case "template_ta3.html", "template_ta17.html":
				if strings.Contains(renderedWithout, `<section id="gift">`) {
					t.Errorf("%s rendered <section id=\"gift\"> when bank accounts are empty", target)
				}
				if strings.Contains(renderedWithout, `href="#gift"`) {
					t.Errorf("%s rendered navbar link #gift when bank accounts are empty", target)
				}
			case "template_pn4.html", "template_jawa3.html":
				if strings.Contains(renderedWithout, `<section id="gift"`) {
					t.Errorf("%s rendered <section id=\"gift\" when bank accounts are empty", target)
				}
				if strings.Contains(renderedWithout, `href="#gift"`) {
					t.Errorf("%s rendered navbar link #gift when bank accounts are empty", target)
				}
			}

			// 2. With gift: should contain BankAccount1
			var bufWith bytes.Buffer
			if err := tmpl.Execute(&bufWith, testDataWithGift); err != nil {
				t.Fatalf("Failed to execute template %s with gift: %v", target, err)
			}
			renderedWith := bufWith.String()
			if !strings.Contains(renderedWith, "987654321") {
				t.Errorf("%s did NOT render BankAccount1 (987654321) when provided", target)
			}
		})
	}
}

func TestSampleInvitationForTemplate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	templates := []struct {
		id       string
		fileName string
	}{
		{"ta17", "template_ta17.html"},
		{"jawa3", "template_jawa3.html"},
		{"ta3", "template_ta3.html"},
		{"pn4", "template_pn4.html"},
	}

	for _, tc := range templates {
		t.Run(tc.id, func(t *testing.T) {
			inv, wishes := getSampleInvitationForTemplate(tc.id)
			if inv.Title == "" {
				t.Errorf("Title is empty for template %s", tc.id)
			}
			if inv.GroomName == "" || inv.BrideName == "" {
				t.Errorf("Names are empty for template %s", tc.id)
			}
			if len(wishes) == 0 {
				t.Errorf("Wishes list is empty for template %s", tc.id)
			}

			tmplPath := filepath.Join(wd, "..", "..", "web", "templates", tc.fileName)
			tmpl, err := template.ParseFiles(tmplPath)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", tc.fileName, err)
			}

			data := struct {
				models.Invitation
				GuestTo             string
				HasGuestTo          bool
				GalleryList         []string
				Wishes              []models.InvitationWish
				YoutubeEmbedURL     string
				TurutMengundangList []string
				OGTitle             string
				OGDescription       string
				OGImage             string
				OGURL               string
			}{
				Invitation:          inv,
				GuestTo:             "Tamu Undangan",
				HasGuestTo:          true,
				GalleryList:         strings.Split(inv.GalleryImages, ","),
				Wishes:              wishes,
				YoutubeEmbedURL:     "",
				TurutMengundangList: []string{"Keluarga Besar"},
				OGTitle:             inv.Title,
				OGDescription:       inv.AkadDate,
				OGImage:             inv.CoverImage,
				OGURL:               "https://undangan.dekorin.my.id/demo",
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("Failed to execute template %s with sample data: %v", tc.fileName, err)
			}
			html := buf.String()
			if !strings.Contains(html, inv.GroomNickname) && !strings.Contains(html, inv.GroomName) {
				t.Errorf("Groom name not rendered in %s", tc.fileName)
			}
		})
	}
}

// TestJawa3UserProvidedPhotos memverifikasi template_jawa3 mengambil foto pengantin dan galeri dari data yang diisi user
func TestJawa3UserProvidedPhotos(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tmplPath := filepath.Join(wd, "..", "..", "web", "templates", "template_jawa3.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		t.Fatalf("Failed to parse template_jawa3.html: %v", err)
	}

	// 1. Uji dengan foto yang diisi oleh user
	userBridePhoto := "https://dekorin.my.id/uploads/foto_pengantin_wanita.jpg"
	userGroomPhoto := "https://dekorin.my.id/uploads/foto_pengantin_pria.jpg"
	userGallery := []string{
		"https://dekorin.my.id/uploads/galeri_user_1.jpg",
		"https://dekorin.my.id/uploads/galeri_user_2.jpg",
		"https://dekorin.my.id/uploads/galeri_user_3.jpg",
	}

	dataWithUserPhotos := struct {
		models.Invitation
		GuestTo             string
		HasGuestTo          bool
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
		OGTitle             string
		OGDescription       string
		OGImage             string
		OGURL               string
	}{
		Invitation: models.Invitation{
			ID:             "inv_user_photos",
			Slug:           "user-wedding",
			Title:          "The Wedding of Yogi & Ratna",
			Template:       "jawa3",
			EventType:      "wedding",
			BrideName:      "Ratna Permata, S.Pd.",
			BrideNickname:  "Ratna",
			BridePhoto:     userBridePhoto,
			GroomName:      "Yogi Pratama, S.T.",
			GroomNickname:  "Yogi",
			GroomPhoto:     userGroomPhoto,
			AkadDate:       "Sabtu, 20 Desember 2026",
			ResepsiDate:    "Sabtu, 20 Desember 2026",
		},
		GuestTo:     "Sahabat Tercinta",
		HasGuestTo:  true,
		GalleryList: userGallery,
	}

	var bufWith bytes.Buffer
	if err := tmpl.Execute(&bufWith, dataWithUserPhotos); err != nil {
		t.Fatalf("Failed to execute template_jawa3.html with user photos: %v", err)
	}

	rendered := bufWith.String()

	// Pastikan foto mempelai wanita & pria menggunakan input dari user
	if !strings.Contains(rendered, userBridePhoto) {
		t.Errorf("template_jawa3.html did NOT render user bride photo: %s", userBridePhoto)
	}
	if !strings.Contains(rendered, userGroomPhoto) {
		t.Errorf("template_jawa3.html did NOT render user groom photo: %s", userGroomPhoto)
	}

	// Pastikan foto default orang lain TIDAK muncul
	if strings.Contains(rendered, "bride-default.jpg") {
		t.Errorf("template_jawa3.html rendered bride-default.jpg when user provided their own photo")
	}
	if strings.Contains(rendered, "groom-default.jpg") {
		t.Errorf("template_jawa3.html rendered groom-default.jpg when user provided their own photo")
	}

	// Pastikan semua foto galeri yang diisi user ditampilkan
	for _, gImg := range userGallery {
		if !strings.Contains(rendered, gImg) {
			t.Errorf("template_jawa3.html did NOT render user gallery image: %s", gImg)
		}
	}

	// Pastikan foto galeri dummy bawaan (gallery1 s/d gallery8) TIDAK muncul di dalam #galeri
	galeriStart := strings.Index(rendered, `<section id="galeri">`)
	if galeriStart != -1 {
		galeriHTML := rendered[galeriStart:]
		if galeriEndIdx := strings.Index(galeriHTML, `</section>`); galeriEndIdx != -1 {
			galeriHTML = galeriHTML[:galeriEndIdx]
		}
		for i := 1; i <= 8; i++ {
			dummyName := fmt.Sprintf("gallery%d.jpg", i)
			if strings.Contains(galeriHTML, dummyName) {
				t.Errorf("template_jawa3.html #galeri rendered dummy reference photo %s when user provided their own gallery", dummyName)
			}
		}
	}

	// 2. Uji jika galeri kosong: section galeri tidak boleh menampilkan foto orang lain
	dataEmptyGallery := dataWithUserPhotos
	dataEmptyGallery.GalleryList = nil

	var bufEmpty bytes.Buffer
	if err := tmpl.Execute(&bufEmpty, dataEmptyGallery); err != nil {
		t.Fatalf("Failed to execute template_jawa3.html with empty gallery: %v", err)
	}

	renderedEmpty := bufEmpty.String()
	if strings.Contains(renderedEmpty, `<section id="galeri"`) {
		t.Errorf("template_jawa3.html rendered <section id=\"galeri\"> when GalleryList is empty")
	}
}

func TestJawa3CoverImageLocked(t *testing.T) {
	tmplPath := filepath.Join("..", "..", "web", "templates", "template_jawa3.html")
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Fatalf("Failed to read template_jawa3.html: %v", err)
	}

	// Cover background CSS harus mengarah ke /assets/templates/jawa3/bg-cover.jpg
	if !strings.Contains(string(content), "url('/assets/templates/jawa3/bg-cover.jpg')") {
		t.Errorf("template_jawa3.html does not lock cover background to /assets/templates/jawa3/bg-cover.jpg")
	}

	sampleInv, _ := getSampleInvitationForTemplate("jawa3")
	if sampleInv.CoverImage != "/assets/templates/jawa3/bg-cover.jpg" {
		t.Errorf("expected sample jawa3 cover image to be /assets/templates/jawa3/bg-cover.jpg, got: %s", sampleInv.CoverImage)
	}
}
