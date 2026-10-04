package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"bailian-studio/server/internal/model"
	"bailian-studio/server/internal/seed"
)

func Open(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	// 等待 PG 就绪（docker compose 刚启动时需要几秒）
	for i := 0; i < 30; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	if err := db.AutoMigrate(
		&model.Provider{},
		&model.ModelDef{},
		&model.Task{},
		&model.Asset{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	if err := seed.Seed(db); err != nil {
		return nil, fmt.Errorf("写入模型种子失败: %w", err)
	}
	return db, nil
}

func MustOpen(dsn string) *gorm.DB {
	db, err := Open(dsn)
	if err != nil {
		log.Fatalf("%v", err)
	}
	return db
}
