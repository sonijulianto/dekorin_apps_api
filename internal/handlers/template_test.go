package handlers

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
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
		"template_creamy_latte.html",
		"template_floral.html",
		"template_green_leaves.html",
		"template_khitan_green.html",
		"template_luxury_grey.html",
		"wedding_invitation.html",
	}

	testData := struct {
		models.Invitation
		GuestTo             string
		GalleryList         []string
		Wishes              []models.InvitationWish
		YoutubeEmbedURL     string
		TurutMengundangList []string
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
