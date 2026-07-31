package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig 封装数据库连接参数
type DBConfig struct {
	DSN             string
	LogLevel        string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// Connect 建立并配置 MySQL 数据库连接池（兼容原签名）
func Connect(dsn string, logLevel string) (*gorm.DB, error) {
	return ConnectWithConfig(&DBConfig{
		DSN:             dsn,
		LogLevel:        logLevel,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: time.Hour,
	})
}

// ConnectWithConfig 建立并配置 MySQL 连接池（包含 Fail-Fast 探活）
func ConnectWithConfig(cfg *DBConfig) (*gorm.DB, error) {
	gormLogLevel := logger.Warn
	switch cfg.LogLevel {
	case "silent":
		gormLogLevel = logger.Silent
	case "error":
		gormLogLevel = logger.Error
	case "info":
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100
	}
	idleTime := cfg.ConnMaxIdleTime
	if idleTime <= 0 {
		idleTime = 5 * time.Minute
	}
	lifetime := cfg.ConnMaxLifetime
	if lifetime <= 0 {
		lifetime = time.Hour
	}

	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxIdleTime(idleTime)
	sqlDB.SetConnMaxLifetime(lifetime)

	// Fail-Fast 启动探活 (5s 超时)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL (DSN might be invalid or DB down): %w", err)
	}

	return db, nil
}

// AutoMigrate 自动同步表结构 (建议开发/测试环境使用)
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
