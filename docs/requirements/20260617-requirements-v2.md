# Mihomo WebUI 控制面板 - 需求文档 v3

> 创建日期: 2026-06-17
> 更新日期: 2026-06-17
> 状态: 需求确认阶段（最终版）
> 内核: mihomo (Clash Meta)
> 部署方式: Docker（单容器）

---

## 一、项目概述

基于 mihomo 内核的代理管理 WebUI 控制面板，单 Docker 容器部署，通过 mihomo RESTful API 实现内核操控，集成 Sub-Store 进行订阅管理与协议转换。

### 1.1 核心决策记录

| 决策项 | 结论 | 备注 |
|---|---|---|
| 后端语言 | **Go 或 Rust** | 用户不熟悉，存在学习曲线风险 |
| 前端框架 | **Vue** | 用户较熟悉 |
| 容器模式 | **单容器**（WebUI + mihomo 一体） | 简化部署 |
| 鉴权方案 | **简单密码** | 单容器单实例单用户 |
| 配置存储 | **JSON/YAML 文件** | 数据量小，无需复杂数据库 |
| 内核实例 | **单内核** | 一个 WebUI 管一个 mihomo |
| 订阅管理 | **集成 Sub-Store** | 复用其协议解析和转换能力 |
| UI 风格 | **自行设计** | 不照搬 Clash Dashboard |
| 规则集支持 | **待定** | 需进一步讨论 |

### 1.2 参考项目

| 项目 | 语言 | 作用 | 复用方式 |
|---|---|---|---|
| **mihomo** | Go | 代理内核 | 作为被控进程运行在容器内 |
| **Clash Verge Rev** | Rust+React | 桌面端参考 | 功能设计与交互参考 |
| **Sub-Store** | Node.js | 订阅管理 | 作为订阅引擎集成到容器内 |
| **subconverter** | C++ | 协议转换 | 备选转换方案 |

---

## 二、Mihomo 内核 API 能力盘点

基于源码 `hub/route/` 分析，内核暴露的 RESTful API（Bearer Token 鉴权）：

| 端点 | 方法 | 功能 | 备注 |
|---|---|---|---|
| `GET /` | GET | 探活 | |
| `GET /version` | GET | 版本信息 | |
| `GET /traffic` | GET/WS | 实时上下行流量 | WebSocket 推送 |
| `GET /memory` | GET/WS | 内存使用 | WebSocket 推送 |
| `GET /logs` | GET/WS | 内核日志流 | WebSocket，支持 level 过滤 |
| `GET /configs` | GET | 当前运行时配置 | mode/ipv6/allow-lan/tun 等 |
| `PATCH /configs` | PATCH | 增量更新配置 | 运行时热修改 |
| `PUT /configs` | PUT | 全量重载配置 | path 或 payload(yaml) |
| `POST /configs/geo` | POST | 更新 GeoIP/GeoSite | |
| `GET /proxies` | GET | 所有代理节点列表 | |
| `GET /proxies/{name}` | GET | 单个代理详情 | |
| `PUT /proxies/{name}` | PUT | 切换 Selector 选中节点 | `{"name": "xxx"}` |
| `GET /proxies/{name}/delay` | GET | 单节点测速 | params: url, timeout |
| `DELETE /proxies/{name}` | DELETE | 取消固定选择 | |
| `GET /group` | GET | 代理组列表 | |
| `GET /group/{name}` | GET | 代理组详情 | |
| `GET /group/{name}/delay` | GET | 整组 URLTest 测速 | params: url, timeout |
| `GET /rules` | GET | 规则列表（含命中统计） | |
| `PATCH /rules/disable` | PATCH | 运行时启用/禁用规则 | `{index: bool}` |
| `GET /connections` | GET/WS | 活跃连接列表 | WebSocket 推送 |
| `DELETE /connections` | DELETE | 关闭全部连接 | |
| `DELETE /connections/{id}` | DELETE | 关闭指定连接 | |
| `GET /providers/proxies` | GET | 代理订阅提供者 | |
| `GET /providers/rules` | GET | 规则订阅提供者 | |
| `GET /dns` | GET | DNS 配置 | |
| `GET /cache` | GET | 缓存信息 | |
| `POST /restart` | POST | 重启内核 | |

---

## 三、Sub-Store 集成分析

### 3.1 Sub-Store 能力概览

Sub-Store 是一个成熟的订阅管理 Node.js 服务，具备完整的订阅生命周期管理：

**支持的协议解析（输入）：**
- vmess://、vless://、trojan://、ss://、ssr://
- hysteria2://、anytls://、socks5://、http://、snell://
- Clash YAML 配置（直接导入）
- Surge/Loon/QX 配置格式

**支持的输出格式（Producer）：**
- ✅ **clashmeta.js** — 直接输出 mihomo 兼容格式（核心需求）
- clash.js、sing-box.js、surge.js、loon.js、qx.js 等

**核心功能：**
- 订阅 CRUD（创建/读取/更新/删除）
- 订阅同步（定时拉取远程订阅）
- 节点处理（过滤/重命名/排序/去重）
- 集合管理（多订阅合并）
- RESTful API 完整暴露

### 3.2 集成方案

```
┌──────────────────────────────────────────────────────────────┐
│                     Docker Container                          │
│                                                               │
│  ┌──────────┐   ┌──────────────────┐   ┌────────────────┐   │
│  │ Vue 前端  │   │  WebUI 后端       │   │ Sub-Store 后端  │   │
│  │ (Nginx)  │──►│ (Go/Rust)        │──►│ (Node.js)      │   │
│  │  :80     │   │  :3000           │   │  :3001         │   │
│  └──────────┘   └───────┬──────────┘   └────────────────┘   │
│                         │                                     │
│                ┌────────▼──────────┐                          │
│                │   mihomo 内核     │                          │
│                │   API :9090       │                          │
│                │   Mixed :7890     │                          │
│                └───────────────────┘                          │
│                                                               │
│  Volume: /data                                                │
│  ├── sub-store/     (Sub-Store 数据)                           │
│  ├── webui/         (WebUI 配置: profiles, settings)          │
│  ├── mihomo/        (内核配置文件)                              │
│  └── config.yaml    (启动配置)                                 │
└──────────────────────────────────────────────────────────────┘
```

### 3.3 进程管理

单容器内运行 3 个进程：

| 进程 | 角色 | 管理方式 |
|---|---|---|
| Nginx | 前端静态资源 + 反向代理 | 系统服务/init |
| WebUI Backend (Go/Rust) | 主进程，API + 进程管理 | PID 1，负责启动/监控其他进程 |
| Sub-Store (Node.js) | 订阅管理引擎 | WebUI 后端作为子进程启动 |
| mihomo | 代理内核 | WebUI 后端作为子进程启动 |

> **备选方案**：使用 `s6-overlay` 或 `supervisord` 管理多进程，但推荐 WebUI 后端作为 PID 1 自行管理，减少依赖。

---

## 四、功能需求详细设计

### F01 - 订阅配置管理

**描述：** 通过 Sub-Store 引擎实现订阅导入、转换和管理。

| ID | 需求 | 优先级 | 实现方式 |
|---|---|---|---|
| F01.1 | URL 订阅导入 | P0 | 调用 Sub-Store API 创建订阅，支持各协议 URI |
| F01.2 | 协议自动识别转换 | P0 | Sub-Store parsers 自动识别并解析 |
| F01.3 | 本地文件导入 | P0 | 上传 yaml 文件，通过 Sub-Store 或直接解析 |
| F01.4 | 多订阅管理 | P0 | Sub-Store subscription CRUD API |
| F01.5 | 订阅自动更新 | P1 | Sub-Store cron sync 功能 |
| F01.6 | 订阅健康检查 | P1 | 检查拉取可达性和节点可用性 |
| F01.7 | 节点过滤/重命名/排序 | P1 | Sub-Store processors |
| F01.8 | 多订阅合并（集合） | P1 | Sub-Store collection 功能 |

**与 Sub-Store 的交互流程：**
```
用户添加订阅 URL
    ↓
WebUI 后端 → POST /api/subs (Sub-Store) 创建订阅记录
    ↓
WebUI 后端 → GET /download/{name}?target=clashmeta (Sub-Store) 拉取转换后的 mihomo yaml
    ↓
WebUI 后端 → 将 yaml 与全局规则/覆盖合并 → 写入最终配置文件
    ↓
WebUI 后端 → PUT /configs (mihomo) 推送给内核
```

---

### F02 - 配置管理与切换

**描述：** Profile 管理与配置合并切换。

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F02.1 | Profile 管理 | P0 | 每个 Profile = 订阅源 + 全局规则 + 全局覆盖 |
| F02.2 | Profile 切换 | P0 | 切换后合并配置并 `PUT /configs` 重载 |
| F02.3 | 全局规则 UI 编辑器 | P0 | 可视化编辑 rules 段（拖拽排序/类型选择/策略选择） |
| F02.4 | 全局配置覆盖 YAML 编辑器 | P0 | Monaco Editor，编辑 override/merge 配置 |
| F02.5 | 配置预览 | P1 | 切换前预览最终合并的完整 yaml |
| F02.6 | 配置校验 | P0 | YAML 语法校验 + mihomo 配置项校验 |

**配置合并管道：**
```
订阅原始配置 (Sub-Store 输出的 clashmeta yaml)
    ↓ merge（浅合并）
全局覆盖 (用户 YAML 编辑器内容)
    ↓ prepend rules
全局规则 (UI 编辑器管理的规则)
    ↓
最终配置 → mihomo PUT /configs
```

**数据存储结构（JSON 文件）：**
```
/data/webui/
├── settings.json          # 全局设置（密码、主题、语言）
├── profiles.json          # Profile 注册表
├── profiles/
│   ├── {id}.json          # Profile 元数据
│   ├── {id}.rules.yaml    # 全局规则
│   └── {id}.override.yaml # 全局覆盖
└── subscriptions.json     # 订阅与 Profile 的关联
```

---

### F03 - 规则查看与管理

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F03.1 | 规则列表展示 | P0 | `GET /rules` → index/type/payload/proxy/hitCount |
| F03.2 | 规则来源标注 | P1 | 区分全局规则 vs 订阅配置规则 |
| F03.3 | 规则搜索/过滤 | P1 | 按类型、关键字、策略组过滤 |
| F03.4 | 规则运行时禁用 | P1 | `PATCH /rules/disable`（运行时，不持久化） |
| F03.5 | 命中统计可视化 | P2 | 命中次数排序/热度标识 |

---

### F04 - 连接监控

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F04.1 | 连接列表实时展示 | P0 | WebSocket `GET /connections` → host/chains/rule/traffic |
| F04.2 | 关闭单个连接 | P0 | `DELETE /connections/{id}` |
| F04.3 | 关闭全部连接 | P0 | `DELETE /connections` |
| F04.4 | 连接搜索/过滤 | P1 | 按 host、规则、代理链过滤 |
| F04.5 | 流量排序 | P1 | 按上传/下载/总流量排序 |
| F04.6 | 暂停自动刷新 | P1 | 暂停 WebSocket 更新减少渲染开销 |

---

### F05 - 内核日志

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F05.1 | 日志实时流 | P0 | WebSocket `GET /logs` |
| F05.2 | 日志级别过滤 | P0 | error/warning/info/debug/silent |
| F05.3 | 日志搜索 | P1 | 全文搜索 |
| F05.4 | 日志导出 | P2 | 导出缓冲区为文本 |

---

### F06 - 节点管理与测速

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F06.1 | 代理组列表 | P0 | `GET /group` → Selector/URLTest/Fallback/LoadBalance |
| F06.2 | 组内节点列表 | P0 | 展示节点及当前选中 |
| F06.3 | 切换节点 | P0 | `PUT /proxies/{name}` |
| F06.4 | 单节点测速 | P0 | `GET /proxies/{name}/delay` |
| F06.5 | 整组测速 | P0 | `GET /group/{name}/delay` |
| F06.6 | 测速结果可视化 | P1 | 延迟颜色编码（绿/黄/红），超时标识 |
| F06.7 | 节点搜索 | P1 | 按名称/类型搜索 |
| F06.8 | 节点详情 | P1 | 类型/服务器/端口/TLS 等信息 |

---

### F07 - 运行模式切换

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F07.1 | 模式切换 | P0 | `PATCH /configs` → mode: rule/global/direct |
| F07.2 | 模式状态显示 | P0 | 当前模式高亮 |

---

### F08 - 链式代理

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F08.1 | 链式代理配置 | P0 | 可视化编辑 proxy-groups 嵌套 / proxy-chains |
| F08.2 | 代理链可视化 | P1 | 图形化链路展示 |
| F08.3 | 预设模板 | P2 | 常见场景（本地→中转→落地） |

---

### F09 - 局域网连接模式

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F09.1 | allow-lan 开关 | P0 | `PATCH /configs` → allow-lan |
| F09.2 | 绑定地址 | P1 | bind-address 配置 |
| F09.3 | LAN IP 黑白名单 | P2 | lan-allowed-ips / lan-disallowed-ips |

---

### F10 - IPv6 开关

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F10.1 | IPv6 开关 | P0 | `PATCH /configs` → ipv6 |
| F10.2 | 状态显示 | P0 | 当前状态 |

---

### F11 - 系统代理与 TUN 模式

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F11.1 | 监听端口管理 | P1 | HTTP/SOCKS/Mixed/Redir/TProxy 端口显示与修改 |
| F11.2 | TUN 模式开关 | P0 | `PATCH /configs` → tun.enable |
| F11.3 | TUN 高级配置 | P1 | stack/auto-route/MTU 等 |

> **Docker 环境说明：** 容器内无传统"系统代理"概念，流量接管通过端口映射和 TUN 模式实现。TUN 需要 `--cap-add=NET_ADMIN` 和 `--device /dev/net/tun`。

---

### F12 - 首页仪表盘

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F12.1 | 订阅信息卡片 | P0 | 当前 Profile、订阅来源、更新时间、节点数 |
| F12.2 | 节点选择快捷入口 | P0 | 当前选中节点，快速切换 |
| F12.3 | 网络设置摘要 | P0 | 端口/allow-lan/TUN/IPv6 状态 |
| F12.4 | 代理模式切换 | P0 | Rule/Global/Direct 快速切换 |
| F12.5 | 流量统计 | P0 | 实时速率 + 累计总量（WebSocket） |
| F12.6 | 网站连通性测试 | P1 | Google/YouTube/GitHub 等连通性和延迟 |
| F12.7 | 内核状态 | P0 | 运行状态/版本/内存/运行时长 |
| F12.8 | 快速操作 | P1 | 一键关闭连接/一键测速 |

---

### F13 - 内核启停控制

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F13.1 | 启动内核 | P0 | WebUI 后端拉起 mihomo 子进程 |
| F13.2 | 停止内核 | P0 | SIGTERM/SIGINT 信号 |
| F13.3 | 重启内核 | P0 | `POST /restart` 或进程重启 |
| F13.4 | 存活监控 | P0 | 心跳检测，异常告警 |
| F13.5 | 启动日志 | P1 | 启动过程实时展示 |

---

## 五、非功能需求

| ID | 需求 | 说明 |
|---|---|---|
| NF01 | Docker 单容器 | WebUI + Sub-Store + mihomo 一体 |
| NF02 | 响应式 UI | 桌面和移动端适配 |
| NF03 | 密码鉴权 | 单用户简单密码，首次启动设置 |
| NF04 | 文件持久化 | JSON/YAML 文件存储到 volume |
| NF05 | 国际化 | 中文/英文 |
| NF06 | 暗色模式 | 暗色/亮色主题 |
| NF07 | WebSocket 实时通信 | 流量/连接/日志/内存均 WS 推送 |
| NF08 | 错误处理 | API 失败友好提示 |

---

## 六、技术架构

```
┌──────────────────────────────────────────────────────────┐
│                    Docker Container                       │
│                                                          │
│  ┌────────────┐                                          │
│  │   Nginx    │  :80 (HTTP)                              │
│  │  Vue SPA   │  静态资源 + /api/* 反向代理               │
│  └──────┬─────┘                                          │
│         │                                                │
│  ┌──────▼──────────────────────┐   ┌──────────────────┐ │
│  │  WebUI Backend (Go/Rust)    │   │  Sub-Store        │ │
│  │  PID 1 / 进程管理器          │──►│  (Node.js) :3001  │ │
│  │  :3000                      │   │  订阅引擎         │ │
│  │                             │   └──────────────────┘ │
│  │  ┌───────────────────────┐  │                        │
│  │  │ mihomo 进程管理        │  │                        │
│  │  │ API 代理/WebSocket 转发│  │                        │
│  │  │ 配置合并引擎           │  │                        │
│  │  │ 密码鉴权              │  │                        │
│  │  └───────────┬───────────┘  │                        │
│  └──────────────│──────────────┘                        │
│         ┌───────▼──────────┐                             │
│         │  mihomo 内核     │                             │
│         │  API :9090       │                             │
│         │  Mixed :7890     │                             │
│         │  TUN (optional)  │                             │
│         └──────────────────┘                             │
│                                                          │
│  /data volume                                            │
│  ├── webui/settings.json                                 │
│  ├── webui/profiles.json                                 │
│  ├── webui/profiles/{id}.*                               │
│  ├── sub-store/data.json                                 │
│  └── mihomo/config.yaml                                  │
└──────────────────────────────────────────────────────────┘
```

### 6.1 WebUI 后端职责

| 模块 | 职责 |
|---|---|
| 进程管理器 | 启动/停止/监控 mihomo 和 Sub-Store 子进程 |
| API 路由 | `/api/auth`, `/api/profiles`, `/api/settings`, `/api/kernel/*` |
| 内核代理 | 透传/增强 mihomo API 调用 |
| WebSocket 代理 | 转发 mihomo WS 流（traffic/connections/logs/memory） |
| 配置合并 | 订阅 yaml + 全局覆盖 + 全局规则 → 最终配置 |
| 鉴权中间件 | 简单密码校验，JWT 或 Session |

### 6.2 技术选型

| 层 | 选择 | 说明 |
|---|---|---|
| 前端 | **Vue 3 + Vite** | 用户熟悉 |
| UI 组件库 | **待选**（Naive UI / Element Plus / Ant Design Vue） | 需讨论 |
| 代码编辑器 | **Monaco Editor** | YAML 编辑 |
| 后端 | **Go 或 Rust** | 待最终确认 |
| 订阅引擎 | **Sub-Store (Node.js)** | 作为独立进程 |
| 进程管理 | 自实现（Go os/exec / Rust Command） | PID 1 管理子进程 |
| 数据存储 | JSON/YAML 文件 | 轻量无依赖 |
| 容器初始化 | **自定义 entrypoint.sh** 或 **s6-overlay** | 多进程管理 |

---

## 七、页面路由规划

```
/                       首页仪表盘（F12）
/subscriptions          订阅管理（F01）
/profiles               配置 Profile 管理（F02）
/profiles/:id/rules     全局规则编辑（F02.3）
/profiles/:id/override  全局配置覆盖（F02.4）
/rules                  规则查看（F03）
/proxies                节点管理（F06）
/connections            连接监控（F04）
/logs                   内核日志（F05）
/settings               系统设置
  /settings/general     通用设置（模式 F07、IPv6 F10）
  /settings/network     网络设置（LAN F09、端口 F11、TUN F11）
  /settings/chains      链式代理（F08）
```

---

## 八、里程碑规划

### M1 - 基础骨架（2 周）
- [ ] 项目脚手架（Vue + Go/Rust 后端 + Docker）
- [ ] 多进程管理框架（PID 1 启动 mihomo + Sub-Store）
- [ ] 密码鉴权
- [ ] 内核 API 连通性验证
- [ ] 首页仪表盘框架（内核状态/流量/模式切换）

### M2 - 订阅与节点（2-3 周）
- [ ] Sub-Store 集成与 API 对接
- [ ] 订阅导入/管理（F01）
- [ ] Profile 管理与切换（F02.1-F02.2）
- [ ] 节点查看/切换/测速（F06）
- [ ] 运行模式切换（F07）

### M3 - 监控与规则（1-2 周）
- [ ] 连接监控（F04）
- [ ] 内核日志（F05）
- [ ] 规则查看（F03）

### M4 - 高级配置（2-3 周）
- [ ] 全局规则 UI 编辑器（F02.3）
- [ ] 全局配置覆盖编辑器（F02.4）
- [ ] 链式代理（F08）
- [ ] 网络设置（F09-F11）

### M5 - 打磨（持续）
- [ ] 网站连通性测试（F12.6）
- [ ] 国际化（NF05）
- [ ] 暗色模式（NF06）
- [ ] 移动端适配

---

## 九、待讨论事项

### 9.1 后端语言最终确认

| 选项 | 优势 | 风险 |
|---|---|---|
| **Go** | 与 mihomo 同语言，可复用 config parser；生态成熟；goroutine 天然适合多进程管理和 WS 代理 | 用户不熟，但 Go 学习曲线相对平缓 |
| **Rust** | 性能极致；内存安全 | 学习曲线陡峭，开发周期更长 |

**建议**：Go 更合适。理由：
1. mihomo 本身就是 Go，可以直接 import mihomo 的 config 包做配置校验
2. goroutine 处理多进程管理 + WebSocket 代理 + HTTP 服务非常自然
3. 编译产物是单个静态二进制，容器镜像小
4. 学习曲线比 Rust 平缓很多

### 9.2 Vue UI 组件库

| 选项 | 特点 |
|---|---|
| **Naive UI** | Vue 3 原生，TypeScript 优先，暗色模式好，组件丰富 |
| **Element Plus** | 社区大，文档全，但 Vue 2 痕迹较重 |
| **Ant Design Vue** | 设计规范好，但较重 |

### 9.3 规则集（rule-provider）支持

mihomo 支持 `rule-providers`（远程规则集，如 GeoSite/GeoIP 规则文件），API 提供了 `GET /providers/rules`。

需要讨论是否支持：
- 查看已加载的规则集列表和状态
- 手动触发规则集更新
- 在 Profile 中配置 rule-providers
- 规则集来源管理（添加/删除规则集源）

**建议**：M4 阶段加入，作为规则管理的增强功能。

### 9.4 Sub-Store 集成深度

| 方案 | 描述 | 优劣 |
|---|---|---|
| **A. 完整集成** | 容器内运行 Sub-Store Node.js 进程，WebUI 通过其 API 操作 | ✅ 功能完整，❌ 多一个 Node.js 运行时 |
| **B. 提取核心** | 只提取 Sub-Store 的 parser + producer 逻辑，用 Go 重写 | ✅ 无 Node 依赖，❌ 工作量大且需持续同步 |
| **C. subconverter 替代** | 用 subconverter (C++) 做协议转换 | ✅ 独立二进制，❌ 2021年后维护减少 |

**建议**：方案 A（完整集成 Sub-Store）。理由：
1. Sub-Store 功能最完整，协议支持最广
2. 维护活跃（2026 年仍在更新）
3. Node.js 运行时在容器中开销可接受
4. 避免大量重复开发

### 9.5 其他待确认

- 是否需要 **DNS 配置管理** 的可视化界面？
- 是否需要 **GeoIP/GeoSite 数据库管理**（更新/查看版本）？
- 是否需要 **配置文件导入/导出**（整个 Profile 的备份恢复）？
- 是否需要 **操作审计日志**（谁在什么时候做了什么操作）？
