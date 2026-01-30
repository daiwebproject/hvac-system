package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Kiểm tra xem bảng đã tồn tại chưa
		if _, err := app.FindCollectionByNameOrId("orders"); err == nil {
			return nil
		}

		orders := core.NewBaseCollection("orders")

		// Mã đơn hàng (tự động generate: ORD-20260127-001)
		orders.Fields.Add(&core.TextField{
			Name:     "order_number",
			Required: true,
		})

		// Thông tin khách hàng (JSON)
		// {name, phone, email, address, city, district, ward}
		orders.Fields.Add(&core.JSONField{
			Name:     "customer_info",
			Required: true,
		})

		// Tổng tiền hàng (chưa ship)
		minZero := float64(0)
		orders.Fields.Add(&core.NumberField{
			Name:     "subtotal",
			Required: true,
			Min:      &minZero,
		})

		// Phí vận chuyển
		orders.Fields.Add(&core.NumberField{
			Name: "shipping_fee",
			Min:  &minZero,
		})

		// Giảm giá (nếu có)
		orders.Fields.Add(&core.NumberField{
			Name: "discount",
			Min:  &minZero,
		})

		// Tổng thanh toán
		orders.Fields.Add(&core.NumberField{
			Name:     "total_amount",
			Required: true,
			Min:      &minZero,
		})

		// Trạng thái thanh toán
		orders.Fields.Add(&core.SelectField{
			Name:     "payment_status",
			Required: true,
			Values:   []string{"unpaid", "paid", "refunded"},
		})

		// Phương thức thanh toán
		orders.Fields.Add(&core.SelectField{
			Name:   "payment_method",
			Values: []string{"cod", "bank_transfer", "vietqr", "momo", "zalopay"},
		})

		// Trạng thái đơn hàng
		orders.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"pending", "confirmed", "processing", "shipping", "completed", "cancelled"},
		})

		// Loại đơn hàng
		orders.Fields.Add(&core.SelectField{
			Name:     "order_type",
			Required: true,
			Values:   []string{"product", "service", "combo"},
		})

		// Ghi chú của khách
		orders.Fields.Add(&core.TextField{
			Name: "customer_note",
		})

		// Ghi chú nội bộ
		orders.Fields.Add(&core.TextField{
			Name: "admin_note",
		})

		// Ngày giao hàng dự kiến (cho đơn hàng vật lý)
		orders.Fields.Add(&core.DateField{
			Name: "estimated_delivery",
		})

		// Ngày hẹn (cho đơn dịch vụ)
		orders.Fields.Add(&core.DateField{
			Name: "appointment_date",
		})

		// Tạo indexes
		orders.Indexes = []string{
			"CREATE UNIQUE INDEX idx_orders_number ON orders (order_number)",
			"CREATE INDEX idx_orders_status ON orders (status)",
			"CREATE INDEX idx_orders_payment_status ON orders (payment_status)",
		}

		return app.Save(orders)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("orders"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
