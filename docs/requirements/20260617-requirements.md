# Mihomo WebUI 控制面板 - 需求文档

> 创建日期: 2026-06-17
> 状态: 需求整理阶段
> 内核: mihomo (Clash Meta)
> 部署方式: Docker

---

## 一、项目概述

基于 mihomo 内核的代理管理 WebUI 控制面板，通过 Docker 部署，以 mihomo 的 RESTful API 为底层通信手段，提供完整的代理配置管理、节点控制、连接监控、规则管理等功能。参考项目：Clash Verge Rev。

### 1.1 与 mihomo 的通信方式

mihomo 暴露 RESTful API（默认 `0.0.0.0:9090`），支持 Bearer Token 鉴权（`Authorization: Bearer <secret>`）。WebUI 后端作为中间层与内核通信，前端通过 WebUI 后端间接操作内核。

**关键约束：** WebUI 与内核运行在同一 Docker 容器或同一 Docker 网络中，通过环境变量或配置文件指定内核 API 地址和 Secret。

---

## 二、Mihomo REST API 能力盘点

在开始功能设计前，先梳理内核已提供的 API 能力（源码 `hub/route/`）：

| 端点 | 方法 | 功能 | 备注 |
|---|---|---|---|
| `GET /` | GET | Hello 探活 | |
| `GET /version` | GET | 内核版本信息 | |
| `GET /traffic` | GET/WS | 实时上下行流量 | 支持 WebSocket 推送 |
| `GET /memory` | GET/WS | 内存使用 | 支持 WebSocket 推送 |
| `GET /logs` | GET/WS | 内核日志流 | 支持 WebSocket 推送，可指定 level 过滤 |
| `GET /configs` | GET | 获取当前运行时配置 | 包含 mode、ipv6、allow-lan 等 |
| `PATCH /configs` | PATCH | 增量更新配置 | mode/ipv6/allow-lan/tun/bind-address/log-level 等 |
| `PUT /configs` | PUT | 全量重载配置 | 支持 `path` 或 `payload`（yaml 文本） |
| `POST /configs/geo` | POST | 更新 GeoIP/GeoSite 数据库 | |
| `GET /proxies` | GET | 列出所有代理节点 | 含节点类型、历史延迟等 |
| `GET /proxies/{name}` | GET | 获取单个代理详情 | |
| `PUT /proxies/{name}` | PUT | 切换 Selector 节点 | body: `{"name": "node-name"}` |
| `GET /proxies/{name}/delay` | GET | 单节点测速 | 参数: url, timeout |
| `DELETE /proxies/{name}` | DELETE | 取消固定节点选择 | |
| `GET /group` | GET | 列出所有代理组 | |
| `GET /group/{name}` | GET | 获取单个代理组详情 | |
| `GET /group/{name}/delay` | GET | 整组测速（URLTest） | 参数: url, timeout, expected |
| `GET /rules` | GET | 列出所有生效规则 | 含命中次数等统计 |
| `PATCH /rules/disable` | PATCH | 启用/禁用指定规则 | body: `{index: bool}` |
| `GET /connections` | GET/WS | 当前连接列表 | 支持 WebSocket 实时推送 |
| `DELETE /connections` | DELETE | 关闭所有连接 | |
| `DELETE /connections/{id}` | DELETE | 关闭指定连接 | |
| `GET /providers/proxies` | GET | 代理订阅提供者列表 | |
| `GET /providers/rules` | GET | 规则订阅提供者列表 | |
| `POST /restart` | POST | 重启内核进程 | |
| `GET /dns` | GET | DNS 配置信息 | |
| `GET /cache` | GET | 缓存信息 | |

---

## 三、功能需求详细设计

### F01 - 订阅配置管理

**描述：** 通过订阅链接或本地文件获取代理配置，支持多订阅并存。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F01.1 | 通过 URL 导入订阅 | P0 | 支持常见机场订阅链接（Clash/V2Ray/SS 等协议格式） |
| F01.2 | 协议格式自动识别与转换 | P0 | 识别 vmess://, vless://, ss://, trojan://, clash yaml 等格式，统一转为 mihomo yaml |
| F01.3 | 本地文件导入 | P0 | 上传 .yaml/.yml 配置文件 |
| F01.4 | 多订阅并存管理 | P0 | 订阅列表展示，支持启用/禁用/删除/重命名/排序 |
| F01.5 | 订阅自动更新 | P1 | 可配置更新间隔，手动触发更新，显示上次更新时间 |
| F01.6 | 订阅健康检查 | P1 | 检查订阅是否可拉取、节点是否可用 |

**技术设计：**
- 后端维护订阅注册表（SQLite/JSON 文件），记录每个订阅的 URL、名称、更新时间、原始内容、转换后的 mihomo yaml
- 协议转换模块：解析各协议 URI，生成 mihomo 的 proxies 配置段
- 订阅拉取走后端代理（避免前端 CORS 问题），支持自定义 User-Agent

---

### F02 - 配置管理与切换

**描述：** 在多个配置间切换，支持全局规则编辑和全局配置覆盖。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F02.1 | 多配置 Profile 管理 | P0 | 每个 Profile 由订阅配置 + 全局覆盖 + 全局规则组合而成 |
| F02.2 | Profile 切换 | P0 | 切换后立即通过 `PUT /configs` 重载内核配置 |
| F02.3 | 全局规则 UI 编辑器 | P0 | 可视化编辑 rules 段，支持拖拽排序、类型选择、策略组选择 |
| F02.4 | 全局配置覆盖 YAML 编辑器 | P0 | CodeMirror/Monaco 编辑器，编辑 merge/override 配置段 |
| F02.5 | 配置预览 | P1 | 切换前预览最终合并的完整配置 |
| F02.6 | 配置校验 | P0 | 提交前校验 yaml 语法及 mihomo 配置项合法性 |

**技术设计：**
- 配置合并策略：`订阅原始配置` → `Merge（浅合并）` → `Override（深度覆盖）` → `全局规则 prepend`
- 最终配置写入临时文件，通过 `PUT /configs`（path 或 payload）推送给内核
- 全局规则存储在 WebUI 后端，追加到最终配置的 rules 段头部（prepend）或尾部（append）

---

### F03 - 规则查看与管理

**描述：** 查看当前生效配置的完整规则列表（全局规则 + 订阅自带规则）。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F03.1 | 规则列表展示 | P0 | 通过 `GET /rules` 获取，展示 index/type/payload/proxy/命中次数 |
| F03.2 | 规则来源标注 | P1 | 标注每条规则来自全局规则还是订阅配置 |
| F03.3 | 规则搜索/过滤 | P1 | 按类型、关键字、策略组过滤 |
| F03.4 | 规则启用/禁用 | P1 | 通过 `PATCH /rules/disable` 实现运行时禁用（不持久化） |
| F03.5 | 命中统计可视化 | P2 | 命中次数排序、热度图 |

---

### F04 - 连接监控

**描述：** 实时查看当前活跃连接，支持关闭连接。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F04.1 | 连接列表实时展示 | P0 | 通过 WebSocket `GET /connections` 获取，展示 host/chains/rule/upload/download/start 等 |
| F04.2 | 关闭单个连接 | P0 | 通过 `DELETE /connections/{id}` |
| F04.3 | 关闭全部连接 | P0 | 通过 `DELETE /connections` |
| F04.4 | 连接搜索/过滤 | P1 | 按 host、规则、代理链过滤 |
| F04.5 | 流量排序 | P1 | 按上传/下载/总流量排序 |
| F04.6 | 连接暂停/自动刷新 | P1 | 可暂停 WebSocket 更新以减少渲染开销 |

---

### F05 - 内核日志

**描述：** 实时查看内核运行日志。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F05.1 | 日志实时流 | P0 | 通过 WebSocket `GET /logs` 获取 |
| F05.2 | 日志级别过滤 | P0 | 支持 error/warning/info/debug/silent 级别过滤 |
| F05.3 | 日志搜索 | P1 | 全文搜索日志内容 |
| F05.4 | 日志导出 | P2 | 导出当前日志缓冲区为文本文件 |

---

### F06 - 节点管理与测速

**描述：** 查看节点列表，切换节点，进行延迟测试。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F06.1 | 代理组列表 | P0 | 通过 `GET /group` 获取，展示 Selector/URLTest/Fallback/LoadBalance 等类型 |
| F06.2 | 组内节点列表 | P0 | 展示每个组内的节点及当前选中状态 |
| F06.3 | 切换节点 | P0 | 通过 `PUT /proxies/{name}` 切换 Selector 组的选中节点 |
| F06.4 | 单节点测速 | P0 | 通过 `GET /proxies/{name}/delay` 测试指定节点延迟 |
| F06.5 | 整组测速 | P0 | 通过 `GET /group/{name}/delay` 测试整组节点 |
| F06.6 | 测速结果可视化 | P1 | 延迟条形图/颜色编码（绿/黄/红），显示超时/错误 |
| F06.7 | 节点搜索 | P1 | 按名称、类型搜索节点 |
| F06.8 | 节点信息详情 | P1 | 展示节点类型、服务器、端口、TLS 等详细信息 |

---

### F07 - 运行模式切换

**描述：** 切换内核运行模式。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F07.1 | 模式切换 UI | P0 | 通过 `PATCH /configs` 设置 mode 字段 |
| F07.2 | 支持模式 | P0 | rule（规则模式）、global（全局模式）、direct（直连模式） |
| F07.3 | 模式状态显示 | P0 | 当前模式高亮显示 |

---

### F08 - 链式代理

**描述：** 支持配置代理链（proxy-chains），使流量经过多个代理节点依次转发。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F08.1 | 链式代理配置 UI | P0 | 可视化配置 mihomo 的 proxy-groups 中的 `use` / proxy-chains |
| F08.2 | 代理链可视化 | P1 | 图形化展示流量链路：客户端 → 代理A → 代理B → 目标 |
| F08.3 | 预设链路模板 | P2 | 常见场景模板（如：本地 → 中转 → 落地） |

**技术设计：**
- 利用 mihomo 配置中的 `proxy-groups` 嵌套引用实现链式代理
- 也可通过全局配置覆盖的 YAML 编辑器直接编写 `proxy-chains` 配置

---

### F09 - 局域网连接模式

**描述：** 控制是否允许局域网其他设备连接本代理。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F09.1 | 局域网开关 | P0 | 通过 `PATCH /configs` 设置 `allow-lan` |
| F09.2 | 绑定地址配置 | P1 | 通过 `bind-address` 控制绑定网卡 |
| F09.3 | 认证前缀白名单 | P2 | 通过 `skip-auth-prefixes` 配置免认证 IP 段 |
| F09.4 | LAN 访问 IP 黑白名单 | P2 | 通过 `lan-allowed-ips` / `lan-disallowed-ips` |

---

### F10 - IPv6 开关

**描述：** 控制内核是否启用 IPv6 解析和代理。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F10.1 | IPv6 开关 | P0 | 通过 `PATCH /configs` 设置 `ipv6` 字段 |
| F10.2 | 状态显示 | P0 | 显示当前 IPv6 启用/禁用状态 |

---

### F11 - 系统代理与 TUN 模式

**描述：** 控制系统代理和 TUN 模式的启停。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F11.1 | 系统代理开关 | P1 | Docker 部署下无传统系统代理概念，改为管理 HTTP/SOCKS/Mixed 监听端口 |
| F11.2 | TUN 模式开关 | P0 | 通过 `PATCH /configs` 设置 `tun.enable` |
| F11.3 | TUN 高级配置 | P1 | stack/auto-route/auto-detect-interface 等 |
| F11.4 | 监听端口管理 | P1 | 显示/修改 HTTP/SOCKS/Mixed/Redir/TProxy 端口 |

**技术说明：**
- Docker 环境下"系统代理"概念弱化，主要通过端口映射和 TUN 模式实现流量接管
- TUN 模式需要 Docker 容器具有 `NET_ADMIN` 权限和 `--cap-add=NET_ADMIN`

---

### F12 - 首页仪表盘

**描述：** 首页展示核心信息摘要和快速操作入口。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F12.1 | 订阅信息卡片 | P0 | 当前激活的 Profile 名称、订阅来源、上次更新、节点数 |
| F12.2 | 节点选择快捷入口 | P0 | 显示当前 Selector 组的选中节点，快速切换 |
| F12.3 | 网络设置摘要 | P0 | 监听端口、allow-lan、TUN 状态、IPv6 状态 |
| F12.4 | 代理模式切换 | P0 | Rule/Global/Direct 快速切换按钮 |
| F12.5 | 流量统计 | P0 | 实时上下行速率、累计上传/下载总量 |
| F12.6 | 网站连通性测试 | P1 | 测试常见网站（Google/YouTube/GitHub 等）的连通性和延迟 |
| F12.7 | 内核状态 | P0 | 运行状态、版本号、内存占用、运行时长 |
| F12.8 | 快速操作卡片 | P1 | 一键关闭所有连接、一键测速等 |

---

### F13 - 内核启停控制

**描述：** 通过 WebUI 控制 mihomo 内核进程的启动和停止。

**子需求：**

| ID | 需求 | 优先级 | 说明 |
|---|---|---|---|
| F13.1 | 启动内核 | P0 | WebUI 后端拉起 mihomo 进程 |
| F13.2 | 停止内核 | P0 | 通过信号或 `executor.Shutdown()` 停止内核 |
| F13.3 | 重启内核 | P0 | 通过 `POST /restart` 或进程管理实现 |
| F13.4 | 状态监控 | P0 | 心跳检测内核是否存活，异常时自动告警 |
| F13.5 | 启动日志 | P1 | 内核启动过程中的日志实时展示 |

**技术设计：**
- WebUI 后端作为进程管理者（supervisor 模式），管理 mihomo 子进程生命周期
- 通过 `POST /restart` 实现热重启
- 停止内核通过向子进程发送 SIGTERM/SIGINT

---

## 四、非功能需求

| ID | 需求 | 说明 |
|---|---|---|
| NF01 | Docker 部署 | 单容器（WebUI + mihomo），docker-compose 支持 |
| NF02 | 响应式 UI | 适配桌面和移动端浏览器 |
| NF03 | 鉴权安全 | WebUI 自身需提供登录鉴权（防止局域网未授权访问） |
| NF04 | 数据持久化 | 订阅数据、Profile 配置、用户设置持久化到 volume |
| NF05 | 国际化 | 至少支持中文/英文 |
| NF06 | 暗色模式 | 支持暗色/亮色主题切换 |
| NF07 | WebSocket 实时通信 | 流量、连接、日志等使用 WebSocket 推送 |
| NF08 | 错误处理 | API 调用失败时给出友好提示 |

---

## 五、技术架构建议

```
┌─────────────────────────────────────────────────────┐
│                   Docker Container                   │
│                                                      │
│  ┌──────────────┐    ┌──────────────────────────┐   │
│  │  WebUI 前端   │    │     WebUI 后端           │   │
│  │  (React/Vue)  │◄──►│  (Go/Node.js/Rust)       │   │
│  │  :80/:443     │    │  :3000                   │   │
│  └──────────────┘    └──────────┬───────────────┘   │
│                                 │ HTTP/WS            │
│                       ┌─────────▼──────────────┐     │
│                       │   mihomo 内核           │     │
│                       │   REST API :9090        │     │
│                       │   Mixed Port :7890      │     │
│                       │   TUN (optional)        │     │
│                       └────────────────────────┘     │
│                                                      │
│  Volume: /data (订阅, 配置, 数据库)                    │
└─────────────────────────────────────────────────────┘
```

### 技术选型建议（待讨论）

| 层 | 候选方案 | 推荐 | 理由 |
|---|---|---|---|
| 前端框架 | React + Ant Design / Vue + Element Plus / React + shadcn/ui | React + shadcn/ui | 现代化、组件灵活、社区活跃 |
| 前端构建 | Vite | Vite | 快速开发体验 |
| 代码编辑器 | Monaco Editor / CodeMirror 6 | Monaco | VS Code 同款，YAML 支持好 |
| 后端语言 | Go / Node.js / Rust | Go | 与 mihomo 同语言，便于复用解析逻辑 |
| 后端框架 | Gin / Echo / Chi | Chi | 与 mihomo 同框架，轻量 |
| 数据存储 | SQLite / JSON 文件 / BoltDB | SQLite | 结构化存储，查询灵活 |
| 进程管理 | os/exec + signal | 自实现 | Docker 内单容器，无需 supervisor |
| WebSocket | gorilla/websocket | gorilla/websocket | 成熟稳定 |

---

## 六、页面路由规划

```
/                       首页仪表盘（F12）
/profiles               订阅与配置管理（F01 + F02）
/profiles/:id/rules     全局规则编辑（F02.3）
/profiles/:id/override  全局配置覆盖编辑（F02.4）
/rules                  规则查看（F03）
/proxies                节点管理（F06）
/connections            连接监控（F04）
/logs                   内核日志（F05）
/settings               系统设置（F07 + F09 + F10 + F11）
/settings/mode          运行模式（F07）
/settings/network       网络设置（F09 + F10 + F11）
/settings/chains        链式代理（F08）
```

---

## 七、优先级排序（里程碑规划）

### M1 - 基础骨架（1-2 周）
- 项目脚手架搭建（前端 + 后端 + Docker）
- 内核进程管理（F13：启动/停止/重启/状态）
- 内核 API 连接与鉴权
- 首页仪表盘框架（F12：内核状态、流量统计、模式切换）

### M2 - 配置与节点（2-3 周）
- 订阅导入与管理（F01）
- 多 Profile 切换（F02.1-F02.2）
- 节点查看与切换（F06.1-F06.3）
- 节点测速（F06.4-F06.6）
- 运行模式切换（F07）

### M3 - 监控与规则（1-2 周）
- 连接监控（F04）
- 内核日志（F05）
- 规则查看（F03）

### M4 - 高级配置（2-3 周）
- 全局规则 UI 编辑器（F02.3）
- 全局配置覆盖编辑器（F02.4）
- 链式代理配置（F08）
- 网络设置（F09-F11）

### M5 - 打磨与增强（持续）
- 网站连通性测试（F12.6）
- 国际化（NF05）
- 暗色模式（NF06）
- 移动端适配
- 配置校验与预览（F02.5-F02.6）

---

## 八、待讨论事项

1. **后端语言最终确认**：Go（与内核同语言）、Node.js（前后端同语言）、Rust（性能极致）？
2. **前端框架最终确认**：React vs Vue？UI 组件库选择？
3. **单容器 vs 多容器**：WebUI 与内核同容器（简单）还是分开（灵活）？
4. **WebUI 鉴权方案**：简单密码、用户名+密码、还是多用户？
5. **配置存储格式**：SQLite 还是纯文件（JSON/YAML）？
6. **协议转换库**：自建还是复用开源（如 subconverter）？
7. **是否需要支持多内核实例**（同时管理多个 mihomo）？
8. **是否需要规则集（rule-provider）管理能力**？
9. **是否需要 Clash Dashboard 风格的 UI 布局**还是另起炉灶？
