package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"bailian-studio/server/internal/api"
	"bailian-studio/server/internal/config"
	"bailian-studio/server/internal/database"
	"bailian-studio/server/internal/service"
)

//go:embed all:frontend
var embedded embed.FS

// frontendFS 返回以站点根（index.html 所在目录）为根的 FS；未构建前端时返回 nil（开发模式由 Vite 提供）
func frontendFS() fs.FS {
	sub, err := fs.Sub(embedded, "frontend")
	if err != nil {
		return nil
	}
	f, err := sub.Open("index.html")
	if err != nil {
		return nil
	}
	f.Close()
	return sub
}

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	db := database.MustOpen(cfg.DatabaseURL)
	svc := service.New(db, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartPoller(ctx)

	var frontend fs.FS
	if f := frontendFS(); f != nil {
		frontend = f
		log.Println("已加载内嵌前端资源")
	} else {
		log.Println("未检测到内嵌前端（开发模式），API Only")
	}

	r := api.NewRouter(cfg, svc, frontend)
	log.Printf("百炼创作平台启动: http://127.0.0.1:%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
