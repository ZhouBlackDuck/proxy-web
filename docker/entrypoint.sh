#!/bin/sh
set -e

echo "=== Proxy WebUI Starting ==="

# Initialize data directories
mkdir -p /data/webui/profiles
mkdir -p /data/mihomo/bin
mkdir -p /data/sub-store

# Copy mihomo binary and GeoIP data to data volume if not present
if [ ! -f /data/mihomo/bin/mihomo ]; then
    cp /opt/mihomo/mihomo /data/mihomo/bin/mihomo
    chmod +x /data/mihomo/bin/mihomo
    echo "Copied mihomo binary to /data/mihomo/bin/"
fi
for f in geoip.metadb geosite.dat geoip.dat; do
    if [ ! -f "/data/mihomo/$f" ] && [ -f "/opt/mihomo/data/$f" ]; then
        cp "/opt/mihomo/data/$f" "/data/mihomo/$f"
        echo "Copied $f to /data/mihomo/"
    fi
done

# Generate initial settings.json if not exists
if [ ! -f /data/webui/settings.json ]; then
    cat > /data/webui/settings.json << 'EOF'
{
  "theme": "dark",
  "language": "zh",
  "mihomo": {
    "apiAddr": "127.0.0.1:9090",
    "secret": "",
    "binaryPath": "/data/mihomo/bin/mihomo",
    "configPath": "/data/mihomo/config.yaml"
  },
  "substore": {
    "apiAddr": "127.0.0.1:3001",
    "dataDir": "/data/sub-store"
  }
}
EOF
    echo "Created initial settings.json"
fi

# Generate minimal mihomo config if not exists
if [ ! -f /data/mihomo/config.yaml ]; then
    cat > /data/mihomo/config.yaml << 'EOF'
mixed-port: 7890
allow-lan: false
mode: rule
log-level: info
external-controller: 127.0.0.1:9090
EOF
    echo "Created initial mihomo config.yaml"
fi

# Generate initial profiles.json if not exists
if [ ! -f /data/webui/profiles.json ]; then
    cat > /data/webui/profiles.json << 'EOF'
{
  "activeProfileId": "",
  "profiles": []
}
EOF
    echo "Created initial profiles.json"
fi

# Substitute ENV variables in nginx config
export NGINX_CLIENT_MAX_BODY_SIZE=${NGINX_CLIENT_MAX_BODY_SIZE:-10m}
envsubst '${NGINX_CLIENT_MAX_BODY_SIZE}' < /etc/nginx/http.d/default.conf > /tmp/nginx.conf
mv /tmp/nginx.conf /etc/nginx/http.d/default.conf

# Start Nginx in background
echo "Starting Nginx..."
nginx

# Start WebUI backend (becomes PID 1 responsibility)
echo "Starting WebUI backend..."
exec /app/webui-server
