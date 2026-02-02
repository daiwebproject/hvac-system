// pkg/handlers/render.go

package handlers

import (
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
)

// RenderPage renders a page. It detects HTMX requests to render only the content block.
func RenderPage(t *template.Template, e *core.RequestEvent, layoutName string, pagePath string, data interface{}) error {
	// [NEW] Auto-inject Settings from Context
	settings := e.Get("Settings")

	// Helper to ensure data is a map
	var dataMap map[string]interface{}
	if data == nil {
		dataMap = make(map[string]interface{})
	} else if dm, ok := data.(map[string]interface{}); ok {
		dataMap = dm
	} else {
		// If data is not a map (e.g. struct), we might need to wrap it or handle differently.
		// For now, assuming most handlers pass map[string]interface{} or nil.
		// If struct is passed, we can't easily inject without reflection or key collision.
		// Let's wrap it if valid.
		dataMap = map[string]interface{}{
			"Data": data,
		}
	}

	if settings != nil {
		dataMap["Settings"] = settings
	}

	// [NEW] Inject LogoUrl from Context
	logoUrl := e.Get("LogoUrl")
	if logoUrl != nil {
		dataMap["LogoUrl"] = logoUrl
	}

	// Reassign data to be the updated map
	data = dataMap

	// 1. Clone template gốc
	tmpl, err := t.Clone()
	if err != nil {
		fmt.Println("❌ Template Clone Error:", err)
		return e.String(500, "Template error")
	}

	// 2. Parse file trang cụ thể (VD: views/pages/tech/job_detail.html)
	fullPath := filepath.Join("views", "pages", pagePath)
	_, err = tmpl.ParseFiles(fullPath)
	if err != nil {
		fmt.Printf("❌ Error parsing file %s: %v\n", fullPath, err)
		return e.String(500, "Page not found")
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 3. LOGIC HTMX:
	// Nếu request có Header "HX-Request" và KHÔNG PHẢI là "HX-Boosted" (hoặc tùy logic bạn muốn),
	// ta chỉ render block "content" thay vì toàn bộ layout.
	// Tuy nhiên, để đơn giản và nhất quán cho việc điều hướng trang (Navigation):
	// Ta sẽ quy định client gửi header "HX-Target=main-content" khi muốn swap trang.

	isHtmxNav := e.Request.Header.Get("HX-Request") == "true" && e.Request.Header.Get("HX-Target") == "main-content"

	if isHtmxNav {
		// Chỉ render nội dung, bỏ qua layout (header/footer)
		// Lưu ý: Các file page của bạn phải define "content"
		if err := tmpl.ExecuteTemplate(e.Response, "content", data); err != nil {
			fmt.Println("❌ Render Content Error:", err)
			return e.String(500, "Render error")
		}
	} else {
		// Render Full Layout (bao gồm cả content bên trong)
		if err := tmpl.ExecuteTemplate(e.Response, layoutName, data); err != nil {
			fmt.Println("❌ Render Layout Error:", err)
			return e.String(500, "Render error")
		}
	}

	return nil
}

// RenderPartial renders just a template file/block without any layout logic.
// Useful for small HTMX swaps (like lists, table rows, modals).
func RenderPartial(t *template.Template, e *core.RequestEvent, pagePath string, data interface{}) error {
	tmpl, err := t.Clone()
	if err != nil {
		fmt.Println("❌ Template Clone Error:", err)
		return e.String(500, "Template error")
	}

	fullPath := filepath.Join("views", "pages", pagePath+".html")
	_, err = tmpl.ParseFiles(fullPath)
	if err != nil {
		fmt.Printf("❌ Error parsing file %s: %v\n", fullPath, err)
		return e.String(500, "Page not found")
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Try executing "content" block first if defined, else potentially the whole file or a specific name?
	// For simple partials, usually the file content *is* the block or we define a block inside.
	// But `ExecuteTemplate` needs a name. If the file purely contains HTML without define, `Execute` on root is hard.
	// Convention: All partials should `{{ define "content" }}` or we simply rely on the fact that `ParseFiles` adds it to the set.
	// Let's assume partials are just HTML fragments. ParseFiles returns unique template name associated with filename.

	// Actually, safer pattern:
	// If the user provided "admin/partials/tech_list", we parse "views/pages/admin/partials/tech_list.html".
	// To execute it, we might need to know the template name derived from filename (e.g. "tech_list.html").

	fileName := filepath.Base(fullPath)
	if err := tmpl.ExecuteTemplate(e.Response, fileName, data); err != nil {
		// Callback: If define "content" is standard in your partials
		if err2 := tmpl.ExecuteTemplate(e.Response, "content", data); err2 != nil {
			fmt.Println("❌ Render Partial Error:", err, err2)
			return e.String(500, "Render error")
		}
	}

	return nil
}
