// assets/js/public.js

console.log('✅ Public JS Loaded');

/**
 * 1. BOOKING WIZARD CONTROLLER
 * Quản lý logic của Form đặt lịch 4 bước bên trong Modal
 */
window.bookingWizard = function () {
    return {
        step: 1,
        locationStatus: '',
        selectedDate: '',
        minDate: '',
        loadingSlots: false,
        availableSlots: [],
        submitting: false,

        // Dữ liệu form
        formData: {
            serviceId: '',
            serviceName: '',
            servicePrice: 0,
            name: '',
            phone: '',
            address: '',
            issue: '',
            deviceType: 'ac_split',
            brand: '',
            time: '',      // YYYY-MM-DD
            slotId: '',
            lat: '',
            long: ''
        },

        init() {
            // Cấu hình ngày tối thiểu (ngày mai)
            const tomorrow = new Date();
            tomorrow.setDate(tomorrow.getDate() + 1);
            this.minDate = tomorrow.toISOString().split('T')[0];
            this.selectedDate = this.minDate;
            this.formData.time = this.minDate;

            // [QUAN TRỌNG] Lắng nghe sự kiện mở modal để reset hoặc điền sẵn dữ liệu
            window.addEventListener('open-booking-modal', (e) => {
                this.resetForm();

                // Nếu có dữ liệu truyền vào (từ nút "Chọn dịch vụ này" ở trang chủ)
                if (e.detail && e.detail.serviceId) {
                    this.selectService(e.detail.serviceId, e.detail.serviceName, e.detail.servicePrice);
                }
            });
        },

        // Reset form về trạng thái ban đầu
        resetForm() {
            this.step = 1;
            this.formData.serviceId = '';
            this.formData.slotId = '';
            this.submitting = false;
            // Giữ lại tên/sđt/địa chỉ nếu khách đã nhập để tiện lợi
        },

        // Chọn dịch vụ và tự động chuyển bước 2
        selectService(id, name, price) {
            this.formData.serviceId = id;
            this.formData.serviceName = name;
            this.formData.servicePrice = price;
            // Delay nhẹ tạo trải nghiệm mượt mà
            setTimeout(() => {
                if (this.step === 1) this.nextStep();
            }, 100);
        },

        // Lấy danh sách khung giờ trống từ Backend
        async fetchSlots() {
            if (!this.selectedDate) return;
            this.loadingSlots = true;
            this.availableSlots = [];
            this.formData.slotId = '';
            this.formData.time = this.selectedDate;

            try {
                const response = await fetch(`/api/slots/available?date=${this.selectedDate}`);
                if (response.ok) {
                    this.availableSlots = await response.json();
                }
            } catch (error) {
                console.error('Error fetching slots:', error);
                Swal.fire('Lỗi', 'Không thể tải lịch trống. Vui lòng thử lại sau.', 'error');
            } finally {
                this.loadingSlots = false;
            }
        },

        // Định vị GPS
        getLocation() {
            this.locationStatus = 'Đang lấy vị trí...';
            if (!navigator.geolocation) {
                this.locationStatus = 'Trình duyệt không hỗ trợ định vị.';
                return;
            }

            navigator.geolocation.getCurrentPosition(
                async (position) => {
                    this.formData.lat = position.coords.latitude;
                    this.formData.long = position.coords.longitude;

                    // Gọi API Proxy (Backend) để tránh CORS và bảo mật
                    try {
                        const res = await fetch(`/api/public/reverse-geocode?lat=${this.formData.lat}&lon=${this.formData.long}`);
                        const data = await res.json();
                        if (data && data.display_name) {
                            this.formData.address = data.display_name;
                            this.locationStatus = 'Đã định vị thành công!';
                        } else {
                            this.locationStatus = 'Đã lấy tọa độ. Vui lòng nhập thêm số nhà.';
                        }
                    } catch (e) {
                        this.locationStatus = 'Đã ghim tọa độ. Vui lòng nhập địa chỉ cụ thể.';
                    }
                },
                (err) => {
                    this.locationStatus = 'Không thể lấy vị trí. Vui lòng nhập tay.';
                }
            );
        },

        // Chuyển bước tiếp theo với Validate
        nextStep() {
            // Validate Bước 2 (Thông tin)
            if (this.step === 2) {
                if (!this.formData.name || !this.formData.phone || !this.formData.address) {
                    Swal.fire('Thiếu thông tin', 'Vui lòng điền Họ tên, SĐT và Địa chỉ.', 'warning');
                    return;
                }
                // Pre-fetch slots cho bước 3
                this.fetchSlots();
            }

            // Validate Bước 3 (Thời gian)
            if (this.step === 3 && !this.formData.slotId) {
                Swal.fire('Chưa chọn giờ', 'Vui lòng chọn một khung giờ phù hợp.', 'warning');
                return;
            }

            if (this.step < 4) {
                this.step++;
                // Cuộn lên đầu modal mobile
                const modalBox = document.querySelector('.modal-box');
                if (modalBox) modalBox.scrollTop = 0;
            }
        },

        // Quay lại bước trước
        prevStep() {
            if (this.step > 1) {
                this.step--;
            }
        },

        // Hiển thị thời gian đã chọn dạng text
        getSelectedSlotDisplay() {
            const slot = this.availableSlots.find(s => s.ID === this.formData.slotId);
            if (slot) return `${this.formatDate(this.selectedDate)} | ${slot.StartTime.slice(0, 5)} - ${slot.EndTime.slice(0, 5)}`;
            return 'Chưa chọn';
        },

        // Format ngày tháng (dd/mm/yyyy)
        formatDate(dateStr) {
            if (!dateStr) return '';
            const [y, m, d] = dateStr.split('-');
            return `${d}/${m}/${y}`;
        },

        // Format tiền tệ
        formatMoney(amount) {
            return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
        },

        // Xử lý gửi Form
        async submitBooking() {
            this.submitting = true;

            const data = new FormData();
            // Map dữ liệu vào FormData
            data.append('serviceId', this.formData.serviceId);
            data.append('service_id', this.formData.serviceId); // Backup case
            data.append('customer_name', this.formData.name);
            data.append('customer_phone', this.formData.phone);
            data.append('address', this.formData.address);
            data.append('device_type', this.formData.deviceType);
            data.append('brand', this.formData.brand);
            data.append('issue_description', this.formData.issue);
            data.append('time', this.formData.time);
            data.append('slot_id', this.formData.slotId);
            data.append('lat', this.formData.lat);
            data.append('long', this.formData.long);

            try {
                const response = await fetch('/book', {
                    method: 'POST',
                    body: data
                });

                if (response.ok) {
                    // Đóng modal từ controller cha
                    window.dispatchEvent(new CustomEvent('close-booking-modal'));

                    Swal.fire({
                        title: 'Đã Gửi Yêu Cầu!',
                        html: '<p class="text-lg">Cảm ơn quý khách đã tin tưởng dịch vụ.</p><p class="mt-2 text-slate-600">Kỹ thuật viên sẽ gọi điện xác nhận trong giây lát.<br><strong>Vui lòng để ý điện thoại!</strong> <i class="fa-solid fa-mobile-screen-button text-blue-500 animate-pulse ml-1"></i></p>',
                        icon: 'success',
                        showConfirmButton: false, // Ẩn nút để tập trung vào thông điệp
                        timer: 4000,              // Tự động đóng sau 4s
                        timerProgressBar: true,
                        backdrop: `rgba(0,0,123,0.4)`
                    }).then(() => {
                        // Luôn redirect về trang chủ sau khi xong
                        window.location.href = '/';
                    });
                } else {
                    const text = await response.text();
                    Swal.fire('Lỗi', text || 'Có lỗi xảy ra, vui lòng thử lại.', 'error');
                }
            } catch (error) {
                console.error(error);
                Swal.fire('Lỗi kết nối', 'Vui lòng kiểm tra đường truyền mạng.', 'error');
            } finally {
                this.submitting = false;
            }
        }
    };
};

/**
 * 2. PAGE CONTROLLER
 * Quản lý trạng thái Modal (Mở/Đóng) và các tương tác chung trên trang
 */
window.pageController = function () {
    return {
        bookingModalOpen: false,

        init() {
            // Lắng nghe sự kiện mở modal từ bất kỳ đâu (Navbar, Button...)
            window.addEventListener('open-booking-modal', () => {
                this.bookingModalOpen = true;
            });

            // Lắng nghe sự kiện đóng modal (khi đặt lịch thành công)
            window.addEventListener('close-booking-modal', () => {
                this.bookingModalOpen = false;
            });
        },

        // Hàm gọi modal
        openModal() {
            this.bookingModalOpen = true;
            // Bắn sự kiện để Wizard bên trong reset form
            window.dispatchEvent(new CustomEvent('open-booking-modal'));
        },

        closeModal() {
            this.bookingModalOpen = false;
        },

        // Hàm dùng cho nút "Chọn dịch vụ này" ở danh sách Services
        triggerBooking(id, name, price) {
            this.bookingModalOpen = true;
            // Bắn sự kiện kèm dữ liệu dịch vụ để Wizard tự điền
            window.dispatchEvent(new CustomEvent('open-booking-modal', {
                detail: { serviceId: id, serviceName: name, servicePrice: price }
            }));
        }
    };
};