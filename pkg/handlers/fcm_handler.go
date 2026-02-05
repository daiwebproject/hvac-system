package handlers

import (
	"context"
	"fmt"

	domain "hvac-system/internal/core"
	"hvac-system/pkg/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// FCMHandler manages FCM token registration and notifications
type FCMHandler struct {
	App          *pocketbase.PocketBase
	FCMService   *services.FCMService
	SettingsRepo domain.SettingsRepository // [NEW] Injected
}

// RegisterDeviceTokenRequest represents the request to register a device token
type RegisterDeviceTokenRequest struct {
	Token string `json:"token"`
}

// RegisterDeviceToken registers or updates FCM token for a technician
// POST /api/tech/register-fcm-token
func (h *FCMHandler) RegisterDeviceToken(e *core.RequestEvent) error {
	fmt.Println("👉 [DEBUG] RegisterDeviceToken Request Received") // [DEBUG]

	var req RegisterDeviceTokenRequest
	if err := e.BindBody(&req); err != nil {
		fmt.Printf("❌ [DEBUG] BindBody Error: %v\n", err)
		return e.JSON(400, map[string]string{"error": "Invalid request"})
	}
	fmt.Printf("👉 [DEBUG] Token Received: %s\n", req.Token) // [DEBUG]

	if req.Token == "" {
		return e.JSON(400, map[string]string{"error": "Token is required"})
	}

	// 1. Handle Admin (Auth Check)
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	if authRecord.Collection().Name == "_superusers" { // PocketBase Admin Collection
		if h.FCMService == nil {
			return e.JSON(503, map[string]string{"error": "FCM not configured"})
		}
		// Subscribe Admin to 'admin_alerts' topic (Legacy/Fallback)
		err := h.FCMService.SubscribeToTopic(context.Background(), []string{req.Token}, "admin_alerts")
		if err != nil {
			fmt.Printf("Error subscribing admin to topic: %v\n", err)
		}

		// [NEW] Save Admin Token to Settings for Multicast
		if h.SettingsRepo != nil {
			if err := h.SettingsRepo.AddAdminToken(req.Token); err != nil {
				fmt.Printf("Error saving admin token to settings: %v\n", err)
				// Don't fail the request, just log
			} else {
				fmt.Printf("✅ Saved Admin FCM Token: %s\n", req.Token)
			}
		}

		return e.JSON(200, map[string]interface{}{
			"success": true,
			"message": "Admin FCM token registered",
		})
	}

	// 2. Handle Technician
	// Continued below...

	// [FIX] Prevent Token Leakage: Remove this token from ANY other technician
	// This ensures that if Tech A logs out and Tech B logs in on the same device,
	// Tech A will no longer receive notifications on this device.
	otherTechs, _ := h.App.FindRecordsByFilter("technicians", fmt.Sprintf("fcm_token='%s' && id!='%s'", req.Token, authRecord.Id), "", 100, 0, nil)
	for _, other := range otherTechs {
		other.Set("fcm_token", "")
		h.App.Save(other)
		fmt.Printf("⚠️ Cleared stale FCM token from tech %s (%s)\n", other.GetString("name"), other.Id)
	}

	// Find technician record
	tech, err := h.App.FindRecordById("technicians", authRecord.Id)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Technician not found"})
	}

	// Update FCM token
	tech.Set("fcm_token", req.Token)
	if err := h.App.Save(tech); err != nil {
		return e.JSON(500, map[string]string{"error": "Failed to save token"})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "FCM token registered successfully",
	})
}

// TestNotification sends a test notification to the current user
// POST /api/tech/test-notification
func (h *FCMHandler) TestNotification(e *core.RequestEvent) error {
	if h.FCMService == nil {
		return e.JSON(503, map[string]string{"error": "FCM Service not configured"})
	}

	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	// Find technician record
	tech, err := h.App.FindRecordById("technicians", authRecord.Id)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Technician not found"})
	}

	fcmToken := tech.GetString("fcm_token")
	if fcmToken == "" {
		return e.JSON(400, map[string]string{"error": "No FCM token registered"})
	}

	// Send test notification
	payload := &services.NotificationPayload{
		Title: "✅ Thử nghiệm thông báo",
		Body:  "Thông báo push đang hoạt động bình thường",
		Data: map[string]string{
			"type": "test",
		},
		Icon:  "/assets/icons/icon-192x192.png", // [FIX] path
		Badge: "/assets/icons/icon-192x192.png", // Reuse icon if badge missing
	}

	_, err = h.FCMService.SendNotification(e.Request.Context(), payload)
	if err != nil {
		fmt.Printf("Error sending test notification: %v\n", err)
		return e.JSON(500, map[string]string{"error": "Failed to send notification"})
	}

	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Test notification sent",
	})
}

// NotifyNewJobAssignment broadcasts notification to assigned technician
func (h *FCMHandler) NotifyNewJobAssignment(techID string, jobID string, customerName string) error {
	if h.FCMService == nil {
		return fmt.Errorf("FCM service not configured")
	}

	// Find technician record
	tech, err := h.App.FindRecordById("technicians", techID)
	if err != nil {
		return fmt.Errorf("technician not found: %v", err)
	}

	fcmToken := tech.GetString("fcm_token")
	if fcmToken == "" {
		return fmt.Errorf("no FCM token for technician: %s", techID)
	}

	return h.FCMService.NotifyNewJobAssignment(context.Background(), fcmToken, jobID, customerName)
}

// NotifyJobStatusChange broadcasts status change to technician
func (h *FCMHandler) NotifyJobStatusChange(techID string, jobID string, status string) error {
	if h.FCMService == nil {
		return fmt.Errorf("FCM service not configured")
	}

	// Find technician record
	tech, err := h.App.FindRecordById("technicians", techID)
	if err != nil {
		return fmt.Errorf("technician not found: %v", err)
	}

	fcmToken := tech.GetString("fcm_token")
	if fcmToken == "" {
		return fmt.Errorf("no FCM token for technician: %s", techID)
	}

	return h.FCMService.NotifyJobStatusChange(context.Background(), fcmToken, jobID, status)
}

// GetFCMStatus returns current FCM status for debugging
// GET /api/tech/fcm-status
func (h *FCMHandler) GetFCMStatus(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	// Find technician record
	tech, err := h.App.FindRecordById("technicians", authRecord.Id)
	if err != nil {
		return e.JSON(404, map[string]string{"error": "Technician not found"})
	}

	fcmToken := tech.GetString("fcm_token")
	hasToken := fcmToken != ""

	return e.JSON(200, map[string]interface{}{
		"has_fcm_token": hasToken,
		"token_length":  len(fcmToken),
		"updated_at":    tech.GetString("updated"),
	})
}

// SubscribeTechnicianToTopic subscribes technician to notifications topic
func (h *FCMHandler) SubscribeTechnicianToTopic(techID string, topic string) error {
	if h.FCMService == nil {
		return fmt.Errorf("FCM service not configured")
	}

	// Find technician record
	tech, err := h.App.FindRecordById("technicians", techID)
	if err != nil {
		return fmt.Errorf("technician not found: %v", err)
	}

	fcmToken := tech.GetString("fcm_token")
	if fcmToken == "" {
		return fmt.Errorf("no FCM token for technician: %s", techID)
	}

	return h.FCMService.SubscribeToTopic(context.Background(), []string{fcmToken}, topic)
}
