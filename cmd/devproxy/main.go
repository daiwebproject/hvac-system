package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	// 1. Cấu hình
	backendURL := "http://127.0.0.1:8090" // Địa chỉ mặc định của PocketBase
	listenAddr := ":8443"                 // Port HTTPS proxy
	certFile := "certs/cert.pem"
	keyFile := "certs/key.pem"

	// Kiểm tra certs
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		log.Fatalf("❌ Không tìm thấy file chứng chỉ SSL tại '%s'. Hãy chạy scripts/gen_certs.sh trước!", certFile)
	}

	// 2. Setup Parse URL
	target, err := url.Parse(backendURL)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Setup Proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify response to ensure no weird redirects
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Set headers forwarding
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", req.Host)
	}

	// 4. Server Handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request (optional)
		// fmt.Printf("[Proxy] %s %s\n", r.Method, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	// 5. Start Server
	fmt.Printf("🚀 HTTPS Proxy đang chạy tại: https://192.168.1.12%s (hoặc https://localhost%s)\n", listenAddr, listenAddr)
	fmt.Printf("➡️  Chuyển hướng đến: %s\n", backendURL)
	fmt.Println("⚠️  Đảm bảo bạn đã chạy 'go run main.go serve' ở một terminal khác!")

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: handler,
		TLSConfig: &tls.Config{
			// Có thể thêm cấu hình TLS nếu cần
			MinVersion: tls.VersionTLS12,
		},
	}

	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
