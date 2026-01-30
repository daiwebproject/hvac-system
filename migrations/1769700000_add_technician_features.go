package migrations

import (
	"log"

	"github.com/pocketbase/pocketbase"
)

// init() is called when migrations package is imported
// Note: Manual migration via PocketBase admin panel is recommended:
// 1. Open PocketBase admin: http://localhost:8090/_/
// 2. Go to Collections
// 3. Edit "technicians" collection
// 4. Add new text field: "fcm_token" (Hidden: Yes)
// 5. Save
//
// Alternative: Use this function if you prefer programmatic migration
// But PocketBase auto-migration is simpler for schema changes
func init() {
	// Migration notes for manual setup
	log.Println("TECHNICIAN_FEATURES MIGRATION: Optional - Add fields via admin panel")
	log.Println("Required fields to add:")
	log.Println("  - technicians.fcm_token (text, hidden)")
	log.Println("  - technicians.fcm_token_updated (datetime, hidden)")
	log.Println("  - bookings.offline_synced_at (datetime, hidden)")
}

// EnableTechnicianFeatures can be called to manually add fields to collections
// Usage: Called from main.go during app initialization if needed
func EnableTechnicianFeatures(pb *pocketbase.PocketBase) error {
	// Note: Field manipulation requires lower-level API
	// Recommended approach: Use PocketBase admin panel
	// This is left as documentation for reference

	log.Println("Technician features are enabled via:")
	log.Println("1. Offline support - automatic with Service Worker")
	log.Println("2. QR Scanner - no database changes needed")
	log.Println("3. Maps - uses existing lat/long fields")
	log.Println("4. FCM - manually add 'fcm_token' field to technicians collection")
	log.Println("5. UI - automatic with new HTML/JS files")

	return nil
}
