package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"hvac-system/internal/adapter/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pocketbase/pocketbase/core"
)

// LicenseSecretKey is the digital seal for signing and validating tokens
const LicenseSecretKey = "HVAC_SECURE_V1_@992834_DIGITAL_SEAL_X"

// SettingsMiddleware loads global settings into the request context
// and enforces license expiration logic using JWT Digital Seal.
func SettingsMiddleware(settingsRepo *repository.SettingsRepo) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// 1. Fetch Settings from Database
		settings, err := settingsRepo.GetSettings()
		if err != nil {
			fmt.Printf("Middleware Warning: Failed to load settings: %v\n", err)
		}

		// 2. [DIGITAL SEAL] Validate License JWT
		isValid := false
		var expiryDate time.Time

		if settings.LicenseKey != "" {
			// Parse and Validate Token
			token, err := jwt.Parse(settings.LicenseKey, func(token *jwt.Token) (interface{}, error) {
				// Validate the alg is what we expect: HMAC
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(LicenseSecretKey), nil
			})

			// Check validity and expiration
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if exp, ok := claims["exp"].(float64); ok {
						expiryDate = time.Unix(int64(exp), 0)
						// Check if expired
						if time.Now().Before(expiryDate) {
							isValid = true
							// [SYNC] Override the display date with the TRUTH from the Token
							// This ensures UI always shows the actual token expiry, ignoring DB garbage
							settings.ExpiryDate = expiryDate.Format("2006-01-02")
						}
					}
				}
			} else {
				fmt.Printf("❌ Digital Seal Invalid: %v\n", err)
				fmt.Printf("   -> Key (Len=%d): %q\n", len(settings.LicenseKey), settings.LicenseKey)
			}
		}

		// 3. Store in Context
		e.Set("Settings", settings)

		if settings.Logo != "" {
			logoUrl := fmt.Sprintf("/api/files/settings/%s/%s?thumb=200x0", settings.Id, settings.Logo)
			e.Set("LogoUrl", logoUrl)
		}

		// 4. [ENFORCEMENT] Block access if license is invalid
		path := e.Request.URL.Path

		// Whitelist checks
		isStatic := strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/favicon.ico")
		isLogin := strings.HasPrefix(path, "/login") || strings.HasPrefix(path, "/admin/login") || strings.HasPrefix(path, "/tech/login")

		// Allow accessing Settings page to input new key
		isSettingsPage := strings.HasPrefix(path, "/admin/settings")

		// Also allow internal PocketBase admin UI (protected separately) or maybe block it too?
		// User said: "Chỉ cho phép truy cập ... /admin/settings, /login, /assets/*"
		// So strictly, /_/ should be blocked if license is invalid, unless it falls under "login"?
		// I will block /_/ if license is invalid for safety, forcing them to use /admin/settings to fix it.
		// Wait, /admin/login is the custom login. /_/ is PB admin.

		if !isValid {
			if !isStatic && !isLogin && !isSettingsPage {
				// Render 403 Page
				html := `
				<!DOCTYPE html>
				<html lang="vi">
				<head>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<title>Giấy phép không hợp lệ</title>
					<style>
						body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; background: #eaeff2; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
						.card { background: white; padding: 40px; border-radius: 16px; box-shadow: 0 10px 25px rgba(0,0,0,0.1); text-align: center; max-width: 480px; width: 90%; }
						.icon { font-size: 64px; margin-bottom: 20px; }
						h1 { color: #d63384; margin: 0 0 10px; font-size: 24px; }
						p { color: #526b7a; line-height: 1.6; margin-bottom: 30px; }
						.btn { display: inline-block; background: #206bc4; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; transition: background 0.3s; }
						.btn:hover { background: #1a569d; }
						.meta { font-size: 12px; color: #9aa5b1; margin-top: 20px; }
					</style>
				</head>
				<body>
					<div class="card">
						<div class="icon">🛡️</div>
						<h1>Digital Seal Alert</h1>
						<p>Giấy phép sử dụng phần mềm của bạn <strong>không hợp lệ</strong> hoặc <strong>đã hết hạn</strong>.<br>Vui lòng liên hệ PHẠM ĐẠI 0335942538 để kích hoạt giấy phép.</p>
						<a href="/admin/settings" class="btn">Nhập Mã Kích Hoạt Mới</a>
						<div class="meta">Server ID: HVAC-REQ-NONCE • Contact Support</div>
					</div>
				</body>
				</html>
				`
				return e.HTML(http.StatusForbidden, html)
			}
		}

		return e.Next()
	}
}
