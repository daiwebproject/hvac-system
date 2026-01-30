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
            this.locationStatus = 'Đang lấy vị trí...';
            if (!navigator.geolocation) {
                this.locationStatus = 'Trình duyệt không hỗ trợ vị trí.';
                return;
            }
            navigator.geolocation.getCurrentPosition(
                async (position) => {
                    this.formData.lat = position.coords.latitude;
                    this.formData.long = position.coords.longitude;
                    this.locationStatus = 'Đã lấy tọa độ. Đang tìm địa chỉ...';
                    try {
                        const res = await fetch(`https://nominatim.openstreetmap.org/reverse?format=json&lat=${this.formData.lat}&lon=${this.formData.long}&zoom=18&addressdetails=1`);
                        const data = await res.json();
                        if (data && data.display_name) {
                            this.formData.address = data.display_name;
                            this.locationStatus = 'Đã cập nhật vị trí và địa chỉ!';
                        } else {
                            this.locationStatus = 'Đã ghim tọa độ. Vui lòng nhập địa chỉ cụ thể.';
                        }
                    } catch (e) {
                        console.error(e);
                        this.locationStatus = 'Đã ghim tọa độ. Không thể lấy tên đường (Lỗi mạng).';
                    }
                },
                (err) => {
                    console.error(err);
                    this.locationStatus = 'Không thể lấy vị trí. Hãy kiểm tra quyền truy cập hoặc nhập tay.';
                }
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