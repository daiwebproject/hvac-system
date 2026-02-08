package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Function map to satisfy template parsing
	funcMap := template.FuncMap{
		"dict":        func(values ...interface{}) (map[string]interface{}, error) { return nil, nil },
		"formatMoney": func(val interface{}) string { return "" },
		"sub":         func(a, b float64) float64 { return 0 },
		"mul":         func(a, b float64) float64 { return 0 },
		"div":         func(a, b float64) float64 { return 0 },
		"substr":      func(start, length int, s string) string { return "" },
		"slice":       func(s string, i, j int) string { return "" }, // Mocking slice
	}

	t := template.New("").Funcs(funcMap)

	// List of glob patterns to test
	patterns := []string{
		"views/layouts/*.html",
		"views/components/*.html",
		"views/components/ui/*.html",
		"views/partials/tech/*.html", // This is the suspicious one
		"views/partials/admin/*.html",
	}

	for _, pattern := range patterns {
		fmt.Printf("Testing pattern: %s\n", pattern)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("Glob error: %v\n", err)
			continue
		}

		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				log.Printf("Error reading %s: %v\n", match, err)
				continue
			}

			// Parse individual file to isolate error
			_, parseErr := t.New(match).Parse(string(content))
			if parseErr != nil {
				fmt.Printf("❌ ERROR in file %s:\n%v\n", match, parseErr)
			} else {
				fmt.Printf("✅ OK: %s\n", match)
			}
		}
	}
}
