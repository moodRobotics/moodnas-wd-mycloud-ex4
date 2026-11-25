<p align="center">
  <img src="assets/brand/mood-robotics.png" alt="Mood Robotics" width="360">
</p>

<h1 align="center">MoodNAS</h1>

<p align="center">
  <strong>一款全新的开源固件，让 WD My Cloud EX4 重获新生。<br>
  老硬件，新安全，内置人工智能。</strong>
</p>

<p align="center">
  <a href="README.md">English</a> ·
  <a href="README.es.md">Español</a> ·
  <a href="README.pt.md">Português</a> ·
  <a href="README.fr.md">Français</a> ·
  <a href="README.de.md">Deutsch</a> ·
  <a href="README.it.md">Italiano</a> ·
  <b>简体中文</b> ·
  <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <img alt="许可证：GPL-3.0" src="https://img.shields.io/badge/%E8%AE%B8%E5%8F%AF%E8%AF%81-GPL--3.0-1f5fff">
  <img alt="状态：alpha，模拟器" src="https://img.shields.io/badge/%E7%8A%B6%E6%80%81-alpha%20·%20%E6%A8%A1%E6%8B%9F%E5%99%A8-ffb020">
  <img alt="语言：8" src="https://img.shields.io/badge/%E8%AF%AD%E8%A8%80-8-2fd27a">
</p>

---

## 为什么

**WD My Cloud EX4** 是一款可靠的 4 盘位 NAS，但 Western Digital 已不再为其提供支持。
它的最后一版固件仍运行 Linux 3.2（2013 年）、OpenSSL 1.0.1 和 PHP 5.4，
并且自带默认的 root 密码。数以千计的此类设备至今仍在日益危险的网络环境中，
保存着家庭照片和小企业的文件。

**MoodNAS** 用一套现代、开放的系统取代原有固件。它充分利用每一部分硬件，
并为这台老旧的 ARM 设备带来了一项从未有人期待过的能力：**守护你数据的人工智能**。

<p align="center">
  <img src="docs/img/dashboard.png" alt="MoodNAS 概览" width="900">
</p>

## 亮点

### 👀 有情绪的 NAS
Mood Robotics 的眼睛就是 MoodNAS 的脸：

- 一切正常时，**平静**；
- 忙碌工作时，**专注**；
- 夜间，**休息**；
- 有事需要你关注时，**担忧**；
- 数据面临风险时，**警觉**，并泛起红光。

同样的眼睛也会出现在 NAS 的**前面板 OLED 屏幕**上，即使隔着整个房间，
你也能一眼看出它的状态。

### 🧠 直接在 NAS 上运行的 AI
这些模型无需云端，也不会向家庭网络之外发送任何数据：

| | |
|---|---|
| **硬盘故障预测** | 使用基于 Backblaze 公开故障统计数据的模型，读取每块硬盘的 SMART 数据。在故障发生之前，告诉你哪块硬盘可能出问题，以及原因。 |
| **勒索软件哨兵** | 识别文件被大规模加密的迹象（熵分析加重命名模式），并**在数秒内将共享文件夹冻结为只读**。 |
| **预测式风扇** | 根据温度趋势提前应对发热，夜间保持安静，同时绝不牺牲安全。 |
| **容量预测** | 学习你存储数据的速度，在卷写满前数周发出提醒。 |

### 💬 “Mood”，你的助手
用你自己的话提问：*“我的 NAS 状态怎么样？”*、*“有哪块硬盘需要担心吗？”*、
*“创建一个所有人都能访问的共享文件夹，名为 Photos”*。Mood 可配合
**Claude**（Anthropic API）使用，也可配合**你自己网络中的 Ollama** 使用，
数据完全不出家门。

- 它只查看工具展示给它的内容，任何更改都需要**你的确认**。
- **隐私记录**列出每一个离开 NAS 的请求。

<p align="center">
  <img src="docs/img/ransomware.png" alt="勒索软件哨兵冻结了一个共享文件夹" width="900">
</p>

### 🔒 安全源自设计
- **没有默认密码。** 首次设置需要输入前面板上显示的一次性验证码。
- **默认启用 HTTPS。**
- **密码**采用 PBKDF2-SHA512 哈希，支持双因素认证（TOTP）和登录频率限制。
- **Web 防护：** CSRF 防护、严格的 Content-Security-Policy 以及审计日志。
- **单个 Go 静态二进制文件**，依赖极少。

更多详情请见 [SECURITY.md](SECURITY.md)。

### 🌍 支持你的语言
界面和文档提供**西班牙语、英语、葡萄牙语、法语、德语、意大利语、中文和日语**版本。
每位用户可以选择自己的语言。

### 💾 数据原地保留
MoodNAS **原样**读取由 WD 固件创建的 RAID 卷：无需格式化，无需迁移。

## 项目状态

> **MoodNAS 目前处于 alpha 阶段。** 完整的界面、API、本地 AI 和助手现已可以在
> **模拟的 EX4** 上运行。下一个里程碑是在真实硬件上启动，欢迎你来帮忙。

| 阶段 | | 状态 |
|---|---|---|
| 1 | 固件分析，开源 `wdfw` 镜像工具（逐字节一致的重新打包） | ✅ 已完成 |
| 2 | 硬件清点脚本（Stage 0，只读） | ✅ 已就绪，需要真实设备的报告 |
| 3 | Web 界面（8 种语言）、API、安全、模拟器 | ✅ 已完成 |
| 4 | 设备端 AI + 助手（Claude / Ollama） | ✅ 首个版本 |
| 4b | 带解释的告警、语义搜索、照片标签 | 🔜 计划中 |
| 5 | mainline 内核 + Buildroot、kexec 启动（不写入 flash）、MCU 驱动 | 🚧 进行中：设备树草稿已完成，MCU 协议已解析 |
| 6 | 永久安装 | ⏳ 需要串口控制台恢复途径 |

## 两分钟上手体验

你需要 Docker：

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4.git
cd moodnas-wd-mycloud-ex4
docker compose up --build
```

1. 打开 **http://localhost:8080**。
2. 使用演示账户 **`demo`** / **`moodnas-demo`** 登录。
3. 在概览页面试试模拟器场景：高温、硬盘故障、RAID 降级、勒索软件攻击。留意那双眼睛。

## 首次访问与密码

| 场景 | 用户名 | 密码 |
|---|---|---|
| 演示（`docker compose`，模拟器） | `demo` | `moodnas-demo` |
| 你的真实 NAS | **默认没有** | **默认没有** |

在真实 NAS 上**没有默认用户名和密码**，这是有意为之：WD 原厂固件自带一个众所周知的 root 密码，这正是 MoodNAS 要消除的弱点之一。首次启动时：

1. NAS 会在前面板显示屏（以及系统日志）上显示一个 8 位**设置码**。
2. 打开 `https://<nas-ip>` 并输入该设置码。这证明你就在 NAS 旁边，网络中的其他人无法抢先占用它。
3. 创建**你自己的**管理员账户和密码（至少 10 个字符）。之后可以在“我的账户”中开启双因素认证。

演示账户只在 `moodd -simulate -demo` 下存在。由于其密码是公开的，`moodd` 在真实硬件上会拒绝 `-demo`。

忘记密码？重置方法见 [docs/install](docs/install/README.md)（将随硬件移植一起提供）。

## 工作原理

```
浏览器（Svelte，8 种语言） ──HTTPS──▶ moodd（Go，单个静态二进制文件）
                                      ├─ API · 认证 · 审计
                                      ├─ 本地 AI：硬盘风险 · 哨兵 · 风扇 · 预测
                                      ├─ 助手 ─▶ Claude API | Ollama（局域网）
                                      └─ 平台 ─▶ 模拟器 | EX4 硬件（SATA、SMART、MCU、OLED）
```

| | |
|---|---|
| 架构 | [docs/architecture](docs/architecture/README.md) |
| 深入了解 AI | [docs/ai](docs/ai/README.md) |
| 硬件 | [docs/hardware](docs/hardware/README.md)：[固件格式](docs/hardware/firmware-format.md) · [前面板 MCU 协议](docs/hardware/mcu-protocol.md) |
| 安装计划与 Stage 0 | [docs/install](docs/install/README.md) |

## 硬件

| | WD My Cloud EX4 |
|---|---|
| SoC | Marvell Armada 300 / Kirkwood 88F6282，ARMv5TE，带硬件加密 |
| 内存 | 512 MB |
| 盘位 | 4 × 3.5" SATA |
| 网络 | 2 × 千兆以太网 |
| 前面板 | OLED 屏幕、按键和风扇，由一个微控制器管理 |

其他 WD 型号（EX2、EX4100、PR4100）暂不支持。欢迎贡献。

## 参与贡献

目前最有价值的帮助是提交一份来自真实 EX4 的
**[Stage 0 报告](docs/install/README.md#stage-0-hardware-inventory)**。
它是只读操作，同时还会备份你的 flash。开发和翻译相关内容请参阅
[CONTRIBUTING.md](CONTRIBUTING.md)。

## 法律声明

MoodNAS 是依据 [GNU GPL v3.0 或更高版本](LICENSE)发布的自由软件。它是
**Mood Robotics** 的独立项目，与 Western Digital 无任何关联，也未获得其认可或支持。
“WD”和“My Cloud”是 Western Digital Corporation 的商标，此处仅用于说明硬件兼容性。
本仓库不包含任何 Western Digital 固件或二进制文件；相关工具只处理你自行下载的文件。

更换设备固件存在风险。MoodNAS 不提供任何担保：请先备份你的数据。

<p align="center"><sub>Mood Robotics 用 ♥ 打造：老硬件值得拥有新智能。</sub></p>
