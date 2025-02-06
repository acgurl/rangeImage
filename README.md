# 随机图片 API

基于 Netlify Functions 的随机图片 API 服务，使用 Go 语言实现。

## 功能特点

- 支持多种图片类型（原神、星穹铁道、必应壁纸）
- 支持图片样式转换（webp、jpg、png、缩略图）
- 支持 JSON 和重定向两种返回方式
- 内置多级缓存机制
- 自动请求限流保护
- 完整的健康检查
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

## 请求参数说明

### style参数支持情况

style参数（webp、jpg、png、thumbnail）仅支持以下类型：
- ysh (原神横屏)
- yss (原神竖屏)
- xqh (星穹横屏)
- xqs (星穹竖屏)

注意：必应壁纸（bing）不支持样式转换。

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

## 本地开发

1. 复制环境变量示例文件：
```bash
cp .env.example .env
```

2. 修改 .env 文件中的 MongoDB 连接信息

3. 本地测试：
```bash
go run functions/rangeImage.go
```

4. 环境要求：
- Go 1.19+
- MongoDB 4.0+
- Netlify CLI (可选)

### 健康检查