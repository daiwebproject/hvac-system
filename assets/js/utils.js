// assets/js/utils.js

// Format tiền tệ Việt Nam
window.formatMoney = function (amount) {
    return new Intl.NumberFormat('vi-VN', {
        style: 'currency',
        currency: 'VND'
    }).format(amount || 0);
};

// Scroll mượt đến section
window.scrollToSection = function (sectionId) {
    const element = document.getElementById(sectionId);
    if (element) {
        element.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
};

console.log('✅ Utils loaded');