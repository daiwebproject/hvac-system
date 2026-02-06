/**
 * Toast Notifications - SweetAlert2 wrapper
 * @module core/toast
 */

// Check if SweetAlert2 is available
const Swal = window.Swal;

/**
 * Default toast configuration
 */
const defaultToastConfig = {
    toast: true,
    position: 'top-end',
    showConfirmButton: false,
    timer: 3000,
    timerProgressBar: true,
};

/**
 * Toast notification utilities
 */
export const toast = {
    /**
     * Show success notification
     * @param {string} message
     * @param {Object} [options]
     */
    success(message, options = {}) {
        if (!Swal) {
            console.log('✅', message);
            return;
        }
        Swal.fire({
            ...defaultToastConfig,
            icon: 'success',
            title: message,
            ...options,
        });
    },

    /**
     * Show error notification
     * @param {string} message
     * @param {Object} [options]
     */
    error(message, options = {}) {
        if (!Swal) {
            console.error('❌', message);
            return;
        }
        Swal.fire({
            ...defaultToastConfig,
            icon: 'error',
            title: message,
            timer: 5000,
            ...options,
        });
    },

    /**
     * Show warning notification
     * @param {string} message
     * @param {Object} [options]
     */
    warning(message, options = {}) {
        if (!Swal) {
            console.warn('⚠️', message);
            return;
        }
        Swal.fire({
            ...defaultToastConfig,
            icon: 'warning',
            title: message,
            ...options,
        });
    },

    /**
     * Show info notification
     * @param {string} message
     * @param {Object} [options]
     */
    info(message, options = {}) {
        if (!Swal) {
            console.info('ℹ️', message);
            return;
        }
        Swal.fire({
            ...defaultToastConfig,
            icon: 'info',
            title: message,
            ...options,
        });
    },

    /**
     * Show confirmation dialog
     * @param {string} title
     * @param {string} text
     * @param {Object} [options]
     * @returns {Promise<boolean>}
     */
    async confirm(title, text, options = {}) {
        if (!Swal) {
            return window.confirm(`${title}\n${text}`);
        }

        const result = await Swal.fire({
            title,
            text,
            icon: 'question',
            showCancelButton: true,
            confirmButtonText: options.confirmText || 'Xác nhận',
            cancelButtonText: options.cancelText || 'Hủy',
            confirmButtonColor: '#3085d6',
            cancelButtonColor: '#d33',
            ...options,
        });

        return result.isConfirmed;
    },

    /**
     * Show loading state
     * @param {string} [message='Đang xử lý...']
     */
    loading(message = 'Đang xử lý...') {
        if (!Swal) return;
        Swal.fire({
            title: message,
            allowOutsideClick: false,
            didOpen: () => Swal.showLoading(),
        });
    },

    /**
     * Close any open toast/dialog
     */
    close() {
        if (Swal) Swal.close();
    },
};

export default toast;
