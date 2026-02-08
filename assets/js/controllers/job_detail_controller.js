window.JobDetailController = function (data) {
    return {
        id: data.id,
        lat: data.lat,
        long: data.long,
        address: data.address,
        status: data.status,
        loading: false,
        showCancelModal: false,
        cancelReason: '',
        cancelNote: '',
        newTime: '',
        showNotesModal: false,
        showUpdateModal: false,
        evidenceNote: '',

        init() {
            // Lắng nghe sự kiện SSE từ global
            document.body.addEventListener('job.cancelled', (e) => {
                const data = e.detail;
                if (data && (data.booking_id === this.id || data.job_id === this.id)) {
                    this.handleServerCancellation(data);
                }
            });

            document.body.addEventListener('job.status_changed', (e) => {
                const data = e.detail;
                if (data && (data.job_id === this.id || data.booking_id === this.id)) {
                    console.log('🔄 Job status changed from server:', data.status);
                    this.status = data.status;
                    // Optional: Show toast
                    if (window.pushToast) window.pushToast('info', 'Trạng thái được cập nhật', data.status);
                }
            });
        },

        async handleServerCancellation(data) {
            await Swal.fire({
                title: 'Công việc đã bị hủy!',
                text: `Lý do: ${data.reason || 'Admin đã hủy'}`,
                icon: 'warning',
                allowOutsideClick: false,
                confirmButtonText: 'Về danh sách'
            });
            window.location.href = '/tech/jobs';
        },

        callSupport() {
            Swal.fire({
                title: 'Hỗ trợ kỹ thuật',
                text: 'Bạn cần hỗ trợ gì?',
                icon: 'question',
                showCancelButton: true,
                showDenyButton: true,
                confirmButtonText: 'Gọi tổng đài',
                denyButtonText: 'Hủy/Báo cáo sự cố',
                cancelButtonText: 'Đóng',
                confirmButtonColor: '#3085d6',
                denyButtonColor: '#d33',
                cancelButtonColor: '#aaa'
            }).then((result) => {
                if (result.isConfirmed) {
                    window.location.href = 'tel:19001234';
                } else if (result.isDenied) {
                    this.showCancelModal = true;
                }
            });
        },

        // SỬA LỖI: Link Google Maps chuẩn (giống Admin Dashboard)
        getMapLink() {
            if (this.lat && this.long && this.lat !== 0) {
                // Use standard Google Maps URL
                return `https://www.google.com/maps?q=${this.lat},${this.long}&z=17`;
            }
            // Use standard Google Maps URL for address
            return `https://www.google.com/maps?q=${encodeURIComponent(this.address)}`;
        },

        getStatusLabel(s) {
            const map = {
                'pending': 'Chờ xử lý',
                'assigned': 'Mới nhận',
                'accepted': 'Đã tiếp nhận',
                'moving': 'Đang đi',
                'working': 'Đang làm',
                'completed': 'Hoàn thành',
                'cancelled': 'Đã hủy',
                'arrived': 'Đã đến',
                'failed': 'Thất bại'
            };
            return map[s] || s;
        },

        getStatusOrder(s) {
            const map = {
                'pending': 0,
                'assigned': 1,
                'accepted': 1,
                'moving': 2,
                'arrived': 3,
                'working': 4,
                'completed': 5,
                'cancelled': 6
            };
            return map[s] || 0;
        },

        async updateStatus(newStatus, confirmMsg) {
            this.updateStatusAPI(newStatus, confirmMsg);
        },

        async updateStatusAPI(newStatus, confirmMsg) {
            const label = confirmMsg || `chuyển sang trạng thái: ${this.getStatusLabel(newStatus)}`;
            const confirmResult = await Swal.fire({
                title: 'Xác nhận?',
                text: `Bạn có chắc chắn muốn ${label}?`,
                icon: 'question',
                showCancelButton: true,
                confirmButtonText: 'Đồng ý',
                cancelButtonText: 'Hủy',
                buttonsStyling: false,
                customClass: {
                    confirmButton: 'btn btn-primary ml-2',
                    cancelButton: 'btn btn-ghost mr-2'
                }
            });

            if (!confirmResult.isConfirmed) return;

            this.loading = true;
            const fd = new FormData();
            fd.append('status', newStatus);
            try {
                const res = await fetch(`/api/tech/bookings/${this.id}/status`, { method: 'POST', body: fd });
                if (res.ok) {
                    this.status = newStatus;

                    // Show success message (No reload needed now!)
                    await Swal.fire({
                        title: 'Thành công!',
                        text: 'Đã cập nhật trạng thái',
                        icon: 'success',
                        timer: 1000,
                        showConfirmButton: false
                    });
                } else if (res.status === 409) {
                    // [FIX] Handle Conflict (Server status differs from Client)
                    const data = await res.json();
                    if (data.current_status) {
                        this.status = data.current_status;
                        await Swal.fire({
                            title: 'Cập nhật dữ liệu',
                            text: 'Trạng thái công việc đã được cập nhật từ máy chủ.',
                            icon: 'info',
                            timer: 2000,
                            showConfirmButton: false
                        });
                        return;
                    }
                    Swal.fire('Lỗi', data.error || 'Trạng thái không hợp lệ', 'error');
                } else {
                    Swal.fire('Lỗi', 'Lỗi cập nhật trạng thái', 'error');
                }
            } catch (e) {
                console.error(e);
                Swal.fire('Lỗi', 'Lỗi kết nối', 'error');
            }
            finally { this.loading = false; }
        },

        async checkIn() {
            const confirmResult = await Swal.fire({
                title: 'Xác nhận đã đến?',
                text: "Hệ thống sẽ ghi nhận vị trí GPS hiện tại của bạn.",
                icon: 'info',
                showCancelButton: true,
                confirmButtonText: 'Check-in ngay',
                cancelButtonText: 'Đóng',
                buttonsStyling: false,
                customClass: {
                    confirmButton: 'btn btn-primary ml-2',
                    cancelButton: 'btn btn-ghost mr-2'
                }
            });

            if (!confirmResult.isConfirmed) return;

            this.loading = true;

            // 1. Get GPS
            if (!navigator.geolocation) {
                Swal.fire('Lỗi', 'Trình duyệt không hỗ trợ GPS', 'error');
                this.loading = false;
                return;
            }

            navigator.geolocation.getCurrentPosition(
                async (position) => {
                    const lat = position.coords.latitude;
                    const long = position.coords.longitude;

                    const fd = new FormData();
                    fd.append('lat', lat);
                    fd.append('long', long);

                    try {
                        const res = await fetch(`/api/tech/bookings/${this.id}/checkin`, {
                            method: 'POST',
                            body: fd
                        });

                        const data = await res.json();

                        if (res.ok) {
                            this.status = 'arrived'; // Update local state immediately

                            Swal.fire({
                                title: 'Thành công!',
                                text: data.message || 'Check-in thành công!',
                                icon: 'success',
                                timer: 2000,
                                showConfirmButton: false
                            });
                        } else {
                            Swal.fire('Lỗi', data.error || 'Check-in thất bại', 'error');
                        }
                    } catch (e) {
                        console.error(e);
                        Swal.fire('Lỗi', 'Lỗi kết nối khi Check-in', 'error');
                    } finally {
                        this.loading = false;
                    }
                },
                (error) => {
                    Swal.fire('Lỗi GPS', 'Không lấy được vị trí: ' + error.message, 'error');
                    this.loading = false;
                },
                // Fixed: Removed extra '},' here
                { enableHighAccuracy: true, timeout: 10000 }
            );
        },



        async cancelJob() {
            if (!this.cancelReason) {
                Swal.fire('Thông báo', 'Vui lòng chọn lý do', 'warning');
                return;
            }

            if (this.cancelReason === 'customer_not_home') {
                if (!this.$refs.evidenceInput || !this.$refs.evidenceInput.files.length) {
                    Swal.fire('Thông báo', 'Vui lòng chụp ảnh bằng chứng', 'warning');
                    return;
                }
            }

            if (this.cancelReason === 'reschedule' && !this.newTime) {
                Swal.fire('Thông báo', 'Vui lòng chọn thời gian mới', 'warning');
                return;
            }

            this.loading = true;
            const fd = new FormData();
            fd.append('reason', this.cancelReason);
            fd.append('note', this.cancelNote);

            if (this.newTime) {
                fd.append('new_time', this.newTime);
            }

            if (this.$refs.evidenceInput && this.$refs.evidenceInput.files.length > 0) {
                fd.append('evidence', this.$refs.evidenceInput.files[0]);
            }

            // Get GPS for Verification
            if (navigator.geolocation) {
                try {
                    const pos = await new Promise((resolve, reject) => {
                        navigator.geolocation.getCurrentPosition(resolve, reject, { enableHighAccuracy: true, timeout: 5000 });
                    });
                    fd.append('lat', pos.coords.latitude);
                    fd.append('long', pos.coords.longitude);
                } catch (e) {
                    console.warn("Could not get GPS for cancellation");
                }
            }

            try {
                const res = await fetch(`/api/tech/bookings/${this.id}/cancel`, {
                    method: 'POST',
                    body: fd
                });

                if (res.ok) {
                    let msg = 'Đã hủy công việc';
                    if (this.cancelReason === 'reschedule') {
                        msg = 'Đã đổi lịch thành công';
                    }

                    await Swal.fire({
                        title: 'Thành công',
                        text: msg,
                        icon: 'success'
                    });

                    window.location.href = '/tech/jobs';
                } else {
                    const errorText = await res.text();
                    try {
                        const errJson = JSON.parse(errorText);
                        Swal.fire('Lỗi', errJson.error || errJson.message, 'error');
                    } catch (e) {
                        Swal.fire('Lỗi', errorText, 'error');
                    }
                }
            } catch (e) {
                console.error(e);
                Swal.fire('Lỗi', 'Lỗi kết nối mạng', 'error');
            } finally {
                this.loading = false;
                this.showCancelModal = false;
            }
        }
    };
};