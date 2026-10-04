# 国内构建源（服务器在国外时可用 --build-arg 覆盖回官方源）
ARG GOPROXY_MIRROR=https://goproxy.cn,direct
ARG NPM_REGISTRY=https://registry.npmmirror.com
ARG ALPINE_MIRROR=https://mirrors.aliyun.com

# ---------- 前端构建 ----------
FROM node:22-alpine AS webbuilder
ARG NPM_REGISTRY
RUN npm config set registry ${NPM_REGISTRY}
WORKDIR /build
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------- 后端构建（前端产物嵌入） ----------
FROM golang:1.26-alpine AS gobuilder
ARG GOPROXY_MIRROR
ENV GOPROXY=${GOPROXY_MIRROR}
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN rm -rf cmd/server/frontend
COPY --from=webbuilder /build/dist cmd/server/frontend
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bailian-studio ./cmd/server

# ---------- 运行镜像 ----------
FROM alpine:3.20
ARG ALPINE_MIRROR
RUN sed -i "s#https://dl-cdn.alpinelinux.org#${ALPINE_MIRROR}#g" /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=gobuilder /out/bailian-studio ./bailian-studio
USER app
ENV PORT=8080 DATA_DIR=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["./bailian-studio"]
