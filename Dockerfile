# ========================================
# Stage 1: Build Frontend
# ========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# ========================================
# Stage 2: Build Backend
# ========================================
FROM golang:1.25-alpine AS backend-builder
ARG TARGETARCH
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.Version=$(date +%Y%m%d) -X main.Commit=docker" \
    -o /webui-server ./cmd/server

# ========================================
# Stage 3: Final Image (multi-arch: amd64 + arm64)
# ========================================
FROM alpine:3.20

ARG SUBSTORE_VERSION=2.24.22

LABEL maintainer="proxy-web"
LABEL description="Mihomo WebUI Control Panel"

# Install runtime dependencies
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache \
    ca-certificates \
    tzdata \
    iptables \
    nginx \
    nodejs \
    curl \
    bash

# mihomo: copy from official image (Docker auto-resolves amd64/arm64)
COPY --from=metacubex/mihomo:latest /mihomo /opt/mihomo/mihomo
COPY --from=metacubex/mihomo:latest /root/.config/mihomo /opt/mihomo/data
RUN chmod +x /opt/mihomo/mihomo

# Sub-Store: download from GitHub releases (JS bundle, platform-independent)
RUN mkdir -p /app/sub-store \
    && curl -fsSL "https://github.com/sub-store-org/Sub-Store/releases/download/${SUBSTORE_VERSION}/sub-store.bundle.js" \
       -o /app/sub-store/sub-store.bundle.js
ENV SUB_STORE_BACKEND_API_PORT=3001
ENV SUB_STORE_BACKEND_API_HOST=127.0.0.1
ENV SUB_STORE_DATA_DIR=/data/sub-store

# Copy backend binary
COPY --from=backend-builder /webui-server /app/webui-server

# Copy frontend build to Nginx
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html

# Copy Nginx config
COPY docker/nginx.conf /etc/nginx/http.d/default.conf

# Copy entrypoint (sed strips any CRLF from Windows checkouts)
COPY docker/entrypoint.sh /entrypoint.sh
RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

# Create data directories
RUN mkdir -p /data/webui/profiles /data/mihomo/bin /data/sub-store

# Data volume
VOLUME ["/data"]

EXPOSE 80 7890

ENTRYPOINT ["/entrypoint.sh"]
