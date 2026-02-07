// assets/js/public.js

console.log('✅ Public JS Loaded (v14) - Simplified GPS');

/**
 * 1. BOOKING WIZARD CONTROLLER
 * Quản lý logic của Form đặt lịch 4 bước bên trong Modal
 */
window.bookingWizard = function () {
    return {
        step: 1,
        locationStatus: '',
        showMapModal: false,
        mapInstance: null,
        mapMarker: null,
        mapCenter: { lat: 21.0285, lng: 105.8542 }, // Hanoi default
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

        // Chọn Slot với cảnh báo nếu là Waitlist/Limited
        selectSlot(slot) {
            if (slot.Status === 'full') return;

            // Nếu slot là Waitlist hoặc Limited -> Cảnh báo
            if (slot.Status === 'waitlist' || slot.Status === 'limited') {
                Swal.fire({
                    title: 'Khung giờ cao điểm',
                    html: `Khung giờ <b>${slot.StartTime.substring(0, 5)}</b> đang quá tải.<br>Chúng tôi sẽ cố gắng điều phối thợ và xác nhận lại trong vòng 15 phút.<br><br>Bạn có muốn tiếp tục đặt chờ không?`,
                    icon: 'warning',
                    showCancelButton: true,
                    confirmButtonText: 'Đồng ý đặt chờ',
                    cancelButtonText: 'Chọn giờ khác',
                    confirmButtonColor: '#f97316' // Orange
                }).then((result) => {
                    if (result.isConfirmed) {
                        this.formData.slotId = slot.ID;
                    }
                });
            } else {
                // Available -> Chọn ngay
                this.formData.slotId = slot.ID;
            }
        },

        // Lấy danh sách khung giờ trống từ Backend
        // [Smart Booking] Truyền zone và serviceId để lọc theo khu vực và kỹ năng
        async fetchSlots() {
            if (!this.selectedDate) return;
            this.loadingSlots = true;
            this.availableSlots = [];
            this.formData.slotId = '';
            this.formData.time = this.selectedDate;

            try {
                // Build URL with optional filters
                let url = `/api/slots/available?date=${this.selectedDate}`;

                // Add zone filter (use address as zone identifier)
                if (this.formData.address) {
                    url += `&zone=${encodeURIComponent(this.formData.address)}`;
                }

                // Add service filter for skill-based matching
                if (this.formData.serviceId) {
                    url += `&service_id=${encodeURIComponent(this.formData.serviceId)}`;
                }

                const response = await fetch(url);
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

            // Helper: Detect Environment
            const ua = navigator.userAgent || navigator.vendor || window.opera;
            const isChrome = /Chrome/.test(ua) && /Google Inc/.test(navigator.vendor);
            const isIOS = /iPad|iPhone|iPod/.test(ua) && !window.MSStream;
            const isInApp = /FBAN|FBAV|Instagram|Zalo|Line/.test(ua);

            if (!navigator.geolocation) {
                this.locationStatus = 'Trình duyệt không hỗ trợ.';
                this.suggestChrome(true);
                return;
            }

            const options = {
                enableHighAccuracy: true,
                timeout: 8000,
                maximumAge: 0
            };

            navigator.geolocation.getCurrentPosition(
                async (position) => {
                    this.formData.lat = position.coords.latitude;
                    this.formData.long = position.coords.longitude;
                    this.reverseGeocode(this.formData.lat, this.formData.long);
                },
                (err) => {
                    console.warn(`Geolocation Error (${err.code}): ${err.message}`);

                    let title = 'Lỗi GPS';
                    let html = 'Không thể lấy vị trí. Vui lòng thử lại hoặc chọn trên bản đồ.';
                    let icon = 'warning';

                    if (err.code === 1) { // PERMISSION_DENIED
                        this.locationStatus = 'Quyền vị trí bị chặn.';
                        title = 'Cần quyền truy cập vị trí';

                        if (isIOS) {
                            // iOS Safari Instructions + PWA Hint
                            html = `<div class="text-left text-sm space-y-2">
                                <p><strong>Cách 1 (Nhanh nhất):</strong> Bật vị trí cho Safari:</p>
                                <ol class="list-decimal pl-5 space-y-1">
                                    <li>Bấm <b>'Aa'</b> (hoặc 🔒) trên thanh địa chỉ.</li>
                                    <li>Chọn <b>Cài đặt trang web</b> → <b>Vị trí</b> → <b>Cho phép</b>.</li>
                                </ol>
                                <hr class="my-2"/>
                                <p><strong>Cách 2 (Khuyên dùng):</strong> Thêm vào màn hình chính để tự động bật GPS mỗi khi vào:</p>
                                <ol class="list-decimal pl-5 space-y-1">
                                    <li>Bấm nút <b>Chia sẻ</b> <i class="fa-solid fa-arrow-up-from-bracket"></i></li>
                                    <li>Chọn <b>Thêm vào MH chính</b> (Add to Home Screen)</li>
                                </ol>
                            </div>`;
                            icon = 'info';

                            // [Fallback] Fetch IP Location silently
                            this.getIPLocation().then(data => {
                                if (data) {
                                    console.log('🌍 IP Location Found:', data);
                                    // Optionally update map center even if modal is open
                                    this.mapCenter = { lat: data.lat, lng: data.lon };
                                }
                            });

                        } else {
                            html = 'Bạn đã chặn quyền vị trí. Vui lòng <b>Cho phép</b> trong cài đặt trình duyệt hoặc chuyển sang <b>Google Chrome</b>.';

                            // [Fallback] Fetch IP Location
                            this.getIPLocation().then(data => {
                                if (data) {
                                    this.formData.lat = data.lat;
                                    this.formData.long = data.lon;
                                    this.mapCenter = { lat: data.lat, lng: data.lon };
                                    this.locationStatus = `Đã lấy vị trí gần đúng (IP: ${data.city})`;

                                    // Auto reverse geocode roughly
                                    this.reverseGeocode(data.lat, data.lon);
                                }
                            });
                        }

                    } else if (err.code === 3 || err.code === 2) { // TIMEOUT / UNAVAILABLE
                        this.locationStatus = 'Không tìm thấy GPS.';
                        title = 'Không tìm thấy tín hiệu';
                        html = 'Vui lòng kiểm tra GPS hoặc chuyển sang <b>Google Chrome</b> để chính xác hơn.';
                    }

                    // Auto suggest Chrome or Show iOS Guide
                    Swal.fire({
                        title: title,
                        html: html,
                        icon: icon,
                        confirmButtonText: 'Chọn trên bản đồ 🗺️',
                        showCancelButton: false, // Hide cancel button to focus on Map or Instructions
                        footer: isIOS ? '<span class="text-xs text-gray-500">Mẹo: Thêm vào màn hình chính để dùng App mượt mà hơn!</span>' : ''
                    }).then((result) => {
                        if (result.isConfirmed) {
                            this.showMap(); // Fallback to map immediately
                        } else if (result.dismiss === Swal.DismissReason.cancel) {
                            // Try to open Chrome (Android mainly)
                            const url = window.location.href;
                            if (/Android/i.test(navigator.userAgent)) {
                                window.location.href = `intent://${url.replace(/^https?:\/\//, '')}#Intent;scheme=https;package=com.android.chrome;end`;
                            }
                        }
                    });
                },
                options
            );
        },

        suggestChrome() {
            // Simplified suggestion
        },

        async getIPLocation() {
            // [Cache] Check localStorage first (cache for 1 hour)
            const cached = localStorage.getItem('ipGeoCache');
            if (cached) {
                try {
                    const { data, timestamp } = JSON.parse(cached);
                    if (Date.now() - timestamp < 3600000) { // 1 hour
                        console.log('📍 Using cached IP location:', data.city);
                        return data;
                    }
                } catch (e) { }
            }

            // [Fallback Chain] Try multiple APIs in sequence
            const apis = [
                {
                    name: 'ipwho.is',
                    url: 'https://ipwho.is/',
                    parse: (d) => d.success ? { lat: d.latitude, lon: d.longitude, city: d.city, country: d.country, ip: d.ip } : null
                },
                {
                    name: 'ipapi.co',
                    url: 'https://ipapi.co/json/',
                    parse: (d) => d.latitude ? { lat: d.latitude, lon: d.longitude, city: d.city, country: d.country_name, ip: d.ip } : null
                },
                {
                    name: 'ip-api.com',
                    url: 'http://ip-api.com/json/?fields=status,lat,lon,city,country,query',
                    parse: (d) => d.status === 'success' ? { lat: d.lat, lon: d.lon, city: d.city, country: d.country, ip: d.query } : null
                }
            ];

            for (const api of apis) {
                try {
                    const response = await fetch(api.url, { timeout: 3000 });
                    if (!response.ok) continue;

                    const data = await response.json();
                    const result = api.parse(data);

                    if (result) {
                        console.log(`📍 IP Location from ${api.name}:`, result.city);
                        // Cache the result
                        localStorage.setItem('ipGeoCache', JSON.stringify({ data: result, timestamp: Date.now() }));
                        return result;
                    }
                } catch (err) {
                    console.warn(`IP API ${api.name} failed:`, err.message);
                }
            }

            // [Final Fallback] Return default Hanoi center
            console.log('📍 Using default location (Hanoi)');
            return { lat: 21.0285, lon: 105.8542, city: 'Hà Nội', country: 'Vietnam', ip: 'fallback' };
        },

        // Mở bản đồ chọn vị trí thủ công
        async showMap() {
            // [Logic] 1. Get IP Location first to zone the map (if no data yet)
            if (!this.formData.lat) {
                const ipData = await this.getIPLocation();
                if (ipData) {
                    console.log('🌍 Auto-centering Map via IP:', ipData.city);
                    this.mapCenter = { lat: ipData.lat, lng: ipData.lon };
                }
            }

            this.showMapModal = true;

            this.$nextTick(() => {
                this.initMap();

                // [Logic] 2. Then try GPS automatically (if no data yet)
                // This gives better UX: User sees their city immediately (IP), then zooms to street (GPS)
                if (!this.formData.lat) {
                    // Creating a non-intrusive auto-locate
                    this.locateOnMap(true);
                }
            });
        },

        closeMap() {
            this.showMapModal = false;
            // [Cleanup] Destroy map to prevent memory leaks and state issues
            if (this.mapInstance) {
                this.mapInstance.remove();
                this.mapInstance = null;
            }
        },

        initMap() {
            // [Safety Check] Ensure Leaflet is loaded
            if (typeof L === 'undefined') {
                console.warn('Leaflet (L) is not defined. Using fallback or waiting for reload.');
                return;
            }

            // Cleanup existing instance if any (though closeMap handles it)
            if (this.mapInstance) {
                this.mapInstance.remove();
                this.mapInstance = null;
            }

            // Default center or current formData
            const lat = this.formData.lat || this.mapCenter.lat;
            const lng = this.formData.long || this.mapCenter.lng;

            console.log('🗺️ Initializing Map at:', lat, lng);

            this.mapInstance = L.map('booking-map').setView([lat, lng], 15);

            const tiles = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '© OpenStreetMap'
            });

            tiles.addTo(this.mapInstance);

            // Add center icon behavior
            this.mapInstance.on('moveend', () => {
                const center = this.mapInstance.getCenter();
                this.mapCenter = center;
            });

            // [Fix] Force resize after modal animation
            setTimeout(() => {
                this.mapInstance.invalidateSize();
            }, 300);
        },

        async confirmLocation() {
            const center = this.mapInstance.getCenter();
            this.formData.lat = center.lat;
            this.formData.long = center.lng;

            await this.reverseGeocode(center.lat, center.lng);
            this.closeMap();
        },

        // Locate GPS and fly map to current position
        locateOnMap() {
            if (!navigator.geolocation) {
                Swal.fire('Không hỗ trợ', 'Trình duyệt không hỗ trợ định vị GPS. Vui lòng dùng Google Chrome để có độ chính xác cao nhất.', 'warning');
                return;
            }

            Swal.fire({
                title: 'Đang định vị...',
                text: 'Vui lòng chờ',
                allowOutsideClick: false,
                didOpen: () => Swal.showLoading()
            });

            navigator.geolocation.getCurrentPosition(
                (position) => {
                    Swal.close();
                    const lat = position.coords.latitude;
                    const lng = position.coords.longitude;

                    // Fly to position with animation
                    if (this.mapInstance) {
                        this.mapInstance.flyTo([lat, lng], 16);
                        console.log('🗺️ Flew to GPS:', lat, lng);
                    }
                },
                (err) => {
                    Swal.close();

                    // Helper: Detect Environment
                    const ua = navigator.userAgent || navigator.vendor || window.opera;
                    const isIOS = /iPad|iPhone|iPod/.test(ua) && !window.MSStream;

                    let title = 'Lỗi GPS';
                    let html = 'Không thể lấy vị trí. Vui lòng thử lại hoặc chọn trên bản đồ.';
                    let icon = 'error';

                    if (err.code === 1) {
                        // Permission denied
                        title = 'Cần quyền truy cập vị trí';

                        if (isIOS) {
                            // iOS Safari Instructions + PWA Hint
                            html = `<div class="text-left text-sm space-y-2">
                                <p><strong>Cách 1 (Nhanh nhất):</strong> Bật vị trí cho Safari:</p>
                                <ol class="list-decimal pl-5 space-y-1">
                                    <li>Bấm <b>'Aa'</b> (hoặc 🔒) trên thanh địa chỉ.</li>
                                    <li>Chọn <b>Cài đặt trang web</b> → <b>Vị trí</b> → <b>Cho phép</b>.</li>
                                </ol>
                                <hr class="my-2"/>
                                <p><strong>Cách 2 (Khuyên dùng):</strong> Thêm vào màn hình chính để tự động bật GPS mỗi khi vào:</p>
                                <ol class="list-decimal pl-5 space-y-1">
                                    <li>Bấm nút <b>Chia sẻ</b> <i class="fa-solid fa-arrow-up-from-bracket"></i></li>
                                    <li>Chọn <b>Thêm vào MH chính</b> (Add to Home Screen)</li>
                                </ol>
                            </div>`;
                            icon = 'info';
                        } else {
                            html = 'Bạn đã chặn quyền vị trí. Vui lòng <b>Cho phép</b> trong cài đặt trình duyệt hoặc chuyển sang <b>Google Chrome</b>.';
                            icon = 'warning';
                        }
                    } else if (err.code === 2) {
                        msg = 'Không tìm thấy tín hiệu GPS. Vui lòng thử lại hoặc kéo bản đồ để chọn vị trí.';
                    } else if (err.code === 3) {
                        msg = 'Hết thời gian chờ định vị. Vui lòng kiểm tra kết nối mạng hoặc kéo bản đồ để chọn vị trí.';
                    }

                    Swal.fire({
                        title: title,
                        html: html,
                        icon: icon,
                        confirmButtonText: 'Đã hiểu',
                        footer: isIOS ? '<span class="text-xs text-gray-500">Mẹo: Thêm vào màn hình chính để dùng App mượt mà hơn!</span>' : ''
                    });
                },
                { enableHighAccuracy: true, timeout: 10000, maximumAge: 0 }
            );
        },

        async reverseGeocode(lat, lon) {
            this.locationStatus = 'Đang tìm địa chỉ...';
            // Gọi API Proxy (Backend) để tránh CORS và bảo mật
            try {
                const res = await fetch(`/api/public/reverse-geocode?lat=${lat}&lon=${lon}`);
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