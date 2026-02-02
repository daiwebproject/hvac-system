#!/bin/bash

# Đường dẫn đích trong cấu hình Nginx của bạn
DEST_CERT="/etc/nginx/ssl/192.168.1.12.pem"
DEST_KEY="/etc/nginx/ssl/192.168.1.12-key.pem"

# Đường dẫn file mới tạo
SRC_CERT="certs/cert.pem"
SRC_KEY="certs/key.pem"

echo "🔒 Đang cập nhật chứng chỉ SSL cho Nginx..."

# 1. Backup file cũ
if [ -f "$DEST_CERT" ]; then
    echo "📦 Backup chứng chỉ cũ..."
    sudo cp "$DEST_CERT" "${DEST_CERT}.old"
    sudo cp "$DEST_KEY" "${DEST_KEY}.old"
fi

# 2. Copy file mới vào (cần quyền sudo)
echo "📝 Ghi đè chứng chỉ mới (yêu cầu mật khẩu sudo)..."
sudo cp "$SRC_CERT" "$DEST_CERT"
sudo cp "$SRC_KEY" "$DEST_KEY"

# 3. Restart Nginx
echo "🔄 Khởi động lại Nginx..."
sudo systemctl restart nginx

echo "✅ Hoàn tất! Nginx đang chạy với chứng chỉ mới."
echo "⚠️  Đừng quên cài file rootCA.pem vào điện thoại nếu chưa cài."
