/**
 * Slot Manager Component - Alpine.js data component
 * @module features/slots/slot-manager
 */

import { apiClient } from '../../core/api-client.js';
import { toast } from '../../core/toast.js';
import { formatDate, getDayName } from '../../core/utils.js';

/**
 * Define the Slot Manager Alpine.js component
 * @returns {Object} Alpine.js component
 */
export function slotManager() {
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
                this.slots = await apiClient.get('/admin/api/slots?days=7');
            } catch (e) {
                console.warn('API slots chưa có, hiển thị rỗng');
                this.slots = [];
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
                const response = await apiClient.post('/admin/tools/slots/generate-week', {
                    tech_count: this.techCount
                });

                const result = await response.json();

                if (response.ok) {
                    const msg = `✅ Đã tạo ${result.success_count} khung giờ.` +
                        (result.errors?.length > 0 ? ' (Một số đã tồn tại)' : '');
                    this.showMessage(msg, true);
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
            setTimeout(() => { this.message = ''; }, 5000);
        },

        formatDate(dateStr) {
            return formatDate(dateStr);
        },

        getDayName(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            const days = ['Chủ Nhật', 'Thứ 2', 'Thứ 3', 'Thứ 4', 'Thứ 5', 'Thứ 6', 'Thứ 7'];
            return days[date.getDay()];
        },

        getProgressColor(current, max) {
            if (!max) return 'progress-success';
            const ratio = current / max;
            if (ratio < 0.5) return 'progress-success';
            if (ratio < 0.8) return 'progress-warning';
            return 'progress-error';
        }
    };
}

export default slotManager;
