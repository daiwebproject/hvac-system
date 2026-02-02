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

	// [FIX] Quan trọng: Chỉ trả về Partial khi request nhắm cụ thể vào container danh sách
	// (Ví dụ: khi tìm kiếm hoặc khi tạo thợ xong tự refresh danh sách)
	// Nếu bấm từ Navbar (hx-target="main-content"), nó sẽ bỏ qua dòng này và render trang đầy đủ ở dưới.
	if e.Request.Header.Get("HX-Target") == "tech-list-container" {
		return RenderPartial(h.Templates, e, "admin/partials/tech_list", data)
	}

	// Mặc định render trang đầy đủ (Full Layout)
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

	// Trigger sự kiện để HTMX tự động load lại danh sách
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
