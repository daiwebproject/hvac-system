/**
 * Tech Manager Component - Alpine.js data component
 * @module features/techs/tech-manager
 */

/**
 * Define the Tech Manager Alpine.js component
 * @param {Array} initialServices - List of services for skills selection
 * @returns {Object} Alpine.js component
 */
export function techManager(initialServices = []) {
    return {
        activeTab: 'info',
        isSubmitting: false,
        editModalOpen: false,
        editingTech: null,
        allServices: initialServices,
        searchQuery: '',

        // Location State
        provinces: [],
        districts: [],
        wards: [],
        selectedProvince: '',
        selectedDistrict: '',
        selectedWard: '',
        isLoadingLocation: false,

        init() {
            document.body.addEventListener('techListUpdated', () => {
                // Re-init logic if needed
            });
            this.fetchProvinces();
        },

        // --- Location API Methods ---
        async fetchProvinces() {
            try {
                const res = await fetch('https://esgoo.net/api-tinhthanh/1/0.htm');
                const data = await res.json();
                if (data.error === 0) {
                    this.provinces = data.data;
                }
            } catch (e) {
                console.error('Error fetching provinces:', e);
            }
        },

        async fetchDistricts() {
            this.districts = [];
            this.wards = [];
            this.selectedDistrict = '';
            this.selectedWard = '';
            if (!this.selectedProvince) return;

            this.isLoadingLocation = true;
            try {
                const res = await fetch(`https://esgoo.net/api-tinhthanh/2/${this.selectedProvince}.htm`);
                const data = await res.json();
                if (data.error === 0) {
                    this.districts = data.data;
                }
            } catch (e) {
                console.error(e);
            } finally {
                this.isLoadingLocation = false;
            }
        },

        async fetchWards() {
            this.wards = [];
            this.selectedWard = '';
            if (!this.selectedDistrict) return;

            this.isLoadingLocation = true;
            try {
                const res = await fetch(`https://esgoo.net/api-tinhthanh/3/${this.selectedDistrict}.htm`);
                const data = await res.json();
                if (data.error === 0) {
                    this.wards = data.data;
                }
            } catch (e) {
                console.error(e);
            } finally {
                this.isLoadingLocation = false;
            }
        },

        addZone() {
            if (!this.selectedWard || !this.selectedDistrict || !this.selectedProvince) return;

            // Find names
            const p = this.provinces.find(x => x.id === this.selectedProvince)?.full_name || '';
            const d = this.districts.find(x => x.id === this.selectedDistrict)?.full_name || '';
            const w = this.wards.find(x => x.id === this.selectedWard)?.full_name || '';

            const zoneName = `${w}, ${d}, ${p}`;

            if (!this.editingTech.service_zones) this.editingTech.service_zones = [];

            if (!this.editingTech.service_zones.includes(zoneName)) {
                this.editingTech.service_zones.push(zoneName);
            } else {
                Swal.fire('Thông báo', 'Khu vực này đã được thêm', 'info');
            }
        },

        removeZone(index) {
            this.editingTech.service_zones.splice(index, 1);
        },


        openEditModal(tech) {
            this.editingTech = JSON.parse(JSON.stringify(tech)); // Deep copy to avoid binding issues

            // Ensure arrays exist
            if (!this.editingTech.skills) this.editingTech.skills = [];
            if (!this.editingTech.service_zones) this.editingTech.service_zones = [];

            this.activeTab = 'info';
            this.editModalOpen = true;
        },

        hasSkill(skillId) {
            if (!this.editingTech || !this.editingTech.skills) return false;
            return this.editingTech.skills.includes(skillId);
        },

        hasZone(zoneName) {
            if (!this.editingTech || !this.editingTech.service_zones) return false;
            return this.editingTech.service_zones.includes(zoneName);
        },

        // Helper để chọn tất cả hoặc bỏ chọn tất cả zone (Cần JS can thiệp vào form input vì x-model trong loop phức tạp)
        toggleAllZones(select) {
            const inputs = document.querySelectorAll('input[name="service_zones"]');
            inputs.forEach(input => {
                // Chỉ check những cái đang hiện (theo search)
                if (input.closest('label') && input.closest('label').style.display !== 'none') {
                    input.checked = select;
                }
            });
        },

        async submitForm(e) {
            this.isSubmitting = true;
            const form = e.target.closest('form'); // Handle finding form from button click or submit
            const formData = new FormData(form);

            // Convert Checkboxes to proper JSON array logic for PocketBase/Backend
            // Note: htmx or fetch will send multiple values for same key 'skills', backend must handle []string parsing

            try {
                const response = await fetch(`/admin/techs/${this.editingTech.id}/update`, {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    this.editModalOpen = false;
                    Swal.fire({
                        icon: 'success',
                        title: 'Đã lưu!',
                        text: 'Thông tin kỹ thuật viên đã được cập nhật.',
                        timer: 1500,
                        showConfirmButton: false
                    });
                    if (window.htmx) {
                        htmx.trigger('#tech-list-container', 'techListUpdated'); // Refresh list
                    } else {
                        window.location.reload();
                    }
                } else {
                    throw new Error('Update failed');
                }
            } catch (error) {
                console.error(error);
                Swal.fire('Lỗi', 'Không thể cập nhật thông tin', 'error');
            } finally {
                this.isSubmitting = false;
            }
        },

        formatMoney(amount) {
            if (!amount) return '0 ₫';
            return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(amount);
        }
    };
}
