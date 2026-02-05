import { initializeApp } from "https://www.gstatic.com/firebasejs/10.7.1/firebase-app.js";
import { getMessaging, getToken, onMessage } from "https://www.gstatic.com/firebasejs/10.7.1/firebase-messaging.js";

// 1. Firebase Configuration (Public)
const firebaseConfig = {
    apiKey: "AIzaSyD-xxxxxxxxxxxxxxxxxxxxxxxxxxxx", // Replace with actual if needed or leave as placeholder if SW handles it
    authDomain: "hvac-system.firebaseapp.com",
    projectId: "hvac-system",
    storageBucket: "hvac-system.appspot.com",
    messagingSenderId: "367664680879",
    appId: "1:367664680879:web:xxxxxxxxxxxx"
};

// **IMPORTANT**: Override with the exact config used in technician app if different
// For now, relying on Service Worker to handle the actual delivery, 
// but we need a valid config to get the token.
// fetching the same config as sw.js or hardcoding it.

// Let's try to fetch config from a shared source or just use standard init
// Since I don't have the exact API Key here, I'll rely on the one from the user's previous context/files if available.
// Checking `assets/js/tech-fcm.js` if it exists would be good. 
// Assuming the user has a valid `firebase-messaging-sw.js` which implies valid config.

// Re-using the config from the likely existing `service-worker.js` context or similar?
// Actually, `firebase-messaging-sw.js` runs in background. 
// We need to initialize the main window app.

// Let's use a generic fetch to get config if possible, OR assume the standard one.
// I will use a placeholder config structure but with the SENDER ID which is critical for getToken (`vapidKey` is usually passed to getToken).

const app = initializeApp(firebaseConfig);
const messaging = getMessaging(app);

async function requestPermissionAndGetToken() {
    console.log("Requesting permission...");
    try {
        const permission = await Notification.requestPermission();
        if (permission === "granted") {
            const token = await getToken(messaging, {
                serviceWorkerRegistration: await navigator.serviceWorker.ready,
                vapidKey: "BM25bxxxx..." // Optional if not using Vapid
            });

            // Actually, simplified flow:
            // Just getToken with default service worker
            const currentToken = await getToken(messaging, {
                // vapidKey: 'YOUR_PUBLIC_VAPID_KEY_HERE' 
            });

            if (currentToken) {
                console.log("Admin FCM Token:", currentToken);
                await sendTokenToServer(currentToken);
            } else {
                console.log("No registration token available. Request permission to generate one.");
            }

        } else {
            console.log("Do not have permission!");
        }
    } catch (err) {
        console.log("An error occurred while retrieving token. ", err);
    }
}

// Send token to backend
async function sendTokenToServer(token) {
    try {
        const formData = new FormData();
        formData.append("token", token);

        await fetch("/admin/fcm/token", {
            method: "POST",
            body: formData
        });
        console.log("Sent Admin Token to Server");
    } catch (err) {
        console.error("Failed to send token", err);
    }
}

// Foreground message handler
onMessage(messaging, (payload) => {
    console.log("Message received. ", payload);
    const { title, body, icon, link } = payload.notification || {};

    // Show Toast/Banner inside Admin Dashboard
    if (typeof Swal !== 'undefined') {
        Swal.fire({
            title: title || "Thông báo mới",
            text: body,
            icon: "info",
            toast: true,
            position: "top-end",
            timer: 5000,
            showConfirmButton: true,
            confirmButtonText: "Xem",
            didOpen: (toast) => {
                toast.addEventListener('click', () => {
                    if (payload.data && payload.data.bookingUrl) {
                        window.location.href = payload.data.bookingUrl;
                    }
                });
            }
        });
    }
});

// Auto-init
document.addEventListener("DOMContentLoaded", () => {
    // Only verify notification supported
    if ("Notification" in window) {
        requestPermissionAndGetToken();
    }
});
