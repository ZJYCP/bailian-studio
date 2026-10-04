# ---------- 前端构建 ----------
FROM node:22-alpine AS webbuilder
WORKDIR /build
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# ---------- 后端构建（前端产物嵌入） ----------
FROM golang:1.26-alpine AS gobuilder
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN rm -rf cmd/server/frontend
COPY --from=webbuilder /build/dist cmd/server/frontend
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bailian-studio ./cmd/server

# ---------- 运行镜像 ----------
FROM alpine:3.20
RUN adduser -D -u 10001 app && apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=gobuilder /out/bailian-studio ./bailian-studio
USER app
ENV PORT=8080 DATA_DIR=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["./bailian-studio"]
