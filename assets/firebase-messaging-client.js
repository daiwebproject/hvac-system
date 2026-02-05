// assets/js/firebase-messaging-client.js

class FirebaseMessagingClient {
  constructor(config) {
    this.firebaseConfig = config;
    this.messaging = null;
  }

  async init() {
    try {
      if (typeof firebase === 'undefined') {
        console.error("Firebase SDK missing.");
        return;
      }
      if (!firebase.apps.length) {
        firebase.initializeApp(this.firebaseConfig);
        console.log('Firebase initialized');
      } else {
        firebase.app();
      }

      if (firebase.messaging.isSupported()) {
        this.messaging = firebase.messaging();

        // Checks current permission
        const permission = Notification.permission;

        if (permission === 'granted') {
          // Already granted, get token and setup
          await this.requestPermissionAndGetToken();
          this.setupOnMessage();
        } else if (permission === 'default') {
          // Wait for user gesture
          console.log('FCM: Waiting for user gesture to enable notifications');
          // Dispatch event to show UI button
          window.dispatchEvent(new CustomEvent('fcm:permission-needed'));
        } else {
          console.warn('FCM: Notification permission denied');
        }
      } else {
        console.warn('FCM not supported');
      }
    } catch (error) {
      console.error('FCM Init Error:', error);
    }
  }

  // Exposed method for Button Click
  async manualRequestPermission() {
    await this.requestPermissionAndGetToken();
    this.setupOnMessage();
  }

  async requestPermissionAndGetToken() {
    try {
      const permission = await Notification.requestPermission();
      if (permission === 'granted') {
        console.log('Notification granted.');
        await this.getToken();
        window.dispatchEvent(new CustomEvent('fcm:granted')); // Hide button
      } else {
        console.warn('Notification denied.');
      }
    } catch (error) {
      console.error('Permission error:', error);
    }
  }

  async getToken() {
    try {
      console.log('FCM: Getting Token...');
      // WAITING FOR SERVICE WORKER READY
      const registration = await navigator.serviceWorker.ready;
      console.log('FCM: SW Ready. Scope:', registration.scope);

      const currentToken = await this.messaging.getToken({
        vapidKey: "BM0Uvapd87utXwp2bBC_23HMT3LjtSwWGq6rUU8FnK6DvnJnTDCR_Kj4mGAC-HLgoia-tgjobgSWDpDJkKX_DBk",
        serviceWorkerRegistration: registration // <--- KEY FIX: Sử dụng lại SW chính
      });

      if (currentToken) {
        console.log('FCM Token generated:', currentToken.substring(0, 10) + "...");
        await this.sendTokenToServer(currentToken);
      } else {
        console.warn('No token available from FCM.');
        alert('Lỗi: FCM không trả về Token nào.');
      }
    } catch (error) {
      console.error('Get Token Error:', error);
      alert('Lỗi khi lấy Token: ' + error.message);
    }
  }

  async sendTokenToServer(token) {
    try {
      // --- [SỬA TỪ ĐÂY] ---
      // Bỏ đoạn FormData cũ đi
      // const formData = new FormData();
      // formData.append('token', token);

      // Determine endpoint based on URL
      let endpoint = '/api/tech/fcm/token';
      if (window.location.pathname.startsWith('/admin')) {
        endpoint = '/admin/fcm/token';
        console.log('🔵 [ADMIN] Sending FCM token to:', endpoint);
      } else {
        console.log('🟢 [TECH] Sending FCM token to:', endpoint);
      }

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ token: token }),
        credentials: 'include'
      });

      if (response.ok) {
        console.log('Token sent to server successfully');
        const data = await response.json();
        console.log('Server response:', data);
      } else {
        const errText = await response.text();
        console.error('Failed to send token:', response.status, errText);
        alert('Lỗi gửi Token: ' + errText + '\n(Vui lòng báo cho Admin/Dev)');
      }
    } catch (error) {
      console.error('Send token error:', error);
    }
  }

  setupOnMessage() {
    this.messaging.onMessage((payload) => {
      console.log('Message received: ', payload);
      const { title, body, icon } = payload.notification || {};

      if (typeof Swal !== 'undefined') {
        Swal.fire({
          title: title,
          text: body,
          icon: 'info',
          toast: true,
          position: 'top-end',
          showConfirmButton: false,
          timer: 5000
        });
      } else {
        new Notification(title, { body, icon });
      }
    });
  }
}

function initializeFirebaseClient() {
  if (window.fcmClient) return;

  // Chặn chạy ở trang Login để tránh lỗi 401
  if (window.location.pathname.includes('/login')) {
    console.log('FCM: Skipped on login page');
    return;
  }

  const config = {
    apiKey: "AIzaSyB1zmMjyK6XtqVm8Kcu-EwwAUpfSTkg8AA",
    authDomain: "techapp-hvac.firebaseapp.com",
    projectId: "techapp-hvac",
    storageBucket: "techapp-hvac.firebasestorage.app",
    messagingSenderId: "250596752999",
    appId: "1:250596752999:web:6d810cf577eedfb7d55ec2",
    measurementId: "G-TDF9H77TG2"
  };

  window.fcmClient = new FirebaseMessagingClient(config);
  window.fcmClient.init();

  // Global helper for UI buttons
  window.enableNotifications = () => {
    console.log('enableNotifications called');
    window.fcmClient.manualRequestPermission();
  };
}
// function initializeFirebaseClient() {
//   if (window.fcmClient) return;

//   // Chặn chạy ở trang Login để tránh lỗi 401
//   if (window.location.pathname.includes('/login')) {
//     console.log('FCM: Skipped on login page');
//     return;
//   }

//   const config = {
//     apiKey: "AIzaSyB1zmMjyK6XtqVm8Kcu-EwwAUpfSTkg8AA",
//     authDomain: "techapp-hvac.firebaseapp.com",
//     projectId: "techapp-hvac",
//     storageBucket: "techapp-hvac.firebasestorage.app",
//     messagingSenderId: "250596752999",
//     appId: "1:250596752999:web:6d810cf577eedfb7d55ec2",
//     measurementId: "G-TDF9H77TG2"
//   };

//   window.fcmClient = new FirebaseMessagingClient(config);
//   window.fcmClient.init();
// }

document.addEventListener('DOMContentLoaded', initializeFirebaseClient);