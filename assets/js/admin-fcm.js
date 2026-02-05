// Admin FCM Handler - iOS Push Notification Support
// Lắng nghe sự kiện từ firebase-messaging-client.js

(function () {
    'use strict';

    console.log('Admin FCM Handler loaded');

    // Lắng nghe sự kiện permission cần thiết
    window.addEventListener('fcm:permission-needed', function () {
        console.log('FCM: Permission needed event received');
        showNotificationButton();
    });

    // Hiển thị nút bật thông báo
    function showNotificationButton() {
        const btn = document.getElementById('btn-enable-notify');
        if (btn) {
            btn.classList.remove('hidden');
            console.log('Notification button shown');
        } else {
            console.error('Button #btn-enable-notify not found in DOM');
        }
    }

    // Xử lý khi user bấm nút "Bật thông báo"
    window.requestAdminPermission = async function () {
        console.log('User clicked Enable Notifications');

        if (!('serviceWorker' in navigator)) {
            alert('Trình duyệt không hỗ trợ Service Worker (cần HTTPS)');
            console.error('Service Worker not supported');
            return;
        }

        if (!window.fcmClient) {
            alert('Lỗi: Firebase Client chưa khởi tạo. Vui lòng reload trang.');
            console.error('FCM Client not initialized');
            return;
        }

        try {
            console.log('Requesting permission...');
            const permission = await window.fcmClient.requestPermission();

            if (permission === 'granted') {
                console.log('Permission granted. Getting token...');
                await window.fcmClient.getToken();

                // Ẩn nút sau khi thành công
                const btn = document.getElementById('btn-enable-notify');
                if (btn) btn.classList.add('hidden');

                alert('✅ Đã bật thông báo thành công!');
            } else {
                console.error('Permission denied');
                alert('⚠️ Bạn đã từ chối quyền thông báo.\n\nVui lòng vào Cài đặt trình duyệt để bật lại.');
            }
        } catch (error) {
            console.error('Registration error:', error.message);
            alert('Lỗi: ' + error.message + '\n\nNếu đang dùng iOS, cần truy cập qua HTTPS (không phải IP)\nVui lòng kiểm tra:\n- Đã Add to Home Screen chưa?\n- Đã bật HTTPS chưa?');
        }
    };

    // Auto-init: Kiểm tra permission ngay khi load trang
    document.addEventListener('DOMContentLoaded', async function () {
        console.log('Admin FCM: DOMContentLoaded');

        // Chờ FCM Client khởi tạo (có thể load chậm hơn)
        const waitForFCMClient = setInterval(function () {
            if (window.fcmClient) {
                clearInterval(waitForFCMClient);
                console.log('FCM Client found');

                // Nếu permission đã granted từ trước, tự động sync token
                if (Notification.permission === 'granted') {
                    console.log('Permission already granted. Auto-syncing token...');
                    window.fcmClient.getToken()
                        .then(() => console.log('Auto-sync completed'))
                        .catch(e => console.error('Auto-sync failed:', e));
                } else if (Notification.permission === 'default') {
                    console.log('Permission not granted yet. Waiting for event...');
                } else {
                    console.warn('Permission denied by user');
                    showNotificationButton(); // Vẫn hiện nút để user có thể thử lại
                }
            }
        }, 100);

        // Timeout sau 5s nếu không tìm thấy FCM Client
        setTimeout(function () {
            clearInterval(waitForFCMClient);
            if (!window.fcmClient) {
                console.error('FCM Client not found after 5s');
            }
        }, 5000);
    });

})();
