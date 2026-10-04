# 百炼创作平台（bailian-studio）

本地自用的阿里云百炼（DashScope）AIGC 创作与管理平台。通过 Web 页面调用和管理百炼的 **图像生成/编辑、视频生成、语音合成（TTS，含声音复刻）** 能力，全部参数按百炼官方 API 支持。

## 功能

- **创作中心**：图像（文生图/图片编辑）、视频（文生/图生/参考生）、语音（TTS）三大能力，模型与参数表单由模型目录的 JSON Schema 驱动，不同模型展示不同参数（尺寸/负向词/seed/水印/分辨率/比例/时长/音色/语速/音调/情感指令/SSML 等）
- **任务列表**：状态筛选、参数回显、结果预览（图廊/视频播放/音频播放）、重试/取消/删除
- **资产库**：生成结果画廊，产物全部下载到本地磁盘（百炼结果 URL 24 小时过期），支持预览/下载/删除
- **模型管理**：内置 33 个常用模型（qwen-image 系/wan3.0/wan2.x/cosyvoice/qwen-tts 等），可启用/禁用/设默认；支持新增自定义模型
- **设置**：多套服务配置（apiUrl + API Key）切换、连通性测试；声音复刻（上传音频样本创建专属音色）

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | React 18 + TypeScript + Vite + Tailwind v4 + shadcn/ui + TanStack Query |
| 后端 | Go (Gin + GORM)，单二进制，前端资源可 embed 进二进制 |
| 数据库 | PostgreSQL 16（docker compose 启动） |

## 快速开始

### 前置要求

- Go ≥ 1.24、Node ≥ 20、Docker
- 一个阿里云百炼 API Key（[获取](https://bailian.console.aliyun.com/?apiKey=1)）

### 启动

```bash
# 1. 启动 PostgreSQL（端口 15432，数据落在 docker volume）
docker compose up -d

# 2. 开发模式（两个终端）
cd server && DATA_DIR=../data go run ./cmd/server   # 后端 http://127.0.0.1:8080
cd web && npm install && npm run dev                 # 前端 http://localhost:5173（代理 /api）

# 或用 Makefile
make db-up && make server   # + 另一终端 make web
```

首次启动会自动建表并写入模型种子。打开 http://localhost:5173 → 「设置」→ 新增服务配置：

- **API URL**：北京 `https://dashscope.aliyuncs.com`（新加坡 `https://dashscope-intl.aliyuncs.com`）
- **API Key**：`sk-...`

点「测试」验证连通后即可开始创作。

### 生产（单二进制）

```bash
make build   # 前端构建 → 拷贝到 server/cmd/server/frontend → go embed → bin/bailian-studio
DATA_DIR=./data ./bin/bailian-studio   # http://127.0.0.1:8080
```

### 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | 后端端口 |
| `DATABASE_URL` | `postgres://bailian:bailian123@localhost:15432/bailian_studio?sslmode=disable` | PG 连接串 |
| `DATA_DIR` | `../data` | 数据目录（uploads/ + assets/ 产物落盘） |
| `ALLOWED_ORIGIN` | `http://localhost:5173` | 开发模式 CORS |
| `ACCESS_TOKEN` | （空） | **线上部署必设**：访问口令，保护全部业务 API（登录后种 Cookie，媒体文件也可加载） |

## 线上部署

> ⚠️ 部署前必读：平台默认无登录。**只要端口对公网/内网开放，就必须设置 `ACCESS_TOKEN`**，否则任何知道地址的人都能消耗你的百炼额度。API Key 明文存于数据库，请勿让 PostgreSQL 端口对公网暴露（生产 compose 已默认只走内部网络）。

### 方式一：Docker Compose（推荐）

在服务器上（需安装 Docker）：

```bash
# 1. 同步代码到服务器
rsync -av --exclude data --exclude bin --exclude web/node_modules \
      ./bailian-studio/ user@your-server:/opt/bailian-studio/

# 2. 在服务器上配置口令与数据库密码
cd /opt/bailian-studio
cat > .env <<'EOF'
ACCESS_TOKEN=换成一个足够长的随机口令
POSTGRES_PASSWORD=换成一个强密码
APP_PORT=8080
TZ=Asia/Shanghai
EOF

# 3. 构建并启动（前端+后端在镜像内完成构建）
docker compose -f docker-compose.prod.yml up -d --build

# 4. 验证
curl http://127.0.0.1:8080/api/health
```

浏览器打开 `http://服务器IP:8080`，输入访问口令解锁，再到「设置」里配置百炼 apiUrl + API Key。

说明：

- **国内构建源已内置**：镜像构建走 goproxy.cn（Go 模块）、npmmirror（npm）、阿里云镜像（alpine apk），国内服务器直接 `--build` 不会卡在依赖下载；服务器在海外时可用 `--build-arg GOPROXY_MIRROR=https://proxy.golang.org,direct --build-arg NPM_REGISTRY=https://registry.npmjs.org --build-arg ALPINE_MIRROR=https://dl-cdn.alpinelinux.org` 切回官方源
- 镜像为多阶段构建（node 构建前端 → go 构建后端并 embed → alpine 运行），约 53MB；服务器是 x86 时在 Mac（arm64）上需 `docker buildx build --platform linux/amd64`
- 数据落两个 docker volume：`pgdata-prod`（库）与 `appdata`（上传素材/生成产物），`docker compose down` 不会丢；备份即备份这两个 volume
- 防火墙/安全组只需放行 `APP_PORT`；PostgreSQL 不对宿主机暴露端口

### 方式二：单二进制 + systemd

```bash
# 本机交叉编译（含前端）
make build && cd server && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-s -w" -o ../../bin/bailian-studio-linux ./cmd/server

# 传到服务器
scp bin/bailian-studio-linux user@your-server:/opt/bailian-studio/bailian-studio
```

服务器上用 docker 只跑 PG（`docker compose up -d postgres`，绑定 127.0.0.1），然后 systemd：

```ini
# /etc/systemd/system/bailian-studio.service
[Unit]
Description=Bailian Studio
After=network.target

[Service]
WorkingDirectory=/opt/bailian-studio
Environment=PORT=8080
Environment=DATA_DIR=/opt/bailian-studio/data
Environment=DATABASE_URL=postgres://bailian:你的PG密码@127.0.0.1:15432/bailian_studio?sslmode=disable
Environment=ACCESS_TOKEN=你的访问口令
Environment=GIN_MODE=release
ExecStart=/opt/bailian-studio/bailian-studio
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload && sudo systemctl enable --now bailian-studio
```

### HTTPS（可选）

有域名的话，前面挂一层 Caddy 最省事（自动签发证书）：

```
# Caddyfile
ai.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

没域名也可以直接用 `http://IP:8080`——口令 Cookie 未强制 Secure，HTTP 下功能不受影响，但流量明文，介意的话建议上 HTTPS。

### 部署检查清单

- [ ] `ACCESS_TOKEN` 已设置为强随机口令
- [ ] `POSTGRES_PASSWORD` 已改（且与 DATABASE_URL 一致）
- [ ] 安全组只放行应用端口，5432/15432 未对公网开放
- [ ] `docker compose ps` 两个容器健康，`/api/health` 返回 ok
- [ ] 浏览器访问 → 口令解锁 → 设置页配置 API Key → 跑一次文生图验证

## 架构说明

```
POST /api/tasks ──► 创建任务(queued) ──► goroutine dispatch ──┬─► 同步协议（qwen-image 系/TTS）
                                                              │     成功 → 下载产物 → succeeded
                                                              └─► 异步协议（wan 视频/wanx 图像）
                                                                    拿 task_id → running
                                                                    │
                                       Poller（5s 轮询 GET /api/v1/tasks/{id}）
                                                                    │
                                                       SUCCEEDED → 下载产物落盘 → succeeded
                                                       FAILED/CANCELED → failed（记录 code/message）
```

- **六种协议**：`image_sync`（multimodal-generation 同步，qwen-image 系/wan2.6+/z-image）、`image_async_legacy`（text2image/image-synthesis，wanx 系）、`image_edit_async`（image2image/image-synthesis，wanx-imageedit）、`video_media`（video-synthesis + media[]，wan3.0/wan2.7-i2v,r2v）、`video_classic`（img_url/reference_urls，wan2.6 系）、`tts_http` / `tts_qwen`
- **输入文件**：图片 ≤10MB 直接 base64 内联；视频/音频走百炼临时 OSS 上传（`oss://`，48h 有效，调用带 `X-DashScope-OssResourceResolve: enable`）
- **产物保存**：任务完成后立即下载到 `data/assets/YYYYMM/`（URL 24h 过期前落盘），资产库由本地文件服务（支持 Range，视频可拖动播放）
- **模型目录**：`model_defs` 表，`param_schema` JSONB 驱动前端动态表单与后端参数组装；新增模型只需按协议选型 + 配 schema

## 目录结构

```
bailian-studio/
├── docker-compose.yml      # PostgreSQL 16
├── Makefile                # db-up / server / web / build
├── data/                   # 运行数据（uploads/ 输入素材, assets/ 生成产物）
├── server/                 # Go 后端
│   ├── cmd/server/         # 入口（go:embed 前端）
│   └── internal/
│       ├── api/            # REST handlers（Gin）
│       ├── config/         # 环境变量配置
│       ├── dashscope/      # 百炼 API 客户端（六协议 + 任务轮询 + OSS 上传 + 声音复刻）
│       ├── database/       # GORM 迁移 + 种子
│       ├── model/          # 实体：Provider / ModelDef / Task / Asset
│       ├── seed/           # 内置模型目录（参数 schema）
│       └── service/        # 任务分发 / 轮询器 / 下载器 / 声音复刻
└── web/                    # React 前端（Vite + shadcn）
    └── src/
        ├── components/     # SchemaField 动态表单 / FilePicker / MediaPreview 等
        └── pages/          # Studio / Tasks / Assets / Models / Settings
```

## 注意事项

- **API Key 明文存储**于本地 PostgreSQL（个人工具定位）；请勿把 `data/`、数据库暴露到公网
- cosyvoice-v3.5 系模型仅支持复刻/设计音色（系统音色会报 418），系统音色请用 v3 系
- 视频生成通常 1-5 分钟，异步任务由后台轮询器跟踪，关掉页面不影响
- 本地文件上传走百炼临时存储（48h 有效、100 QPS 限流），不适合生产高并发

## 已验证的链路（真实 API 冒烟）

| 链路 | 模型 | 结果 |
| --- | --- | --- |
| 文生图（同步） | qwen-image-3.0-pro | ✅ 1328×1328 PNG 落盘 |
| 图片编辑（base64 输入） | qwen-image-3.0-pro | ✅ 换装编辑成功 |
| 文生视频（异步 + 轮询 + Range 播放） | wan2.6-t2v | ✅ 720P 8.7MB MP4 |
| 语音合成 | cosyvoice-v3-flash | ✅ mp3 + 字符计量 |
