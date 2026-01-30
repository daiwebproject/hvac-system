package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Tạo bảng Services
		// Kiểm tra xem bảng đã tồn tại chưa để tránh lỗi
		if _, err := app.FindCollectionByNameOrId("services"); err == nil {
			return nil // Đã có rồi thì thôi
		}

		services := core.NewBaseCollection("services")
		
		// Định nghĩa các cột (Fields)
		services.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})
		services.Fields.Add(&core.TextField{
			Name: "description",
		})
		services.Fields.Add(&core.NumberField{
			Name: "price",
		})
		services.Fields.Add(&core.BoolField{
			Name: "active",
		})

		// Lưu bảng services
		if err := app.Save(services); err != nil {
			return err
		}

		// 2. Tạo bảng Bookings
		if _, err := app.FindCollectionByNameOrId("bookings"); err == nil {
			return nil
		}

		bookings := core.NewBaseCollection("bookings")
		
		bookings.Fields.Add(&core.TextField{
			Name:     "customer_name",
			Required: true,
		})
		bookings.Fields.Add(&core.TextField{
			Name:     "customer_phone",
			Required: true,
		})
		bookings.Fields.Add(&core.TextField{
			Name: "status",
		})
		
		// Tạo quan hệ (Relation) tới bảng Services
		bookings.Fields.Add(&core.RelationField{
			Name:         "service_id",
			CollectionId: services.Id,
			MaxSelect:    1,
		})

		return app.Save(bookings)

	}, func(app core.App) error {
		// Logic revert (xóa bảng nếu rollback)
		if collection, err := app.FindCollectionByNameOrId("bookings"); err == nil {
			app.Delete(collection)
		}
		if collection, err := app.FindCollectionByNameOrId("services"); err == nil {
			app.Delete(collection)
		}
		return nil
	})
}