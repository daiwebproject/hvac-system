package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// Config
const (
	BaseURL = "http://localhost:8090"
	TechID  = "sp67aenwbis8fv9" // ID thợ thật (Phạm Nam Định)
	JobID   = "v29k7prax2brcf5" // ID job thật
)

// Hanoi coordinates (Start -> End)
var (
	StartLat = 21.028511
	StartLng = 105.854444 // Hoan Kiem Lake
	EndLat   = 21.003117
	EndLng   = 105.820140 // Royal City
)

type LocationUpdate struct {
	TechID    string  `json:"tech_id"`
	BookingID string  `json:"booking_id"` // Matches backend struct
	Lat       float64 `json:"lat"`
	Long      float64 `json:"long"`
	Heading   float64 `json:"heading"`
	Speed     float64 `json:"speed"`
}

func main() {
	fmt.Println("🚀 Starting GPS Simulation...")
	fmt.Printf("Tech: %s -> Job: %s\n", TechID, JobID)

	steps := 100
	for i := 0; i <= steps; i++ {
		// Interpolate position
		ratio := float64(i) / float64(steps)
		currentLat := StartLat + (EndLat-StartLat)*ratio
		currentLng := StartLng + (EndLng-StartLng)*ratio

		// Calculate heading (simplified)
		heading := math.Atan2(EndLng-StartLng, EndLat-StartLat) * 180 / math.Pi

		// Create payload
		payload := LocationUpdate{
			TechID:    TechID,
			BookingID: JobID,
			Lat:       currentLat,
			Long:      currentLng,
			Heading:   heading,
			Speed:     30.0, // km/h
		}

		// Send Request
		sendLocation(payload)

		// Wait
		time.Sleep(1 * time.Second)
	}

	fmt.Println("✅ Simulation Complete!")
}

func sendLocation(data LocationUpdate) {
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(BaseURL+"/api/tech/location/update", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("📍 Sent: [%.6f, %.6f] Status: %s\n", data.Lat, data.Long, resp.Status)
}
