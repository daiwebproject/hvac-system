package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("settings")
		if err != nil {
			return err
		}

		// 1. Nhóm SEO
		collection.Fields.Add(&core.TextField{Name: "seo_title", Max: 255})
		collection.Fields.Add(&core.TextField{Name: "seo_description", Max: 500})
		collection.Fields.Add(&core.TextField{Name: "seo_keywords", Max: 255})

		// 2. Nhóm Hero Section (Giao diện)
		collection.Fields.Add(&core.TextField{Name: "hero_title", Max: 255})
		collection.Fields.Add(&core.TextField{Name: "hero_subtitle", Max: 255})
		collection.Fields.Add(&core.FileField{
			Name:      "hero_image",
			MaxSelect: 1,
			MaxSize:   5242880, // 5MB
			MimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
		})
		collection.Fields.Add(&core.TextField{Name: "hero_cta_text", Max: 50})
		collection.Fields.Add(&core.TextField{Name: "hero_cta_link", Max: 255})

		return app.Save(collection)
	}, nil)
}
