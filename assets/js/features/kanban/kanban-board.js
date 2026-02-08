/**
 * Kanban Board Component - Alpine.js data component
 * @module features/kanban/kanban-board
 */

import { createSSEConnection } from '../../core/sse-service.js';
import { toast } from '../../core/toast.js';
import { apiClient } from '../../core/api-client.js';
import { formatBookingTime, getStatusLabel } from '../../core/utils.js';

// [FIX] Leaflet Icon 404 - Use CDN
const fixLeafletIcons = () => {
    if (typeof L !== 'undefined' && !L.Icon.Default.prototype._getIconUrl_fixed) {
        delete L.Icon.Default.prototype._getIconUrl;
        L.Icon.Default.mergeOptions({
            iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
            iconUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
            shadowUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png',
        });
        L.Icon.Default.prototype._getIconUrl_fixed = true;
    }
};
// Try to fix immediately if L is global, or wait/call later
fixLeafletIcons();


/**
 * Define the Kanban Board Alpine.js component
 * @param {Array} initialActive - Initial active jobs from server
 * @param {Array} initialCompleted - Initial completed jobs from server
 * @returns {Object} Alpine.js component
 */
export function kanbanBoard(initialActive = [], initialCompleted = []) {
    return {
        // === State ===
        columns: {
            pending: [],
            assigned: [],
            working: [],
            completed: [],
            cancelled: []
        },
        completedJobs: [],
        editingJob: {},
        selectedJob: null,
        searchQuery: '',
        showMapModal: false,
        // Heavy objects stored on instance (non-reactive)
        // fullscreenMapInstance: null,
        // sseConnection: null,
        // locationSSE: null,
        // fullscreenTracker: null,

        mapSidebarTab: 'techs',
        techHoverOn: null,
        jobHoverOn: null,
        selectedTechOnMap: null,
        selectedJobOnMap: null,
        techs: window.initialTechs || [],
        // markerLayerGroup: null, // Heavy object
        showMapSidebar: false,
        // mapMarkers: {}, // Heavy object
        assignTechId: '',
        busyTechs: [], // [NEW] Track busy technicians for assignment modal

        // === Lifecycle ===
        init() {
            // Initialize non-reactive properties
            this.fullscreenMapInstance = null;
            this.sseConnection = null;
            this.locationSSE = null;
            this.fullscreenTracker = null;
            this.markerLayerGroup = null;
            this.mapMarkers = {};

            this.initializeColumns(initialActive, initialCompleted);
            this.setupSSE();
            window.moveJobLocally = this.moveJobLocally.bind(this);

            // Watch for column changes to update map
            this.$watch('columns', () => {
                if (this.showMapModal) this.renderMapMarkers();
            });
        },

        destroy() {
            if (this.sseConnection) {
                this.sseConnection.close();
            }
            if (this.locationSyncHandler) {
                document.removeEventListener('admin:location-updated', this.locationSyncHandler);
            }
            if (this.fullscreenTracker) {
                this.fullscreenTracker.clearAll();
            }
        },

        // === Initialization ===
        initializeColumns(activeJobs, historyJobs) {
            this.columns = {
                pending: [],
                assigned: [],
                working: [],
                completed: [],
                cancelled: []
            };

            activeJobs.forEach(job => {
                // [FIX] Ensure consistent field naming from Backend JSON
                if (!job.staff_id && job.StaffID) job.staff_id = job.StaffID;
                if (!job.time_slot_id && job.TimeSlotID) job.time_slot_id = job.TimeSlotID; // [FIX]

                let status = job.status;
                if (['moving', 'arrived', 'working', 'failed'].includes(status)) {
                    status = 'working';
                }
                if (status === 'accepted') {
                    status = 'assigned';
                }

                if (this.columns[status]) {
                    this.columns[status].push(job);
                } else {
                    this.columns.pending.push(job);
                }
            });

            historyJobs.forEach(job => {
                if (job.status === 'cancelled') {
                    this.columns.cancelled.push(job);
                } else {
                    this.columns.completed.push(job);
                }
            });

            this.completedJobs = historyJobs;
        },

        // === SSE ===
        setupSSE() {
            this.sseConnection = createSSEConnection('/admin/stream', {
                onMessage: (event) => this.handleSSEEvent(event),
                onError: () => console.warn('[Kanban] SSE connection error'),
            });
            this.setupLocationSync(); // [NEW] Sync with Admin Map
        },

        setupLocationSync() {
            // Listen to global events from Admin Map (tracking-integration.js)
            this.locationSyncHandler = (e) => {
                const tech = e.detail;
                if (this.fullscreenTracker && this.showMapModal) {
                    // console.log('[Kanban] Sync update:', tech.name);
                    this.fullscreenTracker.updateTechnicianLocation(
                        tech.id,
                        tech.name,
                        tech.lat,
                        tech.lng,
                        null, null,
                        tech.distance
                    );

                    // [NEW] Sync Status
                    if (tech.status) {
                        this.fullscreenTracker.updateTechnicianStatus(tech.id, tech.status);
                    }

                    // [NEW] Update sidebar list
                    this.updateTechInList({
                        id: tech.id,
                        name: tech.name,
                        lat: tech.lat,
                        long: tech.lng,
                        distance: tech.distance,
                        active: true,
                        status: tech.status // [FIX] Pass status
                    });
                }
            };
            document.addEventListener('admin:location-updated', this.locationSyncHandler);
            console.log('[Kanban] Location sync initialized');
        },

        handleSSEEvent(event) {
            console.log('Admin SSE:', event);

            switch (event.type) {
                case 'job.status_changed':
                    this.moveJobLocally(event.data.booking_id, event.data.status);
                    // [NEW] Update map marker status
                    if (this.fullscreenTracker && event.data.tech_id) {
                        this.fullscreenTracker.updateTechnicianStatus(event.data.tech_id, event.data.status);
                    }
                    if (event.data.tech_id) {
                        // Update sidebar list info via existing helper
                        this.updateTechInList({ id: event.data.tech_id, status: event.data.status });
                    }
                    break;

                case 'job.assigned':
                    const sseTech = this.techs.find(t => t.id === event.data.tech_id);
                    this.moveJobLocally(event.data.booking_id, 'assigned', {
                        staff_id: event.data.tech_id,
                        technician_id: event.data.tech_id,
                        tech_name: sseTech ? sseTech.name : '...'
                    });
                    break;

                case 'job.completed':
                    this.moveJobLocally(event.data.booking_id, 'completed', {
                        status_label: 'completed',
                        invoice_amount: event.data.invoice_amount
                    });
                    break;

                case 'booking.cancelled':
                case 'job.cancelled':
                    this.handleJobCancelled(event.data);
                    break;

                case 'booking.created':
                    this.handleNewBooking(event.data);
                    break;

                case 'tech.status_changed':
                    // [NEW] Update tech status in real-time
                    const tech = this.techs.find(t => t.id === event.data.id);
                    if (tech) {
                        tech.active = event.data.active;
                        tech.lat = event.data.lat; // Update lat/long from event
                        tech.long = event.data.long;

                        // [FIX] Derive status string from active boolean
                        // If they are on a job (working/moving), we might need to preserve that? 
                        // But usually 'status_changed' means online/offline toggle.
                        // Ideally we check if they have active jobs, but for now simple toggle:
                        if (!tech.active) tech.status = 'offline';
                        else if (tech.status === 'offline') tech.status = 'online'; // Only switch back to online if previously offline

                        // Force reactivity
                        this.techs = [...this.techs];
                        if (this.showMapModal) this.renderMapMarkers();

                        // Also update map marker if present
                        if (this.fullscreenTracker) {
                            this.fullscreenTracker.updateTechnicianStatus(tech.id, tech.status);
                            this.fullscreenTracker.updateTechnicianLocation(tech.id, tech.name, tech.lat, tech.long);
                        }
                    }
                    break;
            }
        },

        handleJobCancelled(data) {
            const { id, booking_id, reason, note } = data;
            this.removeJobLocally(id || booking_id);

            if (reason) {
                toast.warning(`Đơn hàng đã hủy. Lý do: ${reason}${note ? ' (' + note + ')' : ''}`);
            }
        },

        handleNewBooking(raw) {
            const newJob = {
                id: raw.id || raw.booking_id,
                customer: raw.customer || raw.customer_name,
                phone: raw.phone || raw.customer_phone || '',
                service: raw.service || raw.device_type,
                time: formatBookingTime(raw.time),
                created: new Date().toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' }),
                status: raw.status || 'pending',
                status_label: raw.status_label || 'Chờ xử lý',
                status_class: 'warning',
                address: raw.address || raw.address_details || '',
                address_details: raw.address_details || '',
                lat: raw.lat,
                long: raw.long,
                issue: raw.issue || '',
                staff_id: null,
                technician_id: null,
                raw_time: raw.raw_time || raw.booking_time,
                duration: raw.duration || 120,
                time_slot_id: raw.time_slot_id // [NEW] Catch slot ID
            };

            this.columns.pending.unshift(newJob);
            toast.success(`🔔 Đơn hàng mới! Khách: ${newJob.customer}`);

            // Geocode for map
            if (typeof window.geocodeAndDraw === 'function' && newJob.address) {
                window.geocodeAndDraw(newJob);
            }
        },

        // === Job Movement ===
        moveJobLocally(jobId, newStatus, extraUpdates = {}) {
            let targetCol = newStatus;
            if (['moving', 'arrived', 'working', 'failed'].includes(newStatus)) {
                targetCol = 'working';
            }
            if (newStatus === 'accepted') {
                targetCol = 'assigned';
            }

            // Find and remove from current column
            let job = null;
            for (const col in this.columns) {
                const idx = this.columns[col].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    job = this.columns[col].splice(idx, 1)[0];
                    break;
                }
            }

            // Fallback to legacy list
            if (!job) {
                const idx = this.completedJobs.findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    job = this.completedJobs.splice(idx, 1)[0];
                }
            }

            if (job) {
                job.status = newStatus;
                job.status_label = extraUpdates.status_label || getStatusLabel(newStatus);
                job.updated = new Date().toLocaleString('vi-VN', {
                    day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit'
                });

                Object.assign(job, extraUpdates);

                if (newStatus === 'cancelled') job.status_class = 'error';
                else if (newStatus === 'completed') job.status_class = 'success';

                if (this.columns[targetCol]) {
                    this.columns[targetCol].unshift(job);
                } else {
                    this.columns.pending.unshift(job);
                }
            } else {
                console.warn('Job not found locally:', jobId);
            }
        },

        removeJobLocally(jobId) {
            for (const col in this.columns) {
                const idx = this.columns[col].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    this.columns[col].splice(idx, 1);
                    return;
                }
            }
        },

        // === Search ===
        matchesSearch(job) {
            if (!this.searchQuery) return true;
            const query = this.searchQuery.toLowerCase();
            return (job.customer && job.customer.toLowerCase().includes(query)) ||
                (job.phone && job.phone.includes(query)) ||
                (job.service && job.service.toLowerCase().includes(query));
        },

        filterJobs() {
            // Placeholder for future enhancements
        },

        // === Drag & Drop ===
        dragStart(e, job) {
            e.dataTransfer.setData('jobId', job.id);
            e.dataTransfer.effectAllowed = 'move';
        },

        async drop(e, targetCol) {
            const jobId = e.dataTransfer.getData('jobId');

            // Find job
            let job = null;
            let sourceCol = null;
            for (const col in this.columns) {
                const idx = this.columns[col].findIndex(j => j.id === jobId);
                if (idx !== -1) {
                    job = this.columns[col][idx];
                    sourceCol = col;
                    break;
                }
            }

            if (!job || sourceCol === targetCol) return;

            // Validate transitions
            if (targetCol === 'assigned' && sourceCol === 'pending') {
                this.openAssignModal(job);
                return;
            }

            if (targetCol === 'cancelled') {
                this.cancelJob(job.id);
                return;
            }

            // Direct status update
            try {
                const response = await apiClient.post(`/admin/api/bookings/${jobId}/status`, {
                    status: targetCol
                });

                if (response.ok) {
                    this.moveJobLocally(jobId, targetCol);
                    toast.success('Đã cập nhật trạng thái');
                } else {
                    toast.error('Không thể cập nhật trạng thái');
                }
            } catch (err) {
                toast.error('Lỗi kết nối');
            }
        },

        // === Modal Actions ===
        viewJob(job) {
            this.selectedJob = job;
            document.getElementById('modal-view-job')?.showModal?.() ||
                (document.getElementById('modal-view-job').checked = true);
        },

        openEdit(job) {
            this.editingJob = { ...job };
            document.getElementById('modal-edit-job')?.showModal?.() ||
                (document.getElementById('modal-edit-job').checked = true);
        },

        // [NEW] Refresh data before assignment
        async refreshBookings() {
            try {
                const response = await apiClient.get('/admin/api/bookings/active');
                if (response.ok) {
                    const jobs = await response.json();
                    jobs.forEach(job => {
                        if (!job.staff_id && job.StaffID) job.staff_id = job.StaffID;
                        if (!job.time_slot_id && job.TimeSlotID) job.time_slot_id = job.TimeSlotID;
                    });
                    return jobs;
                }
            } catch (err) { console.error(err); }
            return this.getAllJobs();
        },

        async openAssignModal(job) {
            this.selectedJob = job;
            this.assignTechId = '';

            // 1. Fetch latest data
            const allJobs = await this.refreshBookings();

            // 2. Calculate Availability
            this.busyTechs = {};
            const reqSlotId = job.time_slot_id;
            const reqStart = job.raw_time ? new Date(job.raw_time).getTime() : 0;
            const reqDuration = job.duration || 120;
            const reqEnd = reqStart + (reqDuration * 60000);

            // Start of conflict check loop
            allJobs.forEach(otherJob => {
                if (otherJob.id === job.id) return;
                const techId = otherJob.staff_id || otherJob.technician_id;
                if (!techId) return;
                if (['completed', 'cancelled'].includes(otherJob.status)) return;

                // Check 1: Slot ID
                if (reqSlotId && otherJob.time_slot_id && otherJob.time_slot_id === reqSlotId) {
                    this.busyTechs[techId] = 'Trùng Slot cố định';
                    return;
                }

                // Check 2: Time Overlap
                if (reqStart > 0 && otherJob.raw_time) {
                    const otherStart = new Date(otherJob.raw_time).getTime();
                    const otherDuration = otherJob.duration || 120;
                    const otherEnd = otherStart + (otherDuration * 60000);

                    if (reqStart < otherEnd && reqEnd > otherStart) {
                        const startStr = new Date(otherStart).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' });
                        const endStr = new Date(otherEnd).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' });
                        this.busyTechs[techId] = `Bận: ${startStr} - ${endStr}`;
                    }
                }
            });

            console.log('Conflicting assignments:', this.busyTechs);
            document.getElementById('modal-assign-generic').checked = true;

        },

        async submitAssignment() {
            if (!this.selectedJob || !this.assignTechId) return;

            const jobId = this.selectedJob.id;
            const techId = this.assignTechId;
            const techName = this.techs.find(t => t.id == techId)?.name || '...';

            try {
                const response = await apiClient.post(`/admin/bookings/${jobId}/assign`, {
                    technician_id: techId
                });

                if (response.ok) {
                    this.moveJobLocally(jobId, 'assigned', {
                        staff_id: techId,        // Sync with backend ID
                        technician_id: techId,
                        tech_name: techName      // [FIX] Populate Name
                    });
                    document.getElementById('modal-assign-generic').checked = false;
                    toast.success('Đã giao việc thành công');
                } else {
                    toast.error('Không thể giao việc');
                }
            } catch (err) {
                toast.error('Lỗi kết nối');
            }
        },

        async cancelJob(id) {
            const confirmed = await toast.confirm(
                'Xác nhận hủy đơn?',
                'Đơn hàng sẽ được chuyển sang trạng thái Đã hủy.'
            );

            if (!confirmed) return;

            try {
                const response = await apiClient.post(`/admin/bookings/${id}/cancel`, {
                    reason: 'Admin hủy',
                    note: ''
                });

                if (response.ok) {
                    this.moveJobLocally(id, 'cancelled');
                    toast.success('Đã hủy đơn hàng');
                } else {
                    toast.error('Không thể hủy đơn');
                }
            } catch (err) {
                toast.error('Lỗi kết nối');
            }
        },

        // === Create Job from Admin Form ===
        async createJob() {
            const form = document.getElementById('create-job-form');
            if (!form) {
                toast.error('Lỗi: Không tìm thấy form');
                return;
            }

            // Validate required fields
            const customerName = form.querySelector('[name="customer_name"]')?.value?.trim();
            const customerPhone = form.querySelector('[name="customer_phone"]')?.value?.trim();
            const bookingTime = form.querySelector('[name="booking_time"]')?.value?.trim();

            if (!customerName || !customerPhone || !bookingTime) {
                toast.error('Vui lòng điền đầy đủ thông tin bắt buộc');
                return;
            }

            const formData = new FormData(form);

            try {
                const response = await apiClient.post('/admin/bookings/create', Object.fromEntries(formData));

                if (response.ok) {
                    const result = await response.json();

                    // Close modal safely
                    const modalToggle = document.getElementById('modal-create-job');
                    if (modalToggle) modalToggle.checked = false;

                    // Reset form
                    form.reset();

                    // Show success toast
                    toast.success('Đã tạo đơn hàng mới!');

                    // SSE will handle adding the job to the board via booking.created event
                } else {
                    const errText = await response.text();
                    toast.error('Lỗi tạo đơn: ' + errText);
                }
            } catch (err) {
                console.error('CreateJob error:', err);
                toast.error('Lỗi kết nối máy chủ');
            }
        },

        // === Map Functions ===
        toggleMapSidebar() {
            this.showMapSidebar = !this.showMapSidebar;
        },

        selectAndPanTo(lat, long, id) {
            if (lat && long && this.fullscreenMapInstance) {
                this.fullscreenMapInstance.flyTo([lat, long], 16);
                this.showMapSidebar = false;

                if (this.mapMarkers[id]) {
                    this.mapMarkers[id].openPopup();
                }
            }
        },

        openMapModal() {
            this.showMapModal = true;
            this.$nextTick(() => this.drawFullscreenMap());
        },

        closeMapModal() {
            this.showMapModal = false;

            if (this.fullscreenTracker) {
                this.fullscreenTracker.clearAll();
                this.fullscreenTracker = null;
            }

            if (this.fullscreenMapInstance) {
                this.fullscreenMapInstance.remove();
                this.fullscreenMapInstance = null;
            }
        },

        drawFullscreenMap(retryCount = 0) {
            const container = document.getElementById('fullscreen-map');
            if (!container) {
                if (retryCount < 5) {
                    setTimeout(() => this.drawFullscreenMap(retryCount + 1), 100);
                }
                return;
            }

            if (typeof L === 'undefined') {
                console.error('Leaflet not loaded');
                return;
            }

            // Ensure icons are fixed
            if (typeof fixLeafletIcons === 'function') fixLeafletIcons();

            if (this.fullscreenMapInstance) {
                this.fullscreenMapInstance.remove();
            }

            this.fullscreenMapInstance = L.map('fullscreen-map').setView([21.0285, 105.8542], 12);
            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '© OpenStreetMap'
            }).addTo(this.fullscreenMapInstance);

            this.markerLayerGroup = L.layerGroup().addTo(this.fullscreenMapInstance);

            // [NEW] Initialize MapTracker
            try {
                const TrackerClass = window.MapTracker || MapTracker;
                if (typeof TrackerClass !== 'undefined') {
                    console.log('[Kanban] Initializing MapTracker for fullscreen map');
                    this.fullscreenTracker = new TrackerClass(this.fullscreenMapInstance, {
                        onMarkerClick: (m) => this.highlightTechOnMap(m.techId)
                    });

                    // Initial render of techs
                    let synced = false;
                    // Try to sync from small map (adminLocationMonitoring) first
                    if (window.adminMapComponent && typeof window.adminMapComponent.getActiveTechnicians === 'function') {
                        try {
                            const activeTechs = window.adminMapComponent.getActiveTechnicians();
                            if (activeTechs && activeTechs.length > 0) {
                                console.log('[Kanban] Syncing from small map:', activeTechs.length, 'techs');
                                activeTechs.forEach(t => {
                                    this.fullscreenTracker.updateTechnicianLocation(
                                        t.id, t.name, t.lat, t.lng, null, null, t.distance
                                    );
                                    // [NEW] Sync sidebar list
                                    this.updateTechInList({
                                        id: t.id, name: t.name, lat: t.lat, long: t.lng, distance: t.distance, active: true
                                    });
                                });
                                synced = true;
                            }
                        } catch (err) { console.warn('[Kanban] Sync failed:', err); }
                    }

                    // Fallback to local state if no sync data
                    if (!synced) {
                        this.techs.forEach(t => {
                            if (t.active && t.lat && t.long) {
                                console.log('[Kanban] Adding initial tech (fallback):', t.name);
                                this.fullscreenTracker.updateTechnicianLocation(t.id, t.name, t.lat, t.long);
                            }
                        });
                    }
                } else {
                    console.warn('[Kanban] MapTracker not defined (global or window)');
                }
            } catch (e) { console.error('[Kanban] MapTracker init error:', e); }

            setTimeout(() => {
                this.fullscreenMapInstance.invalidateSize();
                this.renderMapMarkers();
                // Auto fit
                if (this.fullscreenTracker) this.fullscreenTracker.fitMapToAllMarkers();
            }, 300);
        },

        updateTechInList(data) {
            const idx = this.techs.findIndex(t => t.id === data.id);
            if (idx !== -1) {
                // Update existing
                const oldLat = this.techs[idx].lat;
                const oldLong = this.techs[idx].long;

                this.techs[idx].lat = data.lat;
                this.techs[idx].long = data.long;
                // If data.active is present, use it. Otherwise assume true if receiving location?
                if (typeof data.active !== 'undefined') this.techs[idx].active = data.active;
                else this.techs[idx].active = true;

                if (data.distance) this.techs[idx].distance = data.distance;
                if (data.status) this.techs[idx].status = data.status;

                // Fetch address if location changed significantly (moved > 10m?) 
                // or if no address yet.
                // Simple check: if lat/long changed.
                if (data.lat && data.long && (oldLat !== data.lat || oldLong !== data.long)) {
                    this.fetchAddress(this.techs[idx]);
                }

                // Force Alpine reactivity for array item
                this.techs = [...this.techs];
            } else {
                // Optional: Add if new?
                const newTech = {
                    id: data.id,
                    name: data.name,
                    lat: data.lat,
                    long: data.long,
                    active: true,
                    distance: data.distance,
                    address: 'Đang lấy địa chỉ...'
                };
                this.techs.push(newTech);
                if (newTech.lat && newTech.long) this.fetchAddress(newTech);
            }
        },

        async fetchAddress(tech) {
            if (!tech.lat || !tech.long) return;

            // Debounce or cache could be added here.
            try {
                // Use OpenStreetMap Nominatim
                // IMPORTANT: Use a proper User-Agent header if possible, or respects policy.
                // In browser fetch, we can't easily set User-Agent to a custom value 
                // that overrides browser default, but standard usage is usually okay for low volume.
                const url = `https://nominatim.openstreetmap.org/reverse?format=json&lat=${tech.lat}&lon=${tech.long}&zoom=18&addressdetails=1`;

                const response = await fetch(url, {
                    headers: { 'Accept-Language': 'vi' }
                });

                if (response.ok) {
                    const data = await response.json();
                    tech.address = data.display_name || data.name || 'Không xác định';
                    // Shorten address for UI?
                    if (tech.address.length > 60) tech.address = tech.address.substring(0, 60) + '...';
                }
            } catch (e) {
                console.warn('Geo reverse error:', e);
                tech.address = `${tech.lat.toFixed(4)}, ${tech.long.toFixed(4)}`;
            }
            // Force update
            this.techs = [...this.techs];
        },

        filterJobsByTech(techId) {
            this.mapSidebarTab = 'orders';
            this.searchQuery = '';
            // TODO: Implement actual filtering of the list if needed, or just highlight
            // For now, we just switch to orders tab. 
            // Ideally we should have a filtered view.
        },

        renderMapMarkers() {
            if (!this.fullscreenMapInstance || !this.markerLayerGroup) return;

            // Only handle Job Markers here (Tech markers handled by MapTracker)
            this.markerLayerGroup.clearLayers();
            this.mapMarkers = {};

            const allJobs = this.getAllJobs();
            allJobs.forEach(job => {
                if (job.lat && job.long) {
                    // Custom marker color based on status
                    let markerColor = 'blue';
                    if (job.status === 'pending') markerColor = 'gold';
                    if (job.status === 'working') markerColor = 'violet';
                    if (job.status === 'completed') markerColor = 'green';
                    if (job.status === 'assigned') markerColor = 'blue';

                    // Use Leaflet default icon with hue shift or custom divIcon
                    // Simple solution: Html Icon
                    const iconHtml = `<div class="w-4 h-4 rounded-full border-2 border-white shadow-sm bg-${markerColor}-500"></div>`;

                    const marker = L.marker([job.lat, job.long], {
                        icon: L.divIcon({
                            className: 'custom-job-marker',
                            html: `<span class="relative flex h-4 w-4">
                                      <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-${markerColor}-400 opacity-75" style="display: ${job.status === 'working' ? 'block' : 'none'}"></span>
                                      <span class="relative inline-flex rounded-full h-4 w-4 bg-${markerColor}-500 border-2 border-white shadow-sm"></span>
                                    </span>`,
                            iconSize: [20, 20],
                            iconAnchor: [10, 10]
                        })
                    }).bindPopup(`
                        <div class="p-1">
                            <h3 class="font-bold text-sm">${job.customer || job.customer_name}</h3>
                            <p class="text-xs text-slate-600 mb-1">${job.address}</p>
                            <span class="text-[10px] px-2 py-0.5 rounded bg-${markerColor}-100 text-${markerColor}-700 font-bold uppercase">
                                ${this.getStatusLabel(job.status)}
                            </span>
                        </div>
                    `);

                    marker.addTo(this.markerLayerGroup);
                    this.mapMarkers[job.id] = marker;
                }
            });
        },

        getAllJobs() {
            return [
                ...this.columns.pending,
                ...this.columns.assigned,
                ...this.columns.working
            ];
        },

        getJobsForTech(techId) {
            return this.getAllJobs().filter(j => j.technician_id === techId);
        },

        highlightTechOnMap(techId) {
            this.selectedTechOnMap = techId;
        },

        highlightJobOnMap(jobId) {
            this.selectedJobOnMap = jobId;
            const job = this.getAllJobs().find(j => j.id === jobId);
            if (job && job.lat && job.long) {
                this.selectAndPanTo(job.lat, job.long, jobId);
            }
        },

        getProgressColor(current, max) {
            const ratio = current / max;
            if (ratio < 0.5) return 'progress-success';
            if (ratio < 0.8) return 'progress-warning';
            return 'progress-error';
        },

        // === Helpers ===
        formatBookingTime(rawTime) {
            return formatBookingTime(rawTime);
        },

        getStatusLabel(status) {
            return getStatusLabel(status);
        },

        getTechStatusLabel(status) {
            switch (status) {
                case 'moving': return 'Đang đi';
                case 'working': return 'Đang làm';
                case 'active': // Fallback legacy
                case 'online': return 'Tốc hành/Rãnh';
                case 'offline': return 'Offline';
            }
            return 'Chờ việc';
        },

        getTechStatusClass(status) {
            switch (status) {
                case 'moving': return 'bg-orange-100 text-orange-700';
                case 'working': return 'bg-purple-100 text-purple-700';
                case 'active':
                case 'online': return 'bg-blue-100 text-blue-700';
                case 'offline': return 'bg-gray-100 text-gray-600';
            }
            return 'bg-slate-100 text-slate-600';
        }
    };
}

export default kanbanBoard;
