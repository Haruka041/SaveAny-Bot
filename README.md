<div align="center">

# <img src="docs/static/logo.png" width="45" align="center"> Save Any Bot

**简体中文** | [English](./README_en.md)

> **把 Telegram 上的文件转存到多种存储端。本版本新增分片上传接收端，用于绕过 Cloudflare 100MB 限制并适配 OpenList 本地存储。**

[![Release Date](https://img.shields.io/github/release-date/Haruka041/SaveAny-Bot?label=release)](https://github.com/Haruka041/SaveAny-Bot/releases)
[![tag](https://img.shields.io/github/v/tag/Haruka041/SaveAny-Bot.svg)](https://github.com/Haruka041/SaveAny-Bot/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/Haruka041/SaveAny-Bot/build-release.yml)](https://github.com/Haruka041/SaveAny-Bot/actions/workflows/build-release.yml)
[![Stars](https://img.shields.io/github/stars/Haruka041/SaveAny-Bot?style=flat)](https://github.com/Haruka041/SaveAny-Bot/stargazers)
[![Downloads](https://img.shields.io/github/downloads/Haruka041/SaveAny-Bot/total)](https://github.com/Haruka041/SaveAny-Bot/releases)
[![Issues](https://img.shields.io/github/issues/Haruka041/SaveAny-Bot)](https://github.com/Haruka041/SaveAny-Bot/issues)
[![Pull Requests](https://img.shields.io/github/issues-pr/Haruka041/SaveAny-Bot?label=pr)](https://github.com/Haruka041/SaveAny-Bot/pulls)
[![License](https://img.shields.io/github/license/Haruka041/SaveAny-Bot)](./LICENSE)

</div>

## 概述

SaveAny-Bot 是一个 Telegram 机器人，可将 Telegram 与网站的媒体内容转存到多种存储端。本版本加入了分片接收服务，专门用于 OpenList 本地存储的稳定大文件上传，避免 Cloudflare 的 100MB 请求体限制。

## 🎯 核心特性

- 支持文档/视频/图片/贴纸…甚至还有 [Telegraph](https://telegra.ph/)
- 破解禁止保存的文件
- 批量下载与流式传输
- 多用户与存储规则自动整理
- 监听并自动转存指定聊天的消息, 支持过滤
- 在不同存储端之间转存文件
- 集成 yt-dlp, 从所支持的网站下载并转存媒体文件
- 集成 Aria2, 支持直链/磁力下载和转存
- 使用 JS 编写解析器插件以转存任意网站的文件
- 存储端支持:
  - Alist
  - S3
  - WebDAV
  - 本地磁盘
  - Telegram (重传回指定聊天)

## 本版本新增功能（分片上传）

- WebDAV 存储新增 `receiver_url`，将上传转发到分片接收端。
- 断点续传（服务端校验 offset）。
- staging → final 原子移动，完成后才对外可见。
- 上传清单与日志记录（便于审计/排查）。
- 自动清理过期的 staging 文件。

## 架构与流程

1. Bot 端把文件分片上传到接收端（`/upload_chunk`）。
2. 接收端写入 staging 并记录进度。
3. Bot 端调用 `/complete`，接收端将文件移动到 OpenList 本地目录。
4. OpenList 读取本地存储目录，文件可见。

## 部署

### 1) 接收端（Docker）

```bash
cd file-receiver
docker compose up -d --build
```

### 2) 接收端（systemd）

```bash
sudo cp file-receiver.service /etc/systemd/system/file-receiver.service
sudo systemctl daemon-reload
sudo systemctl enable --now file-receiver
```

### 3) Bot 端（示例）

```bash
go run ./cmd
```

## 配置说明

### 全局配置（节选）

- `stream`: true 为流式下载；false 会缓存到本地文件（更利于断点续传）。
- `workers`: 并发下载任务数。
- `threads`: 单任务最大线程数。

### WebDAV / OpenList 存储（关键）

```toml
[[storages]]
name = "OpenList"
type = "webdav"
enable = true
base_path = "/"
receiver_url = "http://<receiver-host>:8080"
chunk_size_mb = 10
chunk_retries = 3

# 如果仍需 WebDAV 的列目录/读取功能可保留:
# url = "https://example.com/dav"
# username = "username"
# password = "password"
```

注意：
- `base_path` 建议设为 `/`，避免最终目录重复（例如 `/Tele/Tele`）。
- `receiver_url` 启用后，上传会走接收端；不填则回退为 WebDAV 直传。

### 断点续传行为

- **可 seek 的读源（stream=false）**：可从已上传 offset 继续。
- **不可 seek（stream=true）**：中断后会重置 staging 并重新上传。

## 完整 config.toml 示例（已脱敏）

```toml
# 全局配置
stream = true
workers = 3
threads = 4

[telegram]
token = "YOUR_BOT_TOKEN"
app_id = 123456
app_hash = "YOUR_APP_HASH"

[telegram.userbot]
enable = true
session = "data/usersession.db"

[[storages]]
name = "OpenList"
type = "webdav"
enable = true
base_path = "/"
receiver_url = "http://<receiver-host>:8080"
chunk_size_mb = 10
chunk_retries = 3

# 如果仍需 WebDAV 的列目录/读取功能可保留:
# url = "https://example.com/dav"
# username = "username"
# password = "password"

[[users]]
id = 1234567890
storages = ["OpenList"]
blacklist = false
```

## 接收端环境变量

- `FINAL_DIR`: 最终目录（OpenList 本地存储路径）
- `STAGING_DIR`: 分片临时目录
- `MANIFEST_DIR`: 上传清单目录
- `LOG_PATH`: 上传日志
- `STAGING_TTL_HOURS`: 过期清理时间（小时）

## 接收端接口

- `POST /upload_chunk`: 上传分片（参数：`file`, `filename`, `upload_id`, `offset`）
- `POST /complete`: 合并完成（参数：`filename`, `upload_id`）
- `GET /status`: 查询上传进度（参数：`upload_id`）
- `POST /reset`: 清理某个 upload_id 的 staging
- `GET /healthz`: 健康检查

## 常见问题

- **文件路径重复（如 Tele/Tele）**：检查 `base_path` 是否为 `/`。
- **上传卡住/失败**：确认接收端可访问，避免经 Cloudflare 代理上传大文件。
- **OpenList 未显示**：刷新目录或检查 `FINAL_DIR` 是否指向正确路径。

## 安全建议

- 接收端默认不做鉴权，建议只在内网使用。
- 若需要公网访问，请在反代层加鉴权/限速/白名单。

## 鸣谢

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- [tdl](https://github.com/iyear/tdl)
- All the dependencies, contributors and users.
