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
            await window.fcmClient.manualRequestPermission();

            // After successful permission, the token is automatically fetched
            // Ẩn nút sau khi thành công

            const btn = document.getElementById('btn-enable-notify');
            if (btn) btn.classList.add('hidden');

            alert('✅ Đã bật thông báo thành công!');
        } catch (error) {
            console.error('Registration error:', error.message);
            if (error.code === 'messaging/permission-blocked' || error.code === 'messaging/permission-denied') {
                alert('⚠️ Bạn đã từ chối quyền thông báo.\n\nVui lòng vào Cài đặt trình duyệt để bật lại.');
            } else {
                alert('Lỗi: ' + error.message + '\n\nNếu đang dùng iOS, cần truy cập qua HTTPS (không phải IP)\nVui lòng kiểm tra:\n- Đã Add to Home Screen chưa?\n- Đã bật HTTPS chưa?');
            }
        }
    };

    // Auto-init: Kiểm tra permission ngay khi load trang
    document.addEventListener('DOMContentLoaded', async function () {
        console.log('Admin FCM: DOMContentLoaded');

        // Auto-register token if permission already granted
        if (Notification.permission === 'granted') {
            console.log('[ADMIN] Permission granted. Waiting for SW ready...');

            if ('serviceWorker' in navigator) {
                try {
                    // Wait for Service Worker to be ready
                    const registration = await navigator.serviceWorker.ready;
                    console.log('[ADMIN] SW Ready. Scope:', registration.scope);

                    // Check Firebase availability
                    if (typeof firebase === 'undefined' || !firebase.messaging) {
                        throw new Error('Firebase not loaded');
                    }

                    const messaging = firebase.messaging();

                    // Get token directly from Firebase, specifying the Service Worker registration
                    const token = await messaging.getToken({
                        vapidKey: window.VAPID_PUBLIC_KEY,
                        serviceWorkerRegistration: registration
                    });

                    if (!token) {
                        throw new Error('No token received from Firebase');
                    }

                    console.log('[ADMIN] Got FCM token:', token.substring(0, 30) + '...');

                    // Send to server directly
                    const response = await fetch('/admin/fcm/token', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ token: token }),
                        credentials: 'include'
                    });

                    if (response.ok) {
                        const data = await response.json();
                        console.log('[ADMIN] Token sent successfully:', data);
                    } else {
                        const errText = await response.text();
                        console.error('[ADMIN] Server error:', response.status, errText);
                    }

                } catch (error) {
                    console.error('[ADMIN] Error registering FCM token:', error);
                }
            } else {
                console.error('[ADMIN] Service Worker not supported in this browser');
            }
        } else {
            console.log('[ADMIN] Permission not granted, waiting for user action');
        }
    });

})();
