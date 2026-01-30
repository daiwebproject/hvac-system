package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Kiểm tra xem bảng đã tồn tại chưa
		if _, err := app.FindCollectionByNameOrId("order_items"); err == nil {
			return nil
		}

		// Lấy collections để tạo relations
		orders, err := app.FindCollectionByNameOrId("orders")
		if err != nil {
			return err
		}

		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}

		orderItems := core.NewBaseCollection("order_items")

		// Relation tới đơn hàng
		orderItems.Fields.Add(&core.RelationField{
			Name:          "order_id",
			CollectionId:  orders.Id,
			MaxSelect:     1,
			Required:      true,
			CascadeDelete: true, // Xóa đơn hàng thì xóa luôn items
		})

		// Relation tới sản phẩm/dịch vụ
		orderItems.Fields.Add(&core.RelationField{
			Name:         "product_id",
			CollectionId: products.Id,
			MaxSelect:    1,
			Required:     true,
		})

		// Tên sản phẩm (snapshot tại thời điểm mua)
		orderItems.Fields.Add(&core.TextField{
			Name:     "product_name",
			Required: true,
		})

		// Giá tại thời điểm mua
		minZero := float64(0)
		minOne := float64(1)
		orderItems.Fields.Add(&core.NumberField{
			Name:     "price",
			Required: true,
			Min:      &minZero,
		})

		// Số lượng
		orderItems.Fields.Add(&core.NumberField{
			Name:     "quantity",
			Required: true,
			Min:      &minOne,
		})

		// Thành tiền (price * quantity)
		orderItems.Fields.Add(&core.NumberField{
			Name:     "total",
			Required: true,
			Min:      &minZero,
		})

		// Thông số kỹ thuật (snapshot)
		orderItems.Fields.Add(&core.JSONField{
			Name: "specifications",
		})

		// Tạo indexes
		orderItems.Indexes = []string{
			"CREATE INDEX idx_order_items_order ON order_items (order_id)",
			"CREATE INDEX idx_order_items_product ON order_items (product_id)",
		}

		return app.Save(orderItems)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("order_items"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
