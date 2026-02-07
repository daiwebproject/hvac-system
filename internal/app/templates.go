package app

import (
	"errors"
	"html/template"
	"log"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cast"
)

// InitTemplates initializes the HTML templates with custom functions
// NOTE: Moved from pkg/app/config.go to break import cycle
func InitTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"formatMoney": func(val interface{}) string {
			amount := cast.ToFloat64(val)
			return humanize.Commaf(amount)
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"substr": func(start, length int, s string) string {
			if start < 0 {
				start = 0
			}
			if start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
	}

	t := template.New("").Funcs(funcMap)

	// 1. Load Layouts
	if _, err := t.ParseGlob("views/layouts/*.html"); err != nil {
		log.Println("Warning: Layouts error:", err)
	}

	// 2. Load Components (including subdirectories)
	if _, err := t.ParseGlob("views/components/*.html"); err != nil {
		log.Println("Warning: Components error:", err)
	}
	if _, err := t.ParseGlob("views/components/ui/*.html"); err != nil {
		log.Println("Warning: UI Components error:", err)
	}

	// 3. Load Partials (explicit subdirectories)
	if _, err := t.ParseGlob("views/partials/tech/*.html"); err != nil {
		log.Println("Warning: Tech Partials error:", err)
	}
	if _, err := t.ParseGlob("views/partials/admin/*.html"); err != nil {
		log.Println("Warning: Admin Partials error:", err)
	}
	if _, err := t.ParseGlob("views/pages/admin/partials/*.html"); err != nil {
		log.Println("Warning: Admin Page Partials error:", err)
	}

	log.Printf("✅ Loaded Templates: %q", t.DefinedTemplates())

	return t, nil
}
