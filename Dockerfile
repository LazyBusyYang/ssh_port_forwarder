# 默认不设置 HTTP(S) 代理；需要时通过 --build-arg 传入（见 docs/DEPLOYMENT.md）。
# GOPROXY 默认为官方模块代理；勿与「本机 Clash HTTP 代理」混淆。

# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY \
    HTTPS_PROXY=$HTTPS_PROXY \
    ALL_PROXY=$ALL_PROXY \
    NO_PROXY=$NO_PROXY
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY \
    HTTPS_PROXY=$HTTPS_PROXY \
    ALL_PROXY=$ALL_PROXY \
    NO_PROXY=$NO_PROXY
# 官方 Go 模块与校验和；仅在你明确传入 --build-arg GOPROXY=... 时才会覆盖默认值
ARG GOPROXY=https://proxy.golang.org,direct
ARG GOSUMDB=sum.golang.org
ENV GOPROXY=$GOPROXY \
    GOSUMDB=$GOSUMDB
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o spf-server ./cmd/server/

# Stage 3: Final image
FROM alpine:3.19
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG NO_PROXY
ENV HTTP_PROXY=$HTTP_PROXY \
    HTTPS_PROXY=$HTTPS_PROXY \
    ALL_PROXY=$ALL_PROXY \
    NO_PROXY=$NO_PROXY
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/spf-server .
COPY config/config.yaml.example ./config/config.yaml
RUN mkdir -p data

EXPOSE 8080
ENTRYPOINT ["./spf-server"]
CMD ["-config", "config/config.yaml"]
