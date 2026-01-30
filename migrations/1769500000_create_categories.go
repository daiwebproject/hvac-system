package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Kiểm tra xem bảng đã tồn tại chưa
		if _, err := app.FindCollectionByNameOrId("categories"); err == nil {
			return nil // Đã có rồi thì skip
		}

		categories := core.NewBaseCollection("categories")

		// Tên danh mục (VD: Máy lạnh, Máy giặt, Linh kiện)
		categories.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})

		// Slug cho SEO (VD: may-lanh, may-giat)
		categories.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
		})

		// Mô tả danh mục
		categories.Fields.Add(&core.TextField{
			Name: "description",
		})

		// Icon/Image cho danh mục
		categories.Fields.Add(&core.FileField{
			Name:      "icon",
			MaxSelect: 1,
			MaxSize:   5242880, // 5MB
		})

		// Thứ tự hiển thị
		minZero := float64(0)
		categories.Fields.Add(&core.NumberField{
			Name: "sort_order",
			Min:  &minZero,
		})

		// Trạng thái active
		categories.Fields.Add(&core.BoolField{
			Name: "active",
		})

		// Tạo index cho slug để tìm kiếm nhanh
		categories.Indexes = []string{
			"CREATE UNIQUE INDEX idx_categories_slug ON categories (slug)",
		}

		return app.Save(categories)

	}, func(app core.App) error {
		// Rollback: Xóa bảng categories
		if collection, err := app.FindCollectionByNameOrId("categories"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
