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
