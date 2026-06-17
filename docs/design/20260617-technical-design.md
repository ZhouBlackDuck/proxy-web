# Mihomo WebUI - 技术设计文档

> 版本: v1
> 日期: 2026-06-17
> 状态: 设计评审中

---

## 一、总体架构

```
┌──────────────────────────────────────────────────────────────┐
│                      Docker Container                         │
│                                                               │
│   ┌─────────────┐    :80                                      │
│   │   Nginx     │─────────┐                                   │
│   │  Vue SPA    │         │                                   │
│   └─────────────┘         │                                   │
│                           │                                   │
│   ┌───────────────────────▼──────────────────────────────┐   │
│   │              WebUI Backend (Go) :3000                 │   │
│   │              PID 1 — 进程管理者                        │   │
│   │                                                       │   │
│   │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ │   │
│   │  │ HTTP API │ │ WS Proxy │ │ Config   │ │Process  │ │   │
│   │  │ Router   │ │ Relay    │ │ Merger   │ │Manager  │ │   │
│   │  └──────────┘ └────┬─────┘ └──────────┘ └────┬────┘ │   │
│   └─────────────────────│─────────────────────────│──────┘   │
│                         │                         │           │
│              ┌──────────▼────┐    ┌───────────────▼────────┐ │
│              │  Sub-Store    │    │  mihomo (binary)       │ │
│              │  (Node.js)    │    │  API :9090             │ │
│              │  :3001        │    │  Mixed :7890           │ │
│              └───────────────┘    └────────────────────────┘ │
│                                                               │
│   /data volume                                                │
│   ├── webui/                                                  │
│   │   ├── settings.json                                       │
│   │   ├── profiles.json                                       │
│   │   └── profiles/{id}/                                      │
│   │       ├── meta.json                                       │
│   │       ├── rules.yaml                                      │
│   │       └── override.yaml                                   │
│   ├── sub-store/                                              │
│   └── mihomo/                                                 │
│       ├── config.yaml                                         │
│       ├── bin/mihomo                                          │
│       └── Country.mmdb                                        │
└──────────────────────────────────────────────────────────────┘
```

---

## 二、Go 后端项目结构

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # 入口，信号处理，进程编排
├── internal/
│   ├── config/
│   │   ├── paths.go                # 数据目录路径管理
│   │   ├── settings.go             # 全局设置读写 (settings.json)
│   │   └── defaults.go             # 默认配置常量
│   ├── model/
│   │   ├── profile.go              # Profile 数据模型
│   │   ├── subscription.go         # 订阅关联模型
│   │   └── export.go               # 导入导出模型
│   ├── store/
│   │   ├── file.go                 # JSON/YAML 文件读写工具
│   │   ├── profiles.go             # Profile 注册表 CRUD
│   │   └── settings.go             # Settings 持久化
│   ├── process/
│   │   ├── manager.go              # 进程管理器（启动/停止/监控/重启）
│   │   ├── mihomo.go               # mihomo 进程生命周期
│   │   └── substore.go             # Sub-Store 进程生命周期
│   ├── kernel/
│   │   ├── client.go               # mihomo REST API 客户端
│   │   ├── websocket.go            # mihomo WebSocket 客户端
│   │   └── types.go                # API 响应类型定义
│   ├── substore/
│   │   ├── client.go               # Sub-Store API 客户端
│   │   └── types.go                # Sub-Store 响应类型
│   ├── enhance/
│   │   ├── merge.go                # YAML 浅合并
│   │   ├── rules.go                # 全局规则 prepend/append
│   │   ├── pipeline.go             # 配置合并管道编排
│   │   └── validate.go             # 配置校验
│   ├── api/
│   │   ├── router.go               # 路由注册
│   │   ├── middleware/
│   │   │   ├── auth.go             # JWT 鉴权中间件
│   │   │   └── cors.go             # CORS
│   │   ├── handler/
│   │   │   ├── auth.go             # POST /api/auth/login
│   │   │   ├── profile.go          # /api/profiles CRUD
│   │   │   ├── subscription.go     # /api/subscriptions (代理 Sub-Store)
│   │   │   ├── kernel.go           # /api/kernel/* (透传 mihomo)
│   │   │   ├── settings.go         # /api/settings
│   │   │   ├── geo.go              # /api/geo
│   │   │   ├── export.go           # /api/profiles/:id/export, /import
│   │   │   └── health.go           # GET /api/health
│   │   └── ws/
│   │       ├── traffic.go          # WS /api/ws/traffic
│   │       ├── connections.go      # WS /api/ws/connections
│   │       ├── logs.go             # WS /api/ws/logs
│   │       └── memory.go           # WS /api/ws/memory
│   └── export/
│       ├── export.go               # Profile 打包导出
│       └── import.go               # Profile 导入恢复
├── go.mod
├── go.sum
├── Dockerfile
└── Makefile
```

---

## 三、进程管理设计

### 3.1 启动顺序

```
main.go (PID 1)
    │
    ├── 1. 初始化数据目录 (/data/webui, /data/mihomo, /data/sub-store)
    ├── 2. 读取 settings.json
    ├── 3. 启动 Sub-Store 子进程 (node)
    │       └── 等待 :3001 健康检查通过
    ├── 4. 启动 mihomo 子进程
    │       └── 等待 :9090 健康检查通过
    ├── 5. 启动 HTTP API Server (:3000)
    └── 6. 等待信号 (SIGTERM/SIGINT)
            ├── 停止 HTTP Server
            ├── 停止 mihomo (SIGTERM)
            ├── 停止 Sub-Store (SIGTERM)
            └── 退出
```

### 3.2 进程管理器接口

```go
// internal/process/manager.go

type ProcessManager struct {
    mihomo   *MihomoProcess
    substore *SubStoreProcess
    mu       sync.RWMutex
}

type ProcessStatus struct {
    Name    string `json:"name"`
    Running bool   `json:"running"`
    PID     int    `json:"pid"`
    Uptime  int64  `json:"uptime"`  // seconds
    CPU     float64 `json:"cpu"`
    Memory  uint64  `json:"memory"` // bytes
}

func (pm *ProcessManager) StartMihomo(configPath string) error
func (pm *ProcessManager) StopMihomo() error
func (pm *ProcessManager) RestartMihomo() error
func (pm *ProcessManager) StartSubStore() error
func (pm *ProcessManager) StopSubStore() error
func (pm *ProcessManager) Status() []ProcessStatus
func (pm *ProcessManager) MihomoAlive() bool
func (pm *ProcessManager) SubStoreAlive() bool
```

### 3.3 进程健康检查

- mihomo: 轮询 `GET http://127.0.0.1:9090/` 直到返回 200
- Sub-Store: 轮询 `GET http://127.0.0.1:3001/api/subs` 直到返回 200
- 超时: 30 秒，超时则记录错误日志并标记为启动失败
- 运行时: 每 10 秒心跳检测，进程异常退出时自动重启（最多 3 次，指数退避）

---

## 四、API 接口设计

### 4.1 鉴权

```
POST /api/auth/login
  Body: { "password": "xxx" }
  Response: { "token": "jwt-token", "expiresIn": 86400 }

所有其他 /api/* 路由需要 Header: Authorization: Bearer <token>
```

首次启动时若 settings.json 中无密码哈希，则：
- `POST /api/auth/setup` 设置初始密码
- 后续登录走 `/api/auth/login`

### 4.2 完整 API 列表

#### 认证与设置
```
POST   /api/auth/setup           # 首次设置密码
POST   /api/auth/login           # 登录
GET    /api/auth/check           # 验证 token 有效性
PUT    /api/auth/password        # 修改密码

GET    /api/settings             # 获取全局设置
PUT    /api/settings             # 更新全局设置
```

#### Profile 管理
```
GET    /api/profiles             # 列出所有 Profile
POST   /api/profiles             # 创建 Profile
GET    /api/profiles/:id         # 获取 Profile 详情
PUT    /api/profiles/:id         # 更新 Profile
DELETE /api/profiles/:id         # 删除 Profile
POST   /api/profiles/:id/activate  # 激活（切换到此 Profile）
GET    /api/profiles/:id/preview   # 预览合并后的最终配置
POST   /api/profiles/:id/export    # 导出 Profile 包
POST   /api/profiles/import        # 导入 Profile 包
```

#### 全局规则与覆盖（属于某个 Profile）
```
GET    /api/profiles/:id/rules       # 获取全局规则 (yaml)
PUT    /api/profiles/:id/rules       # 更新全局规则
GET    /api/profiles/:id/override    # 获取全局覆盖 (yaml)
PUT    /api/profiles/:id/override    # 更新全局覆盖
```

#### 订阅管理（代理 Sub-Store API）
```
GET    /api/subscriptions              # 列出订阅
POST   /api/subscriptions              # 创建订阅
GET    /api/subscriptions/:name        # 获取订阅详情
PUT    /api/subscriptions/:name        # 更新订阅
DELETE /api/subscriptions/:name        # 删除订阅
POST   /api/subscriptions/:name/sync   # 触发同步
GET    /api/subscriptions/:name/download  # 下载转换后的 mihomo yaml
GET    /api/subscriptions/:name/flow      # 流量信息
```

#### 内核代理（透传 mihomo API）
```
GET    /api/kernel/version
GET    /api/kernel/configs
PATCH  /api/kernel/configs
PUT    /api/kernel/configs             # 全量重载
POST   /api/kernel/configs/geo         # 更新 GeoIP/GeoSite
GET    /api/kernel/proxies
GET    /api/kernel/proxies/:name
PUT    /api/kernel/proxies/:name
GET    /api/kernel/proxies/:name/delay
DELETE /api/kernel/proxies/:name
GET    /api/kernel/group
GET    /api/kernel/group/:name
GET    /api/kernel/group/:name/delay
GET    /api/kernel/rules
PATCH  /api/kernel/rules/disable
GET    /api/kernel/connections
DELETE /api/kernel/connections
DELETE /api/kernel/connections/:id
POST   /api/kernel/restart
```

#### GeoIP 管理
```
GET    /api/geo/status             # GeoIP/GeoSite 版本和更新时间
POST   /api/geo/update             # 触发更新
```

#### WebSocket 代理
```
WS     /api/ws/traffic             # 转发 mihomo /traffic
WS     /api/ws/connections         # 转发 mihomo /connections
WS     /api/ws/logs                # 转发 mihomo /logs（支持 ?level= 参数）
WS     /api/ws/memory              # 转发 mihomo /memory
```

#### 系统
```
GET    /api/health                 # WebUI 后端健康检查
GET    /api/status                 # 所有进程状态
POST   /api/process/mihomo/start
POST   /api/process/mihomo/stop
POST   /api/process/mihomo/restart
```

---

## 五、配置合并管道

参考 Clash Verge Rev 的 enhance 管道，简化为：

```
输入:
  ├── sub_config:   Sub-Store 输出的 clashmeta yaml (Mapping)
  ├── override:     用户全局覆盖 yaml (Mapping)
  └── global_rules: 用户全局规则列表 ([]string)

管道:
  1. parse     → 解析 sub_config yaml 为 serde_yaml Mapping
  2. merge     → 浅合并 override 到 sub_config（override 优先）
  3. rules     → prepend global_rules 到 rules 段
  4. ensure    → 确保必要字段存在（port/mixed-port/external-controller 等）
  5. validate  → YAML 语法校验 + 关键字段校验
  6. serialize → 输出最终 yaml 文本

输出:
  ├── final_config.yaml  → 写入文件
  └── PUT /configs       → 推送给 mihomo
```

Go 实现使用 `gopkg.in/yaml.v3` 的 `yaml.Node` 做结构化操作：

```go
// internal/enhance/pipeline.go

type Pipeline struct {
    store store.Store
    kernel kernel.Client
}

func (p *Pipeline) Build(profileID string) ([]byte, error) {
    // 1. 从 Sub-Store 获取订阅配置
    subYaml, err := p.fetchSubscription(profileID)

    // 2. 读取全局覆盖
    override, err := p.store.ReadOverride(profileID)

    // 3. 读取全局规则
    rules, err := p.store.ReadRules(profileID)

    // 4. 合并
    config, err := parseYAML(subYaml)
    config = mergeOverride(config, override)
    config = prependRules(config, rules)
    config = ensureDefaults(config)

    // 5. 校验
    if err := validate(config); err != nil {
        return nil, err
    }

    // 6. 序列化
    return serializeYAML(config)
}

func (p *Pipeline) Apply(profileID string) error {
    yaml, err := p.Build(profileID)
    if err != nil { return err }

    // 写入文件
    p.store.WriteMihomoConfig(yaml)

    // 推送给内核
    return p.kernel.PutConfig(yaml)
}
```

---

## 六、数据模型

### 6.1 settings.json

```json
{
  "passwordHash": "$2a$10$xxx",
  "theme": "dark",
  "language": "zh",
  "mihomo": {
    "apiAddr": "127.0.0.1:9090",
    "secret": "auto-generated-uuid",
    "binaryPath": "/data/mihomo/bin/mihomo",
    "configPath": "/data/mihomo/config.yaml"
  },
  "substore": {
    "apiAddr": "127.0.0.1:3001",
    "dataDir": "/data/sub-store"
  }
}
```

### 6.2 profiles.json

```json
{
  "activeProfileId": "p-001",
  "profiles": [
    {
      "id": "p-001",
      "name": "日常",
      "description": "日常使用",
      "subscriptionName": "my-airport",
      "createdAt": "2026-06-17T10:00:00Z",
      "updatedAt": "2026-06-17T10:00:00Z",
      "exportSettings": {
        "includeSubscriptions": false
      }
    }
  ]
}
```

### 6.3 profiles/{id}/meta.json

```json
{
  "id": "p-001",
  "name": "日常",
  "description": "日常使用",
  "subscriptionName": "my-airport",
  "createdAt": "2026-06-17T10:00:00Z",
  "updatedAt": "2026-06-17T10:00:00Z",
  "exportSettings": {
    "includeSubscriptions": false
  }
}
```

### 6.4 profiles/{id}/rules.yaml

```yaml
# 全局规则（prepend 到最终配置的 rules 段头部）
- DOMAIN-SUFFIX,ads.example.com,REJECT
- GEOIP,PRIVATE,DIRECT
- MATCH,Proxy
```

### 6.5 profiles/{id}/override.yaml

```yaml
# 全局覆盖（浅合并到订阅配置，此处值优先）
mixed-port: 7890
allow-lan: true
log-level: info
ipv6: false
```

---

## 七、前端架构

### 7.1 技术栈

```
Vue 3 + TypeScript + Vite
Naive UI（组件库）
Vue Router（路由）
Pinia（状态管理）
Vue I18n（国际化）
Monaco Editor（YAML 编辑器）
@vueuse/core（组合式工具）
```

### 7.2 目录结构

```
frontend/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── router/
│   │   └── index.ts               # 路由定义 + 鉴权守卫
│   ├── stores/
│   │   ├── auth.ts                # 登录状态
│   │   ├── kernel.ts              # 内核状态（模式、版本、运行状态）
│   │   ├── profile.ts             # 当前 Profile
│   │   ├── traffic.ts             # 流量数据（WS）
│   │   └── settings.ts            # 全局设置
│   ├── composables/
│   │   ├── useWebSocket.ts        # WS 连接管理
│   │   ├── useKernelApi.ts        # 内核 API 封装
│   │   └── useSubStoreApi.ts      # Sub-Store API 封装
│   ├── api/
│   │   ├── client.ts              # Axios 实例 + 拦截器
│   │   ├── auth.ts
│   │   ├── profiles.ts
│   │   ├── subscriptions.ts
│   │   ├── kernel.ts
│   │   ├── settings.ts
│   │   └── geo.ts
│   ├── views/
│   │   ├── LoginView.vue          # 登录/首次设置密码
│   │   ├── DashboardView.vue      # 首页仪表盘
│   │   ├── SubscriptionsView.vue  # 订阅管理
│   │   ├── ProfilesView.vue       # Profile 管理
│   │   ├── RulesView.vue          # 规则查看
│   │   ├── ProxiesView.vue        # 节点管理
│   │   ├── ConnectionsView.vue    # 连接监控
│   │   ├── LogsView.vue           # 内核日志
│   │   └── SettingsView.vue       # 系统设置
│   ├── components/
│   │   ├── layout/
│   │   │   ├── AppLayout.vue      # 整体布局（侧边栏+内容区）
│   │   │   ├── Sidebar.vue        # 导航侧边栏
│   │   │   └── Header.vue         # 顶部栏
│   │   ├── dashboard/
│   │   │   ├── TrafficCard.vue        # 流量统计卡片
│   │   │   ├── KernelStatusCard.vue   # 内核状态卡片
│   │   │   ├── ModeSwitchCard.vue     # 模式切换卡片
│   │   │   ├── NetworkSummaryCard.vue # 网络设置摘要
│   │   │   ├── CurrentNodeCard.vue    # 当前节点卡片
│   │   │   ├── ConnectivityCard.vue   # 网站连通性测试
│   │   │   └── QuickActionsCard.vue   # 快速操作
│   │   ├── subscriptions/
│   │   │   ├── SubList.vue
│   │   │   ├── SubForm.vue
│   │   │   └── SubDetail.vue
│   │   ├── profiles/
│   │   │   ├── ProfileList.vue
│   │   │   ├── ProfileForm.vue
│   │   │   ├── RulesEditor.vue        # 全局规则可视化编辑
│   │   │   ├── OverrideEditor.vue     # YAML 覆盖编辑器（Monaco）
│   │   │   ├── ConfigPreview.vue      # 合并配置预览
│   │   │   └── ExportDialog.vue       # 导出对话框
│   │   ├── proxies/
│   │   │   ├── ProxyGroupList.vue
│   │   │   ├── ProxyNodeList.vue
│   │   │   ├── ProxyNodeCard.vue
│   │   │   └── DelayBar.vue           # 延迟可视化条
│   │   ├── connections/
│   │   │   ├── ConnectionTable.vue
│   │   │   └── ConnectionDetail.vue
│   │   ├── rules/
│   │   │   ├── RuleList.vue
│   │   │   └── RuleItem.vue
│   │   ├── logs/
│   │   │   └── LogViewer.vue
│   │   └── common/
│   │       ├── TrafficGraph.vue       # 流量折线图
│   │       └── StatusBadge.vue
│   ├── locales/
│   │   ├── zh.ts
│   │   └── en.ts
│   └── styles/
│       └── global.css
└── public/
    └── favicon.ico
```

### 7.3 页面与组件映射

| 页面 | 主组件 | 子组件 |
|---|---|---|
| 仪表盘 | DashboardView | TrafficCard, KernelStatusCard, ModeSwitchCard, NetworkSummaryCard, CurrentNodeCard, ConnectivityCard, QuickActionsCard |
| 订阅管理 | SubscriptionsView | SubList, SubForm, SubDetail |
| Profile 管理 | ProfilesView | ProfileList, ProfileForm, RulesEditor, OverrideEditor, ConfigPreview, ExportDialog |
| 节点管理 | ProxiesView | ProxyGroupList, ProxyNodeList, ProxyNodeCard, DelayBar |
| 连接监控 | ConnectionsView | ConnectionTable, ConnectionDetail |
| 规则查看 | RulesView | RuleList, RuleItem |
| 内核日志 | LogsView | LogViewer |
| 系统设置 | SettingsView | (内嵌表单) |

---

## 八、Docker 构建设计

### 8.1 多阶段构建 Dockerfile

```dockerfile
# ========================================
# Stage 1: 构建前端
# ========================================
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm build

# ========================================
# Stage 2: 构建后端
# ========================================
FROM golang:1.23-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /webui-server ./cmd/server

# ========================================
# Stage 3: 最终镜像
# ========================================
FROM alpine:3.20

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    iptables \
    nginx \
    nodejs \
    npm \
    curl \
    && adduser -D -h /app webui

# 复制 mihomo 二进制（从官方镜像或预构建）
ARG MIHOMO_VERSION=latest
COPY --from=mihomo-builder /mihomo /data/mihomo/bin/mihomo
RUN chmod +x /data/mihomo/bin/mihomo

# 下载 GeoIP/GeoSite 初始数据
RUN mkdir -p /data/mihomo \
    && curl -fsSL -o /data/mihomo/Country.mmdb \
       https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/country.mmdb \
    && curl -fsSL -o /data/mihomo/geosite.dat \
       https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geosite.dat \
    && curl -fsSL -o /data/mihomo/geoip.metadb \
       https://github.com/MetaCubeX/meta-rules-dat/releases/download/latest/geoip.metadb

# 安装 Sub-Store
COPY Sub-Store/backend/ /app/sub-store/
WORKDIR /app/sub-store
RUN npm install --production
ENV SUB_STORE_BACKEND_API_PORT=3001
ENV SUB_STORE_BACKEND_API_HOST=127.0.0.1
ENV SUB_STORE_DATA_DIR=/data/sub-store

# 复制后端二进制
COPY --from=backend-builder /webui-server /app/webui-server

# 复制前端构建产物到 Nginx
COPY --from=frontend-builder /app/frontend/dist /usr/share/nginx/html
COPY docker/nginx.conf /etc/nginx/http.d/default.conf

# 复制启动脚本
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 数据卷
VOLUME ["/data"]

EXPOSE 80

ENTRYPOINT ["/entrypoint.sh"]
```

### 8.2 Nginx 配置

```nginx
# docker/nginx.conf
server {
    listen 80;
    server_name _;

    # Vue SPA 静态资源
    root /usr/share/nginx/html;
    index index.html;

    # SPA 路由 fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理 → Go 后端
    location /api/ {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    # WebSocket 代理（已在 /api/ws/* 下）
    location /api/ws/ {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

### 8.3 entrypoint.sh

```bash
#!/bin/sh
set -e

# 初始化数据目录
mkdir -p /data/webui/profiles
mkdir -p /data/mihomo/bin
mkdir -p /data/sub-store

# 生成初始 settings.json（如不存在）
if [ ! -f /data/webui/settings.json ]; then
    cat > /data/webui/settings.json << 'EOF'
{
  "mihomo": {
    "apiAddr": "127.0.0.1:9090",
    "secret": "",
    "binaryPath": "/data/mihomo/bin/mihomo",
    "configPath": "/data/mihomo/config.yaml"
  },
  "substore": {
    "apiAddr": "127.0.0.1:3001",
    "dataDir": "/data/sub-store"
  },
  "theme": "dark",
  "language": "zh"
}
EOF
fi

# 生成 mihomo 最小配置（如不存在）
if [ ! -f /data/mihomo/config.yaml ]; then
    cat > /data/mihomo/config.yaml << 'EOF'
mixed-port: 7890
allow-lan: false
mode: rule
log-level: info
external-controller: 127.0.0.1:9090
EOF
fi

# 启动 WebUI 后端（PID 1 职责移交）
exec /app/webui-server
```

---

## 九、Go 后端关键依赖

```
# go.mod

module github.com/yourname/proxy-web

go 1.23

require (
    github.com/go-chi/chi/v5       # HTTP router
    github.com/go-chi/cors          # CORS middleware
    github.com/golang-jwt/jwt/v5    # JWT
    github.com/gorilla/websocket    # WebSocket
    golang.org/x/crypto             # bcrypt password hashing
    gopkg.in/yaml.v3                # YAML 解析与操作
    github.com/natefinch/lumberjack # 日志轮转（可选）
)
```

---

## 十、docker-compose.yml

```yaml
version: "3.8"

services:
  proxy-web:
    build: .
    container_name: proxy-web
    restart: unless-stopped
    ports:
      - "9080:80"       # WebUI
      - "7890:7890"     # Mixed port (HTTP + SOCKS5)
      - "7891:7891"     # (可选) 额外监听端口
    volumes:
      - ./data:/data
    cap_add:
      - NET_ADMIN       # TUN 模式需要
    devices:
      - /dev/net/tun:/dev/net/tun  # TUN 模式需要
    environment:
      - TZ=Asia/Shanghai
      - MIHOMO_VERSION=latest  # 可指定 mihomo 版本
```

---

## 十一、关键实现细节

### 11.1 WebSocket 代理转发

Go 后端作为 WS 中间层，连接前端和 mihomo：

```
Browser ←WS→ Go Backend ←WS→ mihomo :9090
```

Go 端维护一个到 mihomo 的长连接，前端连接时复用数据：
- 多前端连接共享一个 mihomo WS 连接（fan-out）
- 无前端连接时断开 mihomo WS（节省资源）
- 断线自动重连

### 11.2 密码存储

- 使用 bcrypt 哈希
- settings.json 中存储 `passwordHash` 字段
- 首次启动无 hash 时进入 setup 模式

### 11.3 配置校验

```go
func ValidateConfig(yamlData []byte) []ValidationError {
    // 1. YAML 语法校验
    // 2. 必须字段检查（proxies 或 proxy-providers 至少有一个）
    // 3. rules 引用的策略组是否存在
    // 4. proxy-groups 引用的节点是否存在
    // 5. port 范围合法性
}
```

### 11.4 Profile 导出包格式

```
profile-export-{id}-{timestamp}.zip
├── manifest.json
├── platform/
│   ├── meta.json
│   ├── rules.yaml
│   └── override.yaml
└── subscriptions/      # 仅当 includeSubscriptions=true
    └── {name}.json     # Sub-Store 订阅定义
```

---

## 十二、开发计划

### Phase 1 — 项目脚手架 + 进程管理 (Week 1-2)
- Go 项目初始化 (go mod, chi router, 基础中间件)
- Vue 项目初始化 (Vite, Naive UI, router, pinia)
- Dockerfile + docker-compose
- 进程管理器（mihomo + Sub-Store 启停）
- 密码鉴权
- 基础 Nginx 配置

### Phase 2 — 内核对接 + 首页 (Week 2-3)
- mihomo API 客户端
- WebSocket 代理转发 (traffic/connections/logs/memory)
- 首页仪表盘全部卡片
- 运行模式切换
- 内核状态监控

### Phase 3 — 订阅与配置 (Week 3-5)
- Sub-Store API 客户端
- 订阅 CRUD 页面
- Profile 管理页面
- 配置合并管道
- Profile 切换（配置重载）

### Phase 4 — 节点与规则 (Week 5-6)
- 节点列表/切换/测速
- 规则查看
- 连接监控
- 内核日志

### Phase 5 — 高级功能 (Week 6-8)
- 全局规则 UI 编辑器
- 全局覆盖 YAML 编辑器 (Monaco)
- 链式代理配置
- 网络设置（LAN/TUN/IPv6/端口）
- GeoIP 管理
- Profile 导出/导入

### Phase 6 — 打磨 (Week 8+)
- 暗色/亮色主题
- 国际化 (zh/en)
- 网站连通性测试
- 响应式适配
- 错误处理优化
