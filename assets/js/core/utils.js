/**
 * Common Utilities
 * @module core/utils
 */

/**
 * Format number as Vietnamese currency
 * @param {number} value
 * @returns {string}
 */
export function formatMoney(value) {
    return new Intl.NumberFormat('vi-VN', {
        style: 'currency',
        currency: 'VND',
    }).format(value);
}

/**
 * Format date as DD/MM/YYYY
 * @param {string|Date} date
 * @returns {string}
 */
export function formatDate(date) {
    if (!date) return '';
    const d = typeof date === 'string' ? new Date(date) : date;
    if (isNaN(d.getTime())) return date;

    const pad = (n) => n.toString().padStart(2, '0');
    return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()}`;
}

/**
 * Format booking time range
 * @param {string} rawTime - YYYY-MM-DD HH:MM format
 * @param {number} [durationHours=2] - Duration in hours
 * @returns {string} - "HH:MM - HH:MM ngày DD/MM/YYYY"
 */
export function formatBookingTime(rawTime, durationHours = 2) {
    if (!rawTime) return '';
    try {
        const date = new Date(rawTime);
        if (isNaN(date.getTime())) return rawTime;

        const endDate = new Date(date.getTime() + durationHours * 60 * 60 * 1000);
        const pad = (n) => n.toString().padStart(2, '0');

        return `${pad(date.getHours())}:${pad(date.getMinutes())} - ${pad(endDate.getHours())}:${pad(endDate.getMinutes())} ngày ${pad(date.getDate())}/${pad(date.getMonth() + 1)}/${date.getFullYear()}`;
    } catch (e) {
        return rawTime;
    }
}

/**
 * Get Vietnamese day name
 * @param {string} dateStr - YYYY-MM-DD format
 * @returns {string}
 */
export function getDayName(dateStr) {
    const days = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];
    return days[new Date(dateStr).getDay()];
}

/**
 * Generate Google Maps link
 * @param {number} lat
 * @param {number} lng
 * @returns {string}
 */
export function getMapLink(lat, lng) {
    if (!lat || !lng) return '#';
    return `https://www.google.com/maps?q=${lat},${lng}`;
}

/**
 * Get status badge class
 * @param {string} status
 * @returns {string}
 */
export function getStatusClass(status) {
    const classes = {
        pending: 'badge-warning',
        assigned: 'badge-info',
        moving: 'badge-primary',
        working: 'badge-secondary',
        completed: 'badge-success',
        cancelled: 'badge-error',
    };
    return classes[status] || 'badge-ghost';
}

/**
 * Get status label in Vietnamese
 * @param {string} status
 * @returns {string}
 */
export function getStatusLabel(status) {
    const labels = {
        pending: 'Chờ xử lý',
        assigned: 'Đã giao thợ',
        moving: 'Đang di chuyển',
        arrived: 'Đã đến',
        working: 'Đang làm',
        completed: 'Hoàn thành',
        cancelled: 'Đã hủy',
    };
    return labels[status] || status;
}

/**
 * Debounce function
 * @param {Function} fn
 * @param {number} delay
 * @returns {Function}
 */
export function debounce(fn, delay = 300) {
    let timeout;
    return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn(...args), delay);
    };
}
