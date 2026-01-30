package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Kiểm tra xem bảng đã tồn tại chưa
		if _, err := app.FindCollectionByNameOrId("reviews"); err == nil {
			return nil
		}

		// Lấy collections để tạo relations
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}

		orders, err := app.FindCollectionByNameOrId("orders")
		if err != nil {
			return err
		}

		reviews := core.NewBaseCollection("reviews")

		// Sản phẩm/dịch vụ được đánh giá
		reviews.Fields.Add(&core.RelationField{
			Name:         "product_id",
			CollectionId: products.Id,
			MaxSelect:    1,
			Required:     true,
		})

		// Đơn hàng liên quan (để verify đã mua)
		reviews.Fields.Add(&core.RelationField{
			Name:         "order_id",
			CollectionId: orders.Id,
			MaxSelect:    1,
			Required:     true,
		})

		// Tên người đánh giá
		reviews.Fields.Add(&core.TextField{
			Name:     "customer_name",
			Required: true,
		})

		// Số sao (1-5)
		minOne := float64(1)
		maxFive := float64(5)
		reviews.Fields.Add(&core.NumberField{
			Name:     "rating",
			Required: true,
			Min:      &minOne,
			Max:      &maxFive,
		})

		// Nội dung đánh giá
		reviews.Fields.Add(&core.TextField{
			Name:     "comment",
			Required: true,
		})

		// Hình ảnh đính kèm
		reviews.Fields.Add(&core.FileField{
			Name:      "images",
			MaxSelect: 5,
			MaxSize:   5242880, // 5MB
		})

		// Trạng thái duyệt
		reviews.Fields.Add(&core.SelectField{
			Name:     "status",
			Required: true,
			Values:   []string{"pending", "approved", "rejected"},
		})

		// Phản hồi từ admin
		reviews.Fields.Add(&core.TextField{
			Name: "admin_reply",
		})

		// Tạo indexes
		reviews.Indexes = []string{
			"CREATE INDEX idx_reviews_product ON reviews (product_id)",
			"CREATE INDEX idx_reviews_status ON reviews (status)",
			"CREATE INDEX idx_reviews_rating ON reviews (rating)",
		}

		return app.Save(reviews)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("reviews"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
