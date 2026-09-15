package handlers

import (
    "github.com/gofiber/fiber/v2"
    "dekorin_apps_api/internal/config"
    "dekorin_apps_api/internal/models"
)

// GetAllUsers handles GET /api/users and returns a list of all users (excluding passwords)
func GetAllUsers(c *fiber.Ctx) error {
    var users []models.User
    // Fetch all users from DB
    if err := config.DB.Find(&users).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data user"})
    }

    // Convert to response struct without password field
    type userResp struct {
        ID       string `json:"id"`
        FullName string `json:"full_name"`
        Email    string `json:"email"`
        CreatedAt string `json:"created_at"`
        UpdatedAt string `json:"updated_at"`
    }
    var resp []userResp
    for _, u := range users {
        resp = append(resp, userResp{
            ID:       u.ID,
            FullName: u.FullName,
            Email:    u.Email,
            CreatedAt: u.CreatedAt.Format("2006-01-02 15:04:05"),
            UpdatedAt: u.UpdatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return c.Status(fiber.StatusOK).JSON(resp)
}
