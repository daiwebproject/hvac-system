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
                    // Không cần reload trang, chỉ cần fetch lại list
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