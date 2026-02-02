#!/bin/sh

# Usage: ./create_customer.sh [customer_name] [port]
NAME=$1
PORT=$2

if [ -z "$NAME" ] || [ -z "$PORT" ]; then
  echo "Usage: ./create_customer.sh [customer_name] [port]"
  echo "Example: ./create_customer.sh hungcuong 8091"
  exit 1
fi

DATA_DIR="/home/customers/$NAME/pb_data"

echo "Creating data directory at $DATA_DIR..."
mkdir -p "$DATA_DIR"

# Ensure permissions (optional, Docker runs as root by default inside, host dir owned by user is usually fine)
# chmod 777 "$DATA_DIR"

echo "Starting Container: hvac-$NAME on Port: $PORT..."

# Run Docker Container
docker run -d \
  --name "hvac-$NAME" \
  --restart unless-stopped \
  -p "$PORT:8090" \
  -v "$DATA_DIR:/pb_data" \
  hvac-app

# Wait for startup
echo "Waiting for service to initialize..."
sleep 5

# Create Default Admin User
ADMIN_EMAIL="admin@$NAME.com"
ADMIN_PASS="1234567890"

echo "Creating Default Admin Account..."
echo "  Email: $ADMIN_EMAIL"
echo "  Pass:  $ADMIN_PASS"

docker exec "hvac-$NAME" /app/hvac-app admin create "$ADMIN_EMAIL" "$ADMIN_PASS"

echo "---------------------------------------------------"
echo "✅ DEPLOYMENT COMPLETE!"
echo "---------------------------------------------------"
echo "URL: http://<vps-ip>:$PORT"
echo "Admin Login: $ADMIN_EMAIL"
echo "Password:    $ADMIN_PASS"
echo "---------------------------------------------------"
echo "NEXT STEPS:"
echo "1. Add to Caddyfile: $NAME.com { reverse_proxy localhost:$PORT }"
echo "2. Reload Caddy: caddy reload"
