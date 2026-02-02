#!/bin/bash
mkdir -p certs

# Kiểm tra mkcert có tồn tại không
if ! command -v mkcert &> /dev/null; then
    echo "mkcert chưa được cài đặt. Vui lòng cài đặt mkcert trước."
    exit 1
fi

echo "Generating certificates..."
# Tạo chứng chỉ cho localhost và IP LAN
# Lưu ý: Bạn cần chạy 'mkcert -install' một lần trên máy để cài CA vào store của máy.
mkcert -key-file certs/key.pem -cert-file certs/cert.pem 192.168.1.12 localhost 127.0.0.1 ::1

echo "✅ Certificates generated in certs/ directory."
echo "⚠️  QUAN TRỌNG: Để trình duyệt tin tưởng, bạn hãy copy file chạy 'mkcert -install' trên máy tính."
echo "⚠️  Nếu test trên điện thoại, hãy tìm file 'rootCA.pem' (chạy 'mkcert -CAROOT' để tìm vị trí) và cài đặt nó vào điện thoại."
