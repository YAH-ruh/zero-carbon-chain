# 零碳微证 全栈镜像（Railway / 任意 Docker 平台部署）
# 三阶段构建：前端 dist → Go 后端编译 → alpine 运行（同时托管 API 与前端静态页）
# 产物为单一域名完整站点：/api/* 走后端接口，其余路径走前端 SPA(hash路由)

# ---- 阶段1：前端构建（Vite + Vue3）----
FROM node:22-alpine AS frontend-builder
WORKDIR /fe
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ .
RUN npm run build \
	# 同源部署：apiBase 置空走相对路径 /api（免跨域，前后端同一域名）
	&& echo "window.__APP_CONFIG__={apiBase:''}" > dist/app.config.js

# ---- 阶段2：后端编译（Go + 纯Go SQLite驱动，无需 CGO）----
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

# ---- 阶段3：运行时 ----
FROM alpine:3.20
WORKDIR /app
# ca-certificates：DeepSeek API 走 HTTPS 需要
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/server ./server
# 前端静态资源：main.go 托管 ./frontend/dist 并做 SPA 回退
COPY --from=frontend-builder /fe/dist ./frontend/dist
# Railway 会注入 PORT 环境变量；config.go 已支持 PORT 回退，此处无需固定
EXPOSE 8080
ENTRYPOINT ["./server"]
