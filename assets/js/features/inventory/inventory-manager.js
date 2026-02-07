/**
 * Inventory Manager Component - Alpine.js data component
 * @module features/inventory/inventory-manager
 */

import { apiClient } from '../../core/api-client.js';
import { toast } from '../../core/toast.js';
import { formatMoney } from '../../core/utils.js';

/**
 * Define the Inventory Manager Alpine.js component
 * @param {Array} initialItems - Initial inventory items from server
 * @returns {Object} Alpine.js component
 */
export function inventoryManager(initialItems = []) {
    return {
        items: Array.isArray(initialItems) ? initialItems : [],
        alerts: [],
        showImportModal: false,
        showAddModal: false,
        searchQuery: '',
        selectedCategory: '',
        isEditing: false,
        editItemId: null,
        newItem: {
            name: '',
            sku: '',
            category: 'capacitors',
            price: '',
            stock_quantity: 0,
            unit: 'cái',
            description: ''
        },
        importData: { product_id: '', quantity: 1, note: '' },
        loading: false,
        message: '',
        success: false,

        get filteredItems() {
            return this.items.filter(item => {
                const matchesSearch = this.searchQuery === ''
                    || item.name.toLowerCase().includes(this.searchQuery.toLowerCase())
                    || (item.sku && item.sku.toLowerCase().includes(this.searchQuery.toLowerCase()));

                const matchesCategory = this.selectedCategory === ''
                    || item.category === this.selectedCategory;

                return matchesSearch && matchesCategory;
            });
        },

        init() {
            console.log('[InventoryManager] Init with', this.items.length, 'items');
            this.loadAlerts();
        },

        async loadAlerts() {
            try {
                const res = await fetch('/admin/tools/inventory/alerts');
                if (res.ok) {
                    const data = await res.json();
                    this.alerts = data.alerts || [];
                }
            } catch (e) {
                console.error('[InventoryManager] Error loading alerts:', e);
            }
        },

        async saveItem() {
            if (!this.newItem.name || !this.newItem.price) {
                this.showMessage('Vui lòng điền tên và giá', false);
                return;
            }

            this.loading = true;

            try {
                let url = '/admin/tools/inventory/create';
                let method = 'POST';

                if (this.isEditing) {
                    url = `/admin/tools/inventory/${this.editItemId}/update`;
                }

                // Convert to form data or json? Handler expects FormValue so usually formData or x-www-form-urlencoded
                // But previous create used JSON? Wait, CreateInventoryItem uses e.Request.FormValue.
                // apiClient.post usually sends JSON if object passed.
                // Let's check apiClient implementation or stick to what worked.
                // Previous addItem used apiClient.post(..., this.newItem).
                // If the handler uses FormValue, it supports both multipart and urlencoded.
                // BUT if apiClient sends JSON, `FormValue` in Go might be empty unless parsed from body manually?
                // Actually PocketBase `e.Request.FormValue` works with multipart/form-data.
                // Let's use FormData to be safe and consistent with other handlers I've seen.

                const fd = new FormData();
                fd.append('name', this.newItem.name);
                fd.append('sku', this.newItem.sku);
                fd.append('category', this.newItem.category);
                fd.append('price', this.newItem.price);
                fd.append('stock_quantity', this.newItem.stock_quantity);
                fd.append('unit', this.newItem.unit);
                fd.append('description', this.newItem.description);

                const response = await fetch(url, { method: 'POST', body: fd });
                const data = await response.json();

                if (response.ok && data.success) {
                    if (this.isEditing) {
                        // Update existing item
                        const index = this.items.findIndex(i => i.id === this.editItemId);
                        if (index !== -1) {
                            this.items[index] = { ...this.items[index], ...data.item };
                        }
                        this.showMessage('✅ Cập nhật thành công', true);
                    } else {
                        // Add new item
                        this.items.push(data.item);
                        this.showMessage('✅ Đã thêm: ' + data.item.name, true);
                    }
                    this.showAddModal = false;
                    this.resetForm();
                } else {
                    this.showMessage(data.error || '❌ Lỗi xảy ra', false);
                }
            } catch (error) {
                this.showMessage('❌ Lỗi: ' + error.message, false);
            } finally {
                this.loading = false;
            }
        },

        openEditModal(item) {
            this.isEditing = true;
            this.editItemId = item.id;
            this.newItem = {
                name: item.name,
                sku: item.sku || '',
                category: item.category || 'other',
                price: item.price,
                stock_quantity: item.stock_quantity,
                unit: item.unit || 'cái',
                description: item.description || ''
            };
            this.showAddModal = true;
        },

        async deleteItem(item) {
            const confirmed = await toast.confirm(
                'Xác nhận xóa?',
                `Bạn có chắc muốn xóa "${item.name}"?`,
                { confirmText: 'Xóa ngay', confirmButtonColor: '#ef4444' }
            );

            if (!confirmed) return;

            try {
                const response = await fetch(`/admin/tools/inventory/${item.id}/delete`, { method: 'POST' });
                const data = await response.json();

                if (response.ok && data.success) {
                    this.items = this.items.filter(i => i.id !== item.id);
                    toast.success('Đã xóa vật tư');
                } else {
                    toast.error(data.error || 'Lỗi khi xóa');
                }
            } catch (e) {
                toast.error('Lỗi kết nối');
            }
        },

        resetForm() {
            this.isEditing = false;
            this.editItemId = null;
            this.newItem = {
                name: '',
                sku: '',
                category: 'capacitors',
                price: '',
                stock_quantity: 0,
                unit: 'cái',
                description: ''
            };
        },

        showStockUpdate(item) {
            const newStock = prompt(`Cập nhật số lượng tồn kho cho "${item.name}":`, item.stock_quantity);
            if (newStock !== null && !isNaN(newStock)) {
                this.updateStock(item.id, newStock);
            }
        },

        async updateStock(itemId, quantity) {
            try {
                const response = await apiClient.post(`/admin/tools/inventory/${itemId}/stock`, {
                    quantity: quantity,
                    operation: 'set'
                });

                if (response.ok) {
                    const item = this.items.find(i => i.id === itemId);
                    if (item) item.stock_quantity = parseFloat(quantity);
                    this.showMessage('✅ Đã cập nhật tồn kho!', true);
                    this.loadAlerts();
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
            return formatMoney(value);
        },

        showMessage(msg, isSuccess) {
            this.message = msg;
            this.success = isSuccess;
            setTimeout(() => { this.message = ''; }, 4000);
        }
    };
}

export default inventoryManager;

