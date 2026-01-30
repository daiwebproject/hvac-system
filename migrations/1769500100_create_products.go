package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Kiểm tra xem bảng đã tồn tại chưa
		if _, err := app.FindCollectionByNameOrId("products"); err == nil {
			return nil
		}

		// Lấy collection categories để tạo relation
		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		products := core.NewBaseCollection("products")

		// Tên sản phẩm
		products.Fields.Add(&core.TextField{
			Name:     "name",
			Required: true,
		})

		// Slug cho SEO
		products.Fields.Add(&core.TextField{
			Name:     "slug",
			Required: true,
		})

		// SKU (Mã sản phẩm)
		products.Fields.Add(&core.TextField{
			Name: "sku",
		})

		// Mô tả ngắn
		products.Fields.Add(&core.TextField{
			Name: "short_description",
		})

		// Mô tả chi tiết (HTML)
		products.Fields.Add(&core.EditorField{
			Name: "description",
		})

		// Giá bán
		minZero := float64(0)
		products.Fields.Add(&core.NumberField{
			Name:     "price",
			Required: true,
			Min:      &minZero,
		})

		// Giá gốc (để gạch đi khi giảm giá)
		products.Fields.Add(&core.NumberField{
			Name: "original_price",
			Min:  &minZero,
		})

		// Tồn kho
		products.Fields.Add(&core.NumberField{
			Name: "stock",
			Min:  &minZero,
		})

		// Ngưỡng cảnh báo hết hàng
		products.Fields.Add(&core.NumberField{
			Name: "low_stock_threshold",
			Min:  &minZero,
		})

		// Hình ảnh sản phẩm (nhiều ảnh)
		products.Fields.Add(&core.FileField{
			Name:      "images",
			MaxSelect: 10,
			MaxSize:   5242880, // 5MB per image
		})

		// Loại sản phẩm (product/service/combo)
		products.Fields.Add(&core.SelectField{
			Name:     "type",
			Required: true,
			Values:   []string{"product", "service", "combo"},
		})

		// Danh mục
		products.Fields.Add(&core.RelationField{
			Name:         "category_id",
			CollectionId: categories.Id,
			MaxSelect:    1,
			Required:     true,
		})

		// Thông số kỹ thuật (JSON)
		products.Fields.Add(&core.JSONField{
			Name: "specifications",
		})

		// Tags (để filter)
		products.Fields.Add(&core.SelectField{
			Name:      "tags",
			MaxSelect: 5,
			Values:    []string{"hot", "new", "sale", "bestseller", "featured"},
		})

		// Trạng thái
		products.Fields.Add(&core.BoolField{
			Name: "active",
		})

		// Featured (Nổi bật)
		products.Fields.Add(&core.BoolField{
			Name: "featured",
		})

		// Tạo indexes
		products.Indexes = []string{
			"CREATE UNIQUE INDEX idx_products_slug ON products (slug)",
			"CREATE INDEX idx_products_category ON products (category_id)",
			"CREATE INDEX idx_products_type ON products (type)",
		}

		return app.Save(products)

	}, func(app core.App) error {
		// Rollback
		if collection, err := app.FindCollectionByNameOrId("products"); err == nil {
			return app.Delete(collection)
		}
		return nil
	})
}
