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
- TCP Relay 中转，UDP 不稳定或被云厂商拦截时仍可承载 SSH/RDP。
- 支持 Linux、macOS、Windows 构建包，Linux 桌面端支持 AppImage。
- 已实测本机与 ARM64 OpenWrt/FriendlyWrt 设备通过 Linkbit 虚拟 IP 通信。

## 界面截图

### 桌面客户端

![Linkbit 桌面客户端](docs/screenshots/desktop-client.svg)

### Web 管理后台

![Linkbit Web 管理后台](docs/screenshots/web-console.svg)

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
- `linkbit-agent`：设备加入、WireGuard 隧道管理、TCP Relay 转发、健康上报。
- `desktop/`：Electron 可视化客户端。
- `web/`：React + TypeScript 管理后台。

## 当前实测状态

已验证内容：

- 控制器健康检查。
- Web 管理后台可用。
- 中继服务健康检查。
- API 烟测、压力测试、中继恢复测试。
- WireGuard Hub 路由安装。
- 本机到 ARM64 目标设备通过 Linkbit TCP Relay 通信。
- SSH 通过 Linkbit 云端中转可用。
- 底层 UDP 正常时，WireGuard 虚拟 IP 可作为加速路径使用。

本次环境实测：

```text
本机 -> 云服务器 Hub -> ARM64 目标设备
TCP Relay SSH 建连：约 0.29s 到 0.84s
目标设备 -> 云服务器 ICMP：约 7 ms
```

远程桌面需要目标设备本身运行 RDP、VNC、NoMachine 或 RustDesk 等桌面服务。Linkbit 负责提供安全 TCP 中转路径；桌面协议服务需要安装在目标系统上。

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

### 4. 登录管理后台

打开控制器地址，例如：

```text
http://120.79.155.227/
```

页面顶部的 `Admin API Key` 不是邀请码。它是管理员密钥，用来读取设备、中继、策略和系统设置。生产部署时它来自控制器环境变量 `LINKBIT_BOOTSTRAP_API_KEY`，或由管理员在 API Key 页面创建。

当前测试部署中，管理员密钥保存在本机的忽略文件中：

```bash
cat .tools/remote-bootstrap-key
```

把输出粘贴到 `Admin API Key`，点击 `连接`。连接成功后才会显示真实设备数量、中继节点和策略。

### 5. 邀请码和加入设备

邀请码只用于新设备第一次加入 Linkbit 网络。已经注册过的设备不会再使用原来的邀请码；它们依靠本机状态文件里的设备 ID 和 device token 继续连接。

在管理后台的 `设备邀请` 区域点击 `生成`，会得到一次性邀请码和等价命令。然后在新设备上运行：


```bash
sudo ./linkbit-agent \
  --controller https://controller.example.com \
  --enrollment-key <token> \
  --name laptop \
  --interface linkbit0
```

也可以使用 Linkbit 桌面客户端，直接填写控制器地址和邀请码启动连接。

当前 FriendlyWrt 已经注册为设备 `friendlywrt`，虚拟 IP 是 `10.88.92.200`，所以它没有“现在要用的邀请码”。要访问它，直接使用桌面客户端的中转功能：

```text
本地监听：127.0.0.1:10022
远端目标：friendlywrt:22
```

## 桌面客户端怎么用

### 连接设备

1. 打开 Linkbit 桌面客户端。
2. 填写 `控制器地址`，例如 `https://controller.example.com` 或 `http://192.0.2.10`。
3. 粘贴 Web 管理后台生成的邀请码。
4. 设置设备名称，例如 `workstation` 或 `friendlywrt`。
5. `WireGuard 接口` 默认保持 `linkbit0`。
6. 点击 `启动连接`。

Linux 上创建 WireGuard 接口需要管理员权限。如果已经安装了系统 Agent，普通桌面客户端不需要再点 `启动连接`，直接使用下面的 SSH/RDP 中转即可。

日志里出现 `device registered`、`loaded device state` 或 `tcp relay target enabled` 后，设备就可以使用了。

### 打开管理后台

点击右上角 `打开管理后台`。如果控制器地址没有写 `http://` 或 `https://`，客户端会自动按 `http://...` 打开。

### 用 Linkbit 中转 SSH

在 `SSH/RDP 中转` 区域填写：

```text
本地监听：127.0.0.1:10022
远端目标：friendlywrt:22
```

然后在本机连接：

```bash
ssh -p 10022 root@127.0.0.1
```

如果提示 `address already in use`，说明 `127.0.0.1:10022` 已经有一个中转或其他服务在监听。可以直接执行上面的 SSH 命令测试，或者把 `本地监听` 改成 `127.0.0.1:10023` 再启动。

也可以直接填虚拟 IP：

```text
远端目标：10.88.92.200:22
```

### 用 Linkbit 中转远程桌面

如果目标 Windows 或 Linux 桌面已经开启 RDP，并监听 `3389`：

```text
本地监听：127.0.0.1:13389
远端目标：desktop-device:3389
```

然后在远程桌面客户端里连接：

```text
127.0.0.1:13389
```

### 控制朋友的电脑

完整流程是：

1. 你打开管理后台，填入 `Admin API Key` 并点击 `连接`。
2. 在 `设备邀请` 区域点击 `生成`，复制邀请码或生成出来的安装命令。
3. 把 Linkbit 客户端安装包、控制器地址和邀请码发给朋友。
4. 朋友安装 Linkbit，以管理员身份启动 Agent，填写：

```text
控制器地址：http://120.79.155.227
邀请码：管理后台生成的 token
设备名称：friend-pc
```

5. 朋友电脑出现在管理后台后，你在自己电脑的 Linkbit 桌面端填写中转：

```text
本地监听：127.0.0.1:13389
远端目标：friend-pc:3389
```

6. 你打开远程桌面客户端，连接：

```text
127.0.0.1:13389
```

注意：Linkbit 负责把你的电脑和朋友电脑通过云服务器中转连起来；朋友电脑本身还需要开启远程桌面服务。Windows 建议开启 RDP，Linux 可以安装 `xrdp`、VNC 或 NoMachine。如果朋友电脑没有桌面服务，只安装 Linkbit 还不能直接看到屏幕。

### CLI 等价命令

桌面客户端的中转功能等价于：

```bash
linkbit-agent forward \
  --controller https://controller.example.com \
  --state ~/.config/linkbit/agent-state.json \
  --listen 127.0.0.1:10022 \
  --target friendlywrt:22
```

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
- 更细的 Relay 可观测性，以及 UDP 退化时的 WireGuard 自动恢复。
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
