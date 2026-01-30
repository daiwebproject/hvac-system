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