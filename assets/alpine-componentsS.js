// Global Alpine.js Component Definitions
// Must be loaded BEFORE Alpine.js initializes
//

/**
 * Offline Indicator Layout Component
 * Shows offline status, SSE connection status, and pending sync count
 */
window.offlineIndicator = function () {
    return {
        isOnline: navigator.onLine,
        isStreamConnected: false, // Trạng thái kết nối SSE Realtime
        pendingCount: 0,
        showSyncStatus: false,

        async updateStatus() {
            this.isOnline = navigator.onLine;
            // Nếu có mạng + Reporter sẵn sàng -> Sync ngay
            if (this.isOnline && window.OfflineJobReporter) {
                await window.OfflineJobReporter.syncPendingReports();
            }
            await this.updatePendingCount();
        },

        async updatePendingCount() {
            if (window.OfflineJobReporter) {
                const reports = await window.OfflineJobReporter.getPendingReports();
                this.pendingCount = reports.length;
            }
        },

        // Helper text hiển thị trạng thái chi tiết
        getStatusText() {
            if (!this.isOnline) return 'Mất kết nối Internet';
            if (this.pendingCount > 0) return `Đang đồng bộ (${this.pendingCount})...`;
            if (this.isStreamConnected) return 'Trực tuyến (Real-time)';
            return 'Đã kết nối';
        },

        init() {
            // 1. Lắng nghe sự kiện mạng
            window.addEventListener('online', () => {
                this.isOnline = true;
                this.updateStatus();
                // Thử kết nối lại SSE nếu cần (HTMX tự xử lý, ta chỉ update UI)
            });
            window.addEventListener('offline', () => {
                this.isOnline = false;
                this.isStreamConnected = false;
            });

            // 2. Lắng nghe trạng thái SSE từ htmx-sse.js
            document.body.addEventListener('htmx:sseOpen', () => {
                this.isStreamConnected = true;
                console.log('✅ SSE Connected');
            });
            document.body.addEventListener('htmx:sseError', () => {
                this.isStreamConnected = false;
                console.warn('⚠️ SSE Disconnected');
            });

            // 3. Lắng nghe sự kiện đồng bộ từ OfflineReporter
            window.addEventListener('report-synced', () => {
                this.updatePendingCount();
                // Hiển thị toast nhỏ nếu muốn
            });

            // Check định kỳ
            this.updatePendingCount();
            setInterval(() => this.updatePendingCount(), 10000);
        }
    };
};

/**
 * Tech Dashboard Alpine Component
 * Handles job list, filtering, offline sync, tips
 */
window.techDashboard = function () {
    return {
        isOnline: navigator.onLine,
        pendingReports: 0,
        activeTab: 'all', // all | new | active | completed
        showRefreshTimer: false,
        refreshCountdown: 30,

        async initDashboard() {
            // Update status & counts
            await this.updatePendingReports();

            // Auto refresh logic (Countdown timer)
            this.startAutoRefresh();
        },

        startAutoRefresh() {
            setInterval(() => {
                if (this.refreshCountdown > 0) {
                    this.refreshCountdown--;
                } else {
                    this.refreshCountdown = 30; // Reset
                    // Trigger HTMX reload silently (nếu đang online)
                    if (this.isOnline) {
                        const listContainer = document.getElementById('job-list-container');
                        if (listContainer) htmx.trigger(listContainer, 'statusUpdated');
                    }
                }
                this.showRefreshTimer = this.refreshCountdown < 5;
            }, 1000);
        },

        async updatePendingReports() {
            if (window.OfflineJobReporter) {
                const reports = await window.OfflineJobReporter.getPendingReports();
                this.pendingReports = reports.length;
            }
        },

        // Manual Sync Button
        async syncNow() {
            if (!this.isOnline) {
                alert('Vui lòng kết nối mạng để đồng bộ.');
                return;
            }
            if (window.OfflineJobReporter) {
                await window.OfflineJobReporter.syncPendingReports();
                await this.updatePendingReports();
                alert('Đã gửi dữ liệu lên máy chủ.');
            }
        },

        getTodayTip() {
            const tips = [
                '⏰ Nhớ check-in đúng giờ để giữ uy tín.',
                '📸 Chụp ảnh "Trước" và "Sau" để tránh tranh cãi.',
                '💬 Gọi điện xác nhận với khách trước khi đi.',
                '⚡ Dùng QR Scanner để nhập vật tư nhanh hơn.',
                '📍 Check Google Maps để tránh tắc đường.'
            ];
            // Lấy tip theo ngày trong năm để không đổi loạn xạ
            const dayOfYear = Math.floor((new Date() - new Date(new Date().getFullYear(), 0, 0)) / 1000 / 60 / 60 / 24);
            return tips[dayOfYear % tips.length];
        }
    };
};

/**
 * Job Detail & Completion Component
 * Handles QR scanner integration, Parts selection
 */
window.jobCompletion = function () {
    return {
        // Data
        step: 1, // 1: Photos -> 2: Parts -> 3: Confirm
        photos: [],
        parts: [], // List vật tư đã chọn {id, name, price, qty}

        // Input binding
        selectedPartId: '',
        selectedQty: 1,
        notes: '',

        // Config
        baseLaborPrice: 0, // Giá nhân công cơ bản (truyền từ server template)

        init() {
            // Lắng nghe sự kiện từ QR Scanner (global event window)
            window.addEventListener('qr-scanned', (e) => {
                this.addPartFromQR(e.detail);
            });
        },

        // --- Logic Vật tư ---
        addPart() {
            if (!this.selectedPartId) return;

            // Tìm option đang chọn để lấy data-name, data-price
            const select = document.querySelector(`select[x-model="selectedPartId"]`);
            if (!select) return;
            const option = select.options[select.selectedIndex];

            this.pushPart({
                id: this.selectedPartId,
                name: option.dataset.name,
                price: parseFloat(option.dataset.price) || 0,
                qty: parseInt(this.selectedQty) || 1
            });

            // Reset form
            this.selectedPartId = '';
            this.selectedQty = 1;
        },

        addPartFromQR(data) {
            // data format: {id, name, price, quantity}
            this.pushPart({
                id: data.id,
                name: data.name,
                price: parseFloat(data.price) || 0,
                qty: parseInt(data.quantity) || 1
            });
            // Show toast/alert
            alert(`Đã thêm từ QR: ${data.name}`);
        },

        pushPart(newItem) {
            // Check trùng lặp -> cộng dồn
            const existing = this.parts.find(p => p.id === newItem.id);
            if (existing) {
                existing.qty += newItem.qty;
            } else {
                this.parts.push(newItem);
            }
        },

        removePart(index) {
            this.parts.splice(index, 1);
        },

        updateQty(index, delta) {
            const item = this.parts[index];
            item.qty += delta;
            if (item.qty <= 0) this.removePart(index);
        },

        // --- Tính toán tiền ---
        get totalPartsCost() {
            return this.parts.reduce((sum, p) => sum + (p.price * p.qty), 0);
        },

        get grandTotal() {
            return this.baseLaborPrice + this.totalPartsCost;
        },

        formatMoney(amount) {
            return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
        },

        // --- Submit ---
        async submitCompletion(jobId) {
            // 1. Validate
            if (this.photos.length === 0) { // Giả sử required
                // alert('Cần ít nhất 1 ảnh nghiệm thu'); 
                // return;
            }

            // 2. Prepare Data (Cho OfflineReporter)
            const jobData = {
                jobId: jobId,
                notes: this.notes,
                parts: this.parts, // Mảng parts đầy đủ
                photos: this.photos // Blob hoặc Base64
            };

            // 3. Save via OfflineReporter
            try {
                if (window.OfflineJobReporter) {
                    await window.OfflineJobReporter.saveJobReport(jobData);
                    alert('Đã lưu báo cáo! Dữ liệu sẽ được gửi khi có mạng.');
                    window.location.href = '/tech/jobs'; // Redirect về list
                } else {
                    // Fallback submit form thường nếu reporter lỗi (hiếm)
                    document.getElementById('completion-form').submit();
                }
            } catch (e) {
                console.error('Submit failed', e);
                alert('Lỗi lưu báo cáo: ' + e.message);
            }
        }
    };
};

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

console.log('✅ Alpine.js components loaded (incl. BookingWizard)');

window.kanbanBoard = function (initialData) {
    return {
        columns: {
            pending: [],
            assigned: [],
            working: [],
            completed: []
        },
        editingJob: {},
        selectedJob: null,

        init() {
            // 1. Phân loại dữ liệu ban đầu
            const rawJobs = initialData || [];

            // Reset columns để tránh duplicate nếu re-init
            this.columns = { pending: [], assigned: [], working: [], completed: [] };

            rawJobs.forEach(job => {
                let status = job.status;
                // Chuẩn hóa status để khớp với tên cột
                if (status === 'moving' || status === 'working') status = 'working';

                if (this.columns[status]) {
                    this.columns[status].push(job);
                } else {
                    // Fallback về pending nếu status lạ
                    this.columns.pending.push(job);
                }
            });

            // 2. Lắng nghe SSE (Realtime)
            const eventSource = new EventSource('/admin/stream');
            eventSource.addEventListener('message', (e) => {
                try {
                    const event = JSON.parse(e.data);
                    // Reload nhẹ nhàng nếu có booking mới/update
                    // (Trong thực tế nên dùng Optimistic Update, nhưng reload an toàn hơn cho MVP)
                    if (event.type === 'booking.created' || event.type === 'booking.updated') {
                        // Debounce reload
                        if (!this._reloadTimeout) {
                            this._reloadTimeout = setTimeout(() => window.location.reload(), 1000);
                        }
                    }
                } catch (err) { console.error('SSE Error', err); }
            });
        },

        // --- Drag & Drop Logic ---
        dragStart(e, job) {
            e.dataTransfer.setData('jobId', job.id);
            e.dataTransfer.effectAllowed = 'move';
        },

        drop(e, targetCol) {
            const jobId = e.dataTransfer.getData('jobId');

            // Tìm job đang nằm ở cột nào
            let sourceCol = null;
            let jobIndex = -1;
            let job = null;

            for (const colName in this.columns) {
                const idx = this.columns[colName].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    sourceCol = colName;
                    jobIndex = idx;
                    job = this.columns[colName][idx];
                    break;
                }
            }

            if (!sourceCol || sourceCol === targetCol) return;

            // Xử lý logic nghiệp vụ

            // 1. Kéo về Pending (Hủy giao việc)
            if (targetCol === 'pending') {
                if (!confirm(`⚠️ HỦY GIAO VIỆC?\n\nĐơn "${job.customer}" sẽ quay lại hàng chờ.`)) return;
            }

            // 2. Kéo vào Assigned (Giao việc) -> Mở Modal
            if (targetCol === 'assigned') {
                const modalCheckbox = document.getElementById('modal-assign-' + jobId);
                if (modalCheckbox) modalCheckbox.checked = true;
                return; // Dừng tại đây, Modal sẽ lo việc submit
            }

            // 3. Cập nhật UI (Optimistic)
            this.columns[sourceCol].splice(jobIndex, 1);
            this.columns[targetCol].push(job);

            // 4. Gọi API
            let newStatus = targetCol;
            if (targetCol === 'working') newStatus = 'moving'; // Default working status start

            fetch(`/admin/api/bookings/${jobId}/status`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: `status=${newStatus}`
            }).then(res => {
                if (!res.ok) {
                    alert('Lỗi cập nhật trạng thái');
                    window.location.reload();
                } else if (targetCol === 'pending') {
                    // Reload để đảm bảo data sạch (xóa tên thợ)
                    setTimeout(() => window.location.reload(), 500);
                }
            });
        },

        // --- Modal Logic ---
        viewJob(job) {
            this.selectedJob = job;
            console.log(this.selectedJob);
            document.getElementById('modal-view-job').checked = true;
        },

        openEdit(job) {
            document.getElementById('modal-view-job').checked = false;
            this.editingJob = { ...job }; // Clone object để không sửa trực tiếp vào UI khi chưa lưu
            document.getElementById('modal-edit-booking').checked = true;
        },

        cancelJob(id) {
            if (confirm('Bạn có chắc chắn muốn HỦY đơn hàng này?')) {
                fetch('/admin/bookings/' + id + '/cancel', { method: 'POST' })
                    .then(res => {
                        if (res.ok) window.location.reload();
                        else alert('Lỗi khi hủy đơn');
                    });
            }
        }
    };
};

console.log('✅ Alpine.js: kanbanBoard loaded');

// Console log check
console.log('✅ Alpine.js components loaded');
window.slotManager = function () {
    return {
        techCount: 3,
        loading: false,
        loadingList: false,
        message: '',
        success: false,
        slots: [],

        init() {
            this.fetchSlots();
        },

        async fetchSlots() {
            this.loadingList = true;
            try {
                // Giả lập hoặc gọi API thật
                const res = await fetch('/admin/api/slots?days=7');
                if (res.ok) {
                    this.slots = await res.json();
                } else {
                    console.warn('API slots chưa có, hiển thị dữ liệu mẫu hoặc rỗng');
                    // this.slots = []; 
                }
            } catch (e) {
                console.error(e);
            } finally {
                this.loadingList = false;
            }
        },

        async generateWeek() {
            this.loading = true;
            this.message = '';

            try {
                // [SMART SCHEDULING] Server automatically uses active tech count
                const response = await fetch('/admin/tools/slots/generate-week', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                    body: 'tech_count=0' // Signal to use auto count
                });

                const result = await response.json();

                if (response.ok) {
                    this.showMessage(
                        `✅ Đã tạo lịch cho ${result.success_count} ngày. ${result.errors?.length > 0 ? '(Một số đã tồn tại)' : ''}`,
                        true
                    );
                    setTimeout(() => this.fetchSlots(), 1000);
                } else {
                    this.showMessage('❌ Lỗi: ' + (result.error || 'Không xác định'), false);
                }
            } catch (error) {
                this.showMessage('❌ Lỗi kết nối: ' + error.message, false);
            } finally {
                this.loading = false;
            }
        },

        showMessage(msg, isSuccess) {
            this.message = msg;
            this.success = isSuccess;
            setTimeout(() => {
                this.message = '';
            }, 5000);
        },

        // Helpers
        formatDate(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            return date.toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' });
        },

        getDayName(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            const days = ['Chủ Nhật', 'Thứ 2', 'Thứ 3', 'Thứ 4', 'Thứ 5', 'Thứ 6', 'Thứ 7'];
            return days[date.getDay()];
        },

        getProgressColor(current, max) {
            const percent = (current / max) * 100;
            if (percent >= 100) return 'progress-error';
            if (percent >= 70) return 'progress-warning';
            return 'progress-success';
        }
    }
};

console.log('✅ Alpine.js: slotManager loaded');

/**
 * Inventory Manager Component
 * @param {Array} initialItems - Dữ liệu danh sách vật tư từ Server
 */
window.inventoryManager = function (initialItems) {
    return {
        // Nhận dữ liệu từ tham số truyền vào, nếu null thì gán mảng rỗng
        items: initialItems || [],

        newItem: {
            name: '',
            sku: '',
            category: 'capacitors',
            price: '',
            stock_quantity: 0,
            unit: 'cái',
            description: ''
        },
        loading: false,
        message: '',
        success: false,

        async addItem() {
            if (!this.newItem.name || !this.newItem.price) {
                this.showMessage('Vui lòng điền tên và giá', false);
                return;
            }

            this.loading = true;
            const formData = new FormData();
            Object.keys(this.newItem).forEach(key => formData.append(key, this.newItem[key]));

            try {
                const response = await fetch('/admin/tools/inventory/create', { method: 'POST', body: formData });
                if (response.ok) {
                    this.showMessage('✅ Đã thêm linh kiện thành công!', true);
                    // Reset form
                    this.newItem = { name: '', sku: '', category: 'capacitors', price: '', stock_quantity: 0, unit: 'cái', description: '' };
                    // Reload trang để cập nhật danh sách
                    setTimeout(() => location.reload(), 1500);
                } else {
                    this.showMessage('❌ Lỗi khi thêm linh kiện', false);
                }
            } catch (error) {
                this.showMessage('❌ Lỗi: ' + error.message, false);
            } finally {
                this.loading = false;
            }
        },

        showStockUpdate(item) {
            const newStock = prompt(`Cập nhật số lượng tồn kho cho "${item.name}":`, item.stock_quantity);
            if (newStock !== null && !isNaN(newStock)) {
                this.updateStock(item.id, newStock);
            }
        },

        async updateStock(itemId, quantity) {
            const formData = new FormData();
            formData.append('quantity', quantity);
            formData.append('operation', 'set');

            try {
                const response = await fetch(`/admin/tools/inventory/${itemId}/stock`, { method: 'POST', body: formData });
                if (response.ok) {
                    this.showMessage('✅ Đã cập nhật tồn kho!', true);
                    setTimeout(() => location.reload(), 500);
                } else {
                    this.showMessage('❌ Cập nhật thất bại', false);
                }
            } catch (error) {
                this.showMessage('❌ Lỗi kết nối', false);
            }
        },

        printQR(item) {
            const qrData = JSON.stringify({ id: item.id, name: item.name, price: item.price });
            const qrUrl = `https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(qrData)}`;

            const win = window.open('', '_blank', 'width=400,height=500');
            win.document.write(`
                <html>
                <head><title>In Tem QR - ${item.name}</title></head>
                <body style="font-family: sans-serif; text-align: center; padding: 20px; border: 2px dashed #ccc; margin: 10px;">
                    <h2 style="margin-bottom: 5px; font-size: 18px;">${item.name}</h2>
                    <p style="margin: 0; color: #666; font-size: 12px;">${item.sku || 'NO-SKU'}</p>
                    <div style="margin: 20px auto;">
                        <img src="${qrUrl}" width="150" height="150" style="border: 1px solid #eee; padding: 5px;" />
                    </div>
                    <p style="font-weight: bold; font-size: 20px; margin: 10px 0;">${this.formatMoney(item.price)}</p>
                    <button onclick="window.print()" style="margin-top: 20px; padding: 10px 20px; cursor: pointer; background: #2563eb; color: white; border: none; border-radius: 4px;">🖨️ IN TEM NGAY</button>
                </body>
                </html>
            `);
        },

        formatMoney(value) {
            return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(value);
        },

        showMessage(msg, isSuccess) {
            this.message = msg;
            this.success = isSuccess;
            setTimeout(() => this.message = '', 4000);
        }
    }
};