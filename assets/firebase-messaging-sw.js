importScripts('https://www.gstatic.com/firebasejs/9.23.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/9.23.0/firebase-messaging-compat.js');

// CẤU HÌNH CHÍNH XÁC
const firebaseConfig = {
  apiKey: "AIzaSyB1zmMjyK6XtqVm8Kcu-EwwAUpfSTkg8AA",
  authDomain: "techapp-hvac.firebaseapp.com",
  projectId: "techapp-hvac",
  storageBucket: "techapp-hvac.firebasestorage.app",
  messagingSenderId: "250596752999",
  appId: "1:250596752999:web:6d810cf577eedfb7d55ec2",
  measurementId: "G-TDF9H77TG2"
};

firebase.initializeApp(firebaseConfig);
const messaging = firebase.messaging();

// 2. Handle Background Messages
messaging.onBackgroundMessage((payload) => {
  console.log('[FCM-SW] Received background message:', payload);

  // Fallback an toàn cho các trường thông báo
  const notificationTitle = payload.notification?.title || 'Thông báo mới';
  const notificationOptions = {
    body: payload.notification?.body || 'Bạn có nhiệm vụ mới',
    icon: payload.notification?.icon || '/assets/icons/icon-192x192.png',
    badge: '/assets/icons/icon-192x192.png', // Icon nhỏ trên thanh status (Android)
    tag: payload.data?.booking_id || 'general-notification', // Group thông báo theo Job ID
    renotify: true, // Rung lại nếu có tin mới cùng tag
    data: payload.data || {},
    requireInteraction: false, // Tự ẩn sau vài giây (true = user phải bấm đóng)
    actions: [
      { action: 'view', title: 'Xem chi tiết' }
    ]
  };

  return self.registration.showNotification(notificationTitle, notificationOptions);
});

// 3. Handle Notification Click (Smart Focus)
self.addEventListener('notificationclick', (event) => {
  console.log('[FCM-SW] Notification clicked', event);
  event.notification.close();

  // URL đích: Ưu tiên URL từ data gửi kèm, nếu không có về danh sách việc
  // Server nên gửi field `url` hoặc `booking_id` trong data
  let urlToOpen = event.notification.data?.url;
  if (!urlToOpen && event.notification.data?.booking_id) {
    urlToOpen = `/tech/job/${event.notification.data.booking_id}`;
  }
  if (!urlToOpen) urlToOpen = '/tech/jobs';

  // Điều hướng thông minh
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      // A. Tìm xem có tab nào của App đang mở không
      for (const client of clientList) {
        // Kiểm tra base URL (đảm bảo đúng là app của mình)
        const clientUrl = new URL(client.url, self.location.origin);

        if (clientUrl.hostname === self.location.hostname && 'focus' in client) {
          // Focus vào tab đó
          return client.focus().then((focusedClient) => {
            // Nếu URL hiện tại khác URL đích -> Điều hướng
            if (focusedClient.url !== new URL(urlToOpen, self.location.origin).href) {
              return focusedClient.navigate(urlToOpen);
            }
            return focusedClient;
          });
        }
      }

      // B. Nếu không có tab nào mở -> Mở cửa sổ mới
      if (clients.openWindow) {
        return clients.openWindow(urlToOpen);
      }
    })
  );
});

// 4. Handle Notification Close (Optional Analytics)
self.addEventListener('notificationclose', (event) => {
  // Có thể log analytics ở đây nếu user gạt bỏ thông báo
  console.log('[FCM-SW] Notification dismissed by user');
});