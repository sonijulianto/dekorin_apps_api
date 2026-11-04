package utils

import (
	"sync"
	"time"

	"dekorin_apps_api/internal/models"
)

type CachedInvitationItem struct {
	Invitation models.Invitation
	Wishes     []models.InvitationWish
	ExpiresAt  time.Time
}

// InvitationCache menyediakan in-memory caching untuk data website undangan publik
// guna mencegah ratusan query ke database PostgreSQL saat pengantin menyebar link di WhatsApp
type InvitationCache struct {
	mu    sync.RWMutex
	items map[string]CachedInvitationItem
	ttl   time.Duration
}

// GlobalInvitationCache instance global dengan TTL 5 menit
var GlobalInvitationCache = NewInvitationCache(5 * time.Minute)

func NewInvitationCache(ttl time.Duration) *InvitationCache {
	c := &InvitationCache{
		items: make(map[string]CachedInvitationItem),
		ttl:   ttl,
	}

	// Pembersihan background berkala setiap 10 menit
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			c.cleanup()
		}
	}()

	return c
}

// Get mengambil data undangan dari cache jika belum kedaluwarsa
func (c *InvitationCache) Get(slug string) (*models.Invitation, []models.InvitationWish, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[slug]
	if !exists || time.Now().After(item.ExpiresAt) {
		return nil, nil, false
	}
	return &item.Invitation, item.Wishes, true
}

// Set menyimpan data undangan ke cache
func (c *InvitationCache) Set(slug string, inv models.Invitation, wishes []models.InvitationWish) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[slug] = CachedInvitationItem{
		Invitation: inv,
		Wishes:     wishes,
		ExpiresAt:  time.Now().Add(c.ttl),
	}
}

// Invalidate menghapus cache suatu undangan saat ada perubahan data atau ucapan baru
func (c *InvitationCache) Invalidate(slug string) {
	if slug == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, slug)
}

// InvalidateAll mengosongkan seluruh cache
func (c *InvitationCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]CachedInvitationItem)
}

func (c *InvitationCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for slug, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, slug)
		}
	}
}
