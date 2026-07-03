<div align="center">

# UTLIVED

**自动录制直播 · 弹幕压制 · 一键投稿 B 站**

自动监测开播，开播即录，录完自动烧录弹幕并投稿到 B 站，像一台永不缺席的直播录像机。

[![Release](https://img.shields.io/github/v/release/YGR1996/utlived-app?label=下载&color=00A1D6)](https://github.com/YGR1996/utlived-app/releases/latest)
[![Platform](https://img.shields.io/badge/平台-Windows%2010%2F11-0078D6)](https://github.com/YGR1996/utlived-app/releases/latest)
[![License](https://img.shields.io/badge/解析层-MIT-green)](LICENSE)

</div>

> 📸 _界面截图待补：把一张主界面截图放到 `assets/screenshot.png` 即可在此显示。_

---

## ✨ 功能亮点

- 🎯 **自动值守录制** —— 监测主播开播状态，开播自动开录、下播自动收尾，无需盯守
- 📡 **多平台录制 + 弹幕采集** —— 一份 URL 自动识别平台，同步抓取弹幕
- 💬 **弹幕压制进画面** —— 弹幕烧录为 ASS 字幕，成品自带弹幕，防遮挡智能排轨
- 🚀 **录完自动投稿 B 站** —— 支持多 P / 合集追加，断点续传
- 🔄 **断流自动重连 + 时间轴修复** —— 网络抖动自动重连，异常时间轴自动修复，避免成品音画错位
- ⏰ **定时录制 / 日切 / 磁盘保护** —— 按时段录制，超长自动分段，磁盘不足自动保护
- 🖥️ **轻量桌面应用** —— 单文件安装，内置 ffmpeg，开箱即用

## 📥 下载

**[⬇️ 下载最新版（Windows 10/11）](https://github.com/YGR1996/utlived-app/releases/latest)**

首次运行会自动安装 WebView2 运行时（若系统缺失）。绿色便携版与安装版均可在 [Releases](https://github.com/YGR1996/utlived-app/releases) 获取。

## 💰 版本与定价

| 版本 | 能力 | 价格 |
|---|---|---|
| **免费版** | 同时录制 **1 个**任务，全部录制/弹幕/压制功能 | ¥0 |
| **Pro 月付** | 多任务同时录制、自动投稿等全部功能 | ¥39 / 月 |
| **Pro 季付** | 同上 | ¥98 / 季 |
| **Pro 年付** | 同上 | ¥298 / 年 |
| **Pro 终身** | 同上，一次买断 | ¥899 |

> 先免费试用，满意再升级。定价以应用内 / 官网为准。

## 🚀 30 秒上手

1. 下载安装并打开 UTLIVED
2. 粘贴直播间地址（如 `https://live.bilibili.com/12345`）
3. 剩下的交给它 —— 自动识别平台、监测开播、开录、压制、投稿

## 🎬 支持平台

**录制 + 弹幕**：B 站、抖音、快手、虎牙、斗鱼、小红书、YY 等国内主站，以及 Twitch、TikTok、SOOP、CHZZK 等海外平台。

**自动投稿**：哔哩哔哩（B 站）。

---

## 🔧 开源组件：直播流地址解析器

本仓库开源了 UTLIVED 的**直播流地址解析层**（MIT 许可）——输入直播间 URL，输出真实播放流地址、房间信息与画质。欢迎直接使用或参与贡献。

> 完整客户端支持的平台更多；此处开源的是其中 **B 站、斗鱼** 两个解析器，作为独立可用的库 / CLI。

### 命令行

```bash
# 安装
go install github.com/YGR1996/utlived-app/cmd/livestream@latest

# 解析一个直播间
livestream https://live.bilibili.com/12345
livestream -q HD -json https://www.douyu.com/9999
```

输出示例：

```
platform : bilibili
anchor   : 某主播
title    : 直播标题
live     : true
quality  : OD
stream   : https://...flv
```

### 作为库调用

```go
import "github.com/YGR1996/utlived-app/parser"

p, _ := parser.Match("https://live.bilibili.com/12345")
info, _ := p.GetRoomInfo(ctx, url, "")
if info.IsLive {
    s, _ := p.GetStreamURL(ctx, url, "OD", "")
    fmt.Println(s.RecordURL)
}
```

画质参数：`OD`（原画）/ `BD`（蓝光）/ `UHD` / `HD` / `SD` / `LD`。

## ❓ FAQ

**Q：免费版能一直用吗？**
能。免费版可无限期使用，仅限同时录制 1 个任务。

**Q：录像存在哪？**
安装版默认存到系统「视频」文件夹下的 UTLIVED 目录；便携版存在程序目录旁。可在设置里改。

**Q：投稿到 B 站需要什么？**
在应用内用 B 站 App 扫码授权即可，凭证仅保存在本地。

**Q：有问题 / 想提需求？**
欢迎在 [Issues](https://github.com/YGR1996/utlived-app/issues) 反馈。

## 📄 许可

- **本仓库内的解析层源码**：[MIT](LICENSE)，可自由使用。
- **UTLIVED 桌面应用**（从 Releases 下载的安装包 / 可执行文件）：专有软件，受 [EULA](EULA.md) 约束，**不开源**，请勿逆向或再分发。
