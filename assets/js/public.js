// assets/js/public.js

window.bookingWizard = function () {
    return {
        step: 1,
        locationStatus: '',
        selectedDate: '',
        minDate: '',
        loadingSlots: false,
        availableSlots: [],
        formData: {
            serviceId: '',
            serviceName: '',
            name: '',
            phone: '',
            address: '',
            issue: '',
            deviceType: 'ac_split',
            brand: '',
            time: '',
            slotId: '',
            lat: '',
            long: ''
        },

        init() {
            // Set min date to tomorrow
            const tomorrow = new Date();
            tomorrow.setDate(tomorrow.getDate() + 1);
            this.minDate = tomorrow.toISOString().split('T')[0];
            this.selectedDate = this.minDate;
        },

        async fetchSlots() {
            if (!this.selectedDate) return;
            this.loadingSlots = true;
            this.availableSlots = [];
            this.formData.slotId = '';
            try {
                const response = await fetch(`/api/slots/available?date=${this.selectedDate}`);
                if (response.ok) this.availableSlots = await response.json();
            } catch (error) {
                console.error('Error fetching slots:', error);
            } finally {
                this.loadingSlots = false;
            }
        },

        getLocation() {
            this.locationStatus = 'Đang lấy vị trí (vui lòng cho phép)...';

            if (!navigator.geolocation) {
                this.locationStatus = 'Trình duyệt không hỗ trợ định vị.';
                return;
            }

            const options = {
                enableHighAccuracy: true,
                timeout: 10000,
                maximumAge: 0
            };

            navigator.geolocation.getCurrentPosition(
                async (position) => {
                    this.formData.lat = position.coords.latitude;
                    this.formData.long = position.coords.longitude;
                    this.locationStatus = 'Đã lấy tọa độ. Đang tìm địa chỉ...';

                    try {
                        const res = await fetch(`https://nominatim.openstreetmap.org/reverse?format=json&lat=${this.formData.lat}&lon=${this.formData.long}&zoom=18&addressdetails=1`);
                        const data = await res.json();
                        if (data && data.display_name) {
                            // Format address nicely if possible, or just use display_name
                            this.formData.address = data.display_name;
                            this.locationStatus = 'Định vị thành công!';
                        } else {
                            this.locationStatus = 'Đã ghim tọa độ. Vui lòng điền địa chỉ cụ thể.';
                        }
                    } catch (e) {
                        console.error(e);
                        this.locationStatus = 'Đã lấy tọa độ (Lỗi tìm tên đường). Vui lòng nhập địa chỉ.';
                    }
                },
                (err) => {
                    console.error(err);
                    let msg = 'Không thể lấy vị trí.';
                    switch (err.code) {
                        case err.PERMISSION_DENIED:
                            msg = 'Bạn đã từ chối quyền truy cập vị trí. Hãy bật lại trong cài đặt Safari/Trình duyệt.';
                            break;
                        case err.POSITION_UNAVAILABLE:
                            msg = 'Không tìm thấy vị trí hiện tại.';
                            break;
                        case err.TIMEOUT:
                            msg = 'Quá thời gian chờ lấy vị trí. Hãy thử lại.';
                            break;
                    }
                    this.locationStatus = msg;
                },
                options
            );
        },

        nextStep() {
            if (this.step === 2) this.fetchSlots();
            if (this.step < 4) this.step++;
        },

        setService(name) {
            this.formData.serviceName = name;
        },

        getServiceName() {
            return this.formData.serviceName || "Dịch vụ đã chọn";
        },

        getSelectedSlotDisplay() {
            const slot = this.availableSlots.find(s => s.ID === this.formData.slotId);
            if (slot) return `${this.selectedDate} | ${slot.StartTime} - ${slot.EndTime}`;
            return '';
        }
    };
};
// File: assets/js/public.js

console.log('✅ Public JS Loaded');

// --- Page Controller cho Trang Chủ (Đặt lịch) ---
window.pageController = function () {
    return {
        bookingModalOpen: false,
        step: 1,
        formData: {
            serviceId: '',
            serviceName: '',
            customerName: '',
            phone: '',
            address: '',
            time: '',
            lat: '',
            long: ''
        },

        // Mở modal (mặc định vào bước 1)
        openModal() {
            this.step = 1;
            this.bookingModalOpen = true;
        },

        // Đóng modal
        closeModal() {
            this.bookingModalOpen = false;
        },

        // Khi user chọn "Chọn dịch vụ này" ở trang chủ
        prefillService(id, name) {
            this.formData.serviceId = id;
            this.formData.serviceName = name;
            this.step = 2; // Nhảy thẳng vào bước điền thông tin
            this.bookingModalOpen = true;
        },

        // Khi user chọn dịch vụ trong Modal (Step 1)
        selectService(id, name) {
            this.formData.serviceId = id;
            this.formData.serviceName = name;
            this.step = 2;
        },

        validateStep2() {
            if (!this.formData.customerName || !this.formData.phone || !this.formData.address || !this.formData.time) {
                Swal.fire('Thiếu thông tin', 'Vui lòng điền đầy đủ các trường bắt buộc (*)', 'warning');
                return false;
            }
            return true;
        },

        getCurrentLocation() {
            if (navigator.geolocation) {
                navigator.geolocation.getCurrentPosition(pos => {
                    this.formData.lat = pos.coords.latitude;
                    this.formData.long = pos.coords.longitude;
                    Swal.fire({
                        title: 'Đã lấy vị trí!',
                        text: 'Tọa độ chính xác giúp thợ đến nhanh hơn.',
                        icon: 'success',
                        timer: 1500,
                        showConfirmButton: false
                    });
                }, () => Swal.fire('Lỗi', 'Không thể lấy vị trí. Vui lòng nhập địa chỉ thủ công.', 'error'));
            } else {
                Swal.fire('Lỗi', 'Trình duyệt không hỗ trợ vị trí.', 'error');
            }
        },

        formatDate(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            return date.toLocaleString('vi-VN', {
                hour: '2-digit', minute: '2-digit',
                day: '2-digit', month: '2-digit', year: 'numeric'
            });
        },

        submitBooking() {
            const data = new FormData();
            for (const key in this.formData) {
                data.append(key, this.formData[key]);
            }

            // Hiển thị loading
            Swal.fire({ title: 'Đang xử lý...', didOpen: () => Swal.showLoading() });

            fetch('/book', { method: 'POST', body: data })
                .then(res => {
                    if (res.ok) {
                        this.closeModal();
                        Swal.fire({
                            title: 'Đặt Lịch Thành Công!',
                            text: 'Cảm ơn bạn! Chúng tôi sẽ gọi lại xác nhận trong ít phút.',
                            icon: 'success'
                        });
                        // Reset form sau khi gửi thành công
                        this.formData = {
                            serviceId: '', serviceName: '',
                            customerName: '', phone: '', address: '', time: '',
                            lat: '', long: ''
                        };
                        this.step = 1;
                    } else {
                        res.text().then(text => Swal.fire('Lỗi', text || 'Có lỗi xảy ra', 'error'));
                    }
                })
                .catch(err => Swal.fire('Lỗi', 'Lỗi kết nối mạng', 'error'));
        }
    };
};