package middleware

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JwtSecret adalah secret key untuk enkripsi dan verifikasi JWT token
var JwtSecret = initJwtSecret()

func initJwtSecret() []byte {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret != "" {
		if len(secret) < 32 {
			log.Println("⚠️ [SECURITY WARNING] JWT_SECRET is shorter than 32 characters. Consider using a 256-bit secret.")
		}
		return []byte(secret)
	}
	log.Println("⚠️ [SECURITY WARNING] JWT_SECRET is not set in environment! Using fallback key. Please configure JWT_SECRET in production.")
	return []byte("super-secret-key-dekorin-123")
}

// Protected memvalidasi Bearer JWT token dari header Authorization
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// Coba ambil dari cookie jika ada (untuk web superadmin)
			cookieToken := c.Cookies("superadmin_token")
			if cookieToken != "" {
				authHeader = "Bearer " + cookieToken
			}
		}

		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Akses ditolak: Token autentikasi tidak ditemukan",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Format Authorization header tidak valid (Harus: Bearer <token>)",
			})
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Sesi login tidak valid atau sudah kedaluwarsa. Silakan login kembali.",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Klaim token tidak valid",
			})
		}

		c.Locals("user_id", claims["id"])
		c.Locals("company_id", claims["company_id"])
		c.Locals("user_role", claims["role"])
		c.Locals("user_phone", claims["phone"])
		c.Locals("user_email", claims["email"])

		return c.Next()
	}
}

// OptionalProtected mencoba membaca token JWT jika ada, tanpa memblokir request jika tidak ada
func OptionalProtected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			cookieToken := c.Cookies("superadmin_token")
			if cookieToken != "" {
				authHeader = "Bearer " + cookieToken
			}
		}

		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString := parts[1]
				token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fiber.ErrUnauthorized
					}
					return JwtSecret, nil
				})

				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						c.Locals("user_id", claims["id"])
						c.Locals("company_id", claims["company_id"])
						c.Locals("user_role", claims["role"])
						c.Locals("user_phone", claims["phone"])
						c.Locals("user_email", claims["email"])
					}
				}
			}
		}

		return c.Next()
	}
}

// RequireRole memeriksa apakah user yang login memiliki salah satu role yang diizinkan
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("user_role")
		if roleVal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Akses ditolak",
			})
		}

		userRole, _ := roleVal.(string)
		// Superadmin selalu memiliki akses penuh
		if strings.EqualFold(userRole, "superadmin") {
			return c.Next()
		}

		for _, r := range allowedRoles {
			if strings.EqualFold(r, userRole) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Anda tidak memiliki hak akses untuk tindakan ini",
		})
	}
}

// RequireSuperadmin memastikan hanya role superadmin yang dapat mengakses endpoint
func RequireSuperadmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("user_role")
		if roleVal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Akses ditolak: Hanya Superadmin",
			})
		}

		userRole, _ := roleVal.(string)
		if !strings.EqualFold(userRole, "superadmin") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Akses khusus Superadmin Platform Dekorin",
			})
		}

		return c.Next()
	}
}

// GetCompanyID mengambil CompanyID user yang sedang login dari konteks
func GetCompanyID(c *fiber.Ctx) string {
	cid, _ := c.Locals("company_id").(string)
	return strings.TrimSpace(cid)
}

// GetUserID mengambil UserID user yang sedang login dari konteks
func GetUserID(c *fiber.Ctx) string {
	uid, _ := c.Locals("user_id").(string)
	return strings.TrimSpace(uid)
}

// RequireVendor membatasi akses hanya untuk akun internal vendor/tim (owner, admin, crew, superadmin)
// dan memblokir akun calon pengantin ('couple') agar data internal vendor tetap terisolasi.
func RequireVendor() fiber.Handler {
	return RequireRole("superadmin", "owner", "admin", "crew", "staff")
}

// RequireCouple membatasi akses hanya untuk akun Calon Pengantin ('couple') dan Superadmin.
func RequireCouple() fiber.Handler {
	return RequireRole("couple")
}
