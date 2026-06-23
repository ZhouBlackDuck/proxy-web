#!/bin/sh
set -e

echo "=== Proxy WebUI Starting ==="

# Initialize data directories
mkdir -p /data/webui
mkdir -p /data/mihomo/bin

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

# subconverter is already installed in the base image at /usr/bin/subconverter
# Create pref.ini from example if not present (subconverter requires it)
if [ ! -f /base/pref.ini ]; then
    cp /base/pref.example.ini /base/pref.ini
    # Fix template_path to point to base templates
    sed -i 's|^template_path=$|template_path=base|' /base/pref.ini
    echo "Created subconverter pref.ini"
fi

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
  "subconverter": {
    "apiAddr": "127.0.0.1:25500",
    "binaryPath": "/usr/bin/subconverter"
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
