<p align="center">
  <img src="assets/logo.svg" width="132" alt="Linkbit 标志" />
</p>

<h1 align="center">Linkbit</h1>

<p align="center">
  自托管、安全、低延时的私有网络与远程访问平台，面向 SSH、远程桌面、服务共享和多设备协同。
</p>

<p align="center">
  <a href="README.md">English</a>
  ·
  <a href="docs/deployment.md">部署文档</a>
  ·
  <a href="docs/openapi.yaml">OpenAPI</a>
  ·
  <a href="https://github.com/RobotXTeam/Linkbit/releases">下载发布包</a>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-React-3178C6?logo=typescript&logoColor=white">
  <img alt="WireGuard" src="https://img.shields.io/badge/WireGuard-数据平面-88171A?logo=wireguard&logoColor=white">
  <img alt="Self Hosted" src="https://img.shields.io/badge/自托管-ready-0F766E">
  <img alt="License" src="https://img.shields.io/badge/License-TBD-lightgrey">
</p>

---

## Linkbit 是什么？

Linkbit 是一套可自托管的 VPN 与应用集成平台，目标是让用户用自己的公网服务器，把多台设备安全、低延时地连接到同一个私有网络里。

它包含公网控制器、云服务器 Hub、中继服务、客户端 Agent、Web 管理后台和可视化桌面客户端。设备通过邀请码加入网络，自动获得虚拟 IP，在无法直连或网络环境复杂时，通过你的云服务器进行中转。

## 核心能力

- 通过一次性邀请码安全加入设备。
- 控制器统一管理用户、设备组、设备、策略、中继和 API Key。
- WireGuard 数据平面，默认虚拟网段 `10.88.0.0/16`。
- 云服务器 Hub 中转，适合 NAT、家庭宽带、路由器、内网设备场景。
- Linux Agent 自动生成 WireGuard 密钥并持久化设备状态。
- 可视化桌面客户端，不再只能依赖 CLI。
- 支持 Linux、macOS、Windows 构建包，Linux 桌面端支持 AppImage。
- 已实测本机与 ARM64 OpenWrt/FriendlyWrt 设备通过 Linkbit 虚拟 IP 通信。

## 架构

```text
┌──────────────────────┐
│  Linkbit Controller  │
│  API · Web · SQLite  │
│  WireGuard Hub       │
└──────────┬───────────┘
           │ 云服务器中转路径
    ┌──────┴──────┐
    │             │
┌───▼───┐     ┌───▼───┐
│设备 A │     │设备 B │
│10.88.x│     │10.88.x│
└───────┘     └───────┘
```

组件说明：

- `linkbit-controller`：控制平面、认证、设备注册、网络策略、Web 管理后台、WireGuard Hub。
- `linkbit-relay`：DERP 风格中继服务，支持注册和心跳。
- `linkbit-agent`：设备加入、WireGuard 隧道管理、健康上报。
- `desktop/`：Electron 可视化客户端。
- `web/`：React + TypeScript 管理后台。

## 当前实测状态

已验证内容：

- 控制器健康检查。
- Web 管理后台可用。
- 中继服务健康检查。
- API 烟测、压力测试、中继恢复测试。
- WireGuard Hub 路由安装。
- 本机到 ARM64 目标设备通过 Linkbit 虚拟 IP 通信。
- SSH 通过 Linkbit 虚拟 IP 可用。

本次环境实测：

```text
本机 -> 云服务器 Hub -> ARM64 目标设备
ICMP 平均延时：约 30 ms
SSH：可通过 Linkbit 虚拟 IP 登录
```

远程桌面需要目标设备本身运行 RDP、VNC、NoMachine 或 RustDesk 等桌面服务。Linkbit 负责提供安全私有网络路径；桌面协议服务需要安装在目标系统上。

## 快速开始

### 1. 测试

```bash
make test
make smoke
make stress
make recovery-smoke
```

### 2. 打包

```bash
LINKBIT_VERSION=v0.2.0 ./scripts/package-release.sh
```

产物位置：

```text
artifacts/release/
```

### 3. 部署控制器

复制配置模板：

```bash
cp deploy/controller.env.example /etc/linkbit/controller.env
```

关键生产配置：

```env
LINKBIT_LISTEN_ADDR=:80
LINKBIT_PUBLIC_URL=https://controller.example.com
LINKBIT_API_KEY_PEPPER=replace-with-random-secret
LINKBIT_BOOTSTRAP_API_KEY=replace-with-admin-key
LINKBIT_HUB_WG_ENABLED=true
LINKBIT_HUB_WG_INTERFACE=linkbit-hub
LINKBIT_HUB_WG_IP=10.88.0.1
LINKBIT_HUB_WG_NETWORK=10.88.0.0/16
LINKBIT_HUB_WG_PORT=443
LINKBIT_HUB_WG_PRIVATE_KEY=replace-with-wireguard-private-key
LINKBIT_HUB_WG_ENDPOINT=controller.example.com:443
```

安装服务：

```bash
./deploy/install-controller.sh
./deploy/install-relay.sh
```

### 4. 加入设备

在 Web 管理后台生成邀请码后运行：

```bash
sudo ./linkbit-agent \
  --controller https://controller.example.com \
  --enrollment-key <token> \
  --name laptop \
  --interface linkbit0
```

也可以使用 Linkbit 桌面客户端，直接填写控制器地址和邀请码启动连接。

## 仓库结构

```text
cmd/                    Go 可执行入口
internal/controller/    控制器 API 与 WireGuard Hub
internal/agent/         Agent、WireGuard、状态、健康上报
internal/relay/         中继服务
internal/store/         SQLite 存储
web/                    Web 管理后台
desktop/                Electron 桌面客户端
deploy/                 systemd 安装脚本和配置模板
scripts/                构建、测试、打包和部署脚本
docs/                   架构、API、打包、部署文档
assets/                 品牌资源
```

## 安全设计

- API Key 和邀请码使用安全随机数生成。
- Token 以 HMAC-SHA256 摘要形式存储，不保存明文。
- 设备 Token 只允许访问设备级 API。
- 控制器会校验 WireGuard endpoint，避免非法配置下发到客户端。
- 生产环境建议使用 HTTPS，并妥善保护 Admin API Key。
- 运行时密钥通过环境文件加载，不提交到 Git。

## 路线图

- Windows MSI 和 macOS DMG 图形安装包签名。
- 桌面客户端系统托盘状态图标。
- UDP 被云厂商拦截时的 TCP fallback 中继。
- RustDesk 打包与 Linkbit 身份集成。
- 多租户 RBAC、审计日志。
- PostgreSQL 与高可用控制器。

## 商业定位

Linkbit 面向需要私有化部署的个人、团队和企业：

- 不把 SSH/RDP 暴露到公网。
- 用自己的云服务器掌控中转路径。
- 快速添加设备并统一管理。
- 控制平面可审计、可部署、可迁移。

## License

许可证尚未最终确定。公开商业分发前请补充正式 License。
