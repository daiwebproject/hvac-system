// Wrap in alpine:init to ensure Alpine is loaded first
if (window.Alpine) {
    defineKanbanBoard();
} else {
    window.addEventListener('alpine:init', defineKanbanBoard);
}

function defineKanbanBoard() {
    window.kanbanBoard = function (initialActive, initialCompleted) {
        return {
            columns: {
                pending: [],
                assigned: [],
                working: [],
                completed: [],
                cancelled: []
            },
            completedJobs: [], // Keep for reference if needed, but primary data is now in columns
            editingJob: {},
            selectedJob: null,
            searchQuery: '',
            showMapModal: false,
            fullscreenMapInstance: null,
            mapSidebarTab: 'techs',  // 'techs' or 'orders'
            techHoverOn: null,
            jobHoverOn: null,
            selectedTechOnMap: null,
            selectedJobOnMap: null,

        init() {
            // 1. Setup Active Jobs (Kanban)
            const activeJobs = initialActive || [];

            // Reset columns
            this.columns = {
                pending: [],
                assigned: [],
                working: [],
                completed: [],
                cancelled: []
            };

            activeJobs.forEach(job => {
                let status = job.status;
                // Normalize status
                if (status === 'moving' || status === 'arrived' || status === 'working' || status === 'failed') status = 'working';

                if (this.columns[status]) {
                    this.columns[status].push(job);
                } else {
                    // Fallback to pending if unknown status (shouldn't happen for active jobs)
                    this.columns.pending.push(job);
                }
            });

            // 2. Setup Completed Jobs (History List)
            // Split into completed vs cancelled columns
            const historyJobs = initialCompleted || [];

            historyJobs.forEach(job => {
                let status = job.status; // 'completed' or 'cancelled'
                if (status === 'cancelled') {
                    this.columns.cancelled.push(job);
                } else {
                    // Default to completed for anything else in this list
                    this.columns.completed.push(job);
                }
            });

            this.completedJobs = historyJobs; // Keep reference just in case

            // 3. Listen to SSE
            this.setupSSE();

            // 4. Expose helpers globally (for Modals outside Alpine scope)
            window.moveJobLocally = this.moveJobLocally.bind(this);
        },

        // Search Filter Helper
        matchesSearch(job) {
            if (!this.searchQuery) return true;
            const query = this.searchQuery.toLowerCase();
            return (job.customer && job.customer.toLowerCase().includes(query)) ||
                (job.phone && job.phone.includes(query)) ||
                (job.service && job.service.toLowerCase().includes(query));
        },

        // Helper to trigger UI update (if needed)
        filterJobs() {
            // Alpine x-show with matchesSearch handles the UI, 
            // this is just a placeholder if we need side effects
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
                    // Handle Job Completion (Payment)
                    else if (event.type === 'job.completed') {
                        const { booking_id, invoice_amount } = event.data;
                        this.moveJobLocally(booking_id, 'completed', {
                            status_label: 'completed',
                            invoice_amount: invoice_amount
                        });
                    }
                    // Handle Cancellations
                    else if (event.type === 'booking.cancelled' || event.type === 'job.cancelled') {
                        const { id, booking_id, reason, note } = event.data;
                        this.removeJobLocally(id || booking_id);

                        // [NEW] Notify Admin
                        if (reason) {
                            Swal.fire({
                                title: 'Thông báo',
                                text: `Đơn hàng đã hủy. Lý do: ${reason} ${note ? '(' + note + ')' : ''}`,
                                icon: 'warning',
                                toast: true,
                                position: 'top-end',
                                showConfirmButton: false,
                                timer: 5000
                            });
                        }
                    }
                    // Handle New Bookings
                    else if (event.type === 'booking.created') {
                        // [FIX] Add to list directly without reload
                        const newJob = event.data;

                        // Default properties if missing
                        if (!newJob.status) newJob.status = 'pending';
                        if (!newJob.id) newJob.id = newJob.booking_id;

                        // Add to pending column
                        this.columns.pending.unshift(newJob);

                        Swal.fire({
                            title: '🔔 Đơn hàng mới!',
                            text: `Khách hàng: ${newJob.customer}`,
                            icon: 'success',
                            toast: true,
                            position: 'top-end',
                            showConfirmButton: false,
                            timer: 5000
                        });

                        // Try to geocode if address exists (optional, reusing existing logic)
                        if (typeof geocodeAndDraw === 'function' && newJob.address) {
                            geocodeAndDraw(newJob);
                        }
                    }
                } catch (err) { console.error('SSE Error', err); }
            });
        },

        // Helper to move job between columns without reload
        moveJobLocally(jobId, newStatus, extraUpdates = {}) {
            // 1. Determine target info
            let targetCol = newStatus;
            if (['moving', 'arrived', 'working', 'failed'].includes(newStatus)) targetCol = 'working';

            // 2. Find and remove from current list (check all columns)
            let job = null;
            for (const col in this.columns) {
                const idx = this.columns[col].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    job = this.columns[col].splice(idx, 1)[0];
                    break;
                }
            }

            // Check legacy list just in case (optional fallback)
            if (!job) {
                const idx = this.completedJobs.findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    job = this.completedJobs.splice(idx, 1)[0];
                }
            }

            // 3. Update and Add to new location
            if (job) {
                job.status = newStatus;
                job.status_label = extraUpdates.status_label || newStatus; // Use provided label or raw status
                job.updated = new Date().toLocaleString('vi-VN', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }); // Set recent update time

                // Apply extra updates (e.g. staff_id, invoice info)
                Object.assign(job, extraUpdates);

                // Add status class for completed/cancelled
                if (newStatus === 'cancelled') job.status_class = 'error';
                else if (newStatus === 'completed') job.status_class = 'success';

                if (this.columns[targetCol]) {
                    this.columns[targetCol].unshift(job);
                } else {
                    // Fallback
                    this.columns.pending.unshift(job);
                }
            } else {
                // If job not found locally, reload to be safe
                console.warn('Job not found locally for update:', jobId);
                // window.location.reload(); // Optional: reload if critical
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

            // Helper function to execute the drop logic
            const executeDrop = () => {
                // 3. Cập nhật UI (Optimistic)
                this.columns[sourceCol].splice(jobIndex, 1);

                // [FIX] Cập nhật thuộc tính Job ngay lập tức
                job.status = targetCol;
                job.status_label = targetCol;

                // Nếu kéo về Pending -> Xóa thông tin thợ
                if (targetCol === 'pending') {
                    job.staff_id = null;
                    job.technician_id = null;
                }

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
                        Swal.fire({
                            title: 'Lỗi',
                            text: 'Không thể cập nhật trạng thái',
                            icon: 'error'
                        });
                        window.location.reload();
                    }
                    // Success
                });
            };

            // Xử lý logic nghiệp vụ

            // 1. Kéo về Pending (Hủy giao việc)
            if (targetCol === 'pending') {
                Swal.fire({
                    title: 'Hủy giao việc?',
                    text: `Đơn "${job.customer}" sẽ quay lại hàng chờ.`,
                    icon: 'warning',
                    showCancelButton: true,
                    confirmButtonColor: '#d33',
                    cancelButtonColor: '#3085d6',
                    confirmButtonText: 'Đồng ý hủy',
                    cancelButtonText: 'Không'
                }).then((result) => {
                    if (result.isConfirmed) {
                        executeDrop();
                    }
                });
                return; // Wait for async confirmation
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

            // 3. Các trường hợp khác -> Thực hiện ngay
            executeDrop();
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
            Swal.fire({
                title: 'Hủy đơn hàng?',
                text: "Bạn có chắc chắn muốn hủy đơn hàng này không? Hành động này không thể hoàn tác.",
                icon: 'warning',
                showCancelButton: true,
                confirmButtonColor: '#d33',
                cancelButtonColor: '#3085d6',
                confirmButtonText: 'Đồng ý hủy',
                cancelButtonText: 'Không'
            }).then((result) => {
                if (result.isConfirmed) {
                    fetch('/admin/bookings/' + id + '/cancel', { method: 'POST' })
                        .then(res => {
                            if (res.ok) {
                                this.removeJobLocally(id);
                                Swal.fire({
                                    title: 'Đã hủy',
                                    text: 'Đơn hàng đã được hủy thành công',
                                    icon: 'success',
                                    toast: true,
                                    position: 'top-end',
                                    showConfirmButton: false,
                                    timer: 3000
                                });
                            } else {
                                Swal.fire('Lỗi', 'Lỗi khi hủy đơn', 'error');
                            }
                        })
                        .catch(() => {
                            Swal.fire('Lỗi', 'Lỗi kết nối', 'error');
                        });
                }
            });
        },

        // --- Fullscreen Map Modal ---
        openMapModal() {
            this.showMapModal = true;
            // Wait for modal & CSS transition, then account for reflow (500ms safe margin)
            setTimeout(() => {
                this.drawFullscreenMap();
                // Trigger map size recalculation after initial render
                if (this.fullscreenMapInstance) {
                    setTimeout(() => this.fullscreenMapInstance.invalidateSize(), 100);
                }
            }, 500);
        },

        closeMapModal() {
            this.showMapModal = false;
            if (this.fullscreenMapInstance) {
                this.fullscreenMapInstance.remove();
                this.fullscreenMapInstance = null;
            }
        },

        drawFullscreenMap() {
            const container = document.getElementById('fullscreen-map');
            if (!container || typeof L === 'undefined') return;

            // Check if container has valid dimensions
            const rect = container.getBoundingClientRect();
            if (rect.width === 0 || rect.height === 0) {
                // Retry with longer wait for DOM reflow
                console.log('Map container loading... retrying in 250ms');
                setTimeout(() => this.drawFullscreenMap(), 250);
                return;
            }

            // Cleanup existing
            if (this.fullscreenMapInstance) {
                this.fullscreenMapInstance.remove();
                this.fullscreenMapInstance = null;
            }

            console.log(`📍 Initializing map in container: ${rect.width}x${rect.height}px`);

            // Create map centered on Hanoi (disable zoom animation initially)
            this.fullscreenMapInstance = L.map('fullscreen-map', {
                zoomAnimation: false
            }).setView([21.0285, 105.8542], 10);

            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '© OpenStreetMap'
            }).addTo(this.fullscreenMapInstance);

            // Status color mapping
            const statusColors = {
                pending: '#eab308',   // yellow-500
                assigned: '#3b82f6', // blue-500
                working: '#a855f7',  // purple-500
                completed: '#22c55e', // green-500
                cancelled: '#ef4444' // red-500
            };

            const statusLabels = {
                pending: 'Chờ xử lý',
                assigned: 'Đã giao',
                working: 'Đang làm',
                completed: 'Hoàn thành',
                cancelled: 'Đã hủy'
            };

            const bounds = [];
            const mapMarkers = {}; // Store marker refs for interaction

            // Add markers for all jobs
            for (const status in this.columns) {
                this.columns[status].forEach(job => {
                    if (job.lat && job.long) {
                        const color = statusColors[status] || '#6b7280';
                        const label = statusLabels[status] || status;

                        // Create colored circle marker
                        const marker = L.circleMarker([job.lat, job.long], {
                            radius: 10,
                            fillColor: color,
                            color: '#ffffff',
                            weight: 2,
                            opacity: 1,
                            fillOpacity: 0.9
                        }).addTo(this.fullscreenMapInstance);

                        // Store marker reference
                        mapMarkers[job.id] = marker;

                        // Create popup with job details
                        const popupContent = `
                            <div style="min-width: 200px;">
                                <div style="font-weight: bold; font-size: 14px; margin-bottom: 4px;">${job.customer_name || job.customer || 'Khách hàng'}</div>
                                <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px;">
                                    <i class="fa-solid fa-phone"></i> ${job.phone || 'N/A'}
                                </div>
                                <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px;">
                                    <i class="fa-solid fa-wrench"></i> ${job.service_type || job.service || 'Dịch vụ'}
                                </div>
                                <div style="font-size: 12px; color: #6b7280; margin-bottom: 8px;">
                                    <i class="fa-solid fa-clock"></i> ${job.created || ''}
                                </div>
                                <div style="display: inline-block; padding: 2px 8px; border-radius: 4px; background: ${color}; color: white; font-size: 11px; font-weight: bold;">
                                    ${label}
                                </div>
                                ${job.technician_id ? `<div style="margin-top: 4px; font-size: 11px;"><i class="fa-solid fa-user-gear"></i> Thợ: ${job.technician_id}</div>` : ''}
                            </div>
                        `;

                        marker.bindPopup(popupContent);

                        // Interactive click handler
                        marker.on('click', () => {
                            this.selectedJobOnMap = job.id;
                            // Open popup
                            marker.openPopup();
                            // Scroll to job in sidebar if orders tab is open
                            if (this.mapSidebarTab === 'orders') {
                                setTimeout(() => {
                                    document.querySelector(`[x-for*="job"][jobid="${job.id}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                                }, 100);
                            }
                        });

                        // Hover effects
                        marker.on('mouseover', function() {
                            this.setRadius(14);
                            this.setStyle({ weight: 3 });
                        });

                        marker.on('mouseout', function() {
                            this.setRadius(10);
                            this.setStyle({ weight: 2 });
                        });

                        bounds.push([job.lat, job.long]);
                    }
                });
            }

            // Store markers for later manipulation
            this.mapMarkers = mapMarkers;

            // Store bounds for later use
            const savedBounds = bounds;

            // Force resize after modal animation, then fit bounds
            setTimeout(() => {
                if (this.fullscreenMapInstance) {
                    this.fullscreenMapInstance.invalidateSize();

                    // Fit bounds after map is fully initialized
                    if (savedBounds.length > 0) {
                        this.fullscreenMapInstance.fitBounds(savedBounds, { padding: [50, 50] });
                    }
                }
            }, 400);
        },

        createJob(event) {
            const form = event.target;
            const formData = new FormData(form);

            fetch(form.action, {
                method: 'POST',
                body: formData
            }).then(res => res.json())
                .then(data => {
                    if (data.success || data.message) { // Handle both simple message and full object
                        Swal.fire({
                            title: 'Thành công',
                            text: 'Đã tạo đơn hàng mới',
                            icon: 'success',
                            timer: 1500,
                            showConfirmButton: false
                        });
                        document.getElementById('modal-create-job').checked = false;
                        form.reset();

                        // [FIX] Add to list directly without reload
                        if (data.booking) {
                            const newJob = data.booking;
                            // Default status if missing
                            if (!newJob.status) newJob.status = 'pending';

                            this.columns.pending.unshift(newJob);

                            // Try geocode if needed
                            if (typeof geocodeAndDraw === 'function' && newJob.address) {
                                geocodeAndDraw(newJob);
                            }
                        } else {
                            // Fallback if no booking data returned (should not happen with new backend)
                            setTimeout(() => window.location.reload(), 1500);
                        }
                    } else {
                        Swal.fire('Lỗi', data.error || 'Lỗi không xác định', 'error');
                    }
                }).catch(err => {
                    console.error(err);
                    Swal.fire('Lỗi', 'Lỗi kết nối', 'error');
                });
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

        // === Map Sidebar Functions ===
        getAllJobs() {
            return [
                ...this.columns.pending,
                ...this.columns.assigned,
                ...this.columns.working,
                ...this.columns.completed
            ].filter(j => j.customer_name);
        },

        getJobsForTech(techId) {
            return this.getAllJobs().filter(j => j.technician_id === techId).length;
        },

        getStatusLabel(status) {
            const labels = {
                pending: 'Chờ xử lý',
                assigned: 'Đã giao',
                working: 'Đang làm',
                completed: 'Hoàn thành',
                cancelled: 'Đã hủy'
            };
            return labels[status] || status;
        },

        highlightTechOnMap(techId) {
            this.selectedTechOnMap = techId;
            // Filter map markers to show only this tech's jobs
            if (this.fullscreenMapInstance) {
                const techJobs = this.getAllJobs().filter(j => j.technician_id === techId);
                console.log(`🔍 Highlighting ${techJobs.length} jobs for tech ${techId}`);
                
                // Zoom to show all tech's jobs
                if (techJobs.length > 0) {
                    const lats = techJobs.map(j => j.lat).filter(l => l);
                    const lons = techJobs.map(j => j.long).filter(l => l);
                    if (lats.length > 0) {
                        const bounds = L.latLngBounds(
                            [[Math.min(...lats), Math.min(...lons)], 
                             [Math.max(...lats), Math.max(...lons)]]
                        );
                        this.fullscreenMapInstance.fitBounds(bounds, { padding: [50, 50] });
                    }
                }
            }
        },

        highlightJobOnMap(jobId) {
            this.selectedJobOnMap = jobId;
            const job = this.getAllJobs().find(j => j.id === jobId);
            if (job && job.lat && job.long && this.fullscreenMapInstance) {
                // Pan to job location
                this.fullscreenMapInstance.setView([job.lat, job.long], 14, { animate: true });
                console.log(`📍 Centered on job ${jobId} at ${job.lat}, ${job.long}`);
            }
        },

        getProgressColor(current, max) {
            const percent = (current / max) * 100;
            if (percent >= 100) return 'progress-error';
            if (percent >= 70) return 'progress-warning';
            return 'progress-success';
        }
    }
};
}  // Close defineKanbanBoard function

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
    if (!mapEl || typeof L === 'undefined') return;

    // 1. Cleanup bản đồ cũ
    if (mapInstance) {
        mapInstance.remove();
        mapInstance = null;
        mapMarkers = [];
    }

    try {
        // 2. Tạo Map
        mapInstance = L.map('fleet-map').setView([10.8231, 106.6297], 13);
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap'
        }).addTo(mapInstance);

        // 3. Xử lý Dữ liệu Đơn hàng
        const bookings = window.initialBookings || [];

        bookings.forEach((job, index) => {
            // Trường hợp A: Đã có tọa độ trong DB
            if (job.lat && job.long) {
                addJobMarker(job, job.lat, job.long);
            }
            // Trường hợp B: Chưa có tọa độ -> Tự động Geocode từ địa chỉ
            else if (job.address && job.address.length > 5) {
                // Delay nhẹ để tránh spam API (OpenStreetMap giới hạn 1req/s)
                setTimeout(() => {
                    geocodeAndDraw(job);
                }, index * 1200);
            }
        });

        // 4. Xử lý Dữ liệu Thợ (Demo/Realtime)
        // Nếu có biến window.initialTechs (cần inject từ backend)
        if (window.initialTechs) {
            window.initialTechs.forEach(tech => {
                if (tech.active) {
                    // Giả lập vị trí nếu chưa có (Demo)
                    // Trong thực tế: dùng tech.last_lat, tech.last_long
                    const lat = tech.lat || (10.8231 + (Math.random() - 0.5) * 0.05);
                    const long = tech.long || (106.6297 + (Math.random() - 0.5) * 0.05);
                    addTechMarker(tech, lat, long);
                }
            });
        }

        // Tự động zoom sau 2s (để chờ geocode xong 1 phần)
        setTimeout(window.fitMapBounds, 2000);

    } catch (e) { console.error("Map init error:", e); }
}

// --- Helpers ---

// Hàm vẽ Marker Khách hàng
function addJobMarker(job, lat, lng) {
    if (!mapInstance) return;
    const iconColor = getJobColor(job.status);
    const marker = L.marker([lat, lng], {
        icon: createCustomIcon(iconColor, 'fa-wrench')
    })
        .addTo(mapInstance)
        .bindPopup(`
        <div class="text-sm">
            <b>${job.customer}</b><br>
            <span class="text-gray-500">${job.address}</span><br>
            <span class="badge badge-xs ${getBadgeClass(job.status)} mt-1">${job.status_label || job.status}</span>
        </div>
    `);
    mapMarkers.push(marker);
}

// Hàm vẽ Marker Thợ
function addTechMarker(tech, lat, lng) {
    if (!mapInstance) return;
    const marker = L.marker([lat, lng], {
        icon: createCustomIcon('#3b82f6', 'fa-user-gear', true) // Màu xanh, icon user
    })
        .addTo(mapInstance)
        .bindPopup(`<b>KTV: ${tech.name}</b><br><span class="text-green-600">● Đang hoạt động</span>`);
    mapMarkers.push(marker);
}

// Hàm Geocode (Tìm tọa độ từ địa chỉ)
async function geocodeAndDraw(job) {
    if (!job.address || job.address.length < 5) return;

    // [VALIDATION] Skip if address is mostly numbers (likely a phone number or ID)
    if (/^\d+$/.test(job.address.replace(/\s|[,\.]/g, ''))) {
        console.warn(`[Geocode] Skipping invalid address (looks like phone/ID): ${job.address}`);
        return;
    }

    try {
        // Thêm "Vietnam" để tìm chính xác hơn
        const query = `${job.address}, Vietnam`;

        // [FIX] Use Backend Proxy to avoid CORS & Header issues
        const url = `/api/public/geocode?q=${encodeURIComponent(query)}`;

        const res = await fetch(url);
        const data = await res.json();

        if (data && data.length > 0) {
            const lat = data[0].lat;
            const lon = data[0].lon;

            // Vẽ marker ngay lập tức
            addJobMarker(job, lat, lon);

            // [TODO]: Gửi tọa độ này về Backend để lưu lại (đỡ phải tìm lần sau)
            // saveCoordinatesToBackend(job.id, lat, lon);
            console.log(`Đã tìm thấy vị trí cho đơn ${job.id}: ${lat}, ${lon}`);
        } else {
            console.warn(`[Geocode] Không tìm thấy địa chỉ: ${job.address}`);
        }
    } catch (err) {
        console.error(`[Geocode] Lỗi khi tìm địa chỉ:`, err);
    }
}

function getJobColor(status) {
    if (status === 'completed') return '#22c55e';
    if (status === 'working' || status === 'moving') return '#a855f7';
    if (status === 'assigned') return '#3b82f6';
    if (status === 'cancelled') return '#ef4444';
    return '#eab308';
}

function getBadgeClass(status) {
    if (status === 'completed') return 'badge-success';
    if (status === 'working') return 'badge-secondary';
    if (status === 'assigned') return 'badge-info';
    return 'badge-warning';
}

// Tạo Icon đẹp hơn (Hỗ trợ FontAwesome class)
function createCustomIcon(color, iconClass = 'fa-circle', isTech = false) {
    const size = isTech ? 40 : 32;
    const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="${size}" height="${size}" class="drop-shadow-md">
        <path fill="${color}" d="M12 0C7.58 0 4 3.58 4 8c0 5.25 8 16 8 16s8-10.75 8-16c0-4.42-3.58-8-8-8z"/>
        <circle cx="12" cy="8" r="3.5" fill="white"/>
    </svg>
    `;

    // Dùng HTML Icon để lồng FontAwesome vào giữa
    return L.divIcon({
        className: 'custom-map-marker-container',
        html: `
            <div style="position: relative; width: ${size}px; height: ${size}px;">
                ${svg}
                <i class="fa-solid ${iconClass}" style="position: absolute; top: ${isTech ? 8 : 6}px; left: 50%; transform: translateX(-50%); font-size: ${isTech ? 14 : 12}px; color: ${color};"></i>
            </div>
        `,
        iconSize: [size, size],
        iconAnchor: [size / 2, size], // Mũi nhọn icon chạm đúng vị trí
        popupAnchor: [0, -size]
    });
}

document.addEventListener('DOMContentLoaded', initFleetMap);
document.addEventListener('htmx:afterSettle', (evt) => {
    if (document.getElementById('fleet-map')) initFleetMap();
});