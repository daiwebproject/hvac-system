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
        await this.requestPermissionAndGetToken();
        this.setupOnMessage();
      } else {
        console.warn('FCM not supported');
      }
    } catch (error) {
      console.error('FCM Init Error:', error);
    }
  }

  async requestPermissionAndGetToken() {
    try {
      const permission = await Notification.requestPermission();
      if (permission === 'granted') {
        console.log('Notification granted.');
        await this.getToken();
      } else {
        console.warn('Notification denied.');
      }
    } catch (error) {
      console.error('Permission error:', error);
    }
  }

  async getToken() {
    try {
      // WAITING FOR SERVICE WORKER READY
      const registration = await navigator.serviceWorker.ready;

      const currentToken = await this.messaging.getToken({
        vapidKey: "BM0Uvapd87utXwp2bBC_23HMT3LjtSwWGq6rUU8FnK6DvnJnTDCR_Kj4mGAC-HLgoia-tgjobgSWDpDJkKX_DBk",
        serviceWorkerRegistration: registration // <--- KEY FIX: Sử dụng lại SW chính
      });

      if (currentToken) {
        console.log('FCM Token:', currentToken.substring(0, 10) + "...");
        await this.sendTokenToServer(currentToken);
      } else {
        console.log('No token available.');
      }
    } catch (error) {
      console.error('Get Token Error:', error);
    }
  }

  async sendTokenToServer(token) {
    try {
      // --- [SỬA TỪ ĐÂY] ---
      // Bỏ đoạn FormData cũ đi
      // const formData = new FormData();
      // formData.append('token', token);

      const response = await fetch('/api/tech/fcm/token', {
        method: 'POST',
        // [QUAN TRỌNG] Thêm Header báo là gửi JSON
        headers: {
          'Content-Type': 'application/json'
        },
        // [QUAN TRỌNG] Gói dữ liệu thành chuỗi JSON
        body: JSON.stringify({ token: token }),

        credentials: 'include' // Giữ nguyên để gửi Cookie
      });
      // --- [HẾT PHẦN SỬA] ---

      if (response.ok) {
        console.log('Token sent to server successfully');
      } else {
        // Log text lỗi ra để dễ debug
        console.error('Failed to send token:', await response.text());
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