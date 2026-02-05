// Service Worker for Offline-First Support
// [Updated: Version v8 - FIX LỖI 401]

const CACHE_VERSION = 'hvac-tech-v9'; // <--- TĂNG LÊN v9 ĐỂ ÉP CẬP NHẬT
const CACHE_NAME = `static-${CACHE_VERSION}`;
const DYNAMIC_CACHE = `dynamic-${CACHE_VERSION}`;
const OFFLINE_DB = 'hvac_offline_db';

const STATIC_ASSETS = [
  '/',
  '/tech/login',
  '/tech/jobs',
  '/tech/dashboard',
  '/assets/style.css',
  '/assets/manifest.json',
  // '/assets/alpine-components.js',
  '/assets/js/utils.js',
  '/assets/js/tech.js',
  '/assets/js/admin.js',
  '/assets/js/public.js',
  '/assets/htmx-sse.js',
  '/assets/qr-scanner.js',
  '/assets/offline-reporter.js',
  '/assets/cart.js',
  // '/assets/icons/icon-192x192.png' // Đảm bảo file tồn tại nếu bật dòng này
];

// 1. INSTALL
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(STATIC_ASSETS))
      .then(() => self.skipWaiting())
  );
});

// 2. ACTIVATE
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME && cacheName !== DYNAMIC_CACHE) {
            console.log('[SW] Xóa cache cũ:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// 3. FETCH - NETWORK ONLY (No Cache)
self.addEventListener('fetch', (event) => {
  // Pass through everything
  return;
});

// ... (Các phần Sync và Notification giữ nguyên)
// --- 4. FIREBASE MESSAGING INTEGRATION ---
importScripts('https://www.gstatic.com/firebasejs/9.23.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/9.23.0/firebase-messaging-compat.js');

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

// Handle Background Messages via Firebase SDK
messaging.onBackgroundMessage((payload) => {
  console.log('[SW] Firebase Message:', payload);
  const title = payload.notification?.title || 'Thông báo mới';
  const options = {
    body: payload.notification?.body || 'Bạn có thông báo mới',
    icon: payload.notification?.icon || '/assets/icons/icon-192x192.png',
    badge: '/assets/icons/icon-192x192.png',
    data: payload.data || {},
    actions: [{ action: 'view', title: 'Xem chi tiết' }]
  };
  return self.registration.showNotification(title, options);
});

// Sync Logic
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-reports' || event.tag === 'sync-offline-jobs') {
    event.waitUntil(syncReports());
  }
});

async function syncReports() {
  const db = await openOfflineDB();
  const pendingReports = await getAllPendingReports(db);

  for (const report of pendingReports) {
    try {
      const formData = new FormData();
      formData.append('notes', report.notes);
      formData.append('parts_json', report.parts_json || JSON.stringify(report.parts));
      if (report.photos && Array.isArray(report.photos)) {
        for (let i = 0; i < report.photos.length; i++) {
          const blob = await getBlobFromStore(db, report.photos[i]);
          if (blob) formData.append('after_images', blob, `evidence_${i}.jpg`);
        }
      }
      const response = await fetch(`/tech/job/${report.jobId}/complete`, {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        await deleteReport(db, report.id);
        notifyClients({ type: 'REPORT_SYNCED', reportId: report.id });
      }
    } catch (error) { console.error('[SW] Sync failed:', error); }
  }
}

function openOfflineDB() {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(OFFLINE_DB, 1);
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    request.onupgradeneeded = (event) => {
      const db = event.target.result;
      if (!db.objectStoreNames.contains('job_reports')) db.createObjectStore('job_reports', { keyPath: 'id' });
      if (!db.objectStoreNames.contains('sync_queue')) db.createObjectStore('sync_queue', { keyPath: 'id' });
    };
  });
}
function getAllPendingReports(db) {
  return new Promise((resolve) => {
    const tx = db.transaction(['job_reports'], 'readonly');
    tx.objectStore('job_reports').getAll().onsuccess = (e) => resolve(e.target.result.filter(r => !r.synced));
  });
}
function getBlobFromStore(db, key) {
  return new Promise((resolve) => {
    const tx = db.transaction(['sync_queue'], 'readonly');
    const req = tx.objectStore('sync_queue').get(key);
    req.onsuccess = (e) => resolve(e.target.result?.data);
    req.onerror = () => resolve(null);
  });
}
function deleteReport(db, id) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(['job_reports'], 'readwrite');
    tx.objectStore('job_reports').delete(id);
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject();
  });
}
async function notifyClients(msg) {
  const allClients = await clients.matchAll({ includeUncontrolled: true });
  allClients.forEach(client => client.postMessage(msg));
}

// Notification Click Handle (Merged)
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const urlToOpen = event.notification.data?.url || '/tech/jobs';
  event.waitUntil(clients.matchAll({ type: 'window' }).then(cl => {
    const client = cl.find(c => c.url === urlToOpen);
    return client ? client.focus() : clients.openWindow(urlToOpen);
  }));
});