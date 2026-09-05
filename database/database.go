// Package database 数据库初始化和连接管理
// 使用SQLite作为演示数据库(单文件、零配置)，可无缝切换MySQL：
// 替换驱动为 mysql.Open(DSN) 并配置连接池即可。
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"blockchain-demo/config"
	"blockchain-demo/models"
	"blockchain-demo/pkg/logger"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 全局数据库连接
var DB *gorm.DB

// InitDB 初始化数据库(建目录 → 连接 → 迁移建表 → 预置演示账号)
func InitDB() {
	cfg := config.AppConfig

	// 确保数据库目录存在
	dbDir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("创建数据库目录失败: %v", err)
	}

	// GORM日志配置(慢查询告警写入分级日志)
	dbLogger := gormlogger.New(
		logger.GormWriter(),
		gormlogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// 连接SQLite数据库
	// 注意: 切换MySQL时替换为 gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: dbLogger})
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// SQLite 为单写者数据库：限制最大连接数为1，从根上规避并发写导致的 database is locked。
	// (切换 MySQL 后可将 MaxOpenConns 放大以提升并发吞吐)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	// 自动迁移数据表
	autoMigrate()

	// 幂等初始化预置演示账号
	initPresetUsers()

	logger.Info("数据库初始化完成: %s", cfg.DBPath)
}

// autoMigrate 自动迁移数据表结构
// 新增表说明：
//   - ReportRecord  AI 报告/建议记录表：企业减排建议与园区报告持久化，供溯源核验；
//   - OperationLog 关键操作审计表：记录操作人/时间/业务单号/区块哈希，支撑"审计留痕"。
//
// 变更说明(v2)：BlockRecord 新增 merkle_root 字段(简化 Merkle 根)。
func autoMigrate() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.EnergyRecord{},
		&models.CarbonCredit{},
		&models.SellOrder{},
		&models.Transaction{},
		&models.BlockRecord{},
		&models.ReportRecord{},
		&models.OperationLog{},
	)
	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	logger.Info("数据表迁移完成")
}

// presetUser 预置演示账号描述
type presetUser struct {
	Username string
	Password string
	Role     string
	Company  string
	ParkID   uint
}

// presetUsers 系统预置的 4 个高权限演示角色账号
// 说明：仅这 4 个账号为系统预置(数据库初始化生成)，网页注册仅允许新增小微企业账号；
// 高权限角色(园区管理员/碳交易所/监管核查)不会通过注册接口产生，杜绝越权。
var presetUsers = []presetUser{
	{"小微企业001", "123456", models.RoleEnterprise, "小微企业001", 1},
	{"园区管理员001", "123456", models.RoleParkAdmin, "园区管理员001", 1},
	{"碳交易所001", "123456", models.RoleExchange, "碳交易所001", 0},
	{"监管核查001", "123456", models.RoleRegulator, "监管核查001", 0},
}

// initPresetUsers 幂等创建预置账号
// 变更说明(v2)：由"整表为空才初始化"改为"按用户名逐个幂等补齐"，
// 解决因整表存在任意用户(如网页注册)导致预置账号永不创建或缺漏的脏态；
// 服务反复重启不会产生重复账号。
func initPresetUsers() {
	for _, u := range presetUsers {
		var count int64
		if err := DB.Model(&models.User{}).Where("username = ?", u.Username).Count(&count).Error; err != nil {
			logger.Error("检查预置账号失败(%s): %v", u.Username, err)
			continue
		}
		if count > 0 {
			continue // 已存在则跳过，保证幂等
		}

		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error("加密密码失败(%s): %v", u.Username, err)
			continue
		}
		user := models.User{
			Username: u.Username,
			Password: string(hashedPwd),
			Role:     u.Role,
			Company:  u.Company,
			ParkID:   u.ParkID,
			Status:   1,
		}
		if err := DB.Create(&user).Error; err != nil {
			logger.Error("创建预置账号失败(%s): %v", u.Username, err)
			continue
		}
		fmt.Printf("✔ 预置账号已创建: %s / %s (角色: %s)\n", u.Username, u.Password, u.Role)
		logger.Info("预置账号创建成功: %s role=%s", u.Username, u.Role)
	}
}
