package services

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMService handles Firebase Cloud Messaging operations
type FCMService struct {
	client *messaging.Client
}

// NewFCMService creates a new FCM service instance
func NewFCMService(credentialsPath string) (*FCMService, error) {
	ctx := context.Background()
	opt := option.WithCredentialsFile(credentialsPath)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing app: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return &FCMService{client: client}, nil
}

// NotificationPayload defines the structure for push notifications
type NotificationPayload struct {
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Data        map[string]string `json:"data,omitempty"`
	DeviceToken string            `json:"-"` // Not sent to FCM, used internally
	Icon        string            `json:"icon,omitempty"`
	Badge       string            `json:"badge,omitempty"`
	Link        string            `json:"link,omitempty"` // [NEW] Action link
}

// SendNotification sends a single notification to a device
func (s *FCMService) SendNotification(ctx context.Context, payload *NotificationPayload) (string, error) {
	link := payload.Link
	if link == "" {
		link = "https://192.168.1.12/tech/jobs" // Default fallback
	}

	message := &messaging.Message{
		Token: payload.DeviceToken,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: payload.Data,
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: payload.Title,
				Body:  payload.Body,
				Icon:  payload.Icon,
				Badge: payload.Badge,
			},
			FCMOptions: &messaging.WebpushFCMOptions{
				Link: link,
			},
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Title: payload.Title,
				Body:  payload.Body,
				Icon:  "icon",
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: payload.Title,
						Body:  payload.Body,
					},
					Sound: "default",
					Badge: intPtr(1),
				},
			},
		},
	}

	response, err := s.client.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("error sending message: %v", err)
	}

	log.Printf("Successfully sent message: %s\n", response)
	return response, nil
}

// SendMulticast sends notifications to multiple devices
func (s *FCMService) SendMulticast(ctx context.Context, deviceTokens []string, payload *NotificationPayload) (*messaging.BatchResponse, error) {
	link := payload.Link
	if link == "" {
		// [FIX] Absolute URL
	}

	// [FIX] SendMulticast using loop to avoid legacy batch API 404 error
	successCount := 0
	failureCount := 0

	for _, token := range deviceTokens {
		// Create a copy of payload with specific token
		singlePayload := *payload
		singlePayload.DeviceToken = token
		singlePayload.Link = link

		// Reuse SendNotification which uses valid HTTP v1 API
		_, err := s.SendNotification(ctx, &singlePayload)
		if err != nil {
			log.Printf("Failed to send to token %s: %v", token, err)
			failureCount++
		} else {
			successCount++
		}
	}

	// Construct a partially fake BatchResponse to satisfy interface
	// Note: We don't have per-message IDs here easily without refactoring return type
	return &messaging.BatchResponse{
		SuccessCount: successCount,
		FailureCount: failureCount,
	}, nil
}

// SendToTopic sends notification to all subscribers of a topic
func (s *FCMService) SendToTopic(ctx context.Context, topic string, payload *NotificationPayload) (string, error) {
	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: payload.Title,
			Body:  payload.Body,
		},
		Data: payload.Data,
	}

	response, err := s.client.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("error sending topic message: %v", err)
	}

	return response, nil
}

// SubscribeToTopic subscribes a device to a topic
func (s *FCMService) SubscribeToTopic(ctx context.Context, deviceTokens []string, topic string) error {
	response, err := s.client.SubscribeToTopic(ctx, deviceTokens, topic)
	if err != nil {
		return fmt.Errorf("error subscribing to topic: %v", err)
	}

	log.Printf("Subscription response: %d succeeded, %d failed\n", response.SuccessCount, response.FailureCount)
	return nil
}

// UnsubscribeFromTopic unsubscribes a device from a topic
func (s *FCMService) UnsubscribeFromTopic(ctx context.Context, deviceTokens []string, topic string) error {
	response, err := s.client.UnsubscribeFromTopic(ctx, deviceTokens, topic)
	if err != nil {
		return fmt.Errorf("error unsubscribing from topic: %v", err)
	}

	log.Printf("Unsubscription response: %d succeeded, %d failed\n", response.SuccessCount, response.FailureCount)
	return nil
}

// NotifyNewJobAssignment notifies technician of new job assignment
func (s *FCMService) NotifyNewJobAssignment(ctx context.Context, techToken string, jobID string, customerName string) error {
	payload := &NotificationPayload{
		DeviceToken: techToken, // [FIX] Must assign token to payload
		Title:       "🚀 Công việc mới được giao",
		Body:        fmt.Sprintf("Khách hàng: %s", customerName),
		Data: map[string]string{
			"job_id": jobID,
			"type":   "job_assignment",
			"action": "open_job",
			"jobUrl": fmt.Sprintf("/tech/job/%s", jobID),
		},
		Icon:  "https://192.168.1.12/assets/icons/icon-192x192.png",
		Badge: "https://192.168.1.12/assets/icons/icon-96x96.png", // Use small icon as badge fallback
	}

	_, err := s.SendNotification(ctx, payload)
	return err
}

// NotifyJobStatusChange notifies technician of job status changes
func (s *FCMService) NotifyJobStatusChange(ctx context.Context, techToken string, jobID string, status string) error {
	statusMessage := map[string]string{
		"assigned":    "✅ Công việc được giao",
		"in_progress": "🔧 Đang thực hiện",
		"completed":   "✨ Hoàn thành",
		"pending":     "⏳ Chờ duyệt",
	}

	title, ok := statusMessage[status]
	if !ok {
		title = "Cập nhật trạng thái công việc"
	}

	payload := &NotificationPayload{
		Title: title,
		Body:  fmt.Sprintf("Job #%s - %s", jobID, title),
		Data: map[string]string{
			"job_id": jobID,
			"status": status,
			"type":   "job_status_change",
			"action": "open_job",
			"jobUrl": fmt.Sprintf("/tech/job/%s", jobID),
		},
		Icon:  "https://192.168.1.12/assets/icons/icon-192x192.png",
		Badge: "https://192.168.1.12/assets/icons/icon-96x96.png", // Use small icon as badge fallback
	}

	_, err := s.SendNotification(ctx, payload)
	return err
}

// NotifyPendingReviewsSync notifies technician to sync pending offline reports
func (s *FCMService) NotifyPendingReviewsSync(ctx context.Context, techToken string, pendingCount int) error {
	payload := &NotificationPayload{
		Title: "📤 Báo cáo chờ đồng bộ",
		Body:  fmt.Sprintf("Bạn có %d báo cáo chờ đẩy lên hệ thống", pendingCount),
		Data: map[string]string{
			"type":   "pending_sync",
			"count":  fmt.Sprintf("%d", pendingCount),
			"action": "open_app",
		},
		Icon:  "https://192.168.1.12/assets/icons/icon-192x192.png",
		Badge: "https://192.168.1.12/assets/icons/icon-96x96.png", // Use small icon as badge fallback
	}

	_, err := s.SendNotification(ctx, payload)
	return err
}

// NotifyPaymentProcessed notifies technician of payment
func (s *FCMService) NotifyPaymentProcessed(ctx context.Context, techToken string, amount float64) error {
	payload := &NotificationPayload{
		Title: "💰 Thanh toán được xác nhận",
		Body:  fmt.Sprintf("Bạn được trả: %.0f VND", amount),
		Data: map[string]string{
			"type":   "payment",
			"amount": fmt.Sprintf("%.2f", amount),
			"action": "open_app",
		},
		Icon:  "https://192.168.1.12/assets/icons/icon-192x192.png",
		Badge: "https://192.168.1.12/assets/icons/icon-96x96.png", // Use small icon as badge fallback
	}

	_, err := s.SendNotification(ctx, payload)
	return err
}

// NotifyNewBooking sends notification to all admins about a new booking
func (s *FCMService) NotifyNewBooking(ctx context.Context, bookingID string, customerName string) error {
	payload := &NotificationPayload{
		Title: "🔔 Đơn hàng mới",
		Body:  fmt.Sprintf("Khách hàng %s vừa đặt lịch", customerName),
		Data: map[string]string{
			"booking_id": bookingID,
			"type":       "new_booking",
			"action":     "open_booking",
			"bookingUrl": fmt.Sprintf("/admin/bookings/%s", bookingID),
		},
		Icon:  "/assets/icons/icon-192x192.png", // [FIX]
		Badge: "/assets/icons/icon-192x192.png", // [FIX]
	}

	_, err := s.SendToTopic(ctx, "admin_alerts", payload)
	return err
}

// NotifyBookingCancelled notifies admins that a booking was cancelled
func (s *FCMService) NotifyBookingCancelled(ctx context.Context, bookingID, customerName, reason, note string) error {
	title := "⚠️ Đơn hàng bị hủy"
	body := fmt.Sprintf("Đơn %s đã bị hủy. Lý do: %s", customerName, reason)
	if note != "" {
		body += fmt.Sprintf(" (%s)", note)
	}

	payload := &NotificationPayload{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":       "booking_cancelled",
			"booking_id": bookingID,
			"reason":     reason,
		},
		Icon:  "/assets/icons/icon-192x192.png",
		Badge: "/assets/icons/icon-192x192.png",
	}

	_, err := s.SendToTopic(ctx, "admin_alerts", payload)
	return err
}

func intPtr(i int) *int {
	return &i
}

// NotifyAdmins sends multicast notification to specific admin devices
func (s *FCMService) NotifyAdmins(ctx context.Context, tokens []string, bookingID, customerName string) error {
	if len(tokens) == 0 {
		return nil
	}
	payload := &NotificationPayload{
		Title: "🔔 Đơn hàng mới (Admin)",
		Body:  fmt.Sprintf("Khách hàng %s vừa đặt lịch", customerName),
		Data: map[string]string{
			"booking_id": bookingID,
			"type":       "new_booking",
			"action":     "open_booking",
			"bookingUrl": fmt.Sprintf("/admin/bookings/%s", bookingID),
		},
		Icon:  "/assets/icons/icon-192x192.png",                                 // [FIX] path
		Badge: "/assets/icons/icon-192x192.png",                                 // Reuse icon if badge missing
		Link:  fmt.Sprintf("https://192.168.1.12/admin/bookings/%s", bookingID), // [FIX] Absolute URL
	}

	response, err := s.SendMulticast(ctx, tokens, payload)
	if err != nil {
		return err
	}
	log.Printf("NotifyAdmins: Success %d, Failure %d", response.SuccessCount, response.FailureCount)
	return nil
}

// NotifyAdminsBookingCancelled sends multicast cancellation to admins
func (s *FCMService) NotifyAdminsBookingCancelled(ctx context.Context, tokens []string, bookingID, customerName, reason, note string) error {
	if len(tokens) == 0 {
		return nil
	}
	title := "⚠️ Đơn hàng bị hủy"
	body := fmt.Sprintf("Đơn %s đã bị hủy. Lý do: %s", customerName, reason)
	if note != "" {
		body += fmt.Sprintf(" (%s)", note)
	}

	payload := &NotificationPayload{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":       "booking_cancelled",
			"booking_id": bookingID,
			"reason":     reason,
		},
		Icon:  "/assets/icons/icon-192x192.png", // [FIX]
		Badge: "/assets/icons/icon-192x192.png", // [FIX]
	}

	response, err := s.SendMulticast(ctx, tokens, payload)
	if err != nil {
		return err
	}
	log.Printf("NotifyAdminsBookingCancelled: Success %d, Failure %d", response.SuccessCount, response.FailureCount)
	return nil
}
