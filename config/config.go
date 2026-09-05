// Package config 加载和管理环境变量配置
//
// 变更说明(v2)：新增日志级别/日志文件、AI 请求超时等配置项，
// 用于支撑分级日志模块与 AI 接口超时容错能力。
package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config 全局配置结构体
type Config struct {
	ServerPort string // HTTP 服务监听端口
	DBPath     string // SQLite 数据库文件路径(演示库；生产可平滑切换 MySQL)

	JWTSecret      string // JWT 签名密钥
	JWTExpireHours int    // JWT 有效期(小时)

	// AI(DeepSeek) 配置
	DeepSeekAPIKey string // API Key，未配置时自动降级为基于规则的参考内容
	DeepSeekAPIURL string // 对话补全接口地址
	DeepSeekModel  string // 使用的模型名称
	AITimeoutSec   int    // 单次 AI 请求超时秒数，超时友好降级，避免卡死接口

	// 日志配置
	LogLevel string // 日志级别: debug/info/warn/error
	LogFile  string // 日志文件路径(空则仅输出控制台)

	DefaultAdmin    string // 保留字段(演示账号已由系统预置脚本管理)
	DefaultAdminPwd string
}

// AppConfig 全局配置实例
var AppConfig *Config

// AITimeout AI 请求超时时长(由 AITimeoutSec 派生)
func (c *Config) AITimeout() time.Duration {
	if c.AITimeoutSec <= 0 {
		return 25 * time.Second
	}
	return time.Duration(c.AITimeoutSec) * time.Second
}

// LoadConfig 从 .env 文件加载配置，缺失项回退到环境变量或默认值
func LoadConfig() *Config {
	// 尝试加载 .env 文件，如果不存在则使用环境变量或默认值
	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，使用环境变量或默认值")
	}

	app := &Config{
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		DBPath:          getEnv("DB_PATH", "./data/carbon_chain.db"),
		JWTSecret:       getEnv("JWT_SECRET", "weitanlian_jwt_secret_2024_change_me"),
		JWTExpireHours:  getEnvInt("JWT_EXPIRE_HOURS", 24),
		DeepSeekAPIKey:  getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekAPIURL:  getEnv("DEEPSEEK_API_URL", "https://api.deepseek.com/v1/chat/completions"),
		DeepSeekModel:   getEnv("DEEPSEEK_MODEL", "deepseek-chat"),
		AITimeoutSec:    getEnvInt("AI_TIMEOUT_SECONDS", 25),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFile:         getEnv("LOG_FILE", "./weitanlian.log"),
		DefaultAdmin:    getEnv("DEFAULT_ADMIN_USER", "admin"),
		DefaultAdminPwd: getEnv("DEFAULT_ADMIN_PASS", "admin123"),
	}

	AppConfig = app
	return app
}

// getEnv 获取环境变量，不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整型环境变量，不存在则返回默认值
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
