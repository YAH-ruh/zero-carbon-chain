# 零碳微证 Go 后端镜像（Railway / 任意 Docker 平台部署）
# 多阶段构建：golang 编译 → alpine 运行（SQLite 驱动为纯 Go 实现，无需 CGO）
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server .

FROM alpine:3.20
WORKDIR /app
# ca-certificates：DeepSeek API 走 HTTPS 需要
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/server ./server
# Railway 会注入 PORT 环境变量；config.go 已支持 PORT 回退，此处无需固定
EXPOSE 8080
ENTRYPOINT ["./server"]
