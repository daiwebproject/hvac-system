package main

import (
	"fmt"
	"html/template"
	"log"
)

func main() {
	funcMap := template.FuncMap{
		"dict":        func(values ...interface{}) (map[string]interface{}, error) { return nil, nil },
		"formatMoney": func(val interface{}) string { return "" },
		"sub":         func(a, b float64) float64 { return 0 },
		"mul":         func(a, b float64) float64 { return 0 },
		"div":         func(a, b float64) float64 { return 0 },
		"substr":      func(start, length int, s string) string { return "" },
		"slice":       func(s string, i, j int) string { return "" },
	}

	t := template.New("").Funcs(funcMap)

	// Simulate InitTemplates logic
	patterns := []string{
		"views/layouts/*.html",
		"views/components/*.html",
		"views/components/ui/*.html",
		"views/partials/tech/*.html",
		"views/partials/admin/*.html",
	}

	for _, pattern := range patterns {
		_, err := t.ParseGlob(pattern)
		if err != nil {
			log.Printf("Error parsing pattern %s: %v\n", pattern, err)
		}
	}

	fmt.Println("Defined Templates:")
	for _, tmpl := range t.Templates() {
		fmt.Println("- " + tmpl.Name())
	}

	// Check specific one
	target := "tech/partials/jobs_list"
	if t.Lookup(target) != nil {
		fmt.Printf("\n✅ FOUND: %s\n", target)
	} else {
		fmt.Printf("\n❌ NOT FOUND: %s\n", target)
	}
}
