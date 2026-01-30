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

// 3. FETCH (ĐÂY LÀ PHẦN QUAN TRỌNG NHẤT)
self.addEventListener('fetch', (event) => {
  const url = event.request.url;

  // A. Bỏ qua request không phải http
  if (!url.startsWith('http')) return;

  // B. [FIX LỖI 401] Bỏ qua API và POST request
  // Service Worker sẽ KHÔNG CHẶN các request này nữa
  // Trình duyệt sẽ tự gửi Cookie đi kèm -> Hết lỗi 401
  if (url.includes('/api/') || event.request.method === 'POST') {
    return;
  }

  // C. [FIX TEMPLATE CACHE] Bỏ qua HTML pages - không cache
  // Chỉ cache static assets (CSS, JS, images)
  const isHTMLPage = event.request.mode === 'navigate' ||
    event.request.destination === 'document' ||
    url.includes('/tech/') ||
    url.includes('/admin/') ||
    url.endsWith('.html');

  if (isHTMLPage) {
    // Network-first cho HTML pages
    return;
  }

  // D. Cache các file tĩnh như bình thường (CSS, JS, images)
  event.respondWith(
    caches.match(event.request).then((response) => {
      if (response) return response;

      return fetch(event.request).then((networkResponse) => {
        if (!networkResponse || networkResponse.status !== 200 || networkResponse.type !== 'basic') {
          return networkResponse;
        }
        const responseToCache = networkResponse.clone();
        caches.open(DYNAMIC_CACHE).then((cache) => {
          cache.put(event.request, responseToCache);
        });
        return networkResponse;
      });
    })
  );
});

// ... (Các phần Sync và Notification giữ nguyên)
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
      // Fetch trong Sync chạy ngầm nên cần credentials nếu server yêu cầu
      // Nhưng vì đây là background sync, cookie có thể không tồn tại nếu user đóng browser.
      // Tuy nhiên với lỗi hiện tại của bạn là ở frontend active, fix ở fetch listener là đủ.
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
self.addEventListener('push', (event) => {
  let data = { title: 'HVAC System', body: 'Thông báo mới', icon: '/assets/icons/icon-192x192.png' };
  if (event.data) {
    try { data = { ...data, ...event.data.json() }; }
    catch (e) { data.body = event.data.text(); }
  }
  event.waitUntil(self.registration.showNotification(data.title, data));
});
self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const urlToOpen = event.notification.data?.url || '/tech/jobs';
  event.waitUntil(clients.matchAll({ type: 'window' }).then(cl => {
    const client = cl.find(c => c.url === urlToOpen);
    return client ? client.focus() : clients.openWindow(urlToOpen);
  }));
});