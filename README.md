# 随机图片 API

基于 Netlify Functions 的随机图片 API 服务，使用 Go 语言实现。

## 功能特点

- 支持多种图片类型（原神、星穹铁道）
- 支持 JSON 和重定向两种返回方式
- 基于 MongoDB 数据存储
- 支持跨域访问

## API 使用

### 图片重定向

```
GET /img?type=ysh
```

### JSON 响应

```
GET /img?type=ysh&json=true
```

### 支持的类型参数

- ysh: 原神横屏
- yss: 原神竖屏
- xqh: 星穹横屏
- xqs: 星穹竖屏

## 开发部署

1. 安装依赖：
```bash
go mod download
```

2. 本地构建：
```bash
go build -o functions/rangeImage ./functions/rangeImage.go
```

3. 部署到 Netlify：
```bash
git push
```