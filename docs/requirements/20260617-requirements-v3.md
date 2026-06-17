# Mihomo WebUI 控制面板 - 需求文档 v3

> 创建日期: 2026-06-17
> 更新日期: 2026-06-17
> 状态: 需求确认（最终版）
> 内核: mihomo (Clash Meta) 二进制
> 部署方式: Docker（单容器）

---

## 一、项目概述

基于 mihomo 内核的代理管理 WebUI 控制面板。单 Docker 容器部署，mihomo 以二进制方式运行（方便版本切换），Sub-Store 作为独立 Node.js 服务提供订阅引擎，Go 后端作为 PID 1 管理所有进程。

### 1.1 核心决策记录

| 决策项 | 结论 | 备注 |
|---|---|---|
| 后端语言 | **Go** | 学习曲线平缓，goroutine 适合多进程+WS |
| mihomo 集成 | **二进制** | 方便版本切换，不 import Go 包 |
| 前端框架 | **Vue 3 + Vite** | 用户熟悉 |
| UI 组件库 | **Naive UI** | Vue 3 原生，暗色模式好 |
| 容器模式 | **单容器** | WebUI + Sub-Store + mihomo 一体 |
| 鉴权方案 | **简单密码** | 单容器单实例单用户 |
| 配置存储 | **JSON/YAML 文件** | 数据量小，无数据库 |
| 内核实例 | **单内核** | 一个 WebUI 管一个 mihomo |
| 订阅管理 | **Sub-Store 完整集成** | 独立 Node.js 服务，API 对接 |
| UI 风格 | **自行设计** | 不照搬现有 Dashboard |
| DNS 可视化 | **不做** | |
| GeoIP 管理 | **做** | 更新 + 查看版本 |
| Profile 导出 | **做** | 导出完整 Profile 包 |
| 审计日志 | **不做** | |
| 规则集管理 | **待讨论** | 见下方规则集讨论 |

### 1.2 参考项目

| 项目 | 语言 | 复用方式 |
|---|---|---|
| **mihomo** | Go | 二进制运行在容器内 |
| **Clash Verge Rev** | Rust+React | 功能设计与交互参考 |
| **Sub-Store** | Node.js | 容器内独立服务，订阅引擎 |
| **subconverter** | C++ | 参考，不直接集成 |

---

## 二、Mihomo 内核 API 能力

| 端点 | 方法 | 功能 |
|---|---|---|
| `GET /` | GET | 探活 |
| `GET /version` | GET | 版本信息 |
| `GET /traffic` | GET/WS | 实时上下行流量 |
| `GET /memory` | GET/WS | 内存使用 |
| `GET /logs` | GET/WS | 日志流（level 过滤） |
| `GET /configs` | GET | 当前运行时配置 |
| `PATCH /configs` | PATCH | 增量更新（mode/ipv6/allow-lan/tun 等） |
| `PUT /configs` | PUT | 全量重载（path 或 yaml payload） |
| `POST /configs/geo` | POST | 更新 GeoIP/GeoSite 数据库 |
| `GET /proxies` | GET | 所有代理节点 |
| `GET /proxies/{name}` | GET | 单节点详情 |
| `PUT /proxies/{name}` | PUT | 切换 Selector 节点 |
| `GET /proxies/{name}/delay` | GET | 单节点测速 |
| `DELETE /proxies/{name}` | DELETE | 取消固定 |
| `GET /group` | GET | 代理组列表 |
| `GET /group/{name}` | GET | 代理组详情 |
| `GET /group/{name}/delay` | GET | 整组测速 |
| `GET /rules` | GET | 规则列表（含命中统计） |
| `PATCH /rules/disable` | PATCH | 运行时启用/禁用规则 |
| `GET /connections` | GET/WS | 活跃连接 |
| `DELETE /connections` | DELETE | 关闭全部连接 |
| `DELETE /connections/{id}` | DELETE | 关闭指定连接 |
| `GET /providers/proxies` | GET | 代理订阅提供者 |
| `PUT /providers/proxies/{name}` | PUT | 更新代理提供者 |
| `GET /providers/proxies/{name}/healthcheck` | GET | 代理提供者健康检查 |
| `GET /providers/rules` | GET | 规则订阅提供者 |
| `PUT /providers/rules/{name}` | PUT | 更新规则提供者 |
| `GET /dns` | GET | DNS 配置 |
| `GET /cache` | GET | 缓存信息 |
| `POST /restart` | POST | 重启内核 |

---

## 三、系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Docker Container                        │
│                                                              │
│  ┌─────────────┐                                            │
│  │   Nginx     │  :80                                       │
│  │   Vue SPA   │  静态资源 + /api/* 反代 → :3000             │
│  └──────┬──────┘                                            │
│         │                                                    │
│  ┌──────▼────────────────────┐    ┌───────────────────────┐ │
│  │ WebUI Backend (Go)        │    │  Sub-Store (Node.js)  │ │
│  │ PID 1                     │    │  :3001                │ │
│  │ :3000                     │───►│  订阅引擎服务          │ │
│  │                           │    └───────────────────────┘ │
│  │ ┌──────────────────────┐  │                              │
│  │ │ 进程管理              │  │    ┌───────────────────────┐ │
│  │ │ API 路由 + 鉴权       │  │    │  mihomo (二进制)      │ │
│  │ │ WS 代理/转发          │  │───►│  API :9090            │ │
│  │ │ 配置合并引擎          │  │    │  Mixed :7890          │ │
│  │ └──────────────────────┘  │    │  TUN (optional)       │ │
│  └───────────────────────────┘    └───────────────────────┘ │
│                                                              │
│  /data volume                                                │
│  ├── webui/settings.json          # 全局设置                  │
│  ├── webui/profiles.json          # Profile 注册表            │
│  ├── webui/profiles/{id}.*        # 各 Profile 数据           │
│  ├── sub-store/                   # Sub-Store 数据            │
│  ├── mihomo/config.yaml           # 当前内核配置              │
│  └── mihomo/bin/mihomo            # 内核二进制                │
└─────────────────────────────────────────────────────────────┘
```

### 3.1 WebUI 后端职责

| 模块 | 职责 |
|---|---|
| 进程管理（PID 1） | 启动/停止/监控 mihomo 和 Sub-Store 子进程 |
| HTTP API | `/api/auth`, `/api/profiles`, `/api/settings`, `/api/geo`, `/api/kernel/*` |
| 内核 API 代理 | 透传 mihomo API，部分增强 |
| WebSocket 代理 | 转发 mihomo WS（traffic/connections/logs/memory） |
| 配置合并 | 订阅 yaml + 全局覆盖 + 全局规则 → 最终配置 |
| 鉴权 | 简单密码，JWT token |

### 3.2 Sub-Store 集成

Sub-Store 作为独立 Node.js 服务运行在 :3001，WebUI 后端通过 HTTP 调用其 API：

```
用户操作 → WebUI 后端 → Sub-Store API
                          ├── POST /api/subs         创建订阅
                          ├── GET  /api/subs         列出订阅
                          ├── PUT  /api/subs/{id}    更新订阅
                          ├── DELETE /api/subs/{id}  删除订阅
                          ├── GET  /download/{name}?target=clashmeta  获取转换后的 mihomo yaml
                          ├── POST /api/collections  创建集合（多订阅合并）
                          └── POST /api/sync         触发同步
```

---

## 四、功能需求

### F01 - 订阅配置管理（via Sub-Store）

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F01.1 | URL 订阅导入 | P0 | 各协议 URI（vmess/vless/trojan/ss/ssr/hysteria2 等） |
| F01.2 | 协议自动识别转换 | P0 | Sub-Store parsers 自动处理 |
| F01.3 | 本地文件导入 | P0 | 上传 .yaml/.yml 配置文件 |
| F01.4 | 多订阅管理 | P0 | CRUD，启用/禁用/排序/重命名 |
| F01.5 | 订阅自动更新 | P1 | 可配置间隔，手动触发，显示更新时间 |
| F01.6 | 订阅健康检查 | P1 | 拉取可达性检查 |
| F01.7 | 节点过滤/重命名/排序 | P1 | Sub-Store processors |
| F01.8 | 多订阅合并（集合） | P1 | Sub-Store collection |

---

### F02 - 配置管理与切换

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F02.1 | Profile 管理 | P0 | Profile = 订阅源 + 全局规则 + 全局覆盖 |
| F02.2 | Profile 切换 | P0 | 合并配置 → `PUT /configs` 重载 |
| F02.3 | 全局规则 UI 编辑器 | P0 | 可视化 rules 编辑（拖拽/类型选择/策略选择） |
| F02.4 | 全局配置覆盖编辑器 | P0 | Monaco Editor，YAML 编辑 override/merge |
| F02.5 | 配置预览 | P1 | 预览最终合并的完整 yaml |
| F02.6 | 配置校验 | P0 | YAML 语法 + mihomo 配置项校验 |
| F02.7 | Profile 导出 | P1 | 导出 Profile 包（含订阅开关，见下方说明） |
| F02.8 | Profile 导入 | P1 | 根据导出开关和备份内容决定是否导入订阅 |

**配置合并管道：**
```
订阅原始配置 (Sub-Store clashmeta yaml)
    ↓ merge（浅合并）
全局覆盖 (用户 YAML)
    ↓ prepend/append rules
全局规则 (UI 编辑器)
    ↓
最终配置 → PUT /configs (mihomo)
```

**Profile 导出/导入设计：**

导出包结构（zip）：
```
profile-export.zip
├── manifest.json              # 导出元数据 + 开关状态
├── platform/
│   ├── profile.json           # Profile 元数据（名称、描述等）
│   ├── rules.yaml             # 全局规则
│   ├── override.yaml          # 全局覆盖
│   └── settings.json          # 关联的平台设置片段
└── subscriptions/             # 仅当开关打开时包含
    ├── sub-1.json             # Sub-Store 订阅定义
    ├── sub-2.json
    └── collections.json       # 集合定义
```

manifest.json 示例：
```json
{
  "version": "1.0",
  "exportTime": "2026-06-17T10:00:00Z",
  "includeSubscriptions": true,
  "profiles": ["profile-1"],
  "subscriptionCount": 3
}
```

**导出行为：**

| 开关状态 | 导出内容 |
|---|---|
| 关闭（默认） | 仅 `platform/` 目录（全局规则 + 覆盖 + Profile 元数据） |
| 打开 | `platform/` + `subscriptions/`（Sub-Store 订阅定义和集合） |

开关本身作为 Profile 导出设置的一部分，存储在 Profile 元数据中。

**导入决策逻辑：**

```
导入 Profile 包
    ↓
读取 manifest.json
    ↓
┌─ 备份中包含订阅数据？
│   ├── 否 → 仅导入平台配置（无论如何）
│   └── 是 ─┐
│            ↓
│   ┌─ manifest.includeSubscriptions == true？
│   │   ├── 否 → 仅导入平台配置（导出时未勾选，虽有数据但不导入）
│   │   └── 是 ─┐
│   │            ↓
│   │   ┌─ 用户确认导入订阅？（UI 提示）
│   │   │   ├── 否 → 仅导入平台配置
│   │   │   └── 是 → 导入平台配置 + 订阅配置
│   │   └───────┘
│   └───────────┘
└───────────────┘
    ↓
订阅导入方式：
  - 通过 Sub-Store API 创建订阅记录（URL/名称/设置）
  - 不导入缓存的节点数据（导入后自动拉取最新）
```

> **关键原则**：开关是导出时的意图标记，导入时尊重这个意图，但最终由用户确认。即使开关关闭，如果备份中有订阅数据，导入时仍可提示用户选择。

---

### F03 - 规则查看与管理

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F03.1 | 规则列表展示 | P0 | index/type/payload/proxy/hitCount |
| F03.2 | 规则来源标注 | P1 | 全局规则 vs 订阅配置规则 |
| F03.3 | 规则搜索/过滤 | P1 | 按类型/关键字/策略组 |
| F03.4 | 规则运行时禁用 | P1 | `PATCH /rules/disable` |
| F03.5 | 命中统计 | P2 | 排序/热度标识 |

---

### F04 - 连接监控

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F04.1 | 连接列表实时展示 | P0 | WS `GET /connections` |
| F04.2 | 关闭单个连接 | P0 | `DELETE /connections/{id}` |
| F04.3 | 关闭全部连接 | P0 | `DELETE /connections` |
| F04.4 | 连接搜索/过滤 | P1 | 按 host/规则/代理链 |
| F04.5 | 流量排序 | P1 | 按上传/下载/总流量 |
| F04.6 | 暂停自动刷新 | P1 | 暂停 WS 更新 |

---

### F05 - 内核日志

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F05.1 | 日志实时流 | P0 | WS `GET /logs` |
| F05.2 | 级别过滤 | P0 | error/warning/info/debug/silent |
| F05.3 | 日志搜索 | P1 | 全文搜索 |
| F05.4 | 日志导出 | P2 | 导出为文本 |

---

### F06 - 节点管理与测速

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F06.1 | 代理组列表 | P0 | Selector/URLTest/Fallback/LoadBalance |
| F06.2 | 组内节点列表 | P0 | 含选中状态 |
| F06.3 | 切换节点 | P0 | `PUT /proxies/{name}` |
| F06.4 | 单节点测速 | P0 | `GET /proxies/{name}/delay` |
| F06.5 | 整组测速 | P0 | `GET /group/{name}/delay` |
| F06.6 | 测速可视化 | P1 | 延迟颜色编码（绿/黄/红） |
| F06.7 | 节点搜索 | P1 | 按名称/类型 |
| F06.8 | 节点详情 | P1 | 类型/服务器/端口/TLS |

---

### F07 - 运行模式切换

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F07.1 | 模式切换 | P0 | `PATCH /configs` → rule/global/direct |
| F07.2 | 状态显示 | P0 | 当前模式高亮 |

---

### F08 - 链式代理

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F08.1 | 链式代理配置 | P0 | proxy-groups 嵌套 / proxy-chains |
| F08.2 | 代理链可视化 | P1 | 图形化链路展示 |
| F08.3 | 预设模板 | P2 | 常见场景模板 |

---

### F09 - 局域网连接模式

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F09.1 | allow-lan 开关 | P0 | `PATCH /configs` |
| F09.2 | 绑定地址 | P1 | bind-address |
| F09.3 | LAN IP 黑白名单 | P2 | lan-allowed-ips / lan-disallowed-ips |

---

### F10 - IPv6 开关

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F10.1 | IPv6 开关 | P0 | `PATCH /configs` → ipv6 |
| F10.2 | 状态显示 | P0 | 当前状态 |

---

### F11 - TUN 模式与端口管理

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F11.1 | 监听端口管理 | P1 | HTTP/SOCKS/Mixed/Redir/TProxy 显示与修改 |
| F11.2 | TUN 模式开关 | P0 | `PATCH /configs` → tun.enable |
| F11.3 | TUN 高级配置 | P1 | stack/auto-route/MTU |

> Docker 下 TUN 需 `--cap-add=NET_ADMIN --device /dev/net/tun`。

---

### F12 - 首页仪表盘

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F12.1 | 订阅信息卡片 | P0 | Profile/来源/更新时间/节点数 |
| F12.2 | 节点选择快捷入口 | P0 | 当前节点/快速切换 |
| F12.3 | 网络设置摘要 | P0 | 端口/allow-lan/TUN/IPv6 |
| F12.4 | 代理模式切换 | P0 | Rule/Global/Direct |
| F12.5 | 流量统计 | P0 | 实时速率 + 累计总量 |
| F12.6 | 网站连通性测试 | P1 | Google/YouTube/GitHub 等 |
| F12.7 | 内核状态 | P0 | 状态/版本/内存/运行时长 |
| F12.8 | 快速操作 | P1 | 一键关连接/一键测速 |

---

### F13 - 内核启停控制

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F13.1 | 启动内核 | P0 | 拉起 mihomo 子进程 |
| F13.2 | 停止内核 | P0 | SIGTERM |
| F13.3 | 重启内核 | P0 | `POST /restart` 或进程重启 |
| F13.4 | 存活监控 | P0 | 心跳检测，异常告警 |
| F13.5 | 启动日志 | P1 | 实时展示 |

---

### F14 - GeoIP/GeoSite 数据库管理

| ID | 需求 | P | 说明 |
|---|---|---|---|
| F14.1 | 查看当前版本 | P0 | 显示 GeoIP/GeoSite 数据库版本和更新时间 |
| F14.2 | 手动更新 | P0 | `POST /configs/geo` 触发更新 |
| F14.3 | 自动更新配置 | P2 | 可配置定期更新间隔 |

---

## 五、非功能需求

| ID | 需求 | 说明 |
|---|---|---|
| NF01 | Docker 单容器 | 多进程统一管理 |
| NF02 | 响应式 UI | 桌面 + 移动端 |
| NF03 | 密码鉴权 | 单用户，首次启动设置 |
| NF04 | 文件持久化 | JSON/YAML 到 volume |
| NF05 | 国际化 | 中/英 |
| NF06 | 暗色/亮色主题 | |
| NF07 | WebSocket 实时通信 | traffic/connections/logs/memory |
| NF08 | 错误处理 | 友好提示 |
| NF09 | mihomo 版本切换 | 支持替换二进制或环境变量指定版本 |

---

## 六、页面路由

```
/                       首页仪表盘
/subscriptions          订阅管理
/profiles               Profile 管理
/profiles/:id/rules     全局规则编辑
/profiles/:id/override  全局配置覆盖
/rules                  规则查看
/proxies                节点管理
/connections            连接监控
/logs                   内核日志
/settings               系统设置
  /settings/general     通用（模式/IPv6/GeoIP）
  /settings/network     网络（LAN/端口/TUN）
  /settings/chains      链式代理
```

---

## 七、里程碑

### M1 - 基础骨架（2 周）
- [ ] Vue 3 + Vite + Naive UI 前端脚手架
- [ ] Go 后端脚手架（Chi router + 进程管理框架）
- [ ] Dockerfile（Nginx + Go binary + Sub-Store + mihomo binary）
- [ ] 多进程管理（PID 1 启动/停止/监控子进程）
- [ ] 密码鉴权
- [ ] 内核 API 连通 + WS 代理
- [ ] 首页仪表盘（内核状态/流量/模式切换）

### M2 - 订阅与节点（2-3 周）
- [ ] Sub-Store 集成（容器内启动 + API 对接）
- [ ] 订阅导入/管理（F01）
- [ ] Profile 管理与切换（F02）
- [ ] 节点查看/切换/测速（F06）
- [ ] 运行模式切换（F07）

### M3 - 监控与规则（1-2 周）
- [ ] 连接监控（F04）
- [ ] 内核日志（F05）
- [ ] 规则查看（F03）
- [ ] GeoIP 管理（F14）

### M4 - 高级配置（2-3 周）
- [ ] 全局规则 UI 编辑器（F02.3）
- [ ] 全局配置覆盖编辑器（F02.4）
- [ ] 链式代理（F08）
- [ ] 网络设置（F09-F11）
- [ ] Profile 导出（F02.7）

### M5 - 打磨（持续）
- [ ] 网站连通性测试
- [ ] 国际化
- [ ] 暗色模式
- [ ] 移动端适配
- [ ] 规则集管理（如确认需要）
