package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Kiểm tra xem bảng đã tồn tại chưa
		if _, err := app.FindCollectionByNameOrId("time_slots"); err == nil {
			return nil
		}

		// Lấy collections để tạo relations
		technicians, err := app.FindCollectionByNameOrId("technicians")
		if err != nil {
			return err
		}

		orders, err := app.FindCollectionByNameOrId("orders")
		if err != nil {
			return err
		}

		timeSlots := core.NewBaseCollection("time_slots")

		// Thợ phụ trách
		timeSlots.Fields.Add(&core.RelationField{
			Name:         "technician_id",
			CollectionId: technicians.Id,
			MaxSelect:    1,
			Required:     true,
		})

		// Ngày làm việc
		timeSlots.Fields.Add(&core.DateField{
			Name:     "work_date",
			Required: true,
		})

		// Giờ bắt đầu (VD: 08:00, 10:00, 14:00)
		timeSlots.Fields.Add(&core.TextField{
			Name:     "start_time",
			Required: true,
		})

		// Giờ kết thúc (VD: 10:00, 12:00, 16:00)
		timeSlots.Fields.Add(&core.TextField{
			Name:     "end_time",
			Required: true,
		})

		// Trạng thái
		timeSlots.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"available", "booked", "completed", "cancelled"},
		})

		// Đơn hàng đã đặt (nếu booked)
		timeSlots.Fields.Add(&core.RelationField{
			Name:         "order_id",
			CollectionId: orders.Id,
			MaxSelect:    1,
		})

		// Ghi chú
		timeSlots.Fields.Add(&core.TextField{
			Name: "note",
		})

		// Tạo indexes để tránh đặt trùng
		timeSlots.Indexes = []string{
			"CREATE INDEX idx_time_slots_technician ON time_slots (technician_id)",
			"CREATE INDEX idx_time_slots_date ON time_slots (work_date)",
			"CREATE INDEX idx_time_slots_status ON time_slots (status)",
			"CREATE UNIQUE INDEX idx_time_slots_unique ON time_slots (technician_id, work_date, start_time)",
		}

		return app.Save(timeSlots)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("time_slots"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
