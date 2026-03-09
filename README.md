---
title: Astrolabe Natal Chart
emoji: ✨
colorFrom: blue
colorTo: indigo
sdk: docker
app_port: 7860
---

# Astrolabe

一个基于 **Go 后端 + Web 前端** 的西方本命盘应用。  
用户输入出生信息后，可生成星盘可视化与结构化解读（人格、关系、爱情、事业、金钱、成长等）。

## 在线预览

- 本地运行：`http://localhost:8080`
- 健康检查：`/healthz`

## 效果预览

> 可在这里放截图/GIF（建议文件放在 `docs/assets/`）
>
> - 首页表单（宇宙风 UI）
> - 星盘绘制（宫位、行星、流光相位）
> - 解读面板（爱情/金钱/成长等）

## 项目亮点

- 极简宇宙风 UI（动态星空背景）
- 星盘可视化（宫位、行星、ASC/MC、相位流光连线）
- 中国省份下拉输入（支持中文）
- 出生时间缺失时自动进入近似模式
- 解读模块化输出：
  - 人格底色
  - 关系模式
  - 爱情解析
  - 事业路径
  - 金钱主题
  - 成长课题 / 行动建议 / 关键相位

## 技术栈

- 后端：Go（标准库 `net/http`）
- 前端：HTML + CSS + Vanilla JavaScript
- 计算：内置星体/宫位/相位计算逻辑（MVP）

## 快速开始

### 1) 安装与启动

```bash
go run ./cmd/server/main.go
```

访问：`http://localhost:8080`

### 2) 运行测试

```bash
go test ./...
```

## API

- `GET /healthz`：健康检查
- `POST /api/v1/chart/natal`：生成本命盘

请求示例：

```json
{
  "birth_date": "1990-01-01",
  "birth_time": "08:15",
  "birth_province": "江苏省",
  "birth_country": "中国",
  "language": "zh-CN"
}
```

返回结构（简化）：

```json
{
  "meta": {},
  "chart": {},
  "reading": {
    "personality": "",
    "relationship": "",
    "love": "",
    "career": "",
    "money": "",
    "growth": "",
    "action": "",
    "focus": ""
  }
}
```

## 目录结构

- `cmd/server`：服务入口
- `internal/astrology`：占星计算与解读逻辑
- `internal/api`：HTTP 路由与接口层
- `web`：前端页面与交互
- `docs`：部署与参考文档

## Roadmap

- [x] 本命盘计算与基础解读
- [x] 宇宙风前端与动态星盘
- [x] 爱情/金钱主题解读
- [ ] 真正 Swiss Ephemeris 高精度计算接入
- [ ] 多语言切换（zh/en）
- [ ] 用户历史记录与导出 PDF
- [ ] 流年/合盘模块

## 贡献指南

欢迎 PR。建议流程：

1. Fork 本仓库
2. 新建分支：`feat/xxx` 或 `fix/xxx`
3. 提交代码并确保 `go test ./...` 通过
4. 发起 Pull Request，描述改动和验证方式

## 常见问题（FAQ）

### 1. 为什么有时提示近似模式？

当未提供 `birth_time` 时，系统会按当地 `12:00` 近似计算，并降低部分解读置信度。

### 2. 为什么省份可选但城市也会出现在请求里？

前端会做兼容映射，避免旧版后端仅识别 `birth_city` 时报错。

### 3. 这是专业建议吗？

不是。本项目用于学习与娱乐。

## 免责声明

本项目用于学习与娱乐，解读结果不构成投资、医疗、法律等专业建议。
