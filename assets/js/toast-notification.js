function toastNotification() {
    return {
        toasts: [],

        addToast(title, message, type = 'info') {
            const id = Date.now();
            this.toasts.push({ id, title, message, type, visible: true });

            // Auto remove after 5 seconds
            setTimeout(() => {
                const index = this.toasts.findIndex(t => t.id === id);
                if (index !== -1) this.toasts[index].visible = false;
                // Clean up array later
                setTimeout(() => {
                    this.toasts = this.toasts.filter(t => t.id !== id);
                }, 300);
            }, 5000);
        },

        handleJobAssigned(event) {
            const data = JSON.parse(event.detail.data);
            console.log('Job Assigned:', data);

            // 1. Play sound (optional)
            // new Audio('/assets/sounds/notification.mp3').play().catch(()=>{});

            // 2. Show Toast
            this.addToast('Công việc mới', `Bạn vừa nhận được việc từ khách hàng ${data.customer_name}`, 'info');

            // 3. Trigger HTMX reload of "New Jobs" list if visible
            const listContainer = document.getElementById('job-list-container');
            if (listContainer) {
                htmx.trigger(listContainer, 'load');
            }

            // 4. Update Badge (if applicable)
            // We can reuse the offlineIndicator logic or a separate store
            // But for now, HTMX reload handles the content.
        },

        handleJobCancelled(event) {
            const data = JSON.parse(event.detail.data);
            console.log('Job Cancelled:', data);

            // 1. Show Toast (Error style)
            this.addToast('Đã hủy đơn', `Đơn hàng #${data.booking_id.substring(0, 8)}... đã bị Admin hủy`, 'error');

            // 2. Remove card from UI immediately if present
            // We try to find any element with data-booking-id="{id}" or simply reload list
            const card = document.querySelector(`[data-job-id="${data.booking_id}"]`);
            if (card) {
                card.remove();
            } else {
                // Reload list to be safe
                const listContainer = document.getElementById('job-list-container');
                if (listContainer) {
                    htmx.trigger(listContainer, 'load');
                }
            }

            // If we are in Job Detail page of that job, redirect out
            if (window.location.pathname.includes(data.booking_id)) {
                alert('Công việc này đã bị hủy!');
                window.location.href = '/tech/jobs';
            }
        },

        handleHtmxError(event) {
            const xhr = event.detail.xhr;
            if (xhr.status === 409) {
                Swal.fire({
                    title: 'Lỗi xung đột!',
                    text: xhr.responseText || 'Trạng thái đơn hàng đã thay đổi.',
                    icon: 'error',
                    confirmButtonText: 'Tải lại trang'
                }).then(() => {
                    window.location.reload();
                });
            } else if (xhr.status >= 400) {
                // Generic error
                Swal.fire({
                    title: 'Lỗi',
                    text: xhr.responseText || 'Có lỗi xảy ra',
                    icon: 'error'
                });
            }
        },

        handleOnlineSync() {
            console.log('Network back online: Syncing UI...');

            // 1. Reload Job List if present
            const listContainer = document.getElementById('job-list-container');
            if (listContainer) {
                htmx.trigger(listContainer, 'load'); // Assumes hx-trigger="load" or compatible
            } else {
                // If checking dashboard counts or generic page
                // window.location.reload(); // Aggressive but safe
                // Less aggressive: Try to reload main content if it's an HTMX boosted page
                // htmx.ajax('GET', window.location.pathname, {target:'#main-content', select:'#main-content'});
            }

            // 2. Re-trigger SSE connection (HTMX SSE extension mostly handles this)
            const body = document.querySelector('body');
            if (body && body.getAttribute('sse-connect')) {
                // HTMX 1.9.x SSE extension reconnect logic
            }

            this.addToast('Đã có mạng', 'Đang đồng bộ lại dữ liệu...', 'success');
        }
    };
}
