FROM golang:1.20-alpine AS builder

# 安装git和构建工具
RUN apk add --no-cache git build-base

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o /aimmit ./cmd/main.go

# 使用官方llama.cpp镜像
FROM ghcr.io/ggml-org/llama.cpp:light

# 安装git和运行时依赖
RUN apt-get update && apt-get install -y git bash ca-certificates && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 创建模型目录
RUN mkdir -p /app/model

# 从builder镜像复制编译好的应用程序
COPY --from=builder /aimmit /usr/local/bin/aimmit

# 设置环境变量
ENV LLAMA_C_PATH="/app"
ENV REPO="/git-repo"

# 配置Git安全目录
RUN git config --global --add safe.directory /git-repo

# 设置入口点
ENTRYPOINT ["aimmit"]