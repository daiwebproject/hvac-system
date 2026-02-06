// Alpine.js module for offline-first job reporting
// Handles IndexedDB storage and sync management

class OfflineJobReporter {
  constructor() {
    this.db = null;
    this.isOnline = navigator.onLine;
    this.pendingSyncs = 0;

    this.initDB();
    this.setupEventListeners();
    this.registerServiceWorker();

    // NEW: Lắng nghe thông báo từ Service Worker khi sync ngầm thành công
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('message', (event) => {
        if (event.data && event.data.type === 'REPORT_SYNCED') {
          console.log('Received sync confirmation from SW:', event.data.reportId);
          this.markReportSynced(event.data.reportId).then(() => {
            // Bắn event để UI (Alpine.js) cập nhật số lượng báo cáo chờ
            window.dispatchEvent(new CustomEvent('report-synced', { detail: { reportId: event.data.reportId } }));
          });
        }
      });
    }
  }

  // Initialize IndexedDB
  async initDB() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open('hvac_offline_db', 1);

      request.onerror = () => {
        console.error('IndexedDB error:', request.error);
        reject(request.error);
      };

      request.onsuccess = () => {
        this.db = request.result;
        resolve(this.db);
      };

      request.onupgradeneeded = (event) => {
        const db = event.target.result;

        // Create object stores
        if (!db.objectStoreNames.contains('job_reports')) {
          const store = db.createObjectStore('job_reports', { keyPath: 'id' });
          store.createIndex('synced', 'synced', { unique: false });
          store.createIndex('timestamp', 'timestamp', { unique: false });
        }

        if (!db.objectStoreNames.contains('sync_queue')) {
          db.createObjectStore('sync_queue', { keyPath: 'id' });
        }
      };
    });
  }

  // Register Service Worker
  async registerServiceWorker() {
    if ('serviceWorker' in navigator) {
      try {
        // Register at root for app-wide scope
        const registration = await navigator.serviceWorker.register('/service-worker.js', {
          scope: '/',
        });
        console.log('Service Worker registered:', registration);
      } catch (error) {
        console.error('Service Worker registration failed:', error);
      }
    }
  }

  // Setup online/offline event listeners
  setupEventListeners() {
    window.addEventListener('online', () => {
      this.isOnline = true;
      console.log('Online - starting sync');
      this.syncPendingReports(); // Manual sync fallback

      // Dispatch event to notify UI to reload data
      window.dispatchEvent(new CustomEvent('network:online'));

      // Reconnect SSE if HTMX is used
      if (typeof htmx !== 'undefined') {
        // Find element with sse-connect (usually body) and trigger connect
        const sseEl = document.querySelector('[sse-connect]');
        if (sseEl) {
          // HTMX SSE extension re-connects on node swap or explicit trigger logic?
          // The extension usually auto-reconnects, but we can force it or just rely on list reload.
          console.log('Online: Triggering UI refresh');
        }
      }
    });

    window.addEventListener('offline', () => {
      this.isOnline = false;
      console.log('Offline - storing locally');
    });
  }

  // Save job report locally
  async saveJobReport(jobData) {
    if (!this.db) await this.initDB();

    const report = {
      id: `report_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      jobId: jobData.jobId,
      // techId: jobData.techId, // Auth handled by Cookie/Session
      notes: jobData.notes,
      parts: jobData.parts || [],
      photos: jobData.photos || [], // Array of keys pointing to 'sync_queue' blobs
      timestamp: Date.now(),
      synced: false,
      syncAttempts: 0,
    };

    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction(['job_reports'], 'readwrite');
      const store = transaction.objectStore('job_reports');
      const request = store.add(report);

      request.onerror = () => {
        console.error('Error saving report:', request.error);
        reject(request.error);
      };

      request.onsuccess = async () => {
        console.log('Report saved locally:', report.id);

        // IMPROVEMENT: Đăng ký Background Sync
        if ('serviceWorker' in navigator && 'sync' in navigator.serviceWorker.ready) {
          try {
            const sw = await navigator.serviceWorker.ready;
            await sw.sync.register('sync-reports');
            console.log('Background Sync registered');
          } catch (e) {
            console.warn('Background Sync registration failed, using manual sync:', e);
          }
        }

        // Nếu đang online thì thử gửi luôn (Fallback)
        if (this.isOnline) {
          this.uploadReport(report).catch(err => console.log('Manual upload deferred:', err));
        }

        resolve(report);
      };
    });
  }

  // Get all pending reports
  async getPendingReports() {
    if (!this.db) await this.initDB();

    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction(['job_reports'], 'readonly');
      const store = transaction.objectStore('job_reports');
      const index = store.index('synced');
      const request = index.getAll();

      request.onerror = () => reject(request.error);
      request.onsuccess = () => {
        const results = request.result.filter(r => r.synced === false);
        resolve(results);
      };
    });
  }

  // Sync pending reports with backend (Manual / Fallback)
  async syncPendingReports() {
    if (!this.isOnline) {
      console.log('Offline - skipping sync');
      return;
    }

    try {
      const pendingReports = await this.getPendingReports();

      if (pendingReports.length === 0) {
        return;
      }

      console.log(`Syncing ${pendingReports.length} reports...`);

      for (const report of pendingReports) {
        try {
          await this.uploadReport(report);
        } catch (error) {
          console.error('Error uploading report:', error);
          await this.incrementSyncAttempts(report.id);
        }
      }
    } catch (error) {
      console.error('Sync error:', error);
    }
  }

  // Upload single report
  async uploadReport(report) {
    // Check if already synced to avoid double submission
    if (report.synced) return;

    const formData = new FormData();
    // formData.append('job_id', report.jobId); // ID nằm trên URL
    formData.append('notes', report.notes);

    // FIX: Match field name with Go handler (parts_json)
    formData.append('parts_json', JSON.stringify(report.parts));

    // Handle photos
    if (report.photos && report.photos.length > 0) {
      for (let i = 0; i < report.photos.length; i++) {
        const blob = await this.getBlob(report.photos[i]);
        if (blob) {
          // FIX: Match field name with Go handler (after_images)
          formData.append('after_images', blob, `evidence_${i}.jpg`);
        }
      }
    }

    // FIX: Match endpoint with Router (/tech/job/{id}/complete)
    const response = await fetch(`/tech/job/${report.jobId}/complete`, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`Upload failed: ${response.statusText}`);
    }

    // Mark as synced
    await this.markReportSynced(report.id);
    console.log('Report synced:', report.id);

    // Notify UI immediately
    window.dispatchEvent(new CustomEvent('report-synced', { detail: { reportId: report.id } }));
  }

  // Mark report as synced
  async markReportSynced(reportId) {
    if (!this.db) await this.initDB();
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction(['job_reports'], 'readwrite');
      const store = transaction.objectStore('job_reports');
      const getRequest = store.get(reportId);

      getRequest.onsuccess = () => {
        const report = getRequest.result;
        if (report) {
          report.synced = true;
          report.syncedAt = Date.now();
          const updateRequest = store.put(report);
          updateRequest.onerror = () => reject(updateRequest.error);
          updateRequest.onsuccess = () => resolve();
        } else {
          resolve(); // Report might have been deleted or not found
        }
      };

      getRequest.onerror = () => reject(getRequest.error);
    });
  }

  // Increment sync attempts
  async incrementSyncAttempts(reportId) {
    if (!this.db) await this.initDB();
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction(['job_reports'], 'readwrite');
      const store = transaction.objectStore('job_reports');
      const getRequest = store.get(reportId);

      getRequest.onsuccess = () => {
        const report = getRequest.result;
        if (report) {
          report.syncAttempts = (report.syncAttempts || 0) + 1;
          const updateRequest = store.put(report);
          updateRequest.onerror = () => reject(updateRequest.error);
          updateRequest.onsuccess = () => resolve();
        } else {
          resolve();
        }
      };

      getRequest.onerror = () => reject(getRequest.error);
    });
  }

  // Get blob from indexedDB
  async getBlob(blobKey) {
    if (!this.db) await this.initDB();
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction(['sync_queue'], 'readonly');
      const store = transaction.objectStore('sync_queue');
      const request = store.get(blobKey);

      request.onsuccess = () => resolve(request.result?.data);
      request.onerror = () => reject(request.error);
    });
  }

  // Check sync status
  async getSyncStatus() {
    const pending = await this.getPendingReports();
    return {
      isOnline: this.isOnline,
      pendingReports: pending.length,
      pendingSyncs: this.pendingSyncs,
    };
  }
}

// Initialize globally
window.OfflineJobReporter = new OfflineJobReporter();

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = OfflineJobReporter;
}
// Alpine.js integration for offline status
function offlineIndicator() {
  return {
    isOnline: navigator.onLine,
    pendingCount: 0,

    init() {
      // Sync initial state
      this.updateStatus();

      // Listen for network changes
      window.addEventListener('online', () => {
        this.isOnline = true;
        this.updateStatus();
      });
      window.addEventListener('offline', () => {
        this.isOnline = false;
      });

      // Listen for sync events
      window.addEventListener('report-synced', () => this.updateStatus());
      window.addEventListener('network:online', () => this.updateStatus());

      // Poll periodically just in case
      setInterval(() => this.updateStatus(), 10000);
    },

    async updateStatus() {
      if (window.OfflineJobReporter) {
        const status = await window.OfflineJobReporter.getSyncStatus();
        this.pendingCount = status.pendingReports || 0;
        // this.isOnline is handled by event listeners effectively, but syncing doesn't hurt
        // this.isOnline = status.isOnline; 
      }
    }
  };
}
