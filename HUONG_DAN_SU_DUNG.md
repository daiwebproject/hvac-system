# HƯỚNG DẪN SỬ DỤNG VÀ BẢO TRÌ HỆ THỐNG HVAC

Tài liệu này cung cấp hướng dẫn chi tiết về cách vận hành, sử dụng và bảo trì hệ thống quản lý dịch vụ điện lạnh (HVAC System).

## 1. Tổng Quan Hệ Thống

Hệ thống bao gồm 3 phân hệ chính:
1.  **Dành cho Khách hàng (Public Web):** Đặt lịch dịch vụ trực tuyến.
2.  **Dành cho Quản trị viên (Admin Dashboard):** Quản lý đơn hàng (Kanban), theo dõi đội thợ (Bản đồ GPS), quản lý kho và báo cáo thống kê.
3.  **Dành cho Kỹ thuật viên (Tech Mobile View):** Nhận việc, cập nhật trạng thái (Di chuyển/Đang làm/Hoàn thành), báo cáo nghiệm thu.

## 2. Yêu Cầu Cài Đặt

*   **Hệ điều hành:** Linux, macOS, hoặc Windows.
*   **Ngôn ngữ lập trình:** Go (Golang) phiên bản 1.22 trở lên.
*   **Cơ sở dữ liệu:** SQLite (Tự động tích hợp sẵn qua PocketBase).

## 3. Khởi Chạy Hệ Thống

Để chạy server, mở terminal tại thư mục gốc của dự án và chạy lệnh:

```bash
go run main.go serve
```

Server sẽ khởi động tại: `http://localhost:8090`

> **Lưu ý:** Nếu cổng 8090 bận, hãy tắt tiến trình cũ hoặc đổi cổng trong code (mặc định PocketBase dùng 8090).

## 4. Hướng Dẫn Sử Dụng

### A. Dành cho Quản Trị Viên (Admin)

*   **Truy cập:** [http://localhost:8090/admin](http://localhost:8090/admin)
*   **Tài khoản mặc định:** `admin@dienlanhpro.vn` / `Admin@123456`

**Các tính năng chính:**
1.  **Dashboard "Cockpit":**
    *   **Kanban Board:** Kéo thả các thẻ đơn hàng để chuyển trạng thái (Pending -> Assigned -> In Progress -> Completed).
    *   **Live Fleet Map:** Bản đồ hiển thị vị trí thực của kỹ thuật viên đang hoạt động.
    *   **Analytics:** Biểu đồ doanh thu 7 ngày và bảng xếp hạng kỹ thuật viên.
2.  **Giao việc:**
    *   Tại cột **Pending**, nhấn nút "Giao việc" trên thẻ đơn hàng.
    *   Chọn kỹ thuật viên từ danh sách. Hệ thống sẽ kiểm tra trùng lịch tự động.
3.  **Quản lý Kho (Inventory):**
    *   Truy cập menu "Quick Tools" -> "Inventory" để nhập/xuất vật tư.

### B. Dành cho Kỹ Thuật Viên (Technician)

*   **Truy cập:** [http://localhost:8090/tech/login](http://localhost:8090/tech/login) (Giao diện tối ưu cho Mobile)
*   **Tài khoản mẫu:** `tech@demo.com` / `12345678`

**Quy trình làm việc:**
1.  **Nhận việc:** Xem danh sách đơn hàng được giao tại trang chủ.
2.  **Di chuyển:** Nhấn "Bắt đầu di chuyển" -> Trạng thái đơn chuyển sang "Moving" (Gửi định vị GPS về cho Admin).
3.  **Check-in:** Khi đến nơi, nhấn "Check-in" -> Trạng thái "Working".
4.  **Hoàn thành:**
    *   Nhấn "Hoàn thành công việc".
    *   Chọn vật tư tiêu hao (Gas, Ống đồng...).
    *   **Quan trọng:** Phải chụp/tải lên ảnh nghiệm thu để đóng đơn.

### C. Dành cho Khách Hàng

*   **Truy cập:** [http://localhost:8090/book](http://localhost:8090/book)
*   Khách hàng chọn dịch vụ, điền thông tin và thời gian mong muốn để tạo yêu cầu đặt lịch.

## 5. Hướng Dẫn Bảo Trì & Xử Lý Sự Cố

### Backup Dữ Liệu
Toàn bộ dữ liệu nằm trong thư mục `pb_data`.
*   **Sao lưu:** Copy thư mục `pb_data` sang nơi an toàn định kỳ.
*   **Phục hồi:** Xóa `pb_data` hiện tại và chép lại bản backup.

### Quản Lý Dữ Liệu (Backend UI)
PocketBase cung cấp giao diện quản lý dữ liệu gốc (Super Admin UI).
*   **Truy cập:** [http://localhost:8090/_/](http://localhost:8090/_/)
*   Dùng tài khoản Admin để đăng nhập.
*   Tại đây bạn có thể sửa trực tiếp dữ liệu trong các bảng `bookings`, `technicians`, `invoices` nếu có sai sót hệ thống.

### Cập Nhật Ứng Dụng
1.  Kéo code mới từ Git.
2.  Chạy lại lệnh `go run main.go serve`.
3.  Hệ thống sẽ **tự động chạy Migrations** (cập nhật cấu trúc database) nếu có thay đổi trong thư mục `migrations/`.

### Các Lỗi Thường Gặp
*   **Lỗi "bind: address already in use":** Do server cũ chưa tắt hẳn.
    *   *Khắc phục:* Chạy `fuser -k 8090/tcp` (Linux) hoặc khởi động lại máy.
*   **Không thấy vị trí trên bản đồ:**
    *   *Nguyên nhân:* Kỹ thuật viên chưa bật chia sẻ vị trí hoặc chưa đăng nhập.
    *   *Khắc phục:* Yêu cầu kỹ thuật viên thao tác trên giao diện Mobile để gửi "Heartbeat".

---
**Hỗ trợ kỹ thuật:** Liên hệ đội ngũ phát triển Antigravity.
