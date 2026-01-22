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

## 🎯 特性

- 支持文档/视频/图片/贴纸…甚至还有 [Telegraph](https://telegra.ph/)
- 破解禁止保存的文件
- 批量下载
- 流式传输
- 多用户使用
- 基于存储规则的自动整理
- 监听并自动转存指定聊天的消息, 支持过滤
- 在不同存储端之间转存文件
- 集成 yt-dlp, 从所支持的网站下载并转存媒体文件
- 集成 Aria2, 支持直链/磁力下载和转存
- 使用 js 编写解析器插件以转存任意网站的文件
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
- 上传清单与日志记录。
- 自动清理过期的 staging 文件。

## 分片上传流程

1. Bot 端把文件分片上传到接收端（`/upload_chunk`）。
2. 接收端写入 staging 并记录进度。
3. Bot 端调用 `/complete`，接收端将文件移动到 OpenList 本地目录。

## 快速开始（分片上传）

### 1) 启动接收端（Docker）

```bash
cd file-receiver
docker compose up -d --build
```

### 2) 配置存储

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

### 3) 启动 Bot

```bash
go run ./cmd
```

## 接收端环境变量

- `FINAL_DIR`: 最终目录（OpenList 本地存储路径）
- `STAGING_DIR`: 分片临时目录
- `MANIFEST_DIR`: 上传清单目录
- `LOG_PATH`: 上传日志
- `STAGING_TTL_HOURS`: 过期清理时间（小时）

## 鸣谢

- [gotd](https://github.com/gotd/td)
- [TG-FileStreamBot](https://github.com/EverythingSuckz/TG-FileStreamBot)
- [gotgproto](https://github.com/celestix/gotgproto)
- [tdl](https://github.com/iyear/tdl)
- All the dependencies, contributors and users.
