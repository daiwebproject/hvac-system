package ui

import (
	"bytes"
	"html/template"

	"github.com/pocketbase/pocketbase/core"
)

// Components handles UI rendering logic separated from business handlers
type Components struct {
	App       core.App
	Templates *template.Template
}

// RenderBookingRow renders a single booking table row for admin dashboard
// This is used by both the initial page load and SSE realtime updates
func (c *Components) RenderBookingRow(record *core.Record) (string, error) {
	// Expand technician relation
	c.App.ExpandRecord(record, []string{"technician_id"}, nil)

	// Fetch available technicians for the assignment dropdown
	techs, _ := c.App.FindRecordsByFilter("technicians", "active=true", "name", 100, 0, nil)

	var buf bytes.Buffer
	// Clone templates to avoid "cannot Clone ... after it has executed" error on shared instance
	tmpl, err := c.Templates.Clone()
	if err != nil {
		return "", err
	}

	err = tmpl.ExecuteTemplate(&buf, "booking_row", map[string]interface{}{
		"Record":      record,
		"Technicians": techs,
	})
	return buf.String(), err
}
