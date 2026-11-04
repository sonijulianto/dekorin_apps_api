package utils

import (
	"testing"
	"time"

	"dekorin_apps_api/internal/models"
)

func TestInvitationCache_GetSetInvalidate(t *testing.T) {
	cache := NewInvitationCache(100 * time.Millisecond)

	inv := models.Invitation{
		ID:    "inv_1",
		Slug:  "test-wedding",
		Title: "The Wedding of Test",
	}
	wishes := []models.InvitationWish{
		{ID: "wsh_1", GuestName: "Budi", Message: "Selamat!"},
	}

	// 1. Get saat belum di-set
	_, _, found := cache.Get("test-wedding")
	if found {
		t.Error("Seharusnya cache kosong untuk test-wedding")
	}

	// 2. Set dan Get
	cache.Set("test-wedding", inv, wishes)
	cachedInv, cachedWishes, found := cache.Get("test-wedding")
	if !found {
		t.Fatal("Seharusnya cache ditemukan setelah di-set")
	}
	if cachedInv.Title != "The Wedding of Test" {
		t.Errorf("Title salah: dapat %s", cachedInv.Title)
	}
	if len(cachedWishes) != 1 || cachedWishes[0].GuestName != "Budi" {
		t.Errorf("Wishes tidak sesuai")
	}

	// 3. Invalidate
	cache.Invalidate("test-wedding")
	_, _, found = cache.Get("test-wedding")
	if found {
		t.Error("Seharusnya cache kosong setelah di-invalidate")
	}

	// 4. Test TTL Expiration
	cache.Set("test-wedding", inv, wishes)
	time.Sleep(120 * time.Millisecond)
	_, _, found = cache.Get("test-wedding")
	if found {
		t.Error("Seharusnya cache sudah kedaluwarsa setelah melewati TTL")
	}
}
