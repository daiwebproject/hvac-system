package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("services")
		if err != nil {
			return err
		}

		// 1. Gallery images (Multiple files)
		if collection.Fields.GetByName("gallery") == nil {
			collection.Fields.Add(&core.FileField{
				Name:      "gallery",
				MaxSelect: 10,
				MaxSize:   5242880, // 5MB each
				MimeTypes: []string{"image/jpeg", "image/png", "image/webp"},
			})
		}

		// 2. Detailed Content (Rich text / HTML)
		if collection.Fields.GetByName("detail_content") == nil {
			collection.Fields.Add(&core.EditorField{
				Name: "detail_content",
			})
		}

		// 3. Service Introduction (Short Intro)
		if collection.Fields.GetByName("intro_text") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "intro_text",
				Max:  500,
			})
		}

		// 4. Video Links (Storing URL strings or JSON list)
		// Assuming we just store a comma-separated string or a single featured video for now,
		// or specifically "video_url" string if user wants just one link
		// "intro service ... video thực tế theo đường dẫn đến youtube, tiktok, facebook"
		if collection.Fields.GetByName("video_url") == nil {
			collection.Fields.Add(&core.TextField{
				Name: "video_url",
			})
		}

		return app.Save(collection)
	}, nil)
}
