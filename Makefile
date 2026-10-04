.PHONY: db-up db-down server web dev build test clean

# 启动 PostgreSQL
db-up:
	docker compose up -d

db-down:
	docker compose down

# 开发模式运行后端（数据目录指向仓库根 data/）
server:
	cd server && DATA_DIR=../data go run ./cmd/server

# 开发模式运行前端（Vite，代理到后端 8080）
web:
	cd web && npm run dev

dev: db-up
	@echo "PostgreSQL 已启动。请分别在两个终端执行: make server / make web"

# 构建单二进制（前端静态资源嵌入 Go）
build:
	cd web && npm install && npm run build
	rm -rf server/cmd/server/frontend
	cp -r web/dist server/cmd/server/frontend
	cd server && go build -o ../bin/bailian-studio ./cmd/server
	@echo "构建完成: bin/bailian-studio"

test:
	cd server && go test ./...

clean:
	rm -rf bin
