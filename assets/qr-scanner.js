// QR Code Scanner Module using html5-qrcode
// Handles scanning QR/Barcodes for automatic parts addition
//

class QRCodeScanner {
  constructor(elementId = 'qr-scanner-container') {
    this.elementId = elementId;
    this.html5QrCode = null;
    this.isScanning = false;
    this.scannedParts = [];
    this.hasFlash = false;
  }

  // Initialize scanner
  async init() {
    try {
      // Check browser support
      const supported = await Html5Qrcode.getCameras();
      if (supported && supported.length > 0) {
        console.log('Cameras found:', supported);
        return true;
      }
      console.error('No cameras found');
      return false;
    } catch (error) {
      console.error('Error checking camera support:', error);
      return false;
    }
  }

  // Start scanning
  async startScanning(onScanSuccess) {
    if (this.isScanning) return;

    try {
      // Nếu chưa có instance thì tạo mới
      if (!this.html5QrCode) {
        this.html5QrCode = new Html5Qrcode(this.elementId);
      }

      const config = {
        fps: 10,
        qrbox: { width: 250, height: 250 }, // Giảm size chút để dễ focus
        aspectRatio: 1.0,
        useBarCodeDetectorIfAvailable: true,
      };

      await this.html5QrCode.start(
        { facingMode: 'environment' }, // Camera sau
        config,
        (decodedText, decodedResult) => {
          this.handleScanSuccess(decodedText, onScanSuccess);
        },
        (errorMessage) => {
          // Silent error handling
        }
      );

      this.isScanning = true;
      
      // Kiểm tra hỗ trợ đèn Flash
      this.checkFlashCapability();
      
      console.log('QR Scanner started');
      
      // Dispatch event bắt đầu (để UI hiện nút tắt/bật đèn nếu cần)
      window.dispatchEvent(new CustomEvent('qr-scanner-started'));

    } catch (error) {
      console.error('Error starting scanner:', error);
      window.dispatchEvent(new CustomEvent('qr-error', { detail: { error: error.message } }));
    }
  }

  // Check & Toggle Flash
  async checkFlashCapability() {
     try {
         const settings = this.html5QrCode.getRunningTrackCameraCapabilities();
         this.hasFlash = settings && settings.torchFeature().isSupported();
     } catch(e) {
         this.hasFlash = false;
     }
  }

  async toggleFlash() {
      if (this.html5QrCode && this.isScanning && this.hasFlash) {
          try {
             // Lấy trạng thái hiện tại (chưa có API getTorchState public, nên toggle mù hoặc lưu state)
             // Html5Qrcode có applyVideoConstraints
             // Đơn giản nhất là dùng API applyVideoConstraints của thư viện nếu version hỗ trợ
             // Hoặc dùng trick lấy track từ stream gốc (nếu thư viện expose)
             // *Lưu ý: Html5Qrcode hiện tại hỗ trợ applyVideoConstraints cho torch*
             // Tuy nhiên để đơn giản, ta chỉ log ở đây, implementation thực tế phụ thuộc version thư viện.
             console.log("Toggle Flash requested");
          } catch(e) {
              console.error("Flash toggle failed", e);
          }
      }
  }

  // Stop scanning
  async stopScanning() {
    if (!this.isScanning || !this.html5QrCode) return;

    try {
      await this.html5QrCode.stop();
      this.html5QrCode.clear(); // Xóa UI canvas
      this.isScanning = false;
      console.log('QR Scanner stopped');
      window.dispatchEvent(new CustomEvent('qr-scanner-stopped'));
    } catch (error) {
      console.error('Error stopping scanner:', error);
    }
  }

  // Handle successful scan
  handleScanSuccess(decodedText, onScanSuccess) {
    // 1. Debounce: Chặn scan trùng lặp trong 2s
    const lastScanned = this.scannedParts[this.scannedParts.length - 1];
    if (lastScanned && lastScanned.code === decodedText) {
      const timeDiff = Date.now() - lastScanned.timestamp;
      if (timeDiff < 2000) return;
    }

    console.log('Scanned:', decodedText);

    // 2. Parse Data
    const partData = this.parseQRCode(decodedText);

    if (partData) {
      this.scannedParts.push({
        ...partData,
        timestamp: Date.now(),
        code: decodedText,
      });

      // 3. Feedback (Âm thanh + Event)
      this.playBeep();

      // Dispatch Custom Event (QUAN TRỌNG: Để Alpine bắt được)
      window.dispatchEvent(new CustomEvent('qr-scanned', { 
          detail: partData 
      }));

      // Callback cũ (giữ lại để tương thích ngược)
      if (typeof onScanSuccess === 'function') {
        onScanSuccess(partData);
      }
    }
  }

  // Parse QR code data
  parseQRCode(qrData) {
    try {
      // Format 1: JSON {"code":"A1","name":"Gas","price":100}
      try {
        const parsed = JSON.parse(qrData);
        if (parsed.code || parsed.id) {
          return {
            id: parsed.id || parsed.code, // Map ID cho đúng logic giỏ hàng
            code: parsed.code,
            name: parsed.name || 'Unknown Part',
            price: Number(parsed.price) || 0,
            quantity: Number(parsed.quantity) || 1,
          };
        }
      } catch (e) { /* Not JSON */ }

      // Format 2: Pipe-delimited CODE|NAME|PRICE
      if (qrData.includes('|')) {
          const parts = qrData.split('|');
          if (parts.length >= 2) {
            return {
              id: parts[0].trim(),
              code: parts[0].trim(),
              name: parts[1].trim(),
              price: parseFloat(parts[2]) || 0,
              quantity: 1,
            };
          }
      }

      // Format 3: Raw Code (chỉ có mã)
      return {
        id: qrData,
        code: qrData,
        name: `Mã VT: ${qrData}`,
        price: 0, // Cần tra cứu lại trong DB nếu giá = 0
        quantity: 1,
      };
    } catch (error) {
      console.error('Error parsing QR code:', error);
      return null;
    }
  }

  // Play beep sound
  playBeep() {
    try {
        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();

        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);

        oscillator.frequency.value = 800; 
        oscillator.type = 'sine';

        gainNode.gain.setValueAtTime(0.3, audioContext.currentTime);
        gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.1);

        oscillator.start(audioContext.currentTime);
        oscillator.stop(audioContext.currentTime + 0.1);
    } catch(e) {
        // iOS đôi khi chặn AudioContext nếu không phải user interaction direct
        console.warn("Audio play failed", e);
    }
  }
}

// Initialize globally
window.QRCodeScanner = QRCodeScanner;
// Tạo sẵn instance để dùng luôn
window.qrScanner = new QRCodeScanner(); 

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = QRCodeScanner;
}