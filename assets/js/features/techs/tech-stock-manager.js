/**
 * Tech Stock Manager Component - Alpine.js data component
 * @module features/techs/tech-stock-manager
 */

import { apiClient } from '../../core/api-client.js';
import { formatMoney } from '../../core/utils.js';
import { toast } from '../../core/toast.js';

/**
 * Define the Tech Stock Manager Alpine.js component
 * @param {Object} initData - Initial data { techs: [], items: [] }
 * @returns {Object} Alpine.js component
 */
export function techStockManager(initData = { techs: [], items: [] }) {
    return {
        techs: Array.isArray(initData.techs) ? initData.techs : [],
        items: Array.isArray(initData.items) ? initData.items : [],
        searchQuery: '',
        showTransferModal: false,
        showDetailModal: false,
        loading: false,

        transfer: {
            techId: '',
            itemId: '',
            quantity: 1
        },

        selectedItem: null,
        selectedTech: null,
        techItems: [],

        init() {
            console.log('[TechStockManager] Initialized');
        },

        get filteredTechs() {
            if (!this.searchQuery) return this.techs;
            const query = this.searchQuery.toLowerCase();
            return this.techs.filter(tech =>
                tech.name.toLowerCase().includes(query) ||
                tech.phone.includes(query)
            );
        },

        formatMoney(amount) {
            return formatMoney(amount);
        },

        updateSelectedItem() {
            this.selectedItem = this.items.find(i => i.id === this.transfer.itemId);
        },

        openTransferModal(techId = '') {
            this.transfer.techId = techId;
            this.transfer.itemId = '';
            this.transfer.quantity = 1;
            this.selectedItem = null;
            this.showTransferModal = true;
        },

        async viewTechDetail(techId) {
            this.selectedTech = this.techs.find(t => t.id === techId);
            this.showDetailModal = true;

            try {
                const data = await apiClient.get(`/admin/tools/tech-stock/${techId}`);
                this.techItems = data.items || [];
            } catch (e) {
                console.error('Error loading tech inventory:', e);
                this.techItems = [];
                toast.error('Không thể tải danh sách vật tư của thợ');
            }
        },

        async submitTransfer() {
            if (!this.transfer.techId || !this.transfer.itemId || !this.transfer.quantity) {
                this.showMessage('Vui lòng điền đầy đủ thông tin', false);
                return;
            }

            this.loading = true;

            try {
                const fd = new FormData();
                fd.append('technician_id', this.transfer.techId);
                fd.append('item_id', this.transfer.itemId);
                fd.append('quantity', this.transfer.quantity);

                const res = await fetch('/admin/tools/tech-stock/transfer', {
                    method: 'POST',
                    body: fd
                });

                const data = await res.json();

                if (res.ok && data.success) {
                    // Update local state
                    const tech = this.techs.find(t => t.id === this.transfer.techId);
                    const item = this.items.find(i => i.id === this.transfer.itemId);

                    if (tech && item) {
                        tech.item_count++;
                        // Calculate added value
                        const addedValue = item.price * parseFloat(this.transfer.quantity);
                        tech.total_value = (tech.total_value || 0) + addedValue;
                    }

                    if (item) item.stock_quantity -= this.transfer.quantity;

                    this.showMessage(data.message || 'Cấp hàng thành công!', true);
                    this.showTransferModal = false;
                    this.transfer = { techId: '', itemId: '', quantity: 1 };
                } else {
                    this.showMessage(data.error || 'Lỗi không xác định', false);
                }
            } catch (e) {
                this.showMessage('Lỗi kết nối: ' + e.message, false);
            } finally {
                this.loading = false;
            }
        },

        showMessage(msg, isSuccess) {
            if (isSuccess) {
                toast.success(msg);
            } else {
                toast.error(msg);
            }
        }
    };
}
