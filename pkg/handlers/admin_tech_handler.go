package handlers

import (
	"github.com/pocketbase/pocketbase/core"
)

// GET /admin/techs
func (h *AdminHandler) TechsList(e *core.RequestEvent) error {
	techs, err := h.TechService.GetAllTechs()
	if err != nil {
		return e.String(500, err.Error())
	}

	data := map[string]interface{}{
		"Techs": techs,
	}

	if e.Request.Header.Get("HX-Request") == "true" {
		return RenderPartial(h.Templates, e, "admin/partials/tech_list", data)
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/tech_management.html", data)
}

// POST /admin/techs/create
func (h *AdminHandler) CreateTech(e *core.RequestEvent) error {
	name := e.Request.FormValue("name")
	email := e.Request.FormValue("email")
	password := e.Request.FormValue("password")

	if err := h.TechService.CreateTech(name, email, password); err != nil {
		return e.String(400, "Error creating tech: "+err.Error())
	}

	e.Response.Header().Set("HX-Trigger", "techListUpdated")
	return e.String(200, "Technician created successfully")
}

// POST /admin/techs/{id}/password
func (h *AdminHandler) ResetTechPassword(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	newPass := e.Request.FormValue("password")

	if err := h.TechService.ResetPassword(id, newPass); err != nil {
		return e.String(400, "Error resetting password: "+err.Error())
	}

	return e.String(200, "Password updated")
}

// POST /admin/techs/{id}/toggle
func (h *AdminHandler) ToggleTechStatus(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if err := h.TechService.ToggleActiveStatus(id); err != nil {
		return e.String(500, err.Error())
	}
	e.Response.Header().Set("HX-Trigger", "techListUpdated")
	return e.String(200, "Status updated")
}
