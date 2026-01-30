package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Lấy collections
		categories, err := app.FindCollectionByNameOrId("categories")
		if err != nil {
			return err
		}

		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}

		// Lấy category IDs
		catMayLanh, _ := app.FindFirstRecordByFilter(categories.Name, "slug = 'may-lanh'")
		catMayGiat, _ := app.FindFirstRecordByFilter(categories.Name, "slug = 'may-giat'")
		catLinhKien, _ := app.FindFirstRecordByFilter(categories.Name, "slug = 'linh-kien'")
		catDichVu, _ := app.FindFirstRecordByFilter(categories.Name, "slug = 'dich-vu'")
		catCombo, _ := app.FindFirstRecordByFilter(categories.Name, "slug = 'combo'")

		// Seed Products
		productsData := []map[string]interface{}{
			// === MÁY LẠNH ===
			{
				"name":                "Máy Lạnh Daikin Inverter 1HP FTKC25UAVMV",
				"slug":                "may-lanh-daikin-inverter-1hp-ftkc25uavmv",
				"sku":                 "DAIKIN-FTKC25U",
				"short_description":   "Máy lạnh Daikin Inverter tiết kiệm điện, công suất 1HP, phù hợp phòng 15m²",
				"description":         "<p>Máy lạnh Daikin Inverter 1HP với công nghệ tiết kiệm điện vượt trội. Thiết kế hiện đại, vận hành êm ái.</p><ul><li>Công suất làm lạnh: 9.000 BTU</li><li>Công nghệ Inverter tiết kiệm điện</li><li>Chế độ hút ẩm</li><li>Gas R32 thân thiện môi trường</li></ul>",
				"price":               8500000,
				"original_price":      10500000,
				"stock":               25,
				"low_stock_threshold": 5,
				"type":                "product",
				"category_id":         catMayLanh.Id,
				"specifications":      mustJSON(map[string]interface{}{"cong_suat": "1HP", "cong_nghe": "Inverter", "gas": "R32", "dien_ap": "220V"}),
				"tags":                []string{"hot", "sale", "featured"},
				"active":              true,
				"featured":            true,
			},
			{
				"name":                "Máy Lạnh LG Inverter 1.5HP V13API1",
				"slug":                "may-lanh-lg-inverter-1-5hp-v13api1",
				"sku":                 "LG-V13API1",
				"short_description":   "Máy lạnh LG Inverter 1.5HP, làm lạnh nhanh, tiết kiệm điện",
				"description":         "<p>Máy lạnh LG Inverter 1.5HP với công nghệ Dual Inverter làm lạnh nhanh gấp đôi.</p>",
				"price":               9800000,
				"original_price":      12000000,
				"stock":               18,
				"low_stock_threshold": 5,
				"type":                "product",
				"category_id":         catMayLanh.Id,
				"specifications":      mustJSON(map[string]interface{}{"cong_suat": "1.5HP", "cong_nghe": "Dual Inverter", "gas": "R32"}),
				"tags":                []string{"new", "bestseller"},
				"active":              true,
				"featured":            true,
			},
			{
				"name":                "Máy Lạnh Multi Daikin 2 Dàn Lạnh 2HP",
				"slug":                "may-lanh-multi-daikin-2-dan-lanh-2hp",
				"sku":                 "DAIKIN-MULTI-2HP",
				"short_description":   "Hệ thống Multi Daikin 1 dàn nóng 2 dàn lạnh, tiết kiệm không gian",
				"description":         "<p>Hệ thống Multi Daikin cho phép kết nối nhiều dàn lạnh với 1 dàn nóng.</p>",
				"price":               25000000,
				"original_price":      28000000,
				"stock":               8,
				"low_stock_threshold": 3,
				"type":                "product",
				"category_id":         catMayLanh.Id,
				"specifications":      mustJSON(map[string]interface{}{"loai": "Multi", "so_dan_lanh": 2, "cong_suat": "2HP"}),
				"tags":                []string{"featured"},
				"active":              true,
				"featured":            false,
			},

			// === MÁY GIẶT ===
			{
				"name":                "Máy Giặt Samsung Inverter 9kg WW90TP44DSH/SV",
				"slug":                "may-giat-samsung-inverter-9kg-ww90tp44dsh-sv",
				"sku":                 "SAMSUNG-WW90TP44",
				"short_description":   "Máy giặt Samsung 9kg, công nghệ AI giặt sạch sâu",
				"description":         "<p>Máy giặt Samsung 9kg với công nghệ AI tự động nhận diện vải và điều chỉnh chế độ giặt phù hợp.</p>",
				"price":               7500000,
				"original_price":      9000000,
				"stock":               15,
				"low_stock_threshold": 5,
				"type":                "product",
				"category_id":         catMayGiat.Id,
				"specifications":      mustJSON(map[string]interface{}{"trong_luong": "9kg", "loai": "Cửa trước", "cong_nghe": "AI Wash"}),
				"tags":                []string{"hot", "bestseller"},
				"active":              true,
				"featured":            true,
			},
			{
				"name":                "Máy Giặt LG Inverter 10.5kg FV1410S4P",
				"slug":                "may-giat-lg-inverter-10-5kg-fv1410s4p",
				"sku":                 "LG-FV1410S4P",
				"short_description":   "Máy giặt LG 10.5kg, công nghệ hơi nước diệt khuẩn",
				"description":         "<p>Máy giặt LG 10.5kg với công nghệ Steam giặt hơi nước diệt khuẩn 99.9%.</p>",
				"price":               8900000,
				"original_price":      10500000,
				"stock":               12,
				"low_stock_threshold": 5,
				"type":                "product",
				"category_id":         catMayGiat.Id,
				"specifications":      mustJSON(map[string]interface{}{"trong_luong": "10.5kg", "loai": "Cửa trước", "cong_nghe": "Steam"}),
				"tags":                []string{"new", "sale"},
				"active":              true,
				"featured":            true,
			},

			// === LINH KIỆN ===
			{
				"name":                "Gas Máy Lạnh R32 - 3kg",
				"slug":                "gas-may-lanh-r32-3kg",
				"sku":                 "GAS-R32-3KG",
				"short_description":   "Gas điều hòa R32 chính hãng, chai 3kg",
				"description":         "<p>Gas R32 thân thiện môi trường, phù hợp cho máy lạnh Inverter.</p>",
				"price":               850000,
				"original_price":      0,
				"stock":               50,
				"low_stock_threshold": 10,
				"type":                "product",
				"category_id":         catLinhKien.Id,
				"specifications":      mustJSON(map[string]interface{}{"loai_gas": "R32", "trong_luong": "3kg"}),
				"tags":                []string{},
				"active":              true,
				"featured":            false,
			},
			{
				"name":                "Remote Máy Lạnh Đa Năng",
				"slug":                "remote-may-lanh-da-nang",
				"sku":                 "REMOTE-UNIVERSAL",
				"short_description":   "Remote điều khiển đa năng cho mọi hãng máy lạnh",
				"description":         "<p>Remote đa năng tương thích với Daikin, LG, Samsung, Panasonic, Mitsubishi...</p>",
				"price":               150000,
				"original_price":      0,
				"stock":               100,
				"low_stock_threshold": 20,
				"type":                "product",
				"category_id":         catLinhKien.Id,
				"specifications":      mustJSON(map[string]interface{}{"loai": "Universal", "pin": "AAA x2"}),
				"tags":                []string{"bestseller"},
				"active":              true,
				"featured":            false,
			},

			// === DỊCH VỤ ===
			{
				"name":                "Vệ Sinh Máy Lạnh Tại Nhà",
				"slug":                "ve-sinh-may-lanh-tai-nha",
				"sku":                 "SV-CLEAN-AC",
				"short_description":   "Dịch vụ vệ sinh máy lạnh chuyên nghiệp, bảo hành 30 ngày",
				"description":         "<p>Vệ sinh máy lạnh sạch sâu bằng máy chuyên dụng, hóa chất an toàn.</p><ul><li>Vệ sinh dàn lạnh, dàn nóng</li><li>Kiểm tra gas</li><li>Bảo hành 30 ngày</li></ul>",
				"price":               200000,
				"original_price":      300000,
				"stock":               999,
				"low_stock_threshold": 0,
				"type":                "service",
				"category_id":         catDichVu.Id,
				"specifications":      mustJSON(map[string]interface{}{"thoi_gian": "60 phút", "bao_hanh": "30 ngày"}),
				"tags":                []string{"hot", "sale"},
				"active":              true,
				"featured":            true,
			},
			{
				"name":                "Sửa Chữa Máy Giặt Tại Nhà",
				"slug":                "sua-chua-may-giat-tai-nha",
				"sku":                 "SV-REPAIR-WM",
				"short_description":   "Sửa chữa máy giặt mọi hư hỏng, bảo hành 90 ngày",
				"description":         "<p>Sửa chữa máy giặt: không vắt, không xả nước, rò rỉ, kêu to...</p>",
				"price":               150000,
				"original_price":      0,
				"stock":               999,
				"low_stock_threshold": 0,
				"type":                "service",
				"category_id":         catDichVu.Id,
				"specifications":      mustJSON(map[string]interface{}{"phi_kiem_tra": "150k", "bao_hanh": "90 ngày"}),
				"tags":                []string{"bestseller"},
				"active":              true,
				"featured":            true,
			},

			// === COMBO ===
			{
				"name":                "Combo Máy Lạnh Daikin 1HP + Lắp Đặt + Bảo Hành 2 Năm",
				"slug":                "combo-may-lanh-daikin-1hp-lap-dat-bao-hanh",
				"sku":                 "COMBO-DAIKIN-1HP",
				"short_description":   "Gói combo tiết kiệm: Máy lạnh + Lắp đặt trọn gói + Bảo hành 2 năm",
				"description":         "<p>Combo bao gồm:</p><ul><li>Máy lạnh Daikin 1HP chính hãng</li><li>Lắp đặt trọn gói (ống đồng 3m, dây điện, khoan tường)</li><li>Bảo hành 2 năm</li><li>Vệ sinh miễn phí 1 lần/năm</li></ul>",
				"price":               9500000,
				"original_price":      11500000,
				"stock":               10,
				"low_stock_threshold": 3,
				"type":                "combo",
				"category_id":         catCombo.Id,
				"specifications":      mustJSON(map[string]interface{}{"bao_gom": "Máy + Lắp đặt + Bảo hành 2 năm", "ve_sinh_mien_phi": "1 lần/năm"}),
				"tags":                []string{"hot", "featured", "sale"},
				"active":              true,
				"featured":            true,
			},
		}

		for _, data := range productsData {
			record := core.NewRecord(products)
			for key, value := range data {
				record.Set(key, value)
			}
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil

	}, func(app core.App) error {
		// Rollback: Xóa tất cả products
		products, err := app.FindCollectionByNameOrId("products")
		if err != nil {
			return nil
		}

		records, err := app.FindAllRecords(products.Name)
		if err != nil {
			return nil
		}

		for _, record := range records {
			app.Delete(record)
		}

		return nil
	})
}

// Helper function để convert map sang JSON
func mustJSON(data map[string]interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}
