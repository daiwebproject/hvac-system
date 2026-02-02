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
            this.columns = { pending: [], assigned: [], working: [], completed: [], cancelled: [] };

            rawJobs.forEach(job => {
                let status = job.status;
                // Chuẩn hóa status để khớp với tên cột
                if (status === 'moving' || status === 'arrived' || status === 'working' || status === 'failed') status = 'working';

                if (this.columns[status]) {
                    this.columns[status].push(job);
                } else {
                    // Fallback về pending nếu status lạ
                    this.columns.pending.push(job);
                }
            });

            // 2. Lắng nghe SSE (Realtime)
            this.setupSSE();
        },

        setupSSE() {
            const eventSource = new EventSource('/admin/stream');
            eventSource.addEventListener('message', (e) => {
                try {
                    const event = JSON.parse(e.data);
                    console.log('Admin SSE:', event);

                    // Handle Job Status Change
                    if (event.type === 'job.status_changed') {
                        const { booking_id, status } = event.data;
                        this.moveJobLocally(booking_id, status);
                    }
                    // Handle Job Assign
                    else if (event.type === 'job.assigned') {
                        const { booking_id, tech_id } = event.data;
                        this.moveJobLocally(booking_id, 'assigned', { staff_id: tech_id });
                    }
                    // Handle Cancellations
                    else if (event.type === 'booking.cancelled' || event.type === 'job.cancelled') {
                        const { id, booking_id } = event.data;
                        this.removeJobLocally(id || booking_id);
                    }
                    // Handle New Bookings
                    else if (event.type === 'booking.created') {
                        if (!this._reloadTimeout) {
                            // Reload nhẹ sau 1.5s để cập nhật danh sách đầy đủ
                            this._reloadTimeout = setTimeout(() => window.location.reload(), 1500);
                        }
                    }
                } catch (err) { console.error('SSE Error', err); }
            });
        },

        // Helper to move job between columns without reload
        moveJobLocally(jobId, newStatus, extraUpdates = {}) {
            // 1. Determine target column
            let targetCol = newStatus;
            if (['moving', 'arrived', 'working', 'failed'].includes(newStatus)) targetCol = 'working';

            // 2. Find and remove from current column
            let job = null;
            for (const col in this.columns) {
                const idx = this.columns[col].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    job = this.columns[col].splice(idx, 1)[0];
                    break;
                }
            }

            // 3. Update and Add to new column
            if (job) {
                job.status = newStatus;
                job.status_label = newStatus; // Update label
                // Apply extra updates (e.g. staff_id)
                Object.assign(job, extraUpdates);

                if (this.columns[targetCol]) {
                    this.columns[targetCol].unshift(job); // Add to top
                } else {
                    this.columns.pending.unshift(job); // Fallback
                }
            } else {
                // If job not found locally, reload to be safe
                window.location.reload();
            }
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
                // Hack nhẹ để mở modal sau khi drop
                setTimeout(() => {
                    const modalCheckbox = document.getElementById('modal-assign-' + jobId);
                    if (modalCheckbox) modalCheckbox.checked = true;
                }, 50);
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
            document.getElementById('modal-view-job').checked = true;
        },

        openEdit(job) {
            document.getElementById('modal-view-job').checked = false;
            // Deep clone để tránh lỗi Alpine reactivity cycle
            this.editingJob = JSON.parse(JSON.stringify(job));
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
        },

        createJob(event) {
            const form = event.target;
            const formData = new FormData(form);

            fetch(form.action, {
                method: 'POST',
                body: formData
            }).then(res => {
                if (res.ok) {
                    Swal.fire({
                        title: 'Thành công',
                        text: 'Đã tạo đơn hàng mới',
                        icon: 'success',
                        timer: 1500,
                        showConfirmButton: false
                    });
                    document.getElementById('modal-create-job').checked = false;
                    form.reset();
                    // Reload to fetch new data
                    setTimeout(() => window.location.reload(), 1500);
                } else {
                    res.text().then(text => Swal.fire('Lỗi', text, 'error'));
                }
            }).catch(err => Swal.fire('Lỗi', 'Lỗi kết nối', 'error'));
        },

        removeJobLocally(jobId) {
            for (const col in this.columns) {
                const idx = this.columns[col].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    this.columns[col].splice(idx, 1);
                    return;
                }
            }
        }
    };
};

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
                const res = await fetch('/admin/api/slots?days=7');
                if (res.ok) {
                    this.slots = await res.json();
                } else {
                    console.warn('API slots chưa có, hiển thị rỗng');
                }
            } catch (e) {
                console.error(e);
            } finally {
                this.loadingList = false;
            }
        },

        async generateWeek() {
            if (this.techCount < 1) {
                this.showMessage('Số thợ phải lớn hơn 0', false);
                return;
            }

            this.loading = true;
            this.message = '';

            try {
                const formData = new FormData();
                formData.append('tech_count', this.techCount);

                const response = await fetch('/admin/tools/slots/generate-week', {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();

                if (response.ok) {
                    this.showMessage(
                        `✅ Đã tạo ${result.success_count} khung giờ. ${result.errors?.length > 0 ? '(Một số đã tồn tại)' : ''}`,
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

window.inventoryManager = function (initialItems) {
    return {
        // Nhận dữ liệu từ tham số truyền vào
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
                    this.newItem = { name: '', sku: '', category: 'capacitors', price: '', stock_quantity: 0, unit: 'cái', description: '' };
                    // Reload trang
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

// ==========================================
// [FIX] MAP LOGIC (AN TOÀN & GLOBAL)
// ==========================================

let mapInstance = null;
let mapMarkers = [];

// Định nghĩa hàm Global NGAY LẬP TỨC để HTML có thể gọi
window.fitMapBounds = function () {
    if (!mapInstance) return; // Nếu chưa có map thì thôi

    try {
        if (mapMarkers.length > 0) {
            const group = new L.featureGroup(mapMarkers);
            mapInstance.fitBounds(group.getBounds(), { padding: [50, 50] });
        } else {
            // Vị trí mặc định nếu không có marker (TP.HCM)
            mapInstance.setView([10.8231, 106.6297], 13);
        }
    } catch (e) {
        console.warn("Map bounds error:", e);
    }
};

function initFleetMap() {
    const mapEl = document.getElementById('fleet-map');

    // Nếu trang hiện tại không có div #fleet-map -> Thoát ngay (tránh lỗi trên các trang khác)
    if (!mapEl) return;

    // Kiểm tra thư viện Leaflet đã load chưa
    if (typeof L === 'undefined') {
        console.warn('Leaflet JS chưa được tải.');
        return;
    }

    // [QUAN TRỌNG] Cleanup bản đồ cũ trước khi tạo mới
    // Việc này cực kỳ quan trọng khi dùng HTMX để swap nội dung trang
    if (mapInstance) {
        mapInstance.remove();
        mapInstance = null;
        mapMarkers = [];
    }

    try {
        // 1. Tạo Map mới
        mapInstance = L.map('fleet-map').setView([10.8231, 106.6297], 13);

        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors',
            maxZoom: 19
        }).addTo(mapInstance);

        // 2. Vẽ Marker từ dữ liệu global (window.initialBookings)
        const bookings = window.initialBookings || [];
        bookings.forEach(job => {
            if (job.lat && job.long) {
                const iconColor = getJobColor(job.status);
                const marker = L.marker([job.lat, job.long], {
                    icon: createCustomIcon(iconColor)
                })
                    .addTo(mapInstance)
                    .bindPopup(`
                    <b>${job.customer}</b><br>
                    <span class="text-xs text-gray-500">${job.service}</span><br>
                    <span class="badge badge-xs ${getBadgeClass(job.status)}">${job.status_label || job.status}</span>
                `);

                mapMarkers.push(marker);
            }
        });

        // 3. Tự động zoom
        if (mapMarkers.length > 0) {
            setTimeout(window.fitMapBounds, 500);
        }
    } catch (err) {
        console.error('Lỗi khởi tạo bản đồ:', err);
    }
}

// --- Map Helpers ---

function getJobColor(status) {
    if (status === 'completed') return '#22c55e'; // Green
    if (status === 'working' || status === 'moving') return '#a855f7'; // Purple
    if (status === 'assigned') return '#3b82f6'; // Blue
    if (status === 'cancelled') return '#ef4444'; // Red
    return '#eab308'; // Yellow (Pending)
}

function getBadgeClass(status) {
    if (status === 'completed') return 'badge-success';
    if (status === 'working') return 'badge-secondary';
    if (status === 'assigned') return 'badge-info';
    return 'badge-warning';
}

function createCustomIcon(color) {
    const svgIcon = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="${color}" stroke="white" stroke-width="2" class="w-8 h-8 drop-shadow-md">
        <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"/>
        <circle cx="12" cy="9" r="2.5" fill="white"/>
    </svg>`;

    return L.divIcon({
        className: 'custom-map-marker',
        html: svgIcon,
        iconSize: [32, 32],
        iconAnchor: [16, 32],
        popupAnchor: [0, -32]
    });
}

// Khởi tạo Map khi load trang VÀ sau khi HTMX swap nội dung (Quan trọng)
document.addEventListener('DOMContentLoaded', initFleetMap);
document.addEventListener('htmx:afterSettle', (evt) => {
    // Chỉ init lại nếu nội dung mới được swap có chứa map
    if (document.getElementById('fleet-map')) {
        initFleetMap();
    }
});