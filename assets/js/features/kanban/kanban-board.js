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
        fullscreenMapInstance: null,
        mapSidebarTab: 'techs',
        techHoverOn: null,
        jobHoverOn: null,
        selectedTechOnMap: null,
        selectedJobOnMap: null,
        techs: window.initialTechs || [],
        markerLayerGroup: null,
        showMapSidebar: false,
        mapMarkers: {},
        assignTechId: '',
        sseConnection: null,

        // === Lifecycle ===
        init() {
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
                let status = job.status;
                if (['moving', 'arrived', 'working', 'failed'].includes(status)) {
                    status = 'working';
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
        },

        handleSSEEvent(event) {
            console.log('Admin SSE:', event);

            switch (event.type) {
                case 'job.status_changed':
                    this.moveJobLocally(event.data.booking_id, event.data.status);
                    break;

                case 'job.assigned':
                    this.moveJobLocally(event.data.booking_id, 'assigned', {
                        staff_id: event.data.tech_id
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
                technician_id: null
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

        openAssignModal(job) {
            this.selectedJob = job;
            this.assignTechId = '';
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
                        staff_id: techName,
                        technician_id: techId
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

            setTimeout(() => {
                this.fullscreenMapInstance.invalidateSize();
                this.renderMapMarkers();
            }, 300);
        },

        renderMapMarkers() {
            if (!this.fullscreenMapInstance || !this.markerLayerGroup) return;

            this.markerLayerGroup.clearLayers();
            this.mapMarkers = {};

            const allJobs = this.getAllJobs();
            allJobs.forEach(job => {
                if (job.lat && job.long) {
                    const marker = L.marker([job.lat, job.long])
                        .bindPopup(`<b>${job.customer}</b><br>${job.address || ''}`);
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
        }
    };
}

export default kanbanBoard;
