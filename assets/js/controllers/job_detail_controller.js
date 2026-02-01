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

        // SỬA LỖI: Link Google Maps chuẩn (giống Admin Dashboard)
        getMapLink() {
            if (this.lat && this.long && this.lat !== 0) {
                // Corrected template literal syntax for lat/long
                return `http://googleusercontent.com/maps.google.com/maps?q=${this.lat},${this.long}&z=17`;
            }
            // Corrected template literal syntax for address
            return `http://googleusercontent.com/maps.google.com/maps?q=${encodeURIComponent(this.address)}`;
        },

        getStatusLabel(s) {
            const map = {
                'pending': 'Chờ xử lý',
                'assigned': 'Mới nhận',
                'moving': 'Đang đi',
                'working': 'Đang làm',
                'completed': 'Hoàn thành',
                'cancelled': 'Đã hủy',
                'arrived': 'Đã đến',
                'failed': 'Thất bại'
            };
            return map[s] || s;
        },

        async updateStatus(newStatus) {
            this.updateStatusAPI(newStatus);
        },

        async updateStatusAPI(newStatus) {
            const confirmResult = await Swal.fire({
                title: 'Xác nhận chuyển trạng thái?',
                text: `Bạn có chắc chắn muốn chuyển sang trạng thái: ${this.getStatusLabel(newStatus)}?`,
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
                    window.scrollTo({ top: 0, behavior: 'smooth' });
                    Swal.fire({
                        title: 'Thành công!',
                        text: 'Đã cập nhật trạng thái',
                        icon: 'success',
                        timer: 1500,
                        showConfirmButton: false
                    });
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
                            window.scrollTo({ top: 0, behavior: 'smooth' });
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

        async reportIssue(type) {
            if (type === 'customer_not_home') {
                const result = await Swal.fire({
                    title: 'Xác nhận khách vắng nhà?',
                    text: "Hệ thống sẽ ghi nhận và thông báo cho Admin.",
                    icon: 'warning',
                    showCancelButton: true,
                    confirmButtonText: 'Xác nhận',
                    cancelButtonText: 'Quay lại',
                    buttonsStyling: false,
                    customClass: {
                        confirmButton: 'btn btn-error ml-2',
                        cancelButton: 'btn btn-ghost mr-2'
                    }
                });

                if (!result.isConfirmed) return;

                this.loading = true;
                const fd = new FormData();
                fd.append('status', 'failed');
                fd.append('reason', 'customer_not_home');

                try {
                    const res = await fetch(`/api/tech/bookings/${this.id}/status`, { method: 'POST', body: fd });
                    if (res.ok) {
                        this.status = 'failed'; // Update local status
                        Swal.fire('Đã báo cáo', 'Đã ghi nhận khách vắng nhà', 'success');
                        window.location.reload();
                    } else {
                        const txt = await res.text();
                        Swal.fire('Lỗi', txt, 'error');
                    }
                } catch (e) {
                    Swal.fire('Lỗi', 'Lỗi kết nối', 'error');
                } finally {
                    this.loading = false;
                }
            }
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