# ========================================
# Stage 1: Build Frontend
# ========================================
FROM node:22-alpine AS frontend-builder
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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /reset-password ./cmd/reset-password

# ========================================
# Stage 3: Final Image — based on subconverter (Alpine 3.16, has all needed libs)
# ========================================
FROM tindy2013/subconverter:latest

LABEL maintainer="proxy-web"
LABEL description="Mihomo WebUI Control Panel"

# Install additional runtime dependencies
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache \
    tzdata \
    iptables \
    nginx \
    curl \
    bash \
    gettext

# mihomo: copy from official image (Docker auto-resolves amd64/arm64)
COPY --from=metacubex/mihomo:latest /mihomo /opt/mihomo/mihomo
COPY --from=metacubex/mihomo:latest /root/.config/mihomo /opt/mihomo/data
RUN chmod +x /opt/mihomo/mihomo

ENV NGINX_CLIENT_MAX_BODY_SIZE=10m

# Copy backend binary
COPY --from=backend-builder /webui-server /app/webui-server
COPY --from=backend-builder /reset-password /usr/local/bin/reset-password

# Copy frontend build to Nginx
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html

# Copy Nginx config
COPY docker/nginx.conf /etc/nginx/http.d/default.conf

# Copy entrypoint (sed strips any CRLF from Windows checkouts)
COPY docker/entrypoint.sh /entrypoint.sh
RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

# Create data directories
RUN mkdir -p /data/webui/profiles /data/mihomo/bin

# Data volume
VOLUME ["/data"]

EXPOSE 80 7890

ENTRYPOINT ["/entrypoint.sh"]
