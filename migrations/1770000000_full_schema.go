package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// ----------------------------------------------------
		// 1. TECHNICIANS (Auth Collection)
		// ----------------------------------------------------
		// Helper to add system fields if missing
		addSystemFields := func(c *core.Collection) {
			c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
			c.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		}

		// ----------------------------------------------------
		// 1. TECHNICIANS (Auth Collection)
		// ----------------------------------------------------
		techs, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			techs = core.NewAuthCollection("technicians")
			addSystemFields(techs) // Explicitly add system dates
			techs.ListRule = types.Pointer("")
			techs.ViewRule = types.Pointer("")

			// Fields
			techs.Fields.Add(&core.TextField{Name: "name", Required: true})
			techs.Fields.Add(&core.BoolField{Name: "active"})
			techs.Fields.Add(&core.BoolField{Name: "verified"})
			techs.Fields.Add(&core.FileField{
				Name:      "avatar",
				MaxSelect: 1,
				MaxSize:   5242880,
			})
			techs.Fields.Add(&core.JSONField{Name: "location"}) // {lat: X, long: Y}

			techs.AddIndex("idx_techs_active", false, "active", "")

			if err := app.Save(techs); err != nil {
				return err
			}
		}

		// ----------------------------------------------------
		// 2. CATEGORIES
		// ----------------------------------------------------
		cats := core.NewBaseCollection("categories")
		addSystemFields(cats)
		cats.ListRule = types.Pointer("")
		cats.ViewRule = types.Pointer("")
		cats.Fields.Add(&core.TextField{Name: "name", Required: true})
		cats.Fields.Add(&core.TextField{Name: "slug", Required: true})
		cats.Fields.Add(&core.TextField{Name: "description"})
		cats.Fields.Add(&core.FileField{Name: "icon", MaxSelect: 1})
		cats.Fields.Add(&core.NumberField{Name: "sort_order"})
		cats.Fields.Add(&core.BoolField{Name: "active"})

		cats.AddIndex("idx_cats_slug", true, "slug", "")

		if err := app.Save(cats); err != nil {
			return err // Or continue
		}

		// ----------------------------------------------------
		// 3. SERVICES
		// ----------------------------------------------------
		services := core.NewBaseCollection("services")
		addSystemFields(services)
		services.ListRule = types.Pointer("")
		services.ViewRule = types.Pointer("")
		services.Fields.Add(&core.TextField{Name: "name", Required: true})
		services.Fields.Add(&core.TextField{Name: "description"})
		services.Fields.Add(&core.NumberField{Name: "price"})
		services.Fields.Add(&core.NumberField{Name: "duration_minutes"})
		services.Fields.Add(&core.BoolField{Name: "active"})
		services.Fields.Add(&core.FileField{Name: "image", MaxSelect: 1})
		services.Fields.Add(&core.RelationField{
			Name:         "category_id",
			CollectionId: cats.Id,
			MaxSelect:    1,
		})

		if err := app.Save(services); err != nil {
			return err
		}

		// ----------------------------------------------------
		// 4. TIME SLOTS
		// ----------------------------------------------------
		slots := core.NewBaseCollection("time_slots")
		addSystemFields(slots)
		slots.ListRule = types.Pointer("")
		slots.ViewRule = types.Pointer("")
		slots.Fields.Add(&core.RelationField{
			Name:         "technician_id",
			CollectionId: techs.Id, // Link to Techs (Optional if global slot)??
			// Original logic: slots are separate, but assigned to booking?
			// Wait, previous migration 1769580000 created it.
			// Let's assume Slots are independent but can serve bookings?
			// Repo uses `date` and `start_time`.
		})
		slots.Fields.Add(&core.TextField{Name: "date", Required: true})       // YYYY-MM-DD
		slots.Fields.Add(&core.TextField{Name: "start_time", Required: true}) // HH:MM
		slots.Fields.Add(&core.TextField{Name: "end_time"})
		slots.Fields.Add(&core.NumberField{Name: "max_capacity"})
		slots.Fields.Add(&core.NumberField{Name: "current_bookings"})
		slots.Fields.Add(&core.BoolField{Name: "is_booked"})

		if err := app.Save(slots); err != nil {
			return err
		}

		// ----------------------------------------------------
		// 5. BOOKINGS
		// ----------------------------------------------------
		bookings := core.NewBaseCollection("bookings")
		addSystemFields(bookings)
		bookings.ListRule = types.Pointer("")
		bookings.ViewRule = types.Pointer("")
		bookings.CreateRule = types.Pointer("")
		bookings.UpdateRule = types.Pointer("")

		bookings.Fields.Add(&core.TextField{Name: "customer_name", Required: true})
		bookings.Fields.Add(&core.TextField{Name: "customer_phone", Required: true})
		bookings.Fields.Add(&core.TextField{Name: "address"})
		bookings.Fields.Add(&core.TextField{Name: "address_details"})
		bookings.Fields.Add(&core.TextField{Name: "booking_time"}) // YYYY-MM-DD HH:MM

		bookings.Fields.Add(&core.TextField{Name: "device_type"})
		bookings.Fields.Add(&core.TextField{Name: "brand"})
		bookings.Fields.Add(&core.TextField{Name: "issue_description"})

		bookings.Fields.Add(&core.SelectField{
			Name:      "job_status",
			Values:    []string{"pending", "assigned", "moving", "arrived", "working", "completed", "cancelled", "failed"},
			MaxSelect: 1,
		})

		// Cost
		bookings.Fields.Add(&core.NumberField{Name: "estimated_cost"})

		// Coordinates
		bookings.Fields.Add(&core.NumberField{Name: "lat"})
		bookings.Fields.Add(&core.NumberField{Name: "long"})

		// Relations
		bookings.Fields.Add(&core.RelationField{
			Name:         "service_id",
			CollectionId: services.Id,
			MaxSelect:    1,
		})
		bookings.Fields.Add(&core.RelationField{
			Name:         "technician_id",
			CollectionId: techs.Id,
			MaxSelect:    1,
		})
		bookings.Fields.Add(&core.RelationField{
			Name:         "time_slot_id",
			CollectionId: slots.Id,
			MaxSelect:    1,
		})

		// Images
		bookings.Fields.Add(&core.FileField{
			Name:      "client_images",
			MaxSelect: 5,
		})

		// Indexes
		bookings.AddIndex("idx_bookings_tech_status", false, "technician_id,job_status", "")
		bookings.AddIndex("idx_bookings_time", false, "booking_time", "")
		bookings.AddIndex("idx_bookings_phone", false, "customer_phone", "")

		if err := app.Save(bookings); err != nil {
			return err
		}

		// ----------------------------------------------------
		// 6. JOB REPORTS
		// ----------------------------------------------------
		reports := core.NewBaseCollection("job_reports")
		addSystemFields(reports)
		reports.Fields.Add(&core.RelationField{
			Name:         "booking_id",
			CollectionId: bookings.Id,
			Required:     true,
			MaxSelect:    1,
		})
		reports.Fields.Add(&core.TextField{Name: "technician_notes"})
		reports.Fields.Add(&core.FileField{Name: "proof_images", MaxSelect: 5})
		reports.Fields.Add(&core.JSONField{Name: "parts_replaced"})
		reports.Fields.Add(&core.NumberField{Name: "final_cost"})

		if err := app.Save(reports); err != nil {
			return err
		}

		// ----------------------------------------------------
		// 7. INVOICES
		// ----------------------------------------------------
		invoices := core.NewBaseCollection("invoices")
		addSystemFields(invoices)
		invoices.Fields.Add(&core.RelationField{
			Name:         "booking_id",
			CollectionId: bookings.Id,
			Required:     true,
			MaxSelect:    1,
		})
		invoices.Fields.Add(&core.TextField{Name: "invoice_code"})
		invoices.Fields.Add(&core.NumberField{Name: "total_amount", Required: true})
		invoices.Fields.Add(&core.NumberField{Name: "discount"})
		invoices.Fields.Add(&core.SelectField{
			Name:   "payment_method",
			Values: []string{"cash", "transfer", "card"},
		})
		invoices.Fields.Add(&core.SelectField{
			Name:     "status",
			Values:   []string{"unpaid", "paid", "cancelled"},
			Required: true,
		})
		invoices.Fields.Add(&core.TextField{Name: "notes"})
		invoices.Fields.Add(&core.FileField{Name: "customer_signature", MaxSelect: 1})

		// Indexes
		invoices.AddIndex("idx_invoices_booking", true, "booking_id", "")
		// We trust 'created' exists on BaseCollection, but to index it we need to ensure table is ready?
		// We can add index on 'status' for now.
		invoices.AddIndex("idx_invoices_status", false, "status", "")

		if err := app.Save(invoices); err != nil {
			return err
		}

		// ----------------------------------------------------
		// 8. INVENTORY
		// ----------------------------------------------------
		inv := core.NewBaseCollection("inventory")
		addSystemFields(inv)
		inv.Fields.Add(&core.TextField{Name: "name", Required: true})
		inv.Fields.Add(&core.TextField{Name: "sku", Required: true})
		inv.Fields.Add(&core.NumberField{Name: "quantity"})
		inv.Fields.Add(&core.NumberField{Name: "price"})
		inv.Fields.Add(&core.TextField{Name: "unit"})

		inv.AddIndex("idx_inv_sku", true, "sku", "")

		if err := app.Save(inv); err != nil {
			return err
		}

		// ----------------------------------------------------
		// SEED DATA
		// ----------------------------------------------------
		// 0. Technicians
		techUser := core.NewRecord(techs)
		techUser.Set("email", "tech@example.com")
		techUser.Set("name", "Nguyễn Văn A")
		techUser.SetPassword("12345678")
		techUser.Set("verified", true)
		techUser.Set("active", true)
		app.Save(techUser)

		// Categories
		catData := []map[string]interface{}{
			{"name": "Máy Lạnh", "slug": "may-lanh", "sort_order": 1, "active": true, "id": "zsiaovqz3xc38db"},
			{"name": "Máy Giặt", "slug": "may-giat", "sort_order": 2, "active": true, "id": "ex6r700aqs07sjo"},
			{"name": "Linh Kiện", "slug": "linh-kien", "sort_order": 3, "active": true, "id": "jl3j4tiutwmr0oz"},
			{"name": "Dịch Vụ", "slug": "dich-vu", "sort_order": 4, "active": true, "id": "jc4h6h5bqnoiv20"},
		}
		for _, c := range catData {
			rec := core.NewRecord(cats)
			rec.Id = c["id"].(string)
			rec.Set("name", c["name"])
			rec.Set("slug", c["slug"])
			rec.Set("sort_order", c["sort_order"])
			rec.Set("active", c["active"])
			app.Save(rec)
		}

		// Services
		svcData := []map[string]interface{}{
			{"name": "Vệ sinh máy lạnh treo tường", "price": 150000, "duration": 60, "cat": "jc4h6h5bqnoiv20"},
			{"name": "Bơm gas R32", "price": 300000, "duration": 30, "cat": "jc4h6h5bqnoiv20"},
			{"name": "Lắp đặt máy lạnh mới", "price": 500000, "duration": 120, "cat": "jc4h6h5bqnoiv20"},
		}
		for _, s := range svcData {
			rec := core.NewRecord(services)
			rec.Set("name", s["name"])
			rec.Set("price", s["price"])
			rec.Set("duration_minutes", s["duration"])
			rec.Set("category_id", s["cat"])
			rec.Set("active", true)
			app.Save(rec)
		}

		return nil
	}, nil)
}
