package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Lấy collection categories
		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		// Seed Categories
		categoriesData := []map[string]interface{}{
			{
				"name":        "Máy Lạnh",
				"slug":        "may-lanh",
				"description": "Máy điều hòa nhiệt độ các loại: Split, Multi, Cassette",
				"sort_order":  1,
				"active":      true,
			},
			{
				"name":        "Máy Giặt",
				"slug":        "may-giat",
				"description": "Máy giặt cửa trước, cửa trên, lồng đứng, lồng ngang",
				"sort_order":  2,
				"active":      true,
			},
			{
				"name":        "Linh Kiện",
				"slug":        "linh-kien",
				"description": "Phụ tùng thay thế: Gas, tụ điện, board mạch, remote",
				"sort_order":  3,
				"active":      true,
			},
			{
				"name":        "Dịch Vụ",
				"slug":        "dich-vu",
				"description": "Vệ sinh, bảo dưỡng, sửa chữa máy lạnh và máy giặt",
				"sort_order":  4,
				"active":      true,
			},
			{
				"name":        "Combo",
				"slug":        "combo",
				"description": "Gói combo tiết kiệm: Máy + Lắp đặt + Bảo hành",
				"sort_order":  5,
				"active":      true,
			},
		}

		for _, data := range categoriesData {
			record := core.NewRecord(categories)
			for key, value := range data {
				record.Set(key, value)
			}
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil

	}, func(app core.App) error {
		// Rollback: Xóa tất cả categories
		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return nil
		}

		records, err := app.FindAllRecords(categories.Name)
		if err != nil {
			return nil
		}

		for _, record := range records {
			app.Delete(record)
		}

		return nil
	})
}
