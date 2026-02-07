package handlers

import (
	"encoding/json"
	"html/template"
	"hvac-system/pkg/services"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
)

// GET /admin/techs
func (h *AdminHandler) TechsList(e *core.RequestEvent) error {
	techs, err := h.TechService.GetAllTechs()
	if err != nil {
		return e.String(500, err.Error())
	}

	// Fetch All Services for "Skills" mapping
	allServices, _ := h.App.FindRecordsByFilter("services", "active=true", "+name", 200, 0, nil)

	// Serialize for Frontend (Alpine.js)
	type ServiceJSON struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var serviceList []ServiceJSON
	for _, s := range allServices {
		serviceList = append(serviceList, ServiceJSON{
			ID:   s.Id,
			Name: s.GetString("name"),
		})
	}
	servicesJSON, _ := json.Marshal(serviceList)
	if len(serviceList) == 0 {
		servicesJSON = []byte("[]")
	}

	// Map Service ID -> Name for template lookup
	// Map Service ID -> Name for template lookup
	skillMap := make(map[string]string)
	for _, s := range allServices {
		skillMap[s.Id] = s.GetString("name")
	}

	// Populate SkillNames for Display
	for _, t := range techs {
		var names []string
		for _, sID := range t.Skills {
			if name, ok := skillMap[sID]; ok {
				names = append(names, name)
			} else {
				names = append(names, sID) // Fallback to ID
			}
		}
		t.SkillNames = names
	}

	data := map[string]interface{}{
		"Techs":           techs,
		"AllServicesJSON": template.JS(string(servicesJSON)),
		"AllZones": []string{
			"Hà Nội", "TP Hồ Chí Minh", "Đà Nẵng", "Hải Phòng", "Cần Thơ",
			"An Giang", "Bà Rịa - Vũng Tàu", "Bắc Giang", "Bắc Kạn", "Bạc Liêu", "Bắc Ninh", "Bến Tre",
			"Bình Định", "Bình Dương", "Bình Phước", "Bình Thuận", "Cà Mau", "Cao Bằng", "Đắk Lắk",
			"Đắk Nông", "Điện Biên", "Đồng Nai", "Đồng Tháp", "Gia Lai", "Hà Giang", "Hà Nam",
			"Hà Tĩnh", "Hải Dương", "Hậu Giang", "Hòa Bình", "Hưng Yên", "Khánh Hòa", "Kiên Giang",
			"Kon Tum", "Lai Châu", "Lâm Đồng", "Lạng Sơn", "Lào Cai", "Long An", "Nam Định", "Nghệ An",
			"Ninh Bình", "Ninh Thuận", "Phú Thọ", "Phú Yên", "Quảng Bình", "Quảng Nam", "Quảng Ngãi",
			"Quảng Ninh", "Quảng Trị", "Sóc Trăng", "Sơn La", "Tây Ninh", "Thái Bình", "Thái Nguyên",
			"Thanh Hóa", "Thừa Thiên Huế", "Tiền Giang", "Trà Vinh", "Tuyên Quang", "Vĩnh Long",
			"Vĩnh Phúc", "Yên Bái",
		},
	}

	// [FIX] Quan trọng: Chỉ trả về Partial khi request nhắm cụ thể vào container danh sách
	if e.Request.Header.Get("HX-Target") == "tech-list-container" {
		return RenderPartial(h.Templates, e, "admin/partials/tech_list", data)
	}

	// Mặc định render trang đầy đủ
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

// POST /admin/techs/{id}/update
// POST /admin/techs/{id}/update
func (h *AdminHandler) UpdateTechInfo(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")

	// Parse Basic Fields
	name := e.Request.FormValue("name")
	email := e.Request.FormValue("email")
	level := e.Request.FormValue("level")

	// Parse Numbers
	baseSalary, _ := strconv.ParseFloat(e.Request.FormValue("base_salary"), 64)
	commission, _ := strconv.ParseFloat(e.Request.FormValue("commission_rate"), 64)

	// Parse Arrays (Skills & Zones)
	// Note: e.Request.Form must be parsed. PocketBase usually does this.
	// For checkboxes with same name "skills", we used e.Request.Form["skills"]
	if e.Request.Form == nil {
		e.Request.ParseMultipartForm(32 << 20)
	}

	skills := e.Request.Form["skills"]
	zones := e.Request.Form["service_zones"]

	input := services.UpdateTechInput{
		Name:           name,
		Email:          email,
		Level:          level,
		BaseSalary:     baseSalary,
		CommissionRate: commission,
		Skills:         skills,
		ServiceZones:   zones,
	}

	if err := h.TechService.UpdateTech(id, input); err != nil {
		return e.String(400, "Error updating tech: "+err.Error())
	}

	e.Response.Header().Set("HX-Trigger", "techListUpdated")
	return e.String(200, "Technician updated successfully")
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

	return e.String(200, "Status updated")
}
