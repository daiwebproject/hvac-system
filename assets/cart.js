// Alpine.js Cart Store for Inventory Management
// Handles parts selection during job completion
//

document.addEventListener('alpine:init', () => {
    Alpine.store('cart', {
        items: [],

        init() {
            this.items = JSON.parse(localStorage.getItem('hvac_cart') || '[]');
        },

        // Thêm vật tư vào giỏ (Hỗ trợ từ QR Scanner hoặc Click chọn)
        add(product, qty = 1) {
            const quantity = parseInt(qty);
            if (quantity <= 0) return;

            // Kiểm tra sản phẩm đã có trong giỏ chưa
            const existing = this.items.find(i => i.id === product.id);

            if (existing) {
                existing.quantity += quantity;
            } else {
                this.items.push({
                    id: product.id,
                    name: product.name,
                    price: parseFloat(product.price || 0),
                    image: product.image || '', // Optional
                    quantity: quantity
                });
            }

            this.save();
            this.showNotification(`Đã thêm: ${product.name}`);
        },

        // Xóa vật tư
        remove(productId) {
            this.items = this.items.filter(i => i.id !== productId);
            this.save();
        },

        // Cập nhật số lượng (+/-)
        updateQuantity(productId, quantity) {
            const qty = parseInt(quantity);
            if (qty <= 0) {
                this.remove(productId);
                return;
            }

            const item = this.items.find(i => i.id === productId);
            if (item) {
                item.quantity = qty;
                this.save();
            }
        },

        // Tính tổng tiền vật tư
        get total() {
            return this.items.reduce((sum, item) =>
                sum + (item.price * item.quantity), 0);
        },

        // Tính tổng số lượng item
        get count() {
            return this.items.reduce((sum, item) =>
                sum + item.quantity, 0);
        },

        // Lưu vào LocalStorage & Bắn sự kiện cập nhật
        save() {
            localStorage.setItem('hvac_cart', JSON.stringify(this.items));

            // 1. Trigger Animation cho Badge (nếu có trong DOM)
            const badge = document.querySelector('.cart-badge');
            if (badge) {
                badge.classList.remove('animate-bounce');
                void badge.offsetWidth; // Force reflow
                badge.classList.add('animate-bounce');
            }

            // 2. Dispatch Custom Event để các component khác (như Navbar) bắt được
            window.dispatchEvent(new CustomEvent('cart-updated', {
                detail: {
                    count: this.count,
                    total: this.total,
                    items: this.items
                }
            }));
        },

        // Xóa sạch giỏ (sau khi Submit Job thành công)
        clear() {
            this.items = [];
            this.save();
        },

        // Hiển thị thông báo Toast
        showNotification(message) {
            const toast = document.createElement('div');
            // Style Tailwind + DaisyUI
            toast.className = 'toast toast-top toast-end z-50';
            toast.innerHTML = `
                <div class="alert alert-success shadow-lg text-white">
                    <div>
                        <i class="fas fa-check-circle"></i>
                        <span>${message}</span>
                    </div>
                </div>
            `;
            document.body.appendChild(toast);

            // Auto remove
            setTimeout(() => {
                toast.classList.add('opacity-0', 'transition-opacity', 'duration-500');
                setTimeout(() => toast.remove(), 500);
            }, 2000);
        }
    });
});

// Helper: Format tiền tệ (Global helper)
window.formatMoney = function (amount) {
    return new Intl.NumberFormat('vi-VN', {
        style: 'currency',
        currency: 'VND'
    }).format(amount || 0);
};

// Helper: Scroll mượt
window.scrollToSection = function (sectionId) {
    const element = document.getElementById(sectionId);
    if (element) {
        element.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
};